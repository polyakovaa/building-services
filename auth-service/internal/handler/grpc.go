package handler

import (
	"building-services/auth-service/internal/service"
	authv1 "building-services/gen/auth/v1"
	"context"
	"log"

	"google.golang.org/grpc/codes"
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
