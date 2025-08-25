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
type OrderService interface {
	ProcessNewOrder(ctx context.Context, order *models.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (*models.Order, error)
	PreloadCache(context.Context, int) error
}

type Consumer struct {
	reader  *kafka.Reader
	logger  *slog.Logger
	service OrderService
}

func NewConsumer(brokers []string, topic, groupID string, MinBytes, MaxBytes int, MaxWait time.Duration, logger *slog.Logger, service OrderService) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: MinBytes,
		MaxBytes: MaxBytes,
		MaxWait:  MaxWait,
	})

	return &Consumer{
		reader:  reader,
		logger:  logger,
		service: service,
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
		//если JSON сломанный, то retry не поможет, значит логгируем об ошибке и прокидываем nil, чтобы осуществить коммит
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
	if order.Payment.Amount < 0 {
		log.Error("invalid order data: payment amount is negative, skipping")
		return nil
	}
	if order.DateCreated.IsZero() {
		log.Error("invalid order data: order date_created is zero, skipping")
		return nil
	}
	if order.OrderUID != order.Payment.Transaction {
		log.Error("invalid order data: order_uid != payment transaction, skipping")
		return nil
	}

	log.Info("processing new order")

	if err := c.service.ProcessNewOrder(ctx, &order); err != nil {
		return fmt.Errorf("%s: failed to process order: %w", op, err)
	}

	return nil
}
