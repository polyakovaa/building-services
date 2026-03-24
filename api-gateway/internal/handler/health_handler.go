package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type ServiceHealth struct {
	Name   string
	Client healthpb.HealthClient
}

type HealthHandler struct {
	services []ServiceHealth
}

func NewHealthHandler(services []ServiceHealth) *HealthHandler {
	return &HealthHandler{services: services}
}
func (h *HealthHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
}

func (h *HealthHandler) Health(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	results := make(map[string]string)
	overallStatus := http.StatusOK

	for _, service := range h.services {
		resp, err := service.Client.Check(ctx, &healthpb.HealthCheckRequest{})

		if err != nil || resp.Status != healthpb.HealthCheckResponse_SERVING {
			results[service.Name] = "unavailable"
			overallStatus = http.StatusServiceUnavailable
		} else {
			results[service.Name] = "ok"
		}
	}

	c.JSON(overallStatus, gin.H{
		"services": results,
	})
}
