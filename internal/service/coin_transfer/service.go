package coin_transfer

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidAmount = errors.New("amount must be positive")
	ErrSelfTransfer  = errors.New("cannot transfer coin to yourself")
)

type UserRepository interface {
	UpdateBalance(ctx context.Context, userID int, amount int) error
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

func (s *Service) SendCoins(ctx context.Context, fromUserID, toUserID, amount int) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if fromUserID == toUserID {
		return ErrSelfTransfer
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

	err = userRepoTx.UpdateBalance(ctx, toUserID, -amount)
	if err != nil {
		return err
	}

	coinTransfer := models.CoinTransfer{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
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
