package item

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

func (m *mockTxManager) DoInTx(ctx context.Context, fn func(repos repository.TxRepositories) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(m.repos)
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

type mockUserRepo struct {
	updateBalanceFunc func(ctx context.Context, userID, amount int) error
}

func (m *mockUserRepo) UpdateBalance(ctx context.Context, userID int, amount int) error {
	if m.updateBalanceFunc != nil {
		return m.updateBalanceFunc(ctx, userID, amount)
	}
	return nil
}

func (m *mockUserRepo) GetUser(ctx context.Context, login string) (*models.User, error) {
	return nil, nil
}

func TestBuyItem_ItemNotFound(t *testing.T) {
	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return nil, apperr.ErrItemNotFound
		},
	}

	svc := NewItemService(nil, itemRepoMock, nil, nil)

	err := svc.BuyItem(context.Background(), "pen", 1)

	if !errors.Is(err, apperr.ErrItemNotFound) {
		t.Errorf("expected %v got %v", apperr.ErrItemNotFound, err)
	}
}

func TestBuyItem_InsufficientFunds(t *testing.T) {
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
			return apperr.ErrInsufficientFunds
		},
	}
	inventoryRepoMock := &mockInventoryRepo{}

	txManagerMock := &mockTxManager{
		repos: repository.TxRepositories{
			User:      userRepoMock,
			Inventory: inventoryRepoMock,
			Item:      itemRepoMock,
		},
	}

	svc := NewItemService(txManagerMock, itemRepoMock, inventoryRepoMock, userRepoMock)

	err := svc.BuyItem(context.Background(), mockItem.Name, 1)

	if !errors.Is(err, apperr.ErrInsufficientFunds) {
		t.Errorf("expected %v got %v", apperr.ErrInsufficientFunds, err)
	}
}

func TestBuyItem_Success(t *testing.T) {
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

	txManagerMock := &mockTxManager{
		repos: repository.TxRepositories{
			User:      userRepoMock,
			Inventory: inventoryRepoMock,
			Item:      itemRepoMock,
		},
	}

	svc := NewItemService(txManagerMock, itemRepoMock, inventoryRepoMock, userRepoMock)

	err := svc.BuyItem(context.Background(), mockItem.Name, 1)
	if err != nil {
		t.Errorf("expected no error got %v", err)
	}
}

func TestBuyItem_TxBeginError(t *testing.T) {
	expected := errors.New("db connection lost")

	txManagerMock := &mockTxManager{
		err: expected,
	}

	itemRepoMock := &mockItemRepo{
		getItemByNameFunc: func(ctx context.Context, name string) (*models.Item, error) {
			return &models.Item{ID: 1, Name: "pen", Cost: 10}, nil
		},
	}

	svc := NewItemService(txManagerMock, itemRepoMock, nil, nil)

	err := svc.BuyItem(context.Background(), "pen", 1)

	if !errors.Is(err, expected) {
		t.Errorf("expected %v got %v", expected, err)
	}
}
