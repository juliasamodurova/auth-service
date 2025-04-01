package jwt

import "github.com/google/uuid"

type GetDataFromTokenParams struct { // используются для извлечения данных из JWT-токена после валидации, сам токен
	Token string
}

type GetDataFromTokenResponse struct { // результат (ID пользователя, закодированный в токене)
	UserId uuid.UUID `json:"userId"`
}
type CreateTokenParams struct { // генерация новой пары токенов (access + refresh)
	UserId uuid.UUID `json:"userId"` // входные параметры (ID пользователя)
}

type CreateTokenResponse struct { // сгенерированные токены
	AccessToken  string
	RefreshToken string
}

type ValidateTokenParams struct { // проверяет валидность токена (не истёк ли, корректная ли подпись)
	Token string
}
