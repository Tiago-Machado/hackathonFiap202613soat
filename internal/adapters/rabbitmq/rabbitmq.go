package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	exchangeName    = "videos"
	routingKey      = "video.created"
	queueName       = "videos.process"
	deadLetterEx    = "videos.dlx"
	deadLetterQueue = "videos.process.dead"
)

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func Dial(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := declareTopology(channel); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Connection{conn: conn, channel: channel}, nil
}

func declareTopology(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(deadLetterEx, amqp.ExchangeFanout, true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(deadLetterQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(deadLetterQueue, "", deadLetterEx, false, nil); err != nil {
		return err
	}
	if err := channel.ExchangeDeclare(exchangeName, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	deadLetterArgs := amqp.Table{"x-dead-letter-exchange": deadLetterEx}
	if _, err := channel.QueueDeclare(queueName, true, false, false, false, deadLetterArgs); err != nil {
		return err
	}
	return channel.QueueBind(queueName, routingKey, exchangeName, false, nil)
}

func (c *Connection) Channel() *amqp.Channel {
	return c.channel
}

func (c *Connection) Close() error {
	return c.conn.Close()
}
