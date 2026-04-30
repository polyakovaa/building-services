package handler

import (
	"net/http"

	"building-services/api-gateway/internal/middleware"
	"building-services/api-gateway/internal/util"
	authv1 "building-services/gen/auth/v1"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	authClient authv1.AuthServiceClient
}

func NewAdminHandler(authClient authv1.AuthServiceClient) *AdminHandler {
	return &AdminHandler{authClient: authClient}
}
func (h *AdminHandler) RegisterRoutes(r *gin.Engine) {
	admin := r.Group("/admin")
	admin.Use(middleware.RequireRole("ROLE_ADMIN"))
	{
		admin.GET("/users", h.ListUsers)
		admin.PUT("/users/:id/role", h.UpdateUserRole)
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	resp, err := h.authClient.ListUsers(ctx, &authv1.ListUsersRequest{})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	ctx, err := util.GetGRPCContext(c)
	if err != nil {
		return
	}

	userID := c.Param("id")
	var req struct {
		Role string `json:"role"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.authClient.UpdateUserRole(ctx, &authv1.UpdateUserRoleRequest{
		UserId:  userID,
		NewRole: req.Role,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
