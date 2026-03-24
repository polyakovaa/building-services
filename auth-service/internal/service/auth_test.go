package service_test

import (
	"building-services/auth-service/internal/model"
	"building-services/auth-service/internal/repository"
	"building-services/auth-service/internal/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepo struct {
	User *model.User
	Err  error
}
type MockTokenRepo struct {
	SavedToken string
	Err        error
}

func (m *MockTokenRepo) SaveRefreshToken(userID, token string, ttl time.Duration) error {
	m.SavedToken = token
	return m.Err
}

func (m *MockTokenRepo) GetUserIDByRefreshToken(token string) (string, error) {
	return "", nil
}

func (m *MockTokenRepo) DeleteRefreshToken(token string) error {
	return nil
}

func (m *MockUserRepo) FindByEmail(email string) (*model.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.User, nil
}

func (m *MockUserRepo) FindByID(email string) (*model.User, error) {
	return m.User, m.Err
}

func (m *MockUserRepo) CreateUser(u *model.User) (*model.User, error) {
	return u, nil
}

type MockPublisher struct {
	CreatedCalls []UserData
	UpdatedCalls []UserData
	ShouldFail   bool
}

type UserData struct {
	UserID   string
	Email    string
	FullName string
	Role     string
}

func (m *MockPublisher) PublishUserCreated(ctx context.Context, userID, email, fullName, role string) error {
	m.CreatedCalls = append(m.CreatedCalls, UserData{
		UserID: userID, Email: email, FullName: fullName, Role: role,
	})
	if m.ShouldFail {
		return errors.New("mock error")
	}
	return nil
}

func (m *MockPublisher) PublishUserUpdated(ctx context.Context, userID, email, fullName, role string) error {
	m.UpdatedCalls = append(m.UpdatedCalls, UserData{
		UserID: userID, Email: email, FullName: fullName, Role: role,
	})
	if m.ShouldFail {
		return errors.New("mock error")
	}
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		email          string
		password       string
		storedPassword string
		role           string
		userExists     bool
		expectedError  error
	}{
		{
			name:           "valid user",
			userID:         "1",
			email:          "test@mail.ru",
			password:       "1234",
			storedPassword: "1234",
			role:           "ROLE_WORKER",
			userExists:     true,
			expectedError:  nil,
		},
		{
			name:          "user not found",
			email:         "no@mail.ru",
			password:      "1234",
			userExists:    false,
			expectedError: service.ErrInvalidCredentials,
		},
		{
			name:           "wrong password",
			userID:         "1",
			email:          "test@mail.ru",
			password:       "wrong",
			storedPassword: "1234",
			role:           "ROLE_WORKER",
			userExists:     true,
			expectedError:  service.ErrInvalidCredentials,
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hashedPassword string
			if tt.userExists {
				hash, err := bcrypt.GenerateFromPassword([]byte(tt.storedPassword), bcrypt.DefaultCost)
				require.NoError(t, err)
				hashedPassword = string(hash)
			}
			mockTokenRepo := &MockTokenRepo{}

			mockUserRepo := &MockUserRepo{}
			if tt.userExists {
				mockUserRepo.User = &model.User{
					ID:           tt.userID,
					Email:        tt.email,
					PasswordHash: hashedPassword,
					Role:         tt.role,
				}
			} else {
				mockUserRepo.Err = repository.ErrUserNotFound
			}
			mockPublisher := &MockPublisher{}

			authService := service.NewAuthService(
				mockUserRepo,
				mockTokenRepo,
				"secret",
				15*time.Minute,
				24*time.Hour,
				mockPublisher,
			)

			user, token, exp, err := authService.Login(tt.email, tt.password)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Nil(t, user)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)
			require.NotNil(t, token)
			require.False(t, exp.IsZero())

			require.Equal(t, tt.email, user.Email)
			require.NotEmpty(t, token.AccessToken)
			require.NotEmpty(t, token.RefreshToken)
			require.Equal(t, token.RefreshToken, mockTokenRepo.SavedToken)
		})
	}
}

func TestRegisterUser(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		userName      string
		email         string
		password      string
		role          string
		expectedError error
		userExists    bool
	}{
		{
			name:          "valid user",
			userID:        "1",
			email:         "test@mail.ru",
			userName:      "testuser",
			password:      "1234567",
			role:          "ROLE_WORKER",
			userExists:    false,
			expectedError: nil,
		},
		{
			name:          "invalid email",
			email:         "nomail.ru",
			userName:      "testuser",
			password:      "1234567",
			expectedError: service.ErrInvalidEmailFormat,
		},
		{
			name:          "password too short",
			userName:      "testuser",
			email:         "test@mail.ru",
			password:      "123",
			expectedError: service.ErrPasswordTooShort,
		},
		{
			name:          "already exists",
			userID:        "1",
			email:         "test@mail.ru",
			userName:      "test user",
			password:      "1234567",
			role:          "ROLE_WORKER",
			userExists:    true,
			expectedError: repository.ErrEmailAlreadyExists,
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := &MockUserRepo{}

			if tt.userExists {
				mockUserRepo.User = &model.User{
					ID:    tt.userID,
					Email: tt.email,
				}
				mockUserRepo.Err = nil
			} else {
				mockUserRepo.User = nil
				mockUserRepo.Err = repository.ErrUserNotFound
			}

			authService := service.NewAuthService(
				mockUserRepo,
				&MockTokenRepo{},
				"secret",
				15*time.Minute,
				24*time.Hour,
				&MockPublisher{},
			)

			user, err := authService.RegisterUser(context.Background(), tt.userName, tt.email, tt.password, tt.role)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Nil(t, user)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)
			require.Equal(t, tt.email, user.Email)
			require.NotEqual(t, tt.password, user.PasswordHash)
		})
	}
}

func TestGenerateTokens_Success(t *testing.T) {
	user := model.User{
		ID:       "1",
		Email:    "test@mail.ru",
		FullName: "testuser",
		Role:     "ROLE_WORKER",
	}
	mockUserRepo := &MockUserRepo{}

	mockUserRepo.User = &user
	mockUserRepo.Err = nil

	authService := service.NewAuthService(
		mockUserRepo,
		&MockTokenRepo{},
		"secret",
		15*time.Minute,
		24*time.Hour,
		&MockPublisher{},
	)

	token, exp, err := authService.GenerateTokens(mockUserRepo.User)

	require.NotNil(t, token)
	require.NoError(t, err)
	require.False(t, exp.IsZero())
	require.NotEmpty(t, token.AccessToken)
	require.NotEmpty(t, token.RefreshToken)

	claims := &service.AccessClaims{}
	parsedToken, err := jwt.ParseWithClaims(
		token.AccessToken,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		},
	)

	require.NoError(t, err)
	require.True(t, parsedToken.Valid)

	assert.Equal(t, user.ID, claims.Subject)
	assert.Equal(t, user.Role, claims.Role)
}

func TestGenerateTokens_SaveRefreshErr(t *testing.T) {
	user := &model.User{
		ID:       "1",
		Email:    "test@mail.ru",
		FullName: "testuser",
		Role:     "ROLE_WORKER",
	}
	mockTokenRepo := &MockTokenRepo{
		Err: errors.New("db error"),
	}

	authService := service.NewAuthService(
		nil,
		mockTokenRepo,
		"secret",
		15*time.Minute,
		24*time.Hour,
		&MockPublisher{},
	)

	token, exp, err := authService.GenerateTokens(user)
	require.Error(t, err)
	require.Nil(t, token)
	require.False(t, exp.IsZero())
}
