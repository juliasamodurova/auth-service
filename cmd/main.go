package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	AuthService "newservice/grpc/genproto"
	"newservice/internal/config"
	"newservice/internal/repo"
	"newservice/internal/service"
	"newservice/pkg/jwt"
	"newservice/pkg/logger"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
)

func main() {

	// загрузка переменных окружения из .env файла
	if err := godotenv.Load("local.env"); err != nil {
		log.Println("Error loading local.env file")
	}

	// загрузка конфигурации
	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// инициализация логгера
	l, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize l: %v", err)
	}
	defer l.Sync()

	// инициализация и подключение к БД
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repository, err := repo.NewRepository(ctx, cfg.PostgreSQL)
	if err != nil {
		l.Fatalf("failed to initialize repository: %v", err)
	}

	// чтение ключей для JWT
	privateKey, err := jwt.ReadPrivateKey()
	if err != nil {
		log.Fatal("failed to read private key")
	}
	publicKey, err := jwt.ReadPublicKey()
	if err != nil {
		log.Fatal("failed to read public key")
	}

	// создание JWT-клиента
	jwtClient := jwt.NewJWTClient(privateKey, publicKey, cfg.System.AccessTokenTimeout, cfg.System.RefreshTokenTimeout)

	// создание сервера аутентификации
	authSrv := service.NewAuthServer(cfg, repository, jwtClient, l)

	// настройка и запуск gRPC-сервера:
	grpcServer := grpc.NewServer()
	AuthService.RegisterAuthServiceServer(grpcServer, authSrv)

	lis, err := net.Listen("tcp", cfg.GRPC.ListenAddress)
	if err != nil {
		l.Fatalf("failed to listen on %s: %v", cfg.GRPC.ListenAddress, err)
	}

	// асинхронный запуск сервера
	go func() {
		l.Infof("gRPC server started on %s", cfg.GRPC.ListenAddress)
		if err := grpcServer.Serve(lis); err != nil {
			l.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("Shutting down gRPC server...")

	// создаём контекст с таймаутом для graceful shutdown
	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// вызов GracefulStop - аналог ShutdownWithContext
	grpcServer.GracefulStop()
	l.Info("gRPC server stopped gracefully")

	l.Info("Closing database connection gracefully...")
	if err := repository.Close(); err != nil {
		l.Fatalf("error shutting down database: %v", err)
	}

	l.Info("Server stopped gracefully")
}
