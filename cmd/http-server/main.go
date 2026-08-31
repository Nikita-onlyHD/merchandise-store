package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nikita-onlyHD/merchandise-store/internal/config"
	"github.com/Nikita-onlyHD/merchandise-store/internal/handler"
	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"

	coinService "github.com/Nikita-onlyHD/merchandise-store/internal/service/coin_transfer"
	itemService "github.com/Nikita-onlyHD/merchandise-store/internal/service/item"
	userService "github.com/Nikita-onlyHD/merchandise-store/internal/service/user"

	coinHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/coin_transfer"
	itemHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/item"
	userHandler "github.com/Nikita-onlyHD/merchandise-store/internal/handler/user"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application runtime error: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping db: %w", err)
	}

	log.Println("connected to db successfully")

	txManager := repository.NewTxManager(pool)

	userRepo := repository.NewUserRepository(pool)
	itemRepo := repository.NewItemRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	coinTransferRepo := repository.NewCoinTransferRepository(pool)

	userSvc := userService.NewUserService(
		userRepo,
		inventoryRepo,
		coinTransferRepo,
		cfg.JWTSecret,
	)

	coinTransferSvc := coinService.NewCoinTransferService(
		txManager,
		userRepo,
		coinTransferRepo,
	)

	itemSvc := itemService.NewItemService(
		txManager,
		itemRepo,
		inventoryRepo,
		userRepo,
	)

	userH := userHandler.NewUserHandler(userSvc)
	coinTransferH := coinHandler.NewCoinTransferHandler(coinTransferSvc)
	itemH := itemHandler.NewItemHandler(itemSvc)

	mux := http.NewServeMux()
	appHandler := handler.RegisterRoutes(
		mux,
		userH,
		itemH,
		coinTransferH,
		cfg.JWTSecret,
	)

	mux.HandleFunc("GET /health", handler.HealthCheckHandler)
	mux.HandleFunc("GET /ready", handler.ReadinessHandler(pool))

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      appHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("server is starting on port %s", cfg.ServerPort)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server launch error: %w", err)
		}

	case sig := <-shutdown:
		log.Printf("start graceful shutdown, signal: %v", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("could not stop server gracefully: %v, force closing", err)
			_ = server.Close()
		}
	}

	log.Println("server stopped successfully")
	return nil
}
