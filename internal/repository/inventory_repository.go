package repository

import (
	"context"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
)

type inventoryRepository struct {
	db DBTX
}

func NewInventoryRepository(db DBTX) *inventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) WithTx(tx DBTX) *inventoryRepository {
	return &inventoryRepository{
		db: tx,
	}
}

func (r *inventoryRepository) GetInventoryByID(ctx context.Context, userID int) ([]models.InventoryItem, error) {
	rows, err := r.db.Query(ctx, "SELECT id, quantity, user_id, item_id FROM inventory WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	defer rows.Close()

	inventory := make([]models.InventoryItem, 0)
	for rows.Next() {
		var item models.InventoryItem
		err := rows.Scan(&item.ID, &item.Quantity, &item.UserID, &item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		inventory = append(inventory, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return inventory, nil
}

func (r *inventoryRepository) AddItem(ctx context.Context, userID int, itemID int, quantity uint) error {
	query := `INSERT INTO inventory (user_id, item_id, quantity)
			  VALUES ($1, $2, $3)
			  ON CONFLICT (user_id, item_id)
			  DO UPDATE SET quantity = inventory.quantity + $3`

	_, err := r.db.Exec(ctx, query, userID, itemID, quantity)
	if err != nil {
		return fmt.Errorf("failed to add item: %w", err)
	}

	return nil
}
