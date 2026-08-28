package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

type coinTransferRepository struct {
	db DBTX
}

func NewCoinTransferRepository(db DBTX) *coinTransferRepository {
	return &coinTransferRepository{db: db}
}

func (r *coinTransferRepository) WithTx(tx DBTX) *coinTransferRepository {
	return &coinTransferRepository{
		db: tx,
	}
}

func (r *coinTransferRepository) AddTransfer(ctx context.Context, coinTransfer *models.CoinTransfer) error {
	query := `INSERT INTO coin_transfers (from_user_id, to_user_id, amount) 
			  VALUES ($1, $2, $3) RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		coinTransfer.FromUserID,
		coinTransfer.ToUserID,
		coinTransfer.Amount,
	).Scan(&coinTransfer.ID, &coinTransfer.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return app_errors.ErrUserNotFound
		}
		return fmt.Errorf("failed to add coin transfer: %w", err)
	}

	return nil
}

func (r *coinTransferRepository) GetTransferHistory(ctx context.Context, userID int) ([]models.CoinTransferDetail, error) {
	query := `SELECT t.id, sender.login AS from_user, receiver.login AS to_user, t.amount, t.created_at 
			  FROM coin_transfers t
			  JOIN users sender ON t.from_user_id = sender.id
			  JOIN users receiver ON t.to_user_id = receiver.id
			  WHERE t.from_user_id = $1 OR t.to_user_id = $1
			  ORDER BY t.created_at;`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin transfer history: %w", err)
	}
	defer rows.Close()

	history := make([]models.CoinTransferDetail, 0)
	for rows.Next() {
		var coinTransfer models.CoinTransferDetail
		err := rows.Scan(
			&coinTransfer.ID,
			&coinTransfer.FromUser,
			&coinTransfer.ToUser,
			&coinTransfer.Amount,
			&coinTransfer.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		history = append(history, coinTransfer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return history, nil
}
