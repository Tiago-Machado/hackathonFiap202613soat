package rabbitmq

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"video-processor/internal/usecase"
)

const prefetchCount = 1

type VideoProcessor interface {
	Execute(ctx context.Context, videoID string) error
}

type Consumer struct {
	channel   *amqp.Channel
	processor VideoProcessor
	log       *slog.Logger
}

func NewConsumer(conn *Connection, processor VideoProcessor, log *slog.Logger) (*Consumer, error) {
	channel := conn.Channel()
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		return nil, err
	}
	return &Consumer{channel: channel, processor: processor, log: log}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handle(ctx, delivery)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, delivery amqp.Delivery) {
	var event usecase.VideoCreated
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.log.Error("mensagem_invalida", "error", err)
		_ = delivery.Nack(false, false)
		return
	}
	if err := c.processor.Execute(ctx, event.VideoID); err != nil {
		c.log.Error("processamento_falhou", "video_id", event.VideoID, "error", err)
		_ = delivery.Nack(false, false)
		return
	}
	_ = delivery.Ack(false)
}
