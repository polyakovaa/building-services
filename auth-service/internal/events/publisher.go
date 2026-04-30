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
	PublishUserCreated(ctx context.Context, userID, email, fullName, role string) error
	PublishUserUpdated(ctx context.Context, userID, email, fullName, role string) error
	Close() error
}

type EventPublisher struct {
	channel *amqp.Channel
	conn    *amqp.Connection
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
		return nil, fmt.Errorf("failed to make event publisher: %w", err)
	}

	err = ch.ExchangeDeclare(
		"user.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &EventPublisher{channel: ch, conn: conn}, nil

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

func (p *EventPublisher) publish(ctx context.Context, routingKey string, event map[string]interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Printf("published message: %s", body)

	return p.channel.PublishWithContext(ctx,
		"user.events",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

}

func (p *EventPublisher) PublishUserCreated(ctx context.Context, userID, email, fullName, role string) error {
	event := map[string]interface{}{
		"event_type": "user.created",
		"user_id":    userID,
		"email":      email,
		"full_name":  fullName,
		"role":       role,
	}
	return p.publish(ctx, "user.created", event)
}

func (p *EventPublisher) PublishUserUpdated(ctx context.Context, userID, email, fullName, role string) error {
	event := map[string]interface{}{
		"event_type": "user.updated",
		"user_id":    userID,
		"email":      email,
		"full_name":  fullName,
		"role":       role,
	}
	return p.publish(ctx, "user.updated", event)
}
