package scheduler

import (
	"context"
	"log"
	"time"

	"building-services/notification-service/internal/service"
)

type DeadlineScheduler struct {
	service  *service.Service
	interval time.Duration
}

func NewDeadlineScheduler(service *service.Service, interval time.Duration) *DeadlineScheduler {
	return &DeadlineScheduler{
		service:  service,
		interval: interval,
	}
}

func (s *DeadlineScheduler) Start(ctx context.Context) {
	go func() {
		s.runOnce(ctx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *DeadlineScheduler) runOnce(ctx context.Context) {
	if err := s.service.ProcessDeadlineReminders(ctx, time.Now().UTC()); err != nil {
		log.Printf("Deadline reminder scheduler failed: %v", err)
	}
}
