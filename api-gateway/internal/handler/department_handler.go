package handler

import (
	"net/http"

	"building-services/api-gateway/internal/util"
	projectv1 "building-services/gen/project/v1"

	"github.com/gin-gonic/gin"
)

func (h *ProjectHandler) CreateDepartment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	var req struct {
		Name       string `json:"name"`
		HeadUserID string `json:"head_user_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.projectClient.Department.CreateDepartment(ctx, &projectv1.CreateDepartmentRequest{
		Name:       req.Name,
		HeadUserId: req.HeadUserID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ProjectHandler) ListDepartments(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	resp, err := h.projectClient.Department.ListDepartments(ctx, &projectv1.ListDepartmentsRequest{})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"departments":  resp.Departments,
		"total_count": resp.TotalCount,
	})
}

func (h *ProjectHandler) AssignUserToDepartment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Param("id")
	userID := c.Param("userId")

	_, err = h.projectClient.Department.AssignUserToDepartment(ctx, &projectv1.AssignUserToDepartmentRequest{
		UserId:       userID,
		DepartmentId: departmentID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user assigned"})
}

func (h *ProjectHandler) UpdateDepartment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Param("id")
	var req struct {
		Name       string `json:"name"`
		HeadUserID string `json:"head_user_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.projectClient.Department.UpdateDepartment(ctx, &projectv1.UpdateDepartmentRequest{
		Id:         departmentID,
		Name:       req.Name,
		HeadUserId: req.HeadUserID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) DeleteDepartment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Param("id")

	_, err = h.projectClient.Department.DeleteDepartment(ctx, &projectv1.DeleteDepartmentRequest{
		Id: departmentID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "department deleted"})
}

func (h *ProjectHandler) RemoveUserFromDepartment(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Param("id")
	userID := c.Param("userId")

	_, err = h.projectClient.Department.RemoveUserFromDepartment(ctx, &projectv1.RemoveUserFromDepartmentRequest{
		UserId:       userID,
		DepartmentId: departmentID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed from department"})
}

func (h *ProjectHandler) GetDepartmentUsers(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	departmentID := c.Param("id")

	resp, err := h.projectClient.Department.GetDepartmentUsers(ctx, &projectv1.GetDepartmentUsersRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
