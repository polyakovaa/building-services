package service_test

import (
	"building-services/auth-service/internal/model"
	"building-services/auth-service/internal/repository"
	"building-services/auth-service/internal/service"
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

			authService := service.NewAuthService(
				mockUserRepo,
				mockTokenRepo,
				"secret",
				15*time.Minute,
				24*time.Hour,
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
			)

			user, err := authService.RegisterUser(tt.userName, tt.email, tt.password, tt.role)

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
		UserName: "testuser",
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
		UserName: "testuser",
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
	)

	token, exp, err := authService.GenerateTokens(user)
	require.Error(t, err)
	require.Nil(t, token)
	require.False(t, exp.IsZero())
}

func TestValidateAccessToken(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		role          string
		secret        string
		exp           time.Time
		expectedError error
	}{
		{
			name:          "valid user",
			userID:        "1",
			role:          "ROLE_WORKER",
			secret:        "secret",
			exp:           time.Now().Add(1 * time.Hour),
			expectedError: nil,
		},
		{
			name:          "wrong secret",
			userID:        "1",
			role:          "ROLE_WORKER",
			secret:        "wrong secret",
			exp:           time.Now().Add(1 * time.Hour),
			expectedError: service.ErrInvalidAccessToken,
		},
		{
			name:          "expired",
			userID:        "1",
			role:          "ROLE_WORKER",
			secret:        "secret",
			exp:           time.Now().Add(-1 * time.Hour),
			expectedError: service.ErrInvalidAccessToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := service.AccessClaims{
				Role: tt.role,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   tt.userID,
					ExpiresAt: jwt.NewNumericDate(tt.exp),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}
			authService := service.NewAuthService(
				nil,
				nil,
				"secret",
				15*time.Minute,
				24*time.Hour,
			)

			access := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
			accessToken, _ := access.SignedString([]byte(tt.secret))

			sub, role, exp, err := authService.ValidateAccessToken(accessToken)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.False(t, exp.IsZero())
			assert.Equal(t, tt.userID, sub)
			assert.Equal(t, tt.role, role)

		})
	}

}
