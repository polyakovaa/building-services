package service

import (
	"building-services/auth-service/internal/events"
	"building-services/auth-service/internal/model"
	"building-services/auth-service/internal/repository"
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo       UserRepo
	tokenRepo      TokenRepo
	jwtSecret      string
	accessTTL      time.Duration
	refreshTTL     time.Duration
	eventPublisher events.Publisher
}

func NewAuthService(
	userRepo UserRepo,
	tokenRepo TokenRepo,
	secret string,
	accessTTL, refreshTTL time.Duration,
	eventPublisher events.Publisher,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		jwtSecret:      secret,
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
		eventPublisher: eventPublisher,
	}
}

type UserRepo interface {
	CreateUser(u *model.User) (*model.User, error)
	FindByID(id string) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindAll(ctx context.Context) ([]*model.User, error)
	UpdateRole(ctx context.Context, userID string, newRole string) error
	UpdateProfile(ctx context.Context, userID string, fullName string, email string) (*model.User, error)
}

type TokenRepo interface {
	SaveRefreshToken(userID, token string, ttl time.Duration) error
	GetUserIDByRefreshToken(token string) (string, error)
	DeleteRefreshToken(token string) error
}

type AccessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

var (
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrPasswordTooShort   = errors.New("password too short")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrNoPermission       = errors.New("permission denied")
)

func (s *AuthService) RegisterUser(ctx context.Context, username, email, password, role string) (*model.User, error) {

	if !isValidEmail(email) {
		return nil, ErrInvalidEmailFormat
	}
	if len(password) < 6 {
		return nil, ErrPasswordTooShort
	}
	if username == "" {
		return nil, fmt.Errorf("full name required")
	}

	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	if existingUser != nil {
		return nil, repository.ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	log.Printf("Creating user: name=%s, email=%s", username, email)

	u := &model.User{
		FullName:     username,
		Email:        email,
		PasswordHash: string(hashed),
		Role:         role,
	}
	user, err := s.userRepo.CreateUser(u)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.eventPublisher.PublishUserCreated(ctx, user.ID, user.Email, user.FullName, user.Role); err != nil {
		log.Printf("Failed to publish event: %v", err)
	}

	return user, nil

}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (s *AuthService) GenerateTokens(user *model.User) (*model.Token, time.Time, error) {
	exp := time.Now().Add(s.accessTTL)

	claims := AccessClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	access := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	accessToken, err := access.SignedString([]byte(s.jwtSecret))

	if err != nil {
		log.Printf("Error signing access token for user %s: %v", user.ID, err)
		return nil, exp, err
	}

	refreshToken := uuid.NewString()

	err = s.tokenRepo.SaveRefreshToken(user.ID, refreshToken, s.refreshTTL)
	if err != nil {
		return nil, exp, err
	}

	return &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, exp, nil

}

func (s *AuthService) GetUserByID(id string) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *AuthService) RefreshToken(oldToken string) (*model.User, *model.Token, time.Time, error) {

	userID, err := s.tokenRepo.GetUserIDByRefreshToken(oldToken)
	if err != nil {
		return nil, nil, time.Time{}, repository.ErrInvalidRefreshToken
	}

	_ = s.tokenRepo.DeleteRefreshToken(oldToken)

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("find user by id: %w", err)
	}

	token, exp, err := s.GenerateTokens(user)
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	return user, token, exp, nil
}

func (s *AuthService) Login(email, password string) (*model.User, *model.Token, time.Time, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, time.Time{}, ErrInvalidCredentials
		}
		return nil, nil, time.Time{}, fmt.Errorf("find user by email: %w", err)
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, nil, time.Time{}, ErrInvalidCredentials
	}

	token, exp, err := s.GenerateTokens(user)
	if err != nil {
		return nil, nil, time.Time{}, err
	}

	return user, token, exp, nil
}

func (s *AuthService) Logout(refreshToken string) error {
	return s.tokenRepo.DeleteRefreshToken(refreshToken)
}

func (s *AuthService) UpdateRole(ctx context.Context, userID, newRole string, adminID string) error {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil || admin.Role != "ROLE_ADMIN" {
		return ErrNoPermission
	}

	return s.userRepo.UpdateRole(ctx, userID, newRole)
}

func (s *AuthService) ListUsers(ctx context.Context, adminID string) ([]*model.User, error) {
	admin, err := s.userRepo.FindByID(adminID)
	if err != nil || admin.Role != "ROLE_ADMIN" {
		return nil, ErrNoPermission
	}
	return s.userRepo.FindAll(ctx)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID string, fullName string, email string) (*model.User, error) {
	if fullName == "" && email == "" {
		return nil, fmt.Errorf("nothing to update")
	}

	if email != "" && !isValidEmail(email) {
		return nil, ErrInvalidEmailFormat
	}

	u, err := s.userRepo.UpdateProfile(ctx, userID, fullName, email)
	if err != nil {
		return nil, err
	}

	if err := s.eventPublisher.PublishUserUpdated(ctx, u.ID, u.Email, u.FullName, u.Role); err != nil {
		log.Printf("Failed to publish user.updated event: %v", err)
	}

	return u, nil
}
