package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	sessionPrefix   = "session:"
	blacklistPrefix = "blacklist:"
)

func (s *RedisStore) Close() error {
	return s.client.Close()
}

// CreateSession создает новую сессию в Redis
func (s *RedisStore) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	// Если время создания не установлено, устанавливаем текущее время
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	// Вычисляем TTL как разницу между временем истечения и текущим временем
	ttl := session.ExpiresAt.Sub(time.Now())
	if ttl <= 0 {
		return nil, fmt.Errorf("session expiration time must be in the future")
	}

	key := sessionPrefix + session.Id
	if err := s.client.Set(ctx, key, sessionData, ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to save session to Redis: %w", err)
	}

	return session, nil
}

// GetSession получает сессию из Redis по ID
func (s *RedisStore) GetSession(ctx context.Context, id string) (*Session, error) {
	key := sessionPrefix + id

	sessionData, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session %s not found", id)
		}
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &session, nil
}

// RevokeSession отзывает сессию, добавляя токен в черный список
func (s *RedisStore) RevokeSession(ctx context.Context, id string) error {
	// Получаем сессию
	session, err := s.GetSession(ctx, id)
	if err != nil {
		return err
	}

	// Добавляем refresh токен в черный список
	blacklistKey := blacklistPrefix + session.RefreshToken
	ttl := session.ExpiresAt.Sub(time.Now())
	if ttl <= 0 {
		ttl = time.Minute // Если токен уже истек, все равно добавляем его на короткое время
	}

	if err := s.client.Set(ctx, blacklistKey, "revoked", ttl).Err(); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	// Обновляем статус сессии
	session.IsRevoked = true

	// Сохраняем обновленную сессию
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session %w", err)
	}

	key := sessionPrefix + id
	if err := s.client.Set(ctx, key, sessionData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update session in Redis: %w", err)
	}

	return nil
}

// DeleteSession удаляет сессию из Redis
func (s *RedisStore) DeleteSession(ctx context.Context, id string) error {
	// Получаем сессию, чтобы добавить токен в черный список перед удалением
	session, err := s.GetSession(ctx, id)
	if err != nil {
		return err
	}

	// Добавляем refresh токен в черный список
	blacklistKey := blacklistPrefix + session.RefreshToken
	ttl := session.ExpiresAt.Sub(time.Now())
	if ttl <= 0 {
		ttl = time.Minute
	}

	if err := s.client.Set(ctx, blacklistKey, "deleted", ttl).Err(); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	// Удаляем сессию
	key := sessionPrefix + id
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}

// IsTokenBlacklisted проверяет, находится ли токен в черном списке
func (s *RedisStore) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + token
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if token is blacklisted: %w", err)
	}

	return exists > 0, nil
}
