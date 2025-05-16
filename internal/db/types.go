package db

import (
	"time"
)

type Session struct {
	Id           string
	UserEmail    string
	RefreshToken string
	IsRevoked    bool
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type RenewAccessTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

type RenewAccessTokenRes struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

type LoginUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserRes struct {
	SessionID             string     `json:"session_id"`
	AccessToken           string     `json:"access_token"`
	RefreshToken          string     `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time  `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time  `json:"refresh_token_expires_at"`
	User                  GetUserRes `json:"user"`
}

type GetUserRes struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}
