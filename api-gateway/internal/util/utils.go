package util

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	projectv1 "building-services/gen/project/v1"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

func GetGRPCContext(c *gin.Context) (context.Context, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("unauthorized"))
	}

	role, exists := c.Get("user_role")
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("unauthorized"))
	}

	md := metadata.New(map[string]string{
		"user_id":   userID.(string),
		"user_role": role.(string),
	})

	return metadata.NewOutgoingContext(c.Request.Context(), md), nil
}

func ConvertStatus(statusStr string) projectv1.ProjectStatus {
	if num, err := strconv.Atoi(statusStr); err == nil {
		switch num {
		case 1:
			return projectv1.ProjectStatus_PROJECT_STATUS_ACTIVE
		case 2:
			return projectv1.ProjectStatus_PROJECT_STATUS_COMPLETED
		case 3:
			return projectv1.ProjectStatus_PROJECT_STATUS_ON_HOLD
		case 4:
			return projectv1.ProjectStatus_PROJECT_STATUS_CANCELLED
		default:
			return projectv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED
		}
	}

	switch statusStr {
	case "active":
		return projectv1.ProjectStatus_PROJECT_STATUS_ACTIVE
	case "completed":
		return projectv1.ProjectStatus_PROJECT_STATUS_COMPLETED
	case "on_hold":
		return projectv1.ProjectStatus_PROJECT_STATUS_ON_HOLD
	case "cancelled":
		return projectv1.ProjectStatus_PROJECT_STATUS_CANCELLED
	default:
		return projectv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED
	}
}

func ConvertTaskStatus(statusStr string) projectv1.TaskStatus {
	switch statusStr {
	case "blocked":
		return projectv1.TaskStatus_TASK_STATUS_BLOCKED
	case "completed":
		return projectv1.TaskStatus_TASK_STATUS_COMPLETED
	case "in_progress":
		return projectv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case "todo":
		return projectv1.TaskStatus_TASK_STATUS_TODO
	default:
		return projectv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func ConvertPriority(priority string) projectv1.TaskPriority {
	switch priority {
	case "urgent":
		return projectv1.TaskPriority_TASK_PRIORITY_URGENT
	case "high":
		return projectv1.TaskPriority_TASK_PRIORITY_HIGH
	case "low":
		return projectv1.TaskPriority_TASK_PRIORITY_LOW
	case "medium":
		return projectv1.TaskPriority_TASK_PRIORITY_MEDIUM
	default:
		return projectv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	}
}
