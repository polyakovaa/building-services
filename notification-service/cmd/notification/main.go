package main

import (
	"context"
	"log"
	"net"
	"time"

	notificationv1 "building-services/gen/notification/v1"
	"building-services/notification-service/config"
	"building-services/notification-service/internal/consumer"
	"building-services/notification-service/internal/handler"
	"building-services/notification-service/internal/repository"
	"building-services/notification-service/internal/scheduler"
	"building-services/notification-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	db, err := config.ConnectToDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewRepository(db)
	notificationService := service.NewService(repo)
	notificationHandler := handler.NewHandler(notificationService)

	eventConsumer, err := consumer.NewEventConsumer(repo, notificationService, cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to create notification consumer: %v", err)
	}
	defer eventConsumer.Close()
	if err := eventConsumer.Start(); err != nil {
		log.Fatalf("Failed to start notification consumer: %v", err)
	}

	deadlineScheduler := scheduler.NewDeadlineScheduler(notificationService, time.Hour)
	deadlineScheduler.Start(context.Background())

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	notificationv1.RegisterNotificationServiceServer(grpcServer, notificationHandler)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("Health server registered, status: %v", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Notification service running on :%s", cfg.Server.Port)
	go func() {
		for {
			ctx := context.Background()
			dbErr := db.PingContext(ctx)
			if dbErr != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				log.Printf("Health check for Notification service failed: %v", dbErr)
			} else {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
