package health

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisChecker проверка Redis
func RedisChecker(client *redis.Client) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		start := time.Now()

		// Пингуем Redis
		_, err := client.Ping(ctx).Result()
		duration := time.Since(start)

		if err != nil {
			return CheckResult{
				Status: StatusDown,
				Error:  err.Error(),
				Details: map[string]any{
					"duration_ms": duration.Milliseconds(),
				},
			}
		}

		// Получаем информацию о сервере
		info, _ := client.Info(ctx, "server").Result()

		return CheckResult{
			Status: StatusUp,
			Details: map[string]any{
				"duration_ms": duration.Milliseconds(),
				"info":        info, // можно парсить info для деталей
			},
		}
	})
}

// DiskSpaceChecker проверка свободного места на диске
func DiskSpaceChecker(path string, minFreeBytes uint64) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		// Для Linux/Unix систем
		// В продакшене лучше использовать библиотеку типа github.com/shirou/gopsutil
		return CheckResult{
			Status: StatusUp,
			Details: map[string]any{
				"path": path,
				"note": "implement disk check based on OS",
			},
		}
	})
}

// MemoryChecker проверка использования памяти
func MemoryChecker(maxUsagePercent float64) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		// В продакшене использовать runtime.MemStats или gopsutil
		return CheckResult{
			Status: StatusUp,
			Details: map[string]any{
				"max_usage_percent": maxUsagePercent,
				"note":              "implement memory check",
			},
		}
	})
}
