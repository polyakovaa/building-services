package handler

import (
	"building-services/api-gateway/internal/util"
	projectv1 "building-services/gen/project/v1"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *ProjectHandler) CreateTask(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}

	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Deadline     string `json:"deadline"`
		AssignedTo   string `json:"assigned_to"`
		ParentTaskId string `json:"parent_task_id"`
		Priority     string `json:"priority"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var deadline *timestamppb.Timestamp

	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format. Use RFC3339: 2024-03-01T00:00:00Z"})
			return
		}
		deadline = timestamppb.New(t)
	}

	resp, err := h.projectClient.Task.CreateTask(ctx, &projectv1.CreateTaskRequest{
		ProjectId:    projectID,
		Title:        req.Title,
		Description:  req.Description,
		Deadline:     deadline,
		AssignedTo:   req.AssignedTo,
		ParentTaskId: req.ParentTaskId,
		Priority:     util.ConvertPriority(req.Priority),
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ProjectHandler) GetTask(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	resp, err := h.projectClient.Task.GetTask(ctx, &projectv1.GetTaskRequest{Id: id})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) DeleteTask(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}
	resp, err := h.projectClient.Task.DeleteTask(ctx, &projectv1.DeleteTaskRequest{Id: id})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) UpdateTask(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	var req struct {
		ProjectId    string `json:"project_id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Deadline     string `json:"deadline"`
		ParentTaskId string `json:"parent_task_id"`
		Priority     string `json:"priority"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	var deadline *timestamppb.Timestamp

	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deadline format. Use RFC3339: 2024-04-15T00:00:00Z"})
			return
		}
		deadline = timestamppb.New(t)
	}

	resp, err := h.projectClient.Task.UpdateTask(ctx, &projectv1.UpdateTaskRequest{
		Id:          id,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    deadline,
		Priority:    util.ConvertPriority(req.Priority),
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) UpdateTaskStatus(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	resp, err := h.projectClient.Task.UpdateTaskStatus(ctx, &projectv1.UpdateTaskStatusRequest{
		Id:     taskID,
		Status: util.ConvertTaskStatus(req.Status),
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) ListTasks(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}

	statusFilter := c.Query("status")
	assignedToFilter := c.Query("assigned_to")
	parentTaskId := c.Query("parent_task_id")
	priorityFilter := c.Query("priority")

	resp, err := h.projectClient.Task.ListTasks(ctx, &projectv1.ListTasksRequest{
		ProjectId:        projectID,
		StatusFilter:     util.ConvertTaskStatus(statusFilter),
		AssignedToFilter: assignedToFilter,
		ParentTaskId:     parentTaskId,
		PriorityFilter:   util.ConvertPriority(priorityFilter),
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) AssignTask(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	var req struct {
		AssigneeId string `json:"assignee_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.projectClient.Task.AssignTask(ctx, &projectv1.AssignTaskRequest{
		TaskId:     taskID,
		AssigneeId: req.AssigneeId,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
func (h *ProjectHandler) ListMyTasks(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	statusFilter := c.Query("status")

	resp, err := h.projectClient.Task.ListMyTasks(ctx, &projectv1.ListMyTasksRequest{
		StatusFilter: util.ConvertTaskStatus(statusFilter),
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) ListAttachments(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	resp, err := h.projectClient.Attachment.ListAttachments(ctx, &projectv1.ListAttachmentsRequest{
		TaskId: taskID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) DeleteAttachment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachment id required"})
		return
	}

	_, err = h.projectClient.Attachment.DeleteAttachment(ctx, &projectv1.DeleteAttachmentRequest{Id: id})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attachment deleted"})
}

func (h *ProjectHandler) GetAttachment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachment id required"})
		return
	}

	resp, err := h.projectClient.Attachment.GetAttachment(ctx, &projectv1.GetAttachmentRequest{Id: id})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
func (h *ProjectHandler) AddAttachment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	var req struct {
		FileURL     string `json:"file_url"`
		Type        string `json:"type"`
		FileName    string `json:"file_name"`
		FileSize    int64  `json:"file_size"`
		Description string `json:"description"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	var attType projectv1.AttachmentType
	switch req.Type {
	case "photo":
		attType = projectv1.AttachmentType_ATTACHMENT_TYPE_PHOTO
	case "document":
		attType = projectv1.AttachmentType_ATTACHMENT_TYPE_DOCUMENT
	case "drawing":
		attType = projectv1.AttachmentType_ATTACHMENT_TYPE_DRAWING
	default:
		attType = projectv1.AttachmentType_ATTACHMENT_TYPE_OTHER
	}

	resp, err := h.projectClient.Attachment.AddAttachment(ctx, &projectv1.AddAttachmentRequest{
		TaskId:      taskID,
		FileUrl:     req.FileURL,
		Type:        attType,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		Description: req.Description,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}
