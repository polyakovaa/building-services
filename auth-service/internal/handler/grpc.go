package handler

import (
	"building-services/auth-service/internal/service"
	authv1 "building-services/gen/auth/v1"
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	authService *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: s,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	role := req.Role.String()

	user, err := h.authService.RegisterUser(ctx, req.FullName, req.Email, req.Password, role)

	if err != nil {
		log.Printf("Failed to register user %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	token, exp, err := h.authService.GenerateTokens(user)
	if err != nil {
		log.Printf("Failed to generate tokens for user %v", err)
		return nil, status.Error(codes.Internal, "failed to generate tokens")
	}

	return &authv1.AuthResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Role:         authv1.Role(authv1.Role_value[user.Role]),
		UserId:       user.ID,
		ExpiresAt:    timestamppb.New(exp),
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {

	user, tokens, exp, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &authv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    timestamppb.New(exp),
		UserId:       user.ID,
		Role:         authv1.Role(authv1.Role_value[user.Role]),
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.AuthResponse, error) {
	user, tokens, exp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &authv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    timestamppb.New(exp),
		UserId:       user.ID,
		Role:         authv1.Role(authv1.Role_value[user.Role]),
	}, nil
}

func (h *AuthHandler) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	userID, err := userIDFromIncomingMetadata(ctx)
	if err != nil {
		return nil, err
	}

	users, err := h.authService.ListUsers(ctx, userID)
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	userInfos := make([]*authv1.UserInfo, 0, len(users))
	for _, u := range users {
		userInfos = append(userInfos, &authv1.UserInfo{
			Id:        u.ID,
			Email:     u.Email,
			FullName:  u.FullName,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
		})
	}

	return &authv1.ListUsersResponse{
		Users:      userInfos,
		TotalCount: int32(len(userInfos)),
	}, nil
}

func (h *AuthHandler) UpdateUserRole(ctx context.Context, req *authv1.UpdateUserRoleRequest) (*authv1.UpdateUserRoleResponse, error) {
	adminID, err := userIDFromIncomingMetadata(ctx)
	if err != nil {
		return nil, err
	}

	err = h.authService.UpdateRole(ctx, req.UserId, req.NewRole, adminID)
	if err != nil {
		log.Printf("Failed to update user role: %v", err)
		return &authv1.UpdateUserRoleResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &authv1.UpdateUserRoleResponse{
		Success: true,
		Message: "Role updated successfully",
	}, nil
}

func (h *AuthHandler) GetInfo(ctx context.Context, req *authv1.GetInfoRequest) (*authv1.UserInfo, error) {
	userID, err := userIDFromIncomingMetadata(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return &authv1.UserInfo{
		Id:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     user.Role,
	}, nil
}

func (h *AuthHandler) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UserInfo, error) {
	userID, err := userIDFromIncomingMetadata(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.authService.UpdateProfile(ctx, userID, req.FullName, req.Email)
	if err != nil {
		log.Printf("Failed to update profile: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &authv1.UserInfo{
		Id:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func userIDFromIncomingMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no metadata")
	}
	values := md.Get("user_id")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "user_id not found")
	}
	return values[0], nil
}
