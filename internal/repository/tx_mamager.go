package repository

import (
	"context"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	UpdateBalance(ctx context.Context, userID int, amount int) error
	GetUser(ctx context.Context, login string) (*models.User, error)
}

type InventoryRepository interface {
	AddItem(ctx context.Context, userID int, itemID int, quantity uint) error
}

type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*models.Item, error)
}

type CoinTransferRepository interface {
	AddTransfer(ctx context.Context, coinTransfer *models.CoinTransfer) error
}

type TxRepositories struct {
	User         UserRepository
	Inventory    InventoryRepository
	Item         ItemRepository
	CoinTransfer CoinTransferRepository
}

type TxManager interface {
	DoInTx(ctx context.Context, fn func(repos TxRepositories) error) error
}

type txManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *txManager {
	return &txManager{pool: pool}
}

func (m *txManager) DoInTx(ctx context.Context, fn func(repos TxRepositories) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	repos := TxRepositories{
		User:         NewUserRepository(tx),
		Inventory:    NewInventoryRepository(tx),
		Item:         NewItemRepository(tx),
		CoinTransfer: NewCoinTransferRepository(tx),
	}

	if err := fn(repos); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
