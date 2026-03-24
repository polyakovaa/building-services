package main

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/config"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/consumer"
	"building-services/project-service/internal/member"
	"building-services/project-service/internal/project"
	"building-services/project-service/internal/user"

	"context"
	"log"
	"net"
	"time"

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

	projectRepo := project.NewRepository(db)
	memberRepo := member.NewRepository(db)
	userRepo := user.NewRepository(db)
	//timelineRepo := timeline.NewRepository(db)
	//taskRepo := task.NewRepository(db)
	//attachmentRepo := attachment.NewRepository(db)
	checker := authz.NewPermissionChecker(userRepo, memberRepo)

	projectService := project.NewService(projectRepo, memberRepo, userRepo, checker)
	memberService := member.NewService(projectRepo, memberRepo)
	//timelineService := timeline.NewService(timelineRepo)
	//taskService := task.NewService(taskRepo, projectRepo, memberRepo)
	//attachmentService := attachment.NewService(attachmentRepo, taskRepo)

	projectHandler := project.NewHandler(projectService)
	memberHandler := member.NewHandler(memberService)
	//timelineHandler := timeline.NewHandler(timelineService)
	//taskHandler := task.NewHandler(taskService)
	//attachmentHandler := attachment.NewHandler(attachmentService)

	userConsumer, err := consumer.NewUserConsumer(userRepo, "amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer userConsumer.Close()
	userConsumer.Start()

	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	projectv1.RegisterProjectServiceServer(grpcServer, projectHandler)
	projectv1.RegisterProjectMemberServiceServer(grpcServer, memberHandler)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("Health server registered, status: %v", healthpb.HealthCheckResponse_SERVING)

	//projectv1.RegisterProjectTimelineServiceServer(grpcServer, timelineHandler)
	//projectv1.RegisterTaskServiceServer(grpcServer, taskHandler)
	//projectv1.RegisterAttachmentServiceServer(grpcServer, attachmentHandler)

	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Project service running on :%s", cfg.Server.Port)
	go func() {
		for {
			ctx := context.Background()

			dbErr := db.PingContext(ctx)

			if dbErr != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				log.Printf("Health check for Project service failed: %v", dbErr)
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
