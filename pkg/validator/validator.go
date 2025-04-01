package validator

import (
	"context"
	"errors"
	"regexp"

	"github.com/go-playground/validator"
)

// Пакет валидации для входных данных
var global *validator.Validate

// константы ошибок
const (
	ErrInvalidFormat      = "Invalid format"
	ErrFieldRequired      = "Field is required"
	ErrFieldExceedsMaxLen = "Field exceeds maximum length"
	ErrFieldBelowMinLen   = "Field is below minimum length"
	ErrFieldExceedsMaxVal = "Field exceeds maximum value"
	ErrFieldBelowMinVal   = "Field is below minimum value"
	ErrUnknownValidation  = "Unknown validation error"
)

// инициализация пакета и создание нового валидатора, установка его в качестве глобального
func init() {
	SetValidator(New())
}

// создаётся новый экземпляр валидатора
func New() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("tag", validateTag)

	return v
}

// функции для установки и получения глобального валидатора
func SetValidator(v *validator.Validate) {
	global = v
}

func Validator() *validator.Validate {
	return global
}

// валидация тегов, проверяет, соответствует ли строка формату хэштега
func validateTag(fl validator.FieldLevel) bool {
	re, _ := regexp.Compile(`^#[a-z0-9_\-]+$`)
	return re.MatchString(fl.Field().String())
}

// основная функция валидации: принимает контекст и структуру для валидации,
// возвращает ошибку если валидация не прошла
func Validate(ctx context.Context, structure any) error {
	return parseValidationErrors(Validator().StructCtx(ctx, structure))
}

// обработка ошибок
func parseValidationErrors(err error) error {
	if err == nil {
		return nil
	}

	vErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(vErrors) == 0 {
		return nil
	}

	validationError := vErrors[0]
	var validationErrorDescription string
	switch validationError.Tag() {
	case "tag":
		validationErrorDescription = ErrInvalidFormat
	case "required":
		validationErrorDescription = ErrFieldRequired
	case "max":
		validationErrorDescription = ErrFieldExceedsMaxLen
	case "min":
		validationErrorDescription = ErrFieldBelowMinLen
	case "lt", "lte":
		validationErrorDescription = ErrFieldExceedsMaxVal
	case "gt", "gte":
		validationErrorDescription = ErrFieldBelowMinVal
	default:
		validationErrorDescription = ErrUnknownValidation
	}

	return errors.New(validationErrorDescription + ": " + validationError.Namespace())
}
