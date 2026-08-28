package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
)

type itemRepository struct {
	db DBTX
}

func NewItemRepository(db DBTX) *itemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) WithTx(tx DBTX) *itemRepository {
	return &itemRepository{
		db: tx,
	}
}

func (r *itemRepository) GetItemByName(ctx context.Context, name string) (*models.Item, error) {
	var item models.Item

	err := r.db.QueryRow(ctx, "SELECT id, name, cost FROM items WHERE name = $1", name).Scan(
		&item.ID,
		&item.Name,
		&item.Cost,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app_errors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return &item, nil
}

func (r *itemRepository) GetAllItems(ctx context.Context) ([]models.Item, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, cost FROM items")
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer rows.Close()

	items := make([]models.Item, 0)
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Cost)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return items, nil
}
