package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// generateTestKeys генерирует пару RSA-ключей для тестов.
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

// TestCreateToken проверяет, что метод CreateToken возвращает непустые токены,
// а также что созданный access token успешно проходит валидацию и из него извлекается правильный userID
func TestCreateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	accessDur := 5 * time.Minute
	refreshDur := 24 * time.Hour

	client := NewJWTClient(privateKey, publicKey, accessDur, refreshDur)
	userID := uuid.New()
	params := &CreateTokenParams{UserId: userID}

	// Создаем токены
	resp, err := client.CreateToken(params)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)

	// Проверяем, что access token валиден
	valid, err := client.ValidateToken(&ValidateTokenParams{Token: resp.AccessToken})
	assert.NoError(t, err)
	assert.True(t, valid)

	// Извлекаем данные из access token
	data, err := client.GetDataFromToken(&GetDataFromTokenParams{Token: resp.AccessToken})
	assert.NoError(t, err)
	assert.Equal(t, userID, data.UserId)
}

// TestValidateToken_Expired проверяет, что при создании токена с отрицательным сроком действия
// валидация возвращает false
func TestValidateToken_Expired(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)

	// Задаем длительность токена так, чтобы он уже был просрочен
	// здесь срок жизни токена = 1 с
	client := NewJWTClient(privateKey, publicKey, time.Second, time.Second)

	userID := uuid.New()
	params := &CreateTokenParams{UserId: userID}

	// Передаем отрицательную длительность для создания просроченного токена.
	tokenStr, err := client.newToken(params, -time.Minute)
	assert.NoError(t, err)

	valid, err := client.ValidateToken(&ValidateTokenParams{Token: tokenStr})
	// Токен просрочен - валидация должна вернуть false
	assert.Error(t, err)
	assert.False(t, valid)
}

// TestGetDataFromToken_Invalid проверяет, что метод GetDataFromToken возвращает ошибку для некорректного токена.
func TestGetDataFromToken_Invalid(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	client := NewJWTClient(privateKey, publicKey, time.Minute, time.Hour)

	_, err := client.GetDataFromToken(&GetDataFromTokenParams{Token: "invalid.token.string"})
	assert.Error(t, err)
}

// TestValidateToken_InvalidSignature проверяет, что токен, подписанный одним ключом,
// не проходит валидацию при проверке другим публичным ключом.
func TestValidateToken_InvalidSignature(t *testing.T) {
	// Генерируем две разные пары ключей.
	privateKey1, publicKey1 := generateTestKeys(t)
	privateKey2, publicKey2 := generateTestKeys(t)

	// Клиент с первой парой ключей создаёт токен.
	client1 := NewJWTClient(privateKey1, publicKey1, time.Minute, time.Hour)
	userID := uuid.New()
	params := &CreateTokenParams{UserId: userID}
	resp, err := client1.CreateToken(params)
	assert.NoError(t, err)

	// Клиент со второй парой ключей пытается валидировать токен.
	client2 := NewJWTClient(privateKey2, publicKey2, time.Minute, time.Hour)
	valid, err := client2.ValidateToken(&ValidateTokenParams{Token: resp.AccessToken})
	// Ожидаем, что валидация не пройдет.
	assert.Error(t, err)
	assert.False(t, valid)
}
