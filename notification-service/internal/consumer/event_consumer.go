package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"building-services/notification-service/internal/repository"
	"building-services/notification-service/internal/service"
	"building-services/notification-service/internal/util"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventConsumer struct {
	repo    *repository.Repository
	service *service.Service
	channel *amqp.Channel
	conn    *amqp.Connection
}

func NewEventConsumer(repo *repository.Repository, svc *service.Service, amqpURL string) (*EventConsumer, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			log.Printf("Notification Service connected to RabbitMQ")
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
		conn.Close()
		return nil, err
	}

	err = ch.ExchangeDeclare("project.events","topic",true,false,false,false,nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"notification_events",
		true, false, false, false, nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	eventTypes := []string{
		"task.created",
		"task.assigned",
		"task.deadline_changed",
		"task.status_changed",
		"task.updated",
		"project.updated",
		"project.member_added",
	}
	for _, routingKey := range eventTypes {
		err = ch.QueueBind(q.Name, routingKey, "project.events", false, nil)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, err
		}
	}

	return &EventConsumer{
		repo:    repo,
		service: svc,
		channel: ch,
		conn:    conn,
	}, nil
}

func (c *EventConsumer) Start() error {
	msgs, err := c.channel.Consume("notification_events","",false,false,false,false,nil)
	if err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}

	log.Printf("Notification Consumer started, waiting for messages...")

	go func() {
		for msg := range msgs {
			c.handleMessage(msg)
		}
	}()

	return nil
}

func (c *EventConsumer) handleMessage(msg amqp.Delivery) {
	ctx := context.Background()

	var event map[string]interface{}
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal notification event: %v", err)
		msg.Nack(false, false)
		return
	}

	eventType, _ := event["event_type"].(string)
	if eventType == "" {
		log.Printf("Notification event without event_type: %s", msg.Body)
		msg.Ack(false)
		return
	}

	eventKey := util.EventKey(msg.Body)
	occurredAt := util.ParseOccurredAt(event)
	inserted, err := c.repo.SaveRawEvent(ctx, repository.RawEvent{
		EventType:  eventType,
		EventKey:   eventKey,
		OccurredAt: occurredAt,
		Payload:    msg.Body,
	})
	if err != nil {
		log.Printf("Failed to save notification raw event: %v", err)
		msg.Nack(false, true)
		return
	}
	if !inserted {
		msg.Ack(false)
		return
	}

	if err := c.service.ProcessProjectEvent(ctx, eventType, eventKey, event, msg.Body); err != nil {
		log.Printf("Failed to process notification event: %v", err)
		msg.Nack(false, true)
		return
	}
	if err := c.repo.MarkEventProcessed(ctx, eventKey); err != nil {
		log.Printf("Failed to mark notification event processed: %v", err)
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
}

func (c *EventConsumer) Close() error {
	var errs []error
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	log.Printf("Notification Consumer closed")
	return nil
}
