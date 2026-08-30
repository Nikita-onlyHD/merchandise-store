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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("failed to get DATABASE_URL environment variable")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key"
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	pool, err := pgxpool.New(ctx, dbURL)
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
		jwtSecret,
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
		jwtSecret,
	)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      appHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("server is starting on port %s", port)
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
