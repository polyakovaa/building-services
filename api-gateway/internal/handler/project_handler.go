package handler

import (
	"building-services/api-gateway/internal/clients"
	projectv1 "building-services/gen/project/v1"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProjectHandler struct {
	projectClient *clients.ProjectClient
}

func NewProjectHandler(client *clients.ProjectClient) *ProjectHandler {
	return &ProjectHandler{projectClient: client}
}

func (h *ProjectHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Проекты
	r.POST("/projects", h.CreateProject)
	/*
		r.GET("/projects/:id", h.GetProject)
		r.GET("/projects", h.ListProjects)
		r.PUT("/projects/:id", h.UpdateProject)
		r.DELETE("/projects/:id", h.DeleteProject)
		r.PATCH("/projects/:id/status", h.ChangeProjectStatus)

		// Участники
		r.POST("/projects/:projectId/members", h.AddMember)
		r.GET("/projects/:projectId/members", h.ListMembers)
		r.DELETE("/projects/:projectId/members/:userId", h.RemoveMember)

		// Задачи
		r.POST("/projects/:projectId/tasks", h.CreateTask)
		r.GET("/tasks/:id", h.GetTask)
		r.GET("/projects/:projectId/tasks", h.ListTasks)
		r.PUT("/tasks/:id", h.UpdateTask)
		r.DELETE("/tasks/:id", h.DeleteTask)
		r.PATCH("/tasks/:id/status", h.UpdateTaskStatus)
		r.POST("/tasks/:id/assign", h.AssignTask)

		// Вложения
		r.POST("/tasks/:taskId/attachments", h.AddAttachment)
		r.GET("/tasks/:taskId/attachments", h.ListAttachments)
		r.DELETE("/attachments/:id", h.DeleteAttachment)
	*/
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	role, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	md := metadata.New(map[string]string{
		"user_id":   userID.(string),
		"user_role": role.(string),
	})

	ctx := metadata.NewOutgoingContext(c.Request.Context(), md)

	var req struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		ObjectAddress string `json:"object_address"`
		Customer      string `json:"customer"`
		StartDate     string `json:"start_date"`
		EndDate       string `json:"end_date"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var startDate, endDate *timestamppb.Timestamp

	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format. Use RFC3339: 2024-03-01T00:00:00Z"})
			return
		}
		startDate = timestamppb.New(t)
	}

	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format. Use RFC3339: 2024-04-15T00:00:00Z"})
			return
		}
		endDate = timestamppb.New(t)
	}

	resp, err := h.projectClient.Project.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Name:          req.Name,
		Description:   req.Description,
		ObjectAddress: req.ObjectAddress,
		Customer:      req.Customer,
		StartDate:     startDate,
		EndDate:       endDate,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}
