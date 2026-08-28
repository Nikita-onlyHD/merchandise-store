package item

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

type mockItemRepo struct {
	getItemByNameFunc func(ctx context.Context, name string) (*models.Item, error)
}

func (m *mockItemRepo) GetItemByName(ctx context.Context, name string) (*models.Item, error) {
	if m.getItemByNameFunc != nil {
		return m.getItemByNameFunc(ctx, name)
	}
	return nil, nil
}

type mockInventoryRepo struct {
	addItem func(ctx context.Context, userID int, itemID int, quantity uint) error
}

func (m *mockInventoryRepo) AddItem(ctx context.Context, userID int, itemID int, quantity uint) error {
	if m.addItem != nil {
		return m.addItem(ctx, userID, itemID, quantity)
	}
	return nil
}

func (m *mockInventoryRepo) WithTx(tx pgx.Tx) InventoryRepository {
	return m
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

func TestBuyItem_ItemNotFound(t *testing.T) {
	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return nil, app_errors.ErrItemNotFound
		},
	}

	svc := NewItemService(nil, itemRepoMock, nil, nil)

	err := svc.BuyItem(context.Background(), "pen", 1)

	if !errors.Is(err, app_errors.ErrItemNotFound) {
		t.Errorf("expected %v got %v", app_errors.ErrItemNotFound, err)
	}
}

func TestBuyItem_InsufficientFunds(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

	mockItem := &models.Item{
		ID:   1,
		Name: "pen",
		Cost: 10,
	}

	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return mockItem, nil
		},
	}
	userRepoMock := &mockUserRepo{
		updateBalanceFunc: func(ctx context.Context, userID, amount int) error {
			return app_errors.ErrInsufficientFunds
		},
	}
	inventoryRepoMock := &mockInventoryRepo{}

	svc := NewItemService(transactorMock, itemRepoMock, inventoryRepoMock, userRepoMock)

	err := svc.BuyItem(context.Background(), mockItem.Name, 1)

	if !errors.Is(err, app_errors.ErrInsufficientFunds) {
		t.Errorf("expected %v got %v", app_errors.ErrInsufficientFunds, err)
	}
}

func TestBuyItem_Success(t *testing.T) {
	txMock := &mockTx{}
	transactorMock := &mockTransactor{tx: txMock}

	mockItem := &models.Item{
		ID:   1,
		Name: "pen",
		Cost: 10,
	}

	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return mockItem, nil
		},
	}
	userRepoMock := &mockUserRepo{}
	inventoryRepoMock := &mockInventoryRepo{}

	svc := NewItemService(transactorMock, itemRepoMock, inventoryRepoMock, userRepoMock)

	err := svc.BuyItem(context.Background(), mockItem.Name, 1)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
}

func TestBuyItem_TxBeginError(t *testing.T) {
	expected := errors.New("db connection lost")

	txMock := &mockTx{}
	transactorMock := &mockTransactor{
		tx:  txMock,
		err: expected,
	}

	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return &models.Item{}, nil
		},
	}

	svc := NewItemService(transactorMock, itemRepoMock, nil, nil)

	err := svc.BuyItem(context.Background(), "pen", 1)

	if !errors.Is(err, expected) {
		t.Errorf("expected %v got %v", expected, err)
	}
}
