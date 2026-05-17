package consumer

import (
	"building-services/analytics-service/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

)

type UserConsumer struct {
	repo    *repository.Repository
	channel *amqp.Channel
	conn    *amqp.Connection
}

func NewUserConsumer(repo *repository.Repository, amqpURL string) (*UserConsumer, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(amqpURL)
		if err == nil {
			log.Printf("Analytics Service connected to RabbitMQ for user events")
			break
		}
		log.Printf("Failed to connect to RabbitMQ for user events (attempt %d/10): %v", i+1, err)
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
	if err := ch.ExchangeDeclare("user.events", "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}
	q, err := ch.QueueDeclare("analytics_service_users", true, false, false, false, nil)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}
	if err := ch.QueueBind(q.Name, "user.*", "user.events", false, nil); err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}
	return &UserConsumer{repo: repo, channel: ch, conn: conn}, nil
}

func (c *UserConsumer) Start() error {
	msgs, err := c.channel.Consume("analytics_service_users", "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume user events: %v", err)
	}
	go func() {
		for msg := range msgs {
			c.handleMessage(msg.Body)
		}
	}()

	return nil
}

func (c *UserConsumer) handleMessage(body []byte) {
	var event struct {
		EventType    string `json:"event_type"`
		UserID       string `json:"user_id"`
		Email        string `json:"email"`
		FullName     string `json:"full_name"`
		Role         string `json:"role"`
		DepartmentID string `json:"department_id"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to unmarshal user event: %v", err)
		return
	}
	switch event.EventType {
	case "user.created", "user.updated":
		if event.UserID == "" {
			log.Printf("Skip user event without user_id")
			return
		}
		if err := c.repo.UpsertUser(repository.User{
			ID:           event.UserID,
			Email:        event.Email,
			FullName:     event.FullName,
			Role:         event.Role,
			DepartmentID: event.DepartmentID,
		}); err != nil {
			log.Printf("Failed to upsert analytics user %s: %v", event.UserID, err)
			return
		}
		if event.DepartmentID != "" {
			if err := c.repo.UpdateTaskDepartmentsForUser(event.UserID, event.DepartmentID); err != nil {
				log.Printf("Failed to sync task departments for user %s: %v", event.UserID, err)
			}
		}
		log.Printf("Analytics user %s upserted successfully", event.UserID)
	}
}

func (c *UserConsumer) Close() error {
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
	log.Printf("Analytics User Consumer closed")
	return nil
}


