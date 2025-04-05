package validator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	RequiredField string `validate:"required"`
	TagField      string `validate:"tag"`
	MaxField      string `validate:"max=5"`
	MinField      string `validate:"min=3"`
	LtField       int    `validate:"lt=10"`
	GteField      int    `validate:"gte=5"`
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		input      TestStruct
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Valid struct", // Валидная структура
			input: TestStruct{
				RequiredField: "value", TagField: "#tag", MaxField: "value", MinField: "val", LtField: 5, GteField: 5,
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "Missing required field", // Отсутствует обязательное поле
			input: TestStruct{
				TagField: "#tag", MaxField: "value", MinField: "val", LtField: 5, GteField: 5,
			},
			wantErr:    true,
			wantErrMsg: ErrFieldRequired + ": TestStruct.RequiredField",
		},
		{
			name: "Invalid tag field", // Недопустимое поле тега
			input: TestStruct{
				RequiredField: "value", TagField: "tag", MaxField: "value", MinField: "val", LtField: 5, GteField: 5,
			},
			wantErr:    true,
			wantErrMsg: ErrInvalidFormat + ": TestStruct.TagField",
		},
		{
			name: "Field exceeds max length", // Поле превышает максимальную длину
			input: TestStruct{
				RequiredField: "value", TagField: "#tag", MaxField: "toolong", MinField: "val", LtField: 5, GteField: 5,
			},
			wantErr:    true,
			wantErrMsg: ErrFieldExceedsMaxLen + ": TestStruct.MaxField",
		},
		{
			name: "Field below min length", // Поле короче минимальной длины
			input: TestStruct{
				RequiredField: "value", TagField: "#tag", MaxField: "value", MinField: "va", LtField: 5, GteField: 5,
			},
			wantErr:    true,
			wantErrMsg: ErrFieldBelowMinLen + ": TestStruct.MinField",
		},
		{
			name: "Field exceeds max value", // Поле превышает максимальное значение
			input: TestStruct{
				RequiredField: "value", TagField: "#tag", MaxField: "value", MinField: "val", LtField: 15, GteField: 5,
			},
			wantErr:    true,
			wantErrMsg: ErrFieldExceedsMaxVal + ": TestStruct.LtField",
		},
		{
			name: "Field below min value", // Поле меньше минимального значения
			input: TestStruct{
				RequiredField: "value", TagField: "#tag", MaxField: "value", MinField: "val", LtField: 5, GteField: 3,
			},
			wantErr:    true,
			wantErrMsg: ErrFieldBelowMinVal + ": TestStruct.GteField",
		},
	}
	// Цикл перебирает все элементы в слайсе tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(context.Background(), tt.input)
			if tt.wantErr {
				assert.NotNil(t, err)                    // Ошибка должна быть не нулевой (т.е. ошибка ДОЛЖНА БЫТЬ)
				assert.EqualError(t, err, tt.wantErrMsg) // Ошибка должна совпадать с ожидаемой ошибкой
			} else {
				assert.NoError(t, err) // Ошибки быть не должно
			}
		})
	}
}
