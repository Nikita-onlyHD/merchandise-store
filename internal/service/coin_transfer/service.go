package coin_transfer

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	UpdateBalance(ctx context.Context, userID int, amount int) error
	GetUser(ctx context.Context, login string) (*models.User, error)
	WithTx(tx pgx.Tx) UserRepository
}

type CoinTransferRepository interface {
	AddTransfer(ctx context.Context, coinTransfer *models.CoinTransfer) error
	WithTx(tx pgx.Tx) CoinTransferRepository
}

type Transactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Service struct {
	txManager        Transactor
	userRepo         UserRepository
	coinTransferRepo CoinTransferRepository
}

func NewCoinTransferService(txManager Transactor, userRepo UserRepository, coinTransferRepo CoinTransferRepository) *Service {
	return &Service{
		txManager:        txManager,
		userRepo:         userRepo,
		coinTransferRepo: coinTransferRepo,
	}
}

func (s *Service) SendCoins(ctx context.Context, fromUserID int, toUserLogin string, amount int) error {
	if amount <= 0 {
		return app_errors.ErrInvalidAmount
	}

	toUser, err := s.userRepo.GetUser(ctx, toUserLogin)
	if err != nil {
		if errors.Is(err, app_errors.ErrUserNotFound) {
			return app_errors.ErrUserNotFound
		}
		return fmt.Errorf("failed to find recipient: %w", err)
	}

	if fromUserID == toUser.ID {
		return app_errors.ErrSelfTransfer
	}

	tx, err := s.txManager.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	userRepoTx := s.userRepo.WithTx(tx)
	coinTransferRepoTx := s.coinTransferRepo.WithTx(tx)

	err = userRepoTx.UpdateBalance(ctx, fromUserID, amount)
	if err != nil {
		return err
	}

	err = userRepoTx.UpdateBalance(ctx, toUser.ID, -amount)
	if err != nil {
		return err
	}

	coinTransfer := models.CoinTransfer{
		FromUserID: fromUserID,
		ToUserID:   toUser.ID,
		Amount:     uint(amount),
	}
	err = coinTransferRepoTx.AddTransfer(ctx, &coinTransfer)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
