package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authPb "github.com/rx3lixir/auth-service/auth-grpc/gen/go"
	"github.com/rx3lixir/auth-service/auth-grpc/server"
	"github.com/rx3lixir/auth-service/internal/config"
	"github.com/rx3lixir/auth-service/internal/db"
	"github.com/rx3lixir/auth-service/pkg/health"
	"github.com/rx3lixir/auth-service/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Загрузка конфигурации
	c, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	logger.Init(c.Service.Env)
	defer logger.Close()

	log := logger.NewLogger()

	// Создаем контекст, который можно отменить при получении сигнала остановки
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов для грациозного завершения
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Логаем инфу для отладки
	log.Info("Configuration loaded",
		"env", c.Service.Env,
		"redis_url", c.Redis.URL,
		"server_address", c.Server.Address,
	)

	// Создание Redis хранилища
	redisStore, err := db.NewRedisStore(c.Redis.RedisURL(), ctx)
	if err != nil {
		log.Error("Failed to initialize Redis store", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	log.Info("Successfully connected to Redis")

	// Создание grpc сервера
	grpcServer := grpc.NewServer()
	authServer := server.NewServer(redisStore, log, c)
	authPb.RegisterAuthServiceServer(grpcServer, authServer)

	// Включение reflection для отладки
	reflection.Register(grpcServer)

	// Запуск gRPC сервера
	listener, err := net.Listen("tcp", c.Server.Address)
	if err != nil {
		log.Error("Failed to start listener", "error", err)
		os.Exit(1)
	}

	log.Info("Server is listening", "address", c.Server.Address)

	// Создаем HealthCheck сервер
	healthServer := health.NewServer(redisStore, log,
		health.WithServiceName("auth-service"),
		health.WithVersion("1.0.0"),
		health.WithPort(":8082"),
		health.WithTimeout(5*time.Second),
	)

	// Запускаем серверы
	errCh := make(chan error, 2)

	// Health check сервер
	go func() {
		errCh <- healthServer.Start()
	}()

	// gRPC сервер
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	// Ждем завершения
	select {
	case <-signalCh:
		log.Info("Shutting down gracefully...")

		// Останавливаем серверы
		grpcServer.GracefulStop()
		if err := healthServer.Shutdown(context.Background()); err != nil {
			log.Error("Health server shutdown error", "error", err)
		}

	case err := <-errCh:
		log.Error("Server error", "error", err)

		grpcServer.GracefulStop()
		if err := healthServer.Shutdown(context.Background()); err != nil {
			log.Error("Health server shutdown error", "error", err)
		}
	}

	log.Info("Server stopped gracefully")
}
