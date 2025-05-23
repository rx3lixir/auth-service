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
	userSessionsIdx = "user_sessions:" // Новый префикс для индекса пользовательских сессий
)

// Close закрывает соединение с Redis
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// CreateSession создает новую сессию в Redis
func (s *RedisStore) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	// Проверяем обязательные поля
	if session.Id == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	if session.UserEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}

	if session.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	// Если время создания не установлено, устанавливаем текущее время
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	// Если время истечения не установлено, возвращаем ошибку
	if session.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("expiration time is required")
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

	// Добавляем sessionID в список сессий пользователя
	userSessionsKey := userSessionsIdx + session.UserEmail
	if err := s.client.SAdd(ctx, userSessionsKey, session.Id).Err(); err != nil {
		// Если не удалось добавить в индекс, удаляем созданную сессию
		s.client.Del(ctx, key)
		return nil, fmt.Errorf("failed to index session for user: %w", err)
	}

	// Устанавливаем TTL для индекса такой же, как для сессии
	if err := s.client.Expire(ctx, userSessionsKey, ttl).Err(); err != nil {
		// Если не удалось установить TTL для индекса, удаляем созданную сессию
		s.client.Del(ctx, key)
		s.client.SRem(ctx, userSessionsKey, session.Id)
		return nil, fmt.Errorf("failed to set expiration for user sessions index: %w", err)
	}

	return session, nil
}

// GetSession получает сессию из Redis по ID
func (s *RedisStore) GetSession(ctx context.Context, id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("session ID is required")
	}

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

// GetSessionsByEmail получает все активные сессии пользователя по email
func (s *RedisStore) GetSessionsByEmail(ctx context.Context, email string) ([]*Session, error) {
	if email == "" {
		return nil, fmt.Errorf("user email is required")
	}

	// Получаем список ID сессий пользователя
	userSessionsKey := userSessionsIdx + email
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*Session{}, nil // Возвращаем пустой список, если сессий нет
	}

	// Формируем ключи для команды MGET
	keys := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		keys[i] = sessionPrefix + id
	}

	// Получаем данные всех сессий одним запросом
	sessionDataList, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions data: %w", err)
	}

	// Десериализуем сессии
	sessions := make([]*Session, 0, len(sessionDataList))
	for _, sessionData := range sessionDataList {
		if sessionData == nil {
			continue // Пропускаем отсутствующие сессии
		}

		var session Session
		sessionStr, ok := sessionData.(string)
		if !ok {
			continue // Пропускаем неверные данные
		}

		if err := json.Unmarshal([]byte(sessionStr), &session); err != nil {
			continue // Пропускаем поврежденные данные
		}

		// Проверяем, что сессия действительно принадлежит запрашиваемому пользователю
		// и не отозвана
		if session.UserEmail == email && !session.IsRevoked {
			sessions = append(sessions, &session)
		}
	}

	return sessions, nil
}

// RevokeSession отзывает сессию, добавляя токен в черный список
func (s *RedisStore) RevokeSession(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("session ID is required")
	}

	// Получаем сессию
	session, err := s.GetSession(ctx, id)
	if err != nil {
		return err
	}

	// Проверяем, не отозвана ли сессия уже
	if session.IsRevoked {
		return fmt.Errorf("session is already revoked")
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
	if id == "" {
		return fmt.Errorf("session ID is required")
	}

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

	// Удаляем сессию из индекса пользователя
	userSessionsKey := userSessionsIdx + session.UserEmail
	if err := s.client.SRem(ctx, userSessionsKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove session from user index: %w", err)
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
	if token == "" {
		return false, fmt.Errorf("token is required")
	}

	key := blacklistPrefix + token
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if token is blacklisted: %w", err)
	}

	return exists > 0, nil
}
