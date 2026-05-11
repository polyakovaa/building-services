package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, event map[string]interface{}) error
	Close() error
}

type EventPublisher struct {
	channel *amqp.Channel
	conn    *amqp.Connection
	exchange string
}

func NewEventPublisher(amqpURL string) (*EventPublisher, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ (attempt %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	const exchangeName = "project.events"
	if err := ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &EventPublisher{channel: ch, conn: conn, exchange: exchangeName}, nil
}

func (p *EventPublisher) Publish(ctx context.Context, routingKey string, event map[string]interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Printf("published project event: key=%s body=%s", routingKey, body)

	return p.channel.PublishWithContext(ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (p *EventPublisher) Close() error {
	var errs []error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

