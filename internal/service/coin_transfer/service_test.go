package transfer

import (
	"context"
	"errors"
	"testing"

	apperr "github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
)

type mockTxManager struct {
	repos repository.TxRepositories
	err   error
}

func (m *mockTxManager) DoInTx(_ context.Context, fn func(repos repository.TxRepositories) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(m.repos)
}

type mockUserRepo struct {
	updateBalanceFunc func(ctx context.Context, userID, amount int) error
	getUser           func(ctx context.Context, login string) (*models.User, error)
}

func (m *mockUserRepo) UpdateBalance(ctx context.Context, userID int, amount int) error {
	if m.updateBalanceFunc != nil {
		return m.updateBalanceFunc(ctx, userID, amount)
	}
	return nil
}

func (m *mockUserRepo) GetUser(ctx context.Context, login string) (*models.User, error) {
	if m.getUser != nil {
		return m.getUser(ctx, login)
	}
	return nil, nil
}

type mockCoinTransferRepo struct {
	addTransferFunc func(ctx context.Context, coinTransfer *models.CoinTransfer) error
}

func (m *mockCoinTransferRepo) AddTransfer(ctx context.Context, coinTransfer *models.CoinTransfer) error {
	if m.addTransferFunc != nil {
		return m.addTransferFunc(ctx, coinTransfer)
	}
	return nil
}

func (m *mockCoinTransferRepo) GetTransferHistory(ctx context.Context, userID int) ([]models.CoinTransfer, error) {
	return nil, nil
}

func TestSendCoins_InvalidAmount(t *testing.T) {
	svc := NewCoinTransferService(nil, nil, nil)

	err := svc.SendCoins(context.Background(), 1, "test_login1", -10)

	if !errors.Is(err, apperr.ErrInvalidAmount) {
		t.Errorf("expected error %v got %v", apperr.ErrInvalidAmount, err)
	}
}

func TestSendCoins_UserNotFound(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return nil, apperr.ErrUserNotFound
		},
	}

	svc := NewCoinTransferService(nil, userRepoMock, nil)

	err := svc.SendCoins(context.Background(), 1, "test_login1", 10)

	if !errors.Is(err, apperr.ErrUserNotFound) {
		t.Errorf("expected error %v got %v", apperr.ErrUserNotFound, err)
	}
}

func TestSendCoins_SelfTransfer(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &models.User{
				ID:       1,
				Login:    "test_login1",
				Password: "test_password",
				Balance:  100,
			}, nil
		},
	}

	svc := NewCoinTransferService(nil, userRepoMock, nil)

	err := svc.SendCoins(context.Background(), 1, "test_login1", 10)

	if !errors.Is(err, apperr.ErrSelfTransfer) {
		t.Errorf("expected error %v got %v", apperr.ErrSelfTransfer, err)
	}
}

func TestSendCoins_InsufficientFunds(t *testing.T) {
	userRepoMock := &mockUserRepo{
		updateBalanceFunc: func(ctx context.Context, userID, amount int) error {
			return apperr.ErrInsufficientFunds
		},
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &models.User{
				ID:       2,
				Login:    "test_login2",
				Password: "test_password",
				Balance:  100,
			}, nil
		},
	}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	txManagerMock := &mockTxManager{
		repos: repository.TxRepositories{
			User:         userRepoMock,
			CoinTransfer: coinTransferRepoMock,
		},
	}

	svc := NewCoinTransferService(txManagerMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login1", 50)

	if !errors.Is(err, apperr.ErrInsufficientFunds) {
		t.Errorf("expected %v got %v", apperr.ErrInsufficientFunds, err)
	}
}

func TestSendCoins_Success(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &models.User{
				ID:       2,
				Login:    "test_login2",
				Password: "test_password",
				Balance:  100,
			}, nil
		},
	}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	txManagerMock := &mockTxManager{
		repos: repository.TxRepositories{
			User:         userRepoMock,
			CoinTransfer: coinTransferRepoMock,
		},
	}

	svc := NewCoinTransferService(txManagerMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login2", 50)
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
}

func TestSendCoins_TxBeginError(t *testing.T) {
	expected := errors.New("db connection lost")

	txManagerMock := &mockTxManager{
		err: expected,
	}

	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &models.User{
				ID:       2,
				Login:    "test_login2",
				Password: "test_password",
				Balance:  100,
			}, nil
		},
	}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	svc := NewCoinTransferService(txManagerMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login2", 50)

	if !errors.Is(err, expected) {
		t.Errorf("expected %v got %v", expected, err)
	}
}
