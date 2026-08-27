package item

import (
	"context"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
)

type ItemRepository interface {
	GetItemByName(ctx context.Context, name string) (*models.Item, error)
}

type InventoryRepository interface {
	AddItem(ctx context.Context, userID int, itemID int, quantity uint) error
	WithTx(tx pgx.Tx) InventoryRepository
}

type UserRepository interface {
	UpdateBalance(ctx context.Context, userID int, amount int) error
	WithTx(tx pgx.Tx) UserRepository
}

type Transactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Service struct {
	txManager     Transactor
	itemRepo      ItemRepository
	inventoryRepo InventoryRepository
	userRepo      UserRepository
}

func NewItemService(
	txManager Transactor,
	itemRepo ItemRepository,
	inventoryRepo InventoryRepository,
	userRepo UserRepository,
) *Service {
	return &Service{
		txManager:     txManager,
		itemRepo:      itemRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
	}
}

func (s *Service) BuyItem(ctx context.Context, itemName string, userID int) error {
	item, err := s.itemRepo.GetItemByName(ctx, itemName)
	if err != nil {
		return err
	}

	tx, err := s.txManager.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	userRepoTx := s.userRepo.WithTx(tx)
	inventoryRepoTx := s.inventoryRepo.WithTx(tx)

	err = userRepoTx.UpdateBalance(ctx, userID, int(item.Cost))
	if err != nil {
		return err
	}

	err = inventoryRepoTx.AddItem(ctx, userID, item.ID, 1)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
