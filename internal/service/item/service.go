package item

import (
	"context"

	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
)

type Service struct {
	txManager     repository.TxManager
	itemRepo      repository.ItemRepository
	inventoryRepo repository.InventoryRepository
	userRepo      repository.UserRepository
}

func NewItemService(
	txManager repository.TxManager,
	itemRepo repository.ItemRepository,
	inventoryRepo repository.InventoryRepository,
	userRepo repository.UserRepository,
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

	return s.txManager.DoInTx(ctx, func(repos repository.TxRepositories) error {
		err = repos.User.UpdateBalance(ctx, userID, int(item.Cost))
		if err != nil {
			return err
		}

		err = repos.Inventory.AddItem(ctx, userID, item.ID, 1)
		if err != nil {
			return err
		}

		return nil
	})
}
