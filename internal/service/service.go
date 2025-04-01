package service

import (
	"context"
	"database/sql"
	"github.com/google/uuid"

	"time"

	AuthService "newservice/grpc/genproto"
	"newservice/internal/config"
	"newservice/internal/repo"
	"newservice/pkg/jwt"
	"newservice/pkg/secure"
	"newservice/pkg/validator"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authServer struct {
	cfg  config.AppConfig
	repo repo.Repository
	log  *zap.SugaredLogger
	jwt  jwt.JWTClient
	AuthService.UnimplementedAuthServiceServer
}

func NewAuthServer(cfg config.AppConfig, repo repo.Repository, jwt jwt.JWTClient, log *zap.SugaredLogger) AuthService.AuthServiceServer {
	return &authServer{
		cfg:  cfg,
		repo: repo,
		log:  log,
		jwt:  jwt,
	}
}

func (a *authServer) Register(ctx context.Context, req *AuthService.RegisterRequest) (*AuthService.RegisterResponse, error) {
	if err := validator.Validate(ctx, req); err != nil {
		a.log.Errorf("validation error: %v", err)

		return nil, status.Error(codes.OK, err.Error())
	}

	passwordValidityCheck, err := secure.IsValidPassword(req.Password)
	if !passwordValidityCheck {
		errMsg := "invalid password"
		if err != nil {
			errMsg = err.Error()
		}
		return nil, status.Error(codes.InvalidArgument, errMsg)
	}

	req.Password, _ = secure.HashPassword(req.Password)

	_, err = a.repo.CreateUser(ctx, &repo.User{
		Username:       req.GetUsername(),
		HashedPassword: req.GetPassword(),
		Email:          req.GetEmail(),
	})
	if err != nil {
		a.log.Error("failed to create user", zap.Error(err))
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return nil, status.Error(codes.AlreadyExists, ErrUserAuthAlreadyExist)
			}
		}

		return nil, errors.Wrap(err, "failed to create user")
	}

	return &AuthService.RegisterResponse{}, nil
}

func (a *authServer) Login(ctx context.Context, req *AuthService.LoginRequest) (*AuthService.LoginResponse, error) {
	if err := validator.Validate(ctx, req); err != nil {
		a.log.Errorf("validation error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := a.repo.GetUserByUsername(ctx, req.GetUsername())
	if err != nil {
		a.log.Errorf("failed to get credentials for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := secure.CheckPassword(user.HashedPassword, req.GetPassword()); err != nil {
		a.log.Errorf("invalid password for user %a: %v", req.GetUsername(), err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	tokens, err := a.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: user.ID,
	})

	authTokenParams := repo.NewAuthTokenParams{
		UserID:           user.ID,                             // ID пользователя
		Tokens:           *tokens,                             // Токены
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour), // Устанавливаем дату истечения refresh токена
	}

	err = a.repo.NewAuthToken(ctx, authTokenParams)
	if err != nil {
		a.log.Errorf("failed to add auth token for user %s: %v", user.ID, err)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &AuthService.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *authServer) Validate(
	ctx context.Context,
	req *AuthService.ValidateRequest,
) (
	*AuthService.ValidateResponse, error,
) {

	check, err := a.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: req.AccessToken,
	})

	if err != nil || !check {
		return nil, status.Error(codes.Unauthenticated, "JWT validation failed")
	}

	accessData, err := a.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.AccessToken,
	})
	if err != nil {

		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	// проверяем наличие refresh токена в БД
	_, err = a.repo.GetRefreshToken(ctx, repo.GetRefreshTokenParams{
		UserID: accessData.UserId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
		}

		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &AuthService.ValidateResponse{
		UserId: accessData.UserId.String(),
	}, nil
}

func (a *authServer) NewJwt(
	ctx context.Context,
	req *AuthService.NewJwtRequest,
) (
	*AuthService.NewJwtResponse, error,
) {
	userID, err := uuid.Parse(req.UserId) // преобразуем string в uuid.UUID
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	if err := validator.Validate(ctx, req); err != nil {
		a.log.Errorf("validation error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tokens, err := a.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: userID,
	})
	if err != nil {
		a.log.Errorf("create tokens err: user_id = %s", req.UserId)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	_, err = a.repo.NewRefreshToken(ctx, repo.NewRefreshTokenParams{
		UserID: userID,
		Token:  tokens.RefreshToken,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return nil, status.Error(codes.NotFound, ErrUserNotFound)
		}
		a.log.Errorf("adding a token to the database: user_id = %s", req.UserId)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &AuthService.NewJwtResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *authServer) RevokeJwt(
	ctx context.Context,
	req *AuthService.RevokeJwtRequest,
) (
	*AuthService.RevokeJwtResponse, error,
) {
	userID, err := uuid.Parse(req.UserId) // преобразуем string в uuid.UUID
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	err = a.repo.DeleteRefreshToken(ctx, repo.DeleteRefreshTokenParams{
		UserID: userID,
	})
	if err != nil {
		a.log.Errorf("remove a token to the database: user_id = %s", req.UserId)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}
	return &AuthService.RevokeJwtResponse{}, nil
}

func (a *authServer) Refresh(
	ctx context.Context,
	req *AuthService.RefreshRequest,
) (
	*AuthService.RefreshResponse, error,
) {

	check, err := a.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: req.RefreshToken,
	})
	if err != nil || !check {
		a.log.Errorf("refresh token validation error")
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	// извлекаем данные из access и refresh токенов
	accessData, err := a.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.AccessToken,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	refreshData, err := a.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.RefreshToken,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}
	if accessData.UserId != refreshData.UserId {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	rtToken, err := a.repo.GetRefreshToken(ctx, repo.GetRefreshTokenParams{
		UserID: refreshData.UserId,
	})
	if err != nil {
		a.log.Errorf("get refresh token err")
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, ErrTokenNotFound)
		}
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	if len(rtToken) == 0 {
		a.log.Errorf("len(rtToken) == 0")
		return nil, status.Error(codes.NotFound, ErrTokenNotFound)
	}

	if rtToken[0] != req.RefreshToken {
		a.log.Errorf("rtToken[0] != req.RefreshToken")
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	// создаём новые токены
	tokens, err := a.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: refreshData.UserId,
	})

	if err != nil {
		a.log.Errorf("create tokens error")
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	err = a.repo.UpdateRefreshToken(ctx, repo.UpdateRefreshTokenParams{
		Token:     tokens.RefreshToken,
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UserID:    refreshData.UserId,
	})

	if err != nil {
		a.log.Errorf("update refresh token err")
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &AuthService.RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
