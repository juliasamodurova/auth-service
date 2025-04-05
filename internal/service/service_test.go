package service

import (
	"context"
	"database/sql"
	"errors"
	"newservice/internal/config"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	AuthService "newservice/grpc/genproto"
	"newservice/internal/repo"
	repoMocks "newservice/internal/repo/mocks"
	"newservice/pkg/jwt"
	jwtMocks "newservice/pkg/jwt/mocks"
)

// newTestAuthServer создаёт экземпляр authServer с моками репозитория и jwt
func newTestAuthServer() (*authServer, *repoMocks.Repository, *jwtMocks.JWTClient) {
	cfg := config.AppConfig{}
	repoMock := new(repoMocks.Repository)
	jwtMock := new(jwtMocks.JWTClient)
	logger := zap.NewNop().Sugar()
	return &authServer{
		cfg:  cfg,
		repo: repoMock,
		jwt:  jwtMock,
		log:  logger,
	}, repoMock, jwtMock
}

func TestAuthServer_Validate(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name        string
		req         *AuthService.ValidateRequest
		jwtMock     func(j *jwtMocks.JWTClient)
		repoMock    func(r *repoMocks.Repository)
		expectedErr error
	}{
		{
			name: "успешная валидация",
			req: &AuthService.ValidateRequest{
				AccessToken: "valid_token",
			},
			jwtMock: func(j *jwtMocks.JWTClient) {
				j.On("ValidateToken", mock.AnythingOfType("*jwt.ValidateTokenParams")).
					Return(true, nil).Once()
				j.On("GetDataFromToken", mock.AnythingOfType("*jwt.GetDataFromTokenParams")).
					Return(&jwt.GetDataFromTokenResponse{
						UserId: userID,
					}, nil).Once()
			},
			repoMock: func(r *repoMocks.Repository) {
				r.On("GetRefreshToken", mock.Anything, repo.GetRefreshTokenParams{UserID: userID}).
					Return([]string{"refresh_token"}, nil).Once()
			},
			expectedErr: nil,
		},
		{
			name: "невалидный токен",
			req: &AuthService.ValidateRequest{
				AccessToken: "invalid_token",
			},
			jwtMock: func(j *jwtMocks.JWTClient) {
				j.On("ValidateToken", mock.AnythingOfType("*jwt.ValidateTokenParams")).
					Return(false, errors.New("invalid token")).Once()
			},
			expectedErr: status.Error(codes.Unauthenticated, "JWT validation failed"),
		},
		{
			name: "ошибка извлечения данных из токена",
			req: &AuthService.ValidateRequest{
				AccessToken: "valid_token",
			},
			jwtMock: func(j *jwtMocks.JWTClient) {
				j.On("ValidateToken", mock.AnythingOfType("*jwt.ValidateTokenParams")).
					Return(true, nil).Once()
				j.On("GetDataFromToken", mock.AnythingOfType("*jwt.GetDataFromTokenParams")).
					Return(nil, errors.New("extraction error")).Once()
			},
			expectedErr: status.Error(codes.Unauthenticated, ErrValidateJwt),
		},
		{
			name: "refresh token не найден",
			req: &AuthService.ValidateRequest{
				AccessToken: "valid_token",
			},
			jwtMock: func(j *jwtMocks.JWTClient) {
				j.On("ValidateToken", mock.AnythingOfType("*jwt.ValidateTokenParams")).
					Return(true, nil).Once()
				j.On("GetDataFromToken", mock.AnythingOfType("*jwt.GetDataFromTokenParams")).
					Return(&jwt.GetDataFromTokenResponse{
						UserId: userID,
					}, nil).Once()
			},
			repoMock: func(r *repoMocks.Repository) {
				r.On("GetRefreshToken", mock.Anything, repo.GetRefreshTokenParams{UserID: userID}).
					Return(nil, sql.ErrNoRows).Once()
			},
			expectedErr: status.Error(codes.Unauthenticated, ErrValidateJwt),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, repoMock, jwtMock := newTestAuthServer()
			if tt.repoMock != nil {
				tt.repoMock(repoMock)
			}
			if tt.jwtMock != nil {
				tt.jwtMock(jwtMock)
			}
			resp, err := srv.Validate(context.Background(), tt.req)
			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, userID.String(), resp.UserId)
			repoMock.AssertExpectations(t)
			jwtMock.AssertExpectations(t)
		})
	}
}
