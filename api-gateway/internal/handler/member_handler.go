package handler

import (
	"building-services/api-gateway/internal/util"
	projectv1 "building-services/gen/project/v1"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ProjectHandler) AddMember(c *gin.Context) {
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
		UserId string `json:"user_id"` // ← только user_id!
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	resp, err := h.projectClient.ProjectMember.AddMember(ctx, &projectv1.AddMemberRequest{
		ProjectId: projectID,
		UserId:    req.UserId,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ProjectHandler) ListMembers(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	projectId := c.Param("id")

	resp, err := h.projectClient.ProjectMember.ListMembers(ctx, &projectv1.ListMembersRequest{
		ProjectId: projectId,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	projectId := c.Param("id")
	if projectId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}

	userId := c.Param("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id required"})
		return
	}

	resp, err := h.projectClient.ProjectMember.RemoveMember(ctx, &projectv1.RemoveMemberRequest{
		UserId:    userId,
		ProjectId: projectId,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)

}
