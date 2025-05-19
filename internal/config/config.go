package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Константы для ключей конфигурации
const (
	envKey                = "service_params.env"
	secretKey             = "server_params.secret_key"
	redisURLKey           = "redis_params.url"
	redisPasswordKey      = "redis_params.password"
	serviceAddress        = "server_params.address"
	sessionTTLDaysKey     = "service_params.session_ttl_days"
	accessTokenTTLMinsKey = "service_params.access_token_ttl_mins"
)

// AppConfig представляет конфигурацию всего приложения
type AppConfig struct {
	Service ServiceParams `mapstructure:"service_params" validate:"required"`
	Server  ServerParams  `mapstructure:"server_params" validate:"required"`
	Redis   RedisParams   `mapstructure:"redis_params" validate:"required"`
}

// ApplicationParams содержит общие параметры приложения
type ServiceParams struct {
	Env                string `mapstructure:"env" validate:"required,oneof=dev prod test"`
	SessionTTLDays     int    `mapstructure:"session_ttl_days" validate:"required,min=1,max=30"`
	AccessTokenTTLMins int    `mapstructure:"access_token_ttl_mins" validate:"required,min=5,max=60"`
}

type ServerParams struct {
	Address   string `mapstructure:"address" validate:"required"`
	SecretKey string `mapstructure:"secret_key" validate:"required"`
}

type RedisParams struct {
	URL      string `mapstructure:"url" validate:"required"`
	Password string `mapstructure:"password"`
}

// RedisURL формирует полный URL для подключения к Redis
func (r *RedisParams) RedisURL() string {
	if r.Password != "" {
		// Если URL уже содержит схему, добавляем пароль
		if len(r.URL) > 6 && r.URL[:6] == "redis:" {
			return fmt.Sprintf("redis://:%s@%s", r.Password, r.URL[8:])
		}
		return fmt.Sprintf("redis://:%s@%s", r.Password, r.URL)
	}

	// Если URL уже содержит схему, возвращаем как есть
	if len(r.URL) > 6 && r.URL[:6] == "redis:" {
		return r.URL
	}

	return fmt.Sprintf("redis://%s", r.URL)
}

// GetSessionTTL возвращает время жизни сессии в виде Duration
func (s *ServiceParams) GetSessionTTL() time.Duration {
	return time.Hour * 24 * time.Duration(s.SessionTTLDays)
}

// GetAccessTokenTTL возвращает время жизни access токена в виде Duration
func (s *ServiceParams) GetAccessTokenTTL() time.Duration {
	return time.Minute * time.Duration(s.AccessTokenTTLMins)
}

// EnvBindings возвращает мапу ключей конфигурации и соответствующих им переменных окружения
func envBindings() map[string]string {
	return map[string]string{
		envKey:                "SERVICE_KEY",
		serviceAddress:        "SERVICE_ADDRESS",
		secretKey:             "SECRET_KEY",
		redisURLKey:           "REDIS_URL",
		redisPasswordKey:      "REDIS_PASSWORD",
		sessionTTLDaysKey:     "SESSION_TTL_DAYS",
		accessTokenTTLMinsKey: "ACCESS_TOKEN_TTL_MINS",
	}
}

// New загружает конфигурацию из файла и переменных окружения
func New() (*AppConfig, error) {
	v := viper.New()

	// Получаем рабочую директорию
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить рабочую директорию: %w", err)
	}

	v.AddConfigPath(filepath.Join(cwd, "internal", "config"))
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	// Привязка переменных окружения
	for configKey, envVar := range envBindings() {
		if err := v.BindEnv(configKey, envVar); err != nil {
			return nil, fmt.Errorf("ошибка привязки переменной окружения %s: %w", envVar, err)
		}
	}

	// Чтение конфигурации
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
	}

	var config AppConfig

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании конфигурации: %w", err)
	}

	// Валидация конфигурации
	validate := validator.New()

	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return &config, nil
}
