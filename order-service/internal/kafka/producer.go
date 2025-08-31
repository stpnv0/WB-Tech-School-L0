package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"time"
)

type Producer struct {
	writer  *kafka.Writer
	logger  *slog.Logger
	timeout time.Duration
}

func NewProducer(brokers []string, timeout time.Duration, log *slog.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}

	return &Producer{
		writer:  writer,
		logger:  log,
		timeout: timeout,
	}
}

// SendMessage - единый метод для отправки сообщений в любой топик.
func (p *Producer) SendMessage(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	const op = "kafka.Producer.SendMessage"

	if p.writer == nil {
		if p.logger != nil {
			p.logger.Warn("kafka writer is nil")
		}
		return fmt.Errorf("writer is nil")
	}

	kafkaHeaders := make([]kafka.Header, 0, len(headers))
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{Key: k, Value: []byte(v)})
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}

	log := p.logger.With(
		slog.String("op", op),
		slog.String("topic", topic),
		slog.String("key", string(key)),
	)

	log.Debug("Attempting to send message to Kafka")

	//применяем таймаут только если его нет и он больше 0
	ctx2 := ctx
	var cancel context.CancelFunc = func() {}
	if p.timeout > 0 {
		if _, has := ctx.Deadline(); !has {
			ctx2, cancel = context.WithTimeout(ctx, p.timeout)
		}
	}
	defer cancel()

	err := p.writer.WriteMessages(ctx2, msg)
	if err != nil {
		log.Error("Failed to write message to Kafka", slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Debug("Message sent successfully")
	return nil
}

func (p *Producer) Close() error {
	if p.writer != nil {
		p.logger.Info("Closing Kafka producer writer")
		return p.writer.Close()
	}
	return fmt.Errorf("Kafka writer is nil, can't close")
}
