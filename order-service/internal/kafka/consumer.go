package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"order-service/internal/models"
	"order-service/internal/validator"
	"strconv"
	"time"
)

// интерфейс сервисного слоя
type OrderService interface {
	ProcessNewOrder(context.Context, *models.Order) error
	GetOrderByUID(context.Context, string) (*models.Order, error)
	PreloadCache(context.Context, int) error
}

type DLQProducer interface {
	SendMessage(ctx context.Context, topic string, key, value []byte, headers map[string]string) error
}

type Consumer struct {
	reader      *kafka.Reader
	logger      *slog.Logger
	service     OrderService
	dlqProducer DLQProducer
	dlqTopic    string
}

func NewConsumer(
	brokers []string,
	topic, groupID string,
	MinBytes, MaxBytes int,
	MaxWait time.Duration,
	logger *slog.Logger,
	service OrderService,
	dlqTopic string,
	dlqProducer DLQProducer,
) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       MinBytes,
		MaxBytes:       MaxBytes,
		MaxWait:        MaxWait,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
	})

	return &Consumer{
		reader:      reader,
		logger:      logger,
		service:     service,
		dlqTopic:    dlqTopic,
		dlqProducer: dlqProducer,
	}
}

// Start - запускает бесконечный цикл чтения сообщений из топика
func (c *Consumer) Start(ctx context.Context) {
	defer c.reader.Close()
	c.logger.Info("Kafka consumer started",
		slog.String("topic", c.reader.Config().Topic),
		slog.String("group", c.reader.Config().GroupID),
		slog.Any("brokers", c.reader.Config().Brokers),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Kafka consumer stopping...")
			return
		default:
			//чтение сообщения
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				c.logger.Error("failed to fetch message",
					slog.Any("error", err),
				)
				continue
			}

			//обработка сообщения
			if err = c.processMessage(ctx, m); err != nil {
				c.logger.Error("failed to process message, will retry",
					slog.Any("error", err),
					slog.Int("partition", m.Partition),
					slog.Int64("offset", m.Offset),
				)
				continue
			}

			//коммит msg
			if err = c.reader.CommitMessages(ctx, m); err != nil {
				c.logger.Error("failed to commit message", slog.Any("error", err))
			}
		}
	}
}

// processMessage - инкапсулирует логику парсинга, валидции и передачи msg в сервис
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	const op = "kafka.processMessage"

	var order models.Order

	if err := json.Unmarshal(msg.Value, &order); err != nil {
		c.logger.Error("invalid json, skipping",
			slog.Any("error", err),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
			slog.String("operation", op),
		)

		//формируем headers для отправки сообщения в DLQ
		headers := map[string]string{
			"error_reason":    "json_unmarshal_failed",
			"error_details":   err.Error(),
			"original_topic":  msg.Topic,
			"original_offset": strconv.FormatInt(msg.Offset, 10),
		}

		if errDLQ := c.dlqProducer.SendMessage(ctx, c.dlqTopic, msg.Key, msg.Value, headers); errDLQ != nil {
			// Возвращаем ошибку, чтобы сообщение не было закоммичено
			c.logger.Error("CRITICAL: FAILED TO SEND MESSAGE TO DLQ", slog.Any("dlq_error", errDLQ))
			return fmt.Errorf("failed to send to DLQ: %w", errDLQ)
		}
		return nil
	}

	log := c.logger.With(
		slog.String("order_uid", order.OrderUID),
		slog.Int("partition", msg.Partition),
		slog.Int64("offset", msg.Offset),
		slog.String("operation", op),
	)

	//валидация данных
	if err := validator.Validate(log, &order); err != nil {
		if errors.Is(err, validator.ErrBadMessage) {
			headers := map[string]string{
				"error_reason":    "validation_failed",
				"error_details":   err.Error(),
				"original_topic":  msg.Topic,
				"original_offset": strconv.FormatInt(msg.Offset, 10),
			}

			if errDLQ := c.dlqProducer.SendMessage(ctx, c.dlqTopic, msg.Key, msg.Value, headers); errDLQ != nil {
				c.logger.Error("CRITICAL: FAILED TO SEND MESSAGE TO DLQ", slog.Any("dlq_error", errDLQ))
				return fmt.Errorf("failed to send to DLQ: %w", errDLQ)
			}
		}
		return nil
	}
	log.Debug("processing new order")

	if err := c.service.ProcessNewOrder(ctx, &order); err != nil {
		// Тут не стоит сразу отправлять в DLQ, потому что может быть временная ошибка (например, бд недоступна)
		return fmt.Errorf("%s: failed to process order: %w", op, err)
	}
	log.Debug("order processed successfully")

	return nil
}
