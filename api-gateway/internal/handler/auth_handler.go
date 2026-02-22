package handler

import (
	authv1 "building-services/gen/auth/v1"
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	authClient authv1.AuthServiceClient
}

func NewAuthHandler(authClient authv1.AuthServiceClient) *AuthHandler {
	return &AuthHandler{authClient: authClient}
}

func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/login", h.Login)
	r.POST("/register", h.Register)
	r.POST("/refresh", h.Refresh)
}

func (h *AuthHandler) Register(c *gin.Context) {
	if h.authClient == nil {
		c.JSON(503, gin.H{"error": "Auth service unavailable"})
		return
	}

	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		UserName string `json:"user_name"`
		Role     string `json:"role"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	role, err := convertRole(request.Role)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authClient.Register(c.Request.Context(), &authv1.RegisterRequest{
		Email:    request.Email,
		Password: request.Password,
		UserName: request.UserName,
		Role:     role,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	setRefreshTokenCookie(c, response.RefreshToken, response.ExpiresAt.AsTime())

	c.JSON(201, gin.H{
		"user_id":       response.UserId,
		"access_token":  response.AccessToken,
		"expires_at":    response.ExpiresAt.AsTime().Format(time.RFC3339),
		"refresh_token": response.RefreshToken,
		"role":          response.Role.String(),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.authClient == nil {
		c.JSON(503, gin.H{"error": "Auth service unavailable"})
		return
	}

	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := h.authClient.Login(c.Request.Context(), &authv1.LoginRequest{
		Email:    request.Email,
		Password: request.Password,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	setRefreshTokenCookie(c, response.RefreshToken, response.ExpiresAt.AsTime())

	c.JSON(200, gin.H{
		"user_id":       response.UserId,
		"access_token":  response.AccessToken,
		"expires_at":    response.ExpiresAt.AsTime().Format(time.RFC3339),
		"refresh_token": response.RefreshToken,
		"role":          response.Role.String(),
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	if h.authClient == nil {
		c.JSON(503, gin.H{"error": "Auth service unavailable"})
		return
	}

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(401, gin.H{"error": "Refresh token required"})
		return
	}

	refreshToken, err = url.QueryUnescape(refreshToken)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid refresh token encoding"})
		return
	}

	response, err := h.authClient.RefreshToken(c.Request.Context(), &authv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})

	if err != nil {
		handleError(c, err)
		return
	}

	setRefreshTokenCookie(c, response.RefreshToken, response.ExpiresAt.AsTime())

	c.JSON(200, gin.H{
		"access_token": response.AccessToken,
		"expires_at":   response.ExpiresAt.AsTime().Format(time.RFC3339),
	})
}

func convertRole(roleStr string) (authv1.Role, error) {
	switch roleStr {
	case "ROLE_GIP":
		return authv1.Role_ROLE_GIP, nil
	case "ROLE_DEPARTMENT_MANAGER":
		return authv1.Role_ROLE_DEPARTMENT_MANAGER, nil
	case "ROLE_DIRECTOR":
		return authv1.Role_ROLE_DIRECTOR, nil
	case "ROLE_WORKER":
		return authv1.Role_ROLE_WORKER, nil
	case "ROLE_PROJECT_MANAGER":
		return authv1.Role_ROLE_PROJECT_MANAGER, nil
	default:
		return authv1.Role_ROLE_UNSPECIFIED, fmt.Errorf("unknown role: %s", roleStr)
	}
}

func setRefreshTokenCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.SetCookie("refresh_token", token, maxAge, "/", "", false, true)
}

func handleError(c *gin.Context, err error) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			c.JSON(401, gin.H{"error": "Invalid credentials"})
		case codes.AlreadyExists:
			c.JSON(409, gin.H{"error": "User already exists"})
		case codes.NotFound:
			c.JSON(404, gin.H{"error": "Resource not found"})
		case codes.InvalidArgument:
			c.JSON(400, gin.H{"error": st.Message()})
		case codes.PermissionDenied:
			c.JSON(403, gin.H{"error": "Permission denied"})
		case codes.DeadlineExceeded:
			c.JSON(504, gin.H{"error": "Request timeout"})
		case codes.Unavailable:
			c.JSON(503, gin.H{"error": "Service temporarily unavailable"})
		default:
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
	} else {
		c.JSON(500, gin.H{"error": err.Error()})
	}
}
