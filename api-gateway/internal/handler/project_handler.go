package handler

import (
	"building-services/api-gateway/internal/clients"
	"building-services/api-gateway/internal/util"
	projectv1 "building-services/gen/project/v1"
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProjectHandler struct {
	projectClient *clients.ProjectClient
}

func NewProjectHandler(client *clients.ProjectClient) *ProjectHandler {
	return &ProjectHandler{projectClient: client}

}

func (h *ProjectHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/projects", h.CreateProject)
	r.GET("/projects/:id", h.GetProject)
	r.GET("/projects", h.ListProjects)
	r.PUT("/projects/:id", h.UpdateProject)
	r.DELETE("/projects/:id", h.DeleteProject)
	r.PATCH("/projects/:id/status", h.ChangeProjectStatus)
	r.POST("/projects/:id/members", h.AddMember)
	r.GET("/projects/:id/members", h.ListMembers)
	r.DELETE("/projects/:id/members/:userId", h.RemoveMember)
	r.POST("/projects/:id/tasks", h.CreateTask)
	r.GET("/tasks/:id", h.GetTask)
	r.GET("/projects/:id/tasks", h.ListTasks)
	r.PUT("/tasks/:id", h.UpdateTask)
	r.DELETE("/tasks/:id", h.DeleteTask)
	r.PATCH("/tasks/:id/status", h.UpdateTaskStatus)
	r.PATCH("/tasks/:id/assign", h.AssignTask)
	r.GET("/tasks/my", h.ListMyTasks)
	r.GET("/projects/:id/timeline", h.GetTimeline)
	r.PATCH("/projects/:id/timeline", h.UpdateTimeline)
	r.POST("/tasks/:id/attachments", h.AddAttachment)
	r.GET("/tasks/:id/attachments", h.ListAttachments)
	r.GET("/attachments/:id", h.GetAttachment)
	r.DELETE("/attachments/:id", h.DeleteAttachment)
	r.POST("/departments", h.CreateDepartment)
	r.GET("/departments", h.ListDepartments)
	r.PUT("/departments/:id", h.UpdateDepartment)
	r.DELETE("/departments/:id", h.DeleteDepartment)
	r.POST("/departments/:id/users/:userId", h.AssignUserToDepartment)
	r.DELETE("/departments/:id/users/:userId", h.RemoveUserFromDepartment)
	r.GET("/departments/:id/users", h.GetDepartmentUsers)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
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

func (h *ProjectHandler) GetProject(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}
	resp, err := h.projectClient.Project.GetProject(ctx, &projectv1.GetProjectRequest{Id: id})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}
	resp, err := h.projectClient.Project.DeleteProject(ctx, &projectv1.DeleteProjectRequest{Id: id})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}

	var req struct {
		Name string `json:"name"`
		Description string `json:"description"`
		ObjectAddress string `json:"object_address"`
		Customer string `json:"customer"`
		EndDate string `json:"end_date"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var endDate *timestamppb.Timestamp
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format. Use RFC3339: 2024-04-15T00:00:00Z"})
			return
		}
		endDate = timestamppb.New(t)
	}

	resp, err := h.projectClient.Project.UpdateProject(ctx, &projectv1.UpdateProjectRequest{
		Id: id,
		Name: req.Name,
		Description: req.Description,
		ObjectAddress: req.ObjectAddress,
		Customer: req.Customer,
		EndDate: endDate,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) ChangeProjectStatus(c *gin.Context) {
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
	projectID := c.Param("id")
	resp, err := h.projectClient.Project.ChangeProjectStatus(ctx, &projectv1.ChangeProjectStatusRequest{
		Id:     projectID,
		Status: util.ConvertStatus(req.Status),
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	statusFilter := c.Query("status")
	managerID := c.Query("manager_id")
	var status projectv1.ProjectStatus
	status = util.ConvertStatus(statusFilter)
	resp, err := h.projectClient.Project.ListProjects(ctx, &projectv1.ListProjectsRequest{
		StatusFilter: status,
		ManagerId:    managerID,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) GetTimeline(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}
	resp, err := h.projectClient.Timeline.GetTimeline(ctx, &projectv1.GetTimelineRequest{
		ProjectId: projectID,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) UpdateTimeline(c *gin.Context) {
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
		ContractDate      string `json:"contract_date"`
		WorkStartDate     string `json:"work_start_date"`
		WorkEndDate       string `json:"work_end_date"`
		HandoverDate      string `json:"handover_date"`
		CommentsDate      string `json:"comments_date"`
		CommentsFixedDate string `json:"comments_fixed_date"`
		AcceptanceDate    string `json:"acceptance_date"`
		FinalPaymentDate  string `json:"final_payment_date"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return

	}
	parseDate := func(s string) *timestamppb.Timestamp {
		if s == "" {
			return nil

		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil
		}
		return timestamppb.New(t)
	}
	resp, err := h.projectClient.Timeline.UpdateTimeline(ctx, &projectv1.UpdateTimelineRequest{
		ProjectId:         projectID,
		ContractDate:      parseDate(req.ContractDate),
		WorkStartDate:     parseDate(req.WorkStartDate),
		WorkEndDate:       parseDate(req.WorkEndDate),
		HandoverDate:      parseDate(req.HandoverDate),
		CommentsDate:      parseDate(req.CommentsDate),
		CommentsFixedDate: parseDate(req.CommentsFixedDate),
		AcceptanceDate:    parseDate(req.AcceptanceDate),
		FinalPaymentDate:  parseDate(req.FinalPaymentDate),
	})

	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)

}

func (h *ProjectHandler) GetUserByID(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	userID := c.Param("id")
	resp, err := h.projectClient.User.GetUser(ctx, &projectv1.GetUserRequest{Id: userID})
	if err != nil {
		handleError(c, err)
		return

	}
	c.JSON(200, resp)
}

func (h *ProjectHandler) FindUsers(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query != "" && len(query) < 2 {
		c.JSON(http.StatusOK, &projectv1.FindUsersResponse{Users: nil})
		return
	}

	resp, err := h.projectClient.Project.FindUsers(ctx, &projectv1.FindUsersRequest{
		Query: query,
		Limit: 100,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) GetUserByEmail(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}
	resp, err := h.projectClient.User.GetUserByEmail(ctx, &projectv1.GetUserByEmailRequest{Email: email})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
