package secure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{
			name:     "Too short",
			password: "Ab1%", // меньше 8 символов
			valid:    false,
		},
		{
			name:     "Too long",
			password: "Averylongpasswordwithmorethanthirtycharacters1%", // больше 30 символов
			valid:    false,
		},
		{
			name:     "Missing digit",
			password: "Password%", // нет цифры
			valid:    false,
		},
		{
			name:     "Missing uppercase",
			password: "password1%", // нет заглавной буквы
			valid:    false,
		},
		{
			name:     "Missing lowercase",
			password: "PASSWORD1%", // нет строчной буквы
			valid:    false,
		},
		{
			name:     "Missing symbol",
			password: "Password1", // нет спец. символа из набора
			valid:    false,
		},
		{
			name:     "Valid password",
			password: "ValidPass1%", // содержит все условия
			valid:    true,
		},
		{
			name:     "Edge case exactly 8 characters",
			password: "Aa1%aaaA", // 8 символов, содержит цифру, верхний и нижний регистр, символ
			valid:    true,
		},
		{
			name:     "Edge case exactly 30 characters",
			password: "Aa1%aaaaaaaaaaaaaaaaaaaaaaAAA", // 30 символов
			valid:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid, err := IsValidPassword(tc.password)

			// Проверяем, что валидность пароля соответствует ожидаемому результату
			assert.Equal(t, tc.valid, isValid)

			// Если пароль не валиден, проверяем, что ошибка не nil и сообщение соответствует ожидаемому
			if !tc.valid {
				assert.Error(t, err)
				assert.Equal(t, PasswordValidateErr, err.Error())
			} else {
				// Если пароль валиден, ошибка должна быть nil
				assert.NoError(t, err)
			}
		})
	}
}
