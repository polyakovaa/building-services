package consumer

import (
	"building-services/analytics-service/internal/repository"
	"building-services/analytics-service/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
			log.Printf("Analytics Service connected to RabbitMQ")
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

	err = ch.ExchangeDeclare(
		"project.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}
	log.Printf("Exchange 'project.events' declared")

	q, err := ch.QueueDeclare(
		"analytics_events",
		true, false, false, false, nil,
	)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}
	log.Printf("Queue 'analytics_events' declared")

	eventTypes := []string{
		"task.*",
		"project.created",
		"project.member_added",
		"project.member_removed",
	}
	for _, routingKey := range eventTypes {
		err = ch.QueueBind(q.Name, routingKey, "project.events", false, nil)
		if err != nil {
			conn.Close()
			ch.Close()
			return nil, err
		}
		log.Printf("Queue bound to routing key: %s", routingKey)
	}

	return &EventConsumer{
		repo:    repo,
		channel: ch,
		service: svc,
		conn:    conn,
	}, nil
}

type RawEvent struct {
	ID           string
	EventType    string
	ProjectID    string
	TaskID       string
	UserID       string
	DepartmentID string
	ActorUserID  string
	OccurredAt   time.Time
	Payload      []byte
}

func (c *EventConsumer) Start() error {
	msgs, err := c.channel.Consume(
		"analytics_events",
		"",
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume: %v", err)
	}

	log.Printf("Analytics Consumer started, waiting for messages...")

	go func() {
		for msg := range msgs {
			c.handleMessage(msg.Body)
		}
	}()

	return nil
}

func (c *EventConsumer) handleMessage(body []byte) {
	log.Printf("Received message from RabbitMQ")

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		return
	}

	eventType, _ := event["event_type"].(string)
	log.Printf("Event type: %s", eventType)

	rawEvent := repository.RawEvent{
		EventType:  eventType,
		OccurredAt: time.Now(),
		Payload:    body,
	}

	if v, ok := event["project_id"].(string); ok {
		rawEvent.ProjectID = v
		log.Printf("   project_id: %s", v)
	}
	if v, ok := event["task_id"].(string); ok {
		rawEvent.TaskID = v
		log.Printf("   task_id: %s", v)
	}
	if v, ok := event["user_id"].(string); ok {
		rawEvent.UserID = v
		log.Printf("   user_id: %s", v)
	}
	if v, ok := event["department_id"].(string); ok {
		rawEvent.DepartmentID = v
		log.Printf("   department_id: %s", v)
	}
	if v, ok := event["actor_user_id"].(string); ok {
		rawEvent.ActorUserID = v
		log.Printf("actor_user_id: %s", v)
	}

	log.Printf("About to save raw event to database...")
	if err := c.repo.SaveRawEvent(rawEvent); err != nil {
		log.Printf("Failed to save raw event: %v", err)
		return
	}
	log.Printf("Raw event saved to database")

	if err := c.service.ProcessEvent(eventType, event); err != nil {
		log.Printf("Failed to process event: %v", err)
		return
	}
	log.Printf("Event processed successfully")
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
	log.Printf("Analytics Consumer closed")
	return nil
}
