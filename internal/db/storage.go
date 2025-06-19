package db

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// SessionStorage определяет интерфейс для работы с сессиями
type SessionStorage interface {
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	GetSession(ctx context.Context, id string) (*Session, error)
	GetSessionsByEmail(ctx context.Context, email string) ([]*Session, error)
	RevokeSession(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id string) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	Close() error
}

// RedisStore реализует методы SessionStorage
type RedisStore struct {
	client *redis.Client
}

// GetClient возвращает Redis клиент (для health checks)
func (s *RedisStore) GetClient() *redis.Client {
	return s.client
}

// NewRedisStore создает новое хранилище Redis
func NewRedisStore(redisURL string, ctx context.Context) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	// Проверка соединения с Redis
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisStore{
		client: client,
	}, nil
}
