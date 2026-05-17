package main

import (
	"building-services/analytics-service/config"
	"building-services/analytics-service/internal/consumer"
	"building-services/analytics-service/internal/handler"
	"building-services/analytics-service/internal/repository"
	"building-services/analytics-service/internal/service"

	"context"
	"log"
	"net"
	"time"

	analyticsv1 "building-services/gen/analytics/v1"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
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
	service := service.NewService(repo)
	handler := handler.NewHandler(service)

	eevntConsumer, err := consumer.NewEventConsumer(repo, service, "amqp://guest:guest@rabbitmq:5672/")

	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer eevntConsumer.Close()
	eevntConsumer.Start()

	userConsumer, err := consumer.NewUserConsumer(repo, "amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to create user consumer: %v", err)
	}
	defer userConsumer.Close()
	userConsumer.Start()

	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, handler)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("Health server registered, status: %v", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Analytics service running on :%s", cfg.Server.Port)
	go func() {
		for {
			ctx := context.Background()

			dbErr := db.PingContext(ctx)

			if dbErr != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				log.Printf("Health check for Analytics service failed: %v", dbErr)
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
