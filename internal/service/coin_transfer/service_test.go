package coin_transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
)

type mockTx struct {
	pgx.Tx
	commitErr error
}

func (m *mockTx) Commit(ctx context.Context) error {
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	return nil
}

type mockTransactor struct {
	tx  pgx.Tx
	err error
}

func (m *mockTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	return m.tx, m.err
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

func (m *mockUserRepo) WithTx(tx pgx.Tx) UserRepository {
	return m
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

func (m *mockCoinTransferRepo) WithTx(tx pgx.Tx) CoinTransferRepository {
	return m
}

func TestSendCoins_InvalidAmount(t *testing.T) {
	svc := NewCoinTransferService(nil, nil, nil)

	err := svc.SendCoins(context.Background(), 1, "test_login1", -10)

	if !errors.Is(err, app_errors.ErrInvalidAmount) {
		t.Errorf("expected error %v got %v", app_errors.ErrInvalidAmount, err)
	}
}

func TestSendCoins_UserNotFound(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return nil, app_errors.ErrUserNotFound
		},
	}

	svc := NewCoinTransferService(nil, userRepoMock, nil)

	err := svc.SendCoins(context.Background(), 1, "test_login1", 10)

	if !errors.Is(err, app_errors.ErrUserNotFound) {
		t.Errorf("expected error %v got %v", app_errors.ErrUserNotFound, err)
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

	if !errors.Is(err, app_errors.ErrSelfTransfer) {
		t.Errorf("expected error %v got %v", app_errors.ErrSelfTransfer, err)
	}
}

func TestSendCoins_InsufficientFunds(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

	userRepoMock := &mockUserRepo{
		updateBalanceFunc: func(ctx context.Context, userID, amount int) error {
			return app_errors.ErrInsufficientFunds
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

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login1", 50)

	if !errors.Is(err, app_errors.ErrInsufficientFunds) {
		t.Errorf("expected %v got %v", app_errors.ErrInsufficientFunds, err)
	}
}

func TestSendCoins_Success(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

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

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login2", 50)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
}

func TestSendCoins_TxBeginError(t *testing.T) {
	expected := errors.New("db connection lost")

	txMock := &mockTx{}
	transactorMock := &mockTransactor{
		tx:  txMock,
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

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, "test_login2", 50)

	if !errors.Is(err, expected) {
		t.Errorf("expected %v got %v", expected, err)
	}
}
