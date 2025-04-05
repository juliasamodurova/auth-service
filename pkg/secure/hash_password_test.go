package secure

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	plainPassword := "SuperSecret123!"

	// Хешируем пароль
	hashed, err := HashPassword(plainPassword)
	assert.NoError(t, err, "Error hashing password")

	// Убедимся, что хеш не равен исходному паролю
	assert.NotEqual(t, plainPassword, hashed, "Hashed password should not equal the plain password")

	// Проверяем, что хеш начинается с ожидаемого префикса bcrypt
	assert.True(t, strings.HasPrefix(hashed, "$2a$") || strings.HasPrefix(hashed, "$2b$") || strings.HasPrefix(hashed, "$2y$"),
		"Hashed password has unexpected prefix: %s", hashed[:4])

	// Проверяем, что функция CheckPassword возвращает nil для корректного пароля
	err = CheckPassword(hashed, plainPassword)
	assert.NoError(t, err, "CheckPassword failed for correct password")

	// Проверяем, что для неверного пароля возвращается ошибка
	err = CheckPassword(hashed, "WrongPassword")
	assert.Error(t, err, "CheckPassword succeeded for wrong password")
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	// Передаем некорректный хеш
	invalidHash := "not_a_valid_hash"
	err := CheckPassword(invalidHash, "anything")
	assert.Error(t, err, "Expected error when comparing with an invalid hash")
}

func TestHashAndCheckPassword_EmptyPassword(t *testing.T) {
	// Проверка на пустой пароль
	plainPassword := ""

	hashed, err := HashPassword(plainPassword)
	assert.NoError(t, err, "Error hashing empty password")

	err = CheckPassword(hashed, plainPassword)
	assert.NoError(t, err, "CheckPassword failed for empty password")
}
