package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	authPb "github.com/rx3lixir/auth-service/auth-grpc/gen/go"
	"github.com/rx3lixir/auth-service/auth-grpc/server"
	"github.com/rx3lixir/auth-service/internal/config"
	"github.com/rx3lixir/auth-service/internal/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Настраиваем логирование
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Создаем контекст, который можно отменить при получении сигнала остановки
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов для грациозного завершения
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		slog.Info("Shutting down gracefully...")
		cancel()
	}()

	// Загрузка конфигурации
	c, err := config.New()
	if err != nil {
		slog.Error("error creating config file", "error", err)
		os.Exit(1)
	}

	// Создание Redis хранилища
	redisStore, err := db.NewRedisStore(c.Redis.RedisURL(), ctx)
	if err != nil {
		slog.Error("Failed to initialize Redis store", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	slog.Info("Successfultty connected to Redis")

	// Создание grpc сервера
	grpcServer := grpc.NewServer()
	authServer := server.NewServer(redisStore)
	authPb.RegisterAuthServer(grpcServer, authServer)

	// Включение reflection для отладки
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", c.Server.Address)
	if err != nil {
		slog.Error("Failed to start listener", "error", err)
		os.Exit(1)
	}

	slog.Info("Server is listening", "address", c.Server.Address)

	// Запускаем сервер в горутине
	serverError := make(chan error, 1)
	go func() {
		serverError <- grpcServer.Serve(listener)
	}()

	// Ждем либо завершения контекста (по сигналу), либо ошибки сервера
	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		slog.Info("Server stopped gracefully")
	case err := <-serverError:
		slog.Error("Server error", "error", err)
	}

}
