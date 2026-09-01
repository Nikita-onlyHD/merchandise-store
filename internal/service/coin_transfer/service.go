package transfer

import (
	"context"
	"errors"
	"fmt"

	apperr "github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
)

type Service struct {
	txManager        repository.TxManager
	userRepo         repository.UserRepository
	coinTransferRepo repository.CoinTransferRepository
}

func NewCoinTransferService(
	txManager repository.TxManager,
	userRepo repository.UserRepository,
	coinTransferRepo repository.CoinTransferRepository,
) *Service {
	return &Service{
		txManager:        txManager,
		userRepo:         userRepo,
		coinTransferRepo: coinTransferRepo,
	}
}

func (s *Service) SendCoins(ctx context.Context, fromUserID int, toUserLogin string, amount int) error {
	if amount <= 0 {
		return apperr.ErrInvalidAmount
	}

	toUser, err := s.userRepo.GetUser(ctx, toUserLogin)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return apperr.ErrUserNotFound
		}
		return fmt.Errorf("failed to find recipient: %w", err)
	}

	if fromUserID == toUser.ID {
		return apperr.ErrSelfTransfer
	}

	return s.txManager.DoInTx(ctx, func(repos repository.TxRepositories) error {
		err = repos.User.UpdateBalance(ctx, fromUserID, amount)
		if err != nil {
			return err
		}

		err = repos.User.UpdateBalance(ctx, toUser.ID, -amount)
		if err != nil {
			return err
		}

		coinTransfer := models.CoinTransfer{
			FromUserID: fromUserID,
			ToUserID:   toUser.ID,
			Amount:     amount,
		}

		err = repos.CoinTransfer.AddTransfer(ctx, &coinTransfer)
		if err != nil {
			return err
		}

		return nil
	})
}
