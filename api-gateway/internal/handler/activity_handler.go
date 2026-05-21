package handler

import (
	"building-services/api-gateway/internal/util"
	projectv1 "building-services/gen/project/v1"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ProjectHandler) ListActivityTypes(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	resp, err := h.projectClient.Activity.ListActivityTypes(ctx, &projectv1.ListActivityTypesRequest{})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity_types": resp.ActivityTypes})
}

func (h *ProjectHandler) CreateActivityType(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	resp, err := h.projectClient.Activity.CreateActivityType(ctx, &projectv1.CreateActivityTypeRequest{
		Name: req.Name,
	})
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
