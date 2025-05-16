package db

import (
	"time"
)

// Session представляет сессию пользователя
type Session struct {
	Id           string    // ID сессии
	UserEmail    string    // Email пользователя
	RefreshToken string    // Refresh токен
	IsRevoked    bool      // Флаг отзыва сессии
	CreatedAt    time.Time // Время создания сессии
	ExpiresAt    time.Time // Время истечения сессии
}

// RenewAccessTokenReq запрос на обновление access токена
type RenewAccessTokenReq struct {
	RefreshToken string `json:"refresh_token"` // Refresh токен
}

// RenewAccessTokenRes ответ с новым access токеном
type RenewAccessTokenRes struct {
	AccessToken          string    `json:"access_token"`            // Новый access токен
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"` // Время истечения токена
}

// LoginUserReq запрос на аутентификацию пользователя
type LoginUserReq struct {
	Email    string `json:"email"`    // Email пользователя
	Password string `json:"password"` // Пароль пользователя
}

// LoginUserRes ответ с данными для аутентифицированного пользователя
type LoginUserRes struct {
	SessionId             string     `json:"session_id"`               // ID сессии
	AccessToken           string     `json:"access_token"`             // Access токен
	RefreshToken          string     `json:"refresh_token"`            // Refresh токен
	AccessTokenExpiresAt  time.Time  `json:"access_token_expires_at"`  // Время истечения access токена
	RefreshTokenExpiresAt time.Time  `json:"refresh_token_expires_at"` // Время истечения refresh токена
	User                  GetUserRes `json:"user"`                     // Информация о пользователе
}

// GetUserRes базовая информация о пользователе
type GetUserRes struct {
	Name    string `json:"name"`     // Имя пользователя
	Email   string `json:"email"`    // Email пользователя
	IsAdmin bool   `json:"is_admin"` // Флаг администратора
}
