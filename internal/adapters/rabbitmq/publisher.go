package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"video-processor/internal/usecase"
)

type Publisher struct {
	channel *amqp.Channel
}

func NewPublisher(conn *Connection) *Publisher {
	return &Publisher{channel: conn.Channel()}
}

func (p *Publisher) Publish(ctx context.Context, event usecase.VideoCreated) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, exchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
