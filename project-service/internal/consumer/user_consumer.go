package consumer

import (
	"building-services/project-service/internal/user"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type UserConsumer struct {
	userRepo user.Repository
	channel  *amqp.Channel
	conn     *amqp.Connection
}

func NewUserConsumer(userRepo user.Repository, amqpURL string) (*UserConsumer, error) {

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
		conn.Close()
		return nil, err
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
		conn.Close()
		ch.Close()
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"project_service_users",
		true, false, false, false, nil,
	)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}

	err = ch.QueueBind(q.Name, "user.*", "user.events", false, nil)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}

	return &UserConsumer{
		userRepo: userRepo,
		channel:  ch,
		conn:     conn,
	}, nil
}

func (c *UserConsumer) Start() error {
	msgs, err := c.channel.Consume(
		"project_service_users",
		"",
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to consume: %v", err)
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

	log.Printf("recieved message: %s", body)

	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		return
	}

	switch event.EventType {
	case "user.created", "user.updated":
		u := &user.User{
			ID:       event.UserID,
			FullName: event.FullName,
			Email:    event.Email,
			Role:     event.Role,
		}
		if event.DepartmentID != "" {
			deptID := event.DepartmentID
			u.DepartmentID = &deptID
		}
		if err := c.userRepo.Upsert(context.Background(), u); err != nil {
			log.Printf("Failed to upsert user: %v", err)
		} else {
			log.Printf("User %s upserted successfully", event.UserID)
		}
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
	return nil
}
