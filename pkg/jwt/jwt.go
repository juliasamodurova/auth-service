package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type JWTClient interface {
	CreateToken(params *CreateTokenParams) (*CreateTokenResponse, error)
	ValidateToken(params *ValidateTokenParams) (bool, error)
	GetDataFromToken(params *GetDataFromTokenParams) (*GetDataFromTokenResponse, error)
}

type jwtClient struct {
	privateKey       *rsa.PrivateKey
	publicKey        *rsa.PublicKey
	accessTokenTime  time.Duration
	refreshTokenTime time.Duration
}

func NewJWTClient( // функция-конструктор, принимает RSA ключи и длительности токенов, возвращает экземпляр jwtClient
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	accessTokenTime time.Duration,
	refreshTokenTime time.Duration,
) *jwtClient {
	return &jwtClient{
		privateKey:       privateKey,
		publicKey:        publicKey,
		accessTokenTime:  accessTokenTime,
		refreshTokenTime: refreshTokenTime,
	}
}

func (a *jwtClient) CreateToken(params *CreateTokenParams) (*CreateTokenResponse, error) {

	accessToken, err := a.newToken(params, a.accessTokenTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshToken, err := a.newToken(params, a.refreshTokenTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &CreateTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *jwtClient) ValidateToken(params *ValidateTokenParams) (bool, error) {
	token, err := jwt.Parse(params.Token, func(token *jwt.Token) (interface{}, error) {
		return a.publicKey, nil
	})

	if err != nil {
		return false, err
	}

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expirationTime := token.Claims.(jwt.MapClaims)["exp"].(float64)
		if int64(expirationTime) > time.Now().Unix() {
			return true, nil
		}
	}
	return false, err
}

func (a *jwtClient) GetDataFromToken(params *GetDataFromTokenParams) (*GetDataFromTokenResponse, error) {
	token, err := jwt.Parse(params.Token, func(token *jwt.Token) (interface{}, error) {
		return a.publicKey, nil
	})
	if err != nil {
		if err.Error() != "Token is expired" {
			return nil, err
		}
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		userIdStr, ok := claims["userId"].(string) // Получаем как строку из токена
		if !ok {
			log.Error().Msg("failed to cast userId to string")
			return nil, fmt.Errorf("invalid token claims")
		}

		// Парсим строку в uuid.UUID
		userId, err := uuid.Parse(userIdStr)
		if err != nil {
			return nil, fmt.Errorf("invalid userId format in token: %w", err)
		}

		return &GetDataFromTokenResponse{
			UserId: userId, // Возвращаем как uuid.UUID
		}, nil
	}
	return nil, errors.New("invalid signing method")
}

func (a *jwtClient) CreateTokenId(params *CreateTokenParams) (string, error) {

	privateKey, err := readPrivateKey()
	if err != nil {
		return "", err
	}
	accessToken := jwt.New(jwt.SigningMethodRS256)

	claims := accessToken.Claims.(jwt.MapClaims)
	claims["userId"] = params.UserId
	accessTokenString, err := accessToken.SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return accessTokenString, nil
}

func (a *jwtClient) newToken(params *CreateTokenParams, lt time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{
		"exp":    time.Now().Add(lt).Unix(),
		"userId": params.UserId.String(),
	}

	tokenString, err := token.SignedString(a.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create signed string from token: %w", err)
	}

	return tokenString, nil
}

func readPrivateKey() (*rsa.PrivateKey, error) {
	privateKeyBytes, err := os.ReadFile("private.pem")
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing the private key")
	}

	// пробуем распарсить сначала как PKCS1
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		// Если получилось, возвращаем privateKey
		return privateKey, nil
	}

	// если не получилось, пробуем как PKCS8
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	// преобразуем и возвращаем как *rsa.PrivateKey
	return privateKeyInterface.(*rsa.PrivateKey), nil
}
