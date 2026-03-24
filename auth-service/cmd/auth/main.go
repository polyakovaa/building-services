package main

import (
	"building-services/auth-service/config"
	"building-services/auth-service/internal/events"
	"building-services/auth-service/internal/handler"
	"building-services/auth-service/internal/repository"
	"building-services/auth-service/internal/service"
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	authv1 "building-services/gen/auth/v1"

	"google.golang.org/grpc"
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

	userRepo := repository.NewUserRepository(db)

	rdb, err := config.ConnectToRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}

	tokenRepo := repository.NewRedisTokenRepository(rdb)

	eventPublisher, err := events.NewEventPublisher("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer eventPublisher.Close()

	authService := service.NewAuthService(
		userRepo,
		tokenRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
		eventPublisher,
	)

	authHandler := handler.NewAuthHandler(authService)
	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	authv1.RegisterAuthServiceServer(grpcServer, authHandler)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	log.Printf("Auth service running on :%s", cfg.Server.Port)

	go func() {
		for {
			ctx := context.Background()

			dbErr := db.PingContext(ctx)
			redisErr := rdb.Ping(ctx).Err()

			if dbErr != nil || redisErr != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
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
