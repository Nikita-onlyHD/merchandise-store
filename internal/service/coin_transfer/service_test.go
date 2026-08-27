package coin_transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/Nikita-onlyHD/merchandise-store/internal/repository"
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
}

func (m *mockUserRepo) UpdateBalance(ctx context.Context, userID int, amount int) error {
	if m.updateBalanceFunc != nil {
		return m.updateBalanceFunc(ctx, userID, amount)
	}
	return nil
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

	err := svc.SendCoins(context.Background(), 1, 2, -10)

	if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected error %v got %v", ErrInvalidAmount, err)
	}
}

func TestSendCoins_SelfTransfer(t *testing.T) {
	svc := NewCoinTransferService(nil, nil, nil)

	err := svc.SendCoins(context.Background(), 1, 1, 10)

	if !errors.Is(err, ErrSelfTransfer) {
		t.Errorf("expected error %v got %v", ErrSelfTransfer, err)
	}
}

func TestSendCoins_Success(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

	userRepoMock := &mockUserRepo{}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, 2, 50)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
}

func TestSendCoins_InsufficientFunds(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

	userRepoMock := &mockUserRepo{
		updateBalanceFunc: func(ctx context.Context, userID, amount int) error {
			return repository.ErrInsufficientFunds
		},
	}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, 2, 50)

	if !errors.Is(err, repository.ErrInsufficientFunds) {
		t.Errorf("expected %v got %v", repository.ErrInsufficientFunds, err)
	}
}

func TestSendCoins_TxBeginError(t *testing.T) {
	expected := errors.New("db connection lost")

	txMock := &mockTx{}
	transactorMock := &mockTransactor{
		tx:  txMock,
		err: expected,
	}

	userRepoMock := &mockUserRepo{}
	coinTransferRepoMock := &mockCoinTransferRepo{}

	svc := NewCoinTransferService(transactorMock, userRepoMock, coinTransferRepoMock)

	err := svc.SendCoins(context.Background(), 1, 2, 50)

	if !errors.Is(err, expected) {
		t.Errorf("expected %v got %v", expected, err)
	}
}
