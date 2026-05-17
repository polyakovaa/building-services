package handler

import (
	"context"
	"errors"
	"log"
	"time"

	notificationv1 "building-services/gen/notification/v1"
	"building-services/notification-service/internal/repository"
	"building-services/notification-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Handler struct {
	notificationv1.UnimplementedNotificationServiceServer
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	start := time.Now()
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := h.service.ListNotifications(ctx, userID, req)
	if err != nil {
		log.Printf("[ERROR] ListNotifications failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] ListNotifications: user_id=%s count=%d duration=%v", userID, len(resp.Notifications), time.Since(start))
	return resp, nil
}

func (h *Handler) GetUnreadCount(ctx context.Context, req *notificationv1.GetUnreadCountRequest) (*notificationv1.UnreadCountResponse, error) {
	start := time.Now()
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := h.service.GetUnreadCount(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] GetUnreadCount failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Printf("[SUCCESS] GetUnreadCount: user_id=%s count=%d duration=%v", userID, resp.Count, time.Since(start))
	return resp, nil
}

func (h *Handler) MarkAsRead(ctx context.Context, req *notificationv1.MarkAsReadRequest) (*notificationv1.Notification, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := h.service.MarkAsRead(ctx, userID, req.GetNotificationId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return resp, nil
}

func (h *Handler) MarkAllAsRead(ctx context.Context, req *notificationv1.MarkAllAsReadRequest) (*notificationv1.UnreadCountResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := h.service.MarkAllAsRead(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func userIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get("user_id")
	if len(values) == 0 || values[0] == "" {
		return "", status.Error(codes.Unauthenticated, "missing user_id")
	}
	return values[0], nil
}
