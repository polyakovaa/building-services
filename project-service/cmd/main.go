package main

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/config"
	"building-services/project-service/internal/activity"
	"building-services/project-service/internal/attachment"
	"building-services/project-service/internal/authz"
	"building-services/project-service/internal/consumer"
	department "building-services/project-service/internal/departmnet"
	"building-services/project-service/internal/events"
	"building-services/project-service/internal/member"
	"building-services/project-service/internal/observability/logger"
	"building-services/project-service/internal/project"
	"building-services/project-service/internal/task"
	"building-services/project-service/internal/timeline"
	"building-services/project-service/internal/user"
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	logger.Init("project-service", env, cfg.Logging.Level)

	db, err := config.ConnectToDB(cfg.Database)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	projectRepo := project.NewRepository(db)
	memberRepo := member.NewRepository(db)
	userRepo := user.NewRepository(db)
	timelineRepo := timeline.NewRepository(db)
	taskRepo := task.NewRepository(db)
	attachmentRepo := attachment.NewRepository(db)
	departmentRepo := department.NewRepository(db)
	activityRepo := activity.NewRepository(db)
	checker := authz.NewPermissionChecker(userRepo, memberRepo, taskRepo, attachmentRepo, departmentRepo)

	eventPublisher, err := events.NewEventPublisher("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		slog.Error("event publisher init failed", "error", err)
		os.Exit(1)
	}
	defer eventPublisher.Close()

	projectService := project.NewService(projectRepo, memberRepo, userRepo, timelineRepo, checker, eventPublisher)
	memberService := member.NewService(projectRepo, memberRepo, userRepo, eventPublisher)
	timelineService := timeline.NewService(timelineRepo, projectRepo)
	taskService := task.NewService(taskRepo, projectRepo, userRepo, activityRepo, checker, eventPublisher)
	attachmentService := attachment.NewService(taskRepo, attachmentRepo, checker)
	departmentService := department.NewService(departmentRepo, userRepo, checker, eventPublisher)

	projectHandler := project.NewHandler(projectService)
	memberHandler := member.NewHandler(memberService)
	timelineHandler := timeline.NewHandler(timelineService)
	taskHandler := task.NewHandler(taskService)
	attachmentHandler := attachment.NewHandler(attachmentService)
	departmentHandler := department.NewHandler(departmentService)
	activityService := activity.NewService(activityRepo, checker, eventPublisher)
	activityHandler := activity.NewHandler(activityService)

	userConsumer, err := consumer.NewUserConsumer(userRepo, "amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		slog.Error("user consumer init failed", "error", err)
		os.Exit(1)
	}
	defer userConsumer.Close()
	userConsumer.Start()

	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	projectv1.RegisterProjectServiceServer(grpcServer, projectHandler)
	projectv1.RegisterProjectMemberServiceServer(grpcServer, memberHandler)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	slog.Info("health server registered", "status", healthpb.HealthCheckResponse_SERVING.String())

	projectv1.RegisterProjectTimelineServiceServer(grpcServer, timelineHandler)
	projectv1.RegisterTaskServiceServer(grpcServer, taskHandler)
	projectv1.RegisterAttachmentServiceServer(grpcServer, attachmentHandler)
	projectv1.RegisterDepartmentServiceServer(grpcServer, departmentHandler)
	projectv1.RegisterActivityTypeServiceServer(grpcServer, activityHandler)

	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		slog.Error("grpc listen failed", "error", err)
		os.Exit(1)
	}

	slog.Info("grpc server started", "port", cfg.Server.Port)
	go func() {
		for {
			ctx := context.Background()
			dbErr := db.PingContext(ctx)
			if dbErr != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				slog.Warn("health check failed", "error", dbErr)
			} else {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("grpc serve failed", "error", err)
		os.Exit(1)
	}
}
