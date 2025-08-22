package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"order-service/internal/models"
	"time"
)

// интерфейс сервисного слоя
type OrderProcessor interface {
	ProcessNewOrder(ctx context.Context, order *models.Order) error
}

type Consumer struct {
	reader    *kafka.Reader
	logger    *slog.Logger
	processor OrderProcessor
}

func NewConsumer(brokers []string, topic, groupID string, logger *slog.Logger, processor OrderProcessor) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  500 * time.Millisecond,
	})

	return &Consumer{
		reader:    reader,
		logger:    logger,
		processor: processor,
	}
}

// Start - запускает бесконечный цикл чтения сообщений из топика
func (c Consumer) Start(ctx context.Context) {
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
func (c Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	const op = "kafka.processMessage"

	var order models.Order

	if err := json.Unmarshal(msg.Value, &order); err != nil {
		//если JSON сломанный, то retry не поможет, значит логгируем, прокидываем nil, чтобы осуществить коммит
		c.logger.Error("invalid json, skipping",
			slog.Any("error", err),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
			slog.String("operation", op),
		)
		return nil
	}

	log := c.logger.With(
		slog.String("order_uid", order.OrderUID),
		slog.Int("partition", msg.Partition),
		slog.Int64("offset", msg.Offset),
		slog.String("operation", op),
	)

	//валидация данных (можно добавить больше проверок)
	if order.OrderUID == "" {
		log.Error("invalid order data: order_uid is empty, skipping")
		return nil
	}

	log.Info("processing new order")

	if err := c.processor.ProcessNewOrder(ctx, &order); err != nil {
		return fmt.Errorf("%s: failed to process order: %w", op, err)
	}

	return nil
}
