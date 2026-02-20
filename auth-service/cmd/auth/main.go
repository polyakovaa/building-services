package main

import (
	"building-services/auth-service/config"
	"building-services/auth-service/internal/handler"
	"building-services/auth-service/internal/repository"
	"building-services/auth-service/internal/service"
	"log"
	"net"

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

	authService := service.NewAuthService(
		userRepo,
		tokenRepo,
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	authHandler := handler.NewAuthHandler(authService)
	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	authv1.RegisterAuthServiceServer(grpcServer, authHandler)

	log.Printf("Auth service running on :%s", cfg.Server.Port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
