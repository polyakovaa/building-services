package handler

import (
	"net/http"
	"strconv"

	"building-services/api-gateway/internal/util"
	notificationv1 "building-services/gen/notification/v1"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	client notificationv1.NotificationServiceClient
}

func NewNotificationHandler(client notificationv1.NotificationServiceClient) *NotificationHandler {
	return &NotificationHandler{client: client}
}

func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/notifications", h.ListNotifications)
	r.GET("/notifications/unread-count", h.GetUnreadCount)
	r.PATCH("/notifications/:id/read", h.MarkAsRead)
	r.PATCH("/notifications/read-all", h.MarkAllAsRead)
}

func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	pageSize := int32(20)
	if value := c.Query("page_size"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			pageSize = int32(parsed)
		}
	}
	resp, err := h.client.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		PageSize:   pageSize,
		PageToken:  c.Query("page_token"),
		UnreadOnly: c.Query("unread_only") == "true",
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	resp, err := h.client.GetUnreadCount(ctx, &notificationv1.GetUnreadCountRequest{})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	resp, err := h.client.MarkAsRead(ctx, &notificationv1.MarkAsReadRequest{NotificationId: c.Param("id")})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	resp, err := h.client.MarkAllAsRead(ctx, &notificationv1.MarkAllAsReadRequest{})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
