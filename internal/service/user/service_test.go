package user

import (
	"context"
	"errors"
	"testing"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	getUser     func(ctx context.Context, login string) (*models.User, error)
	getUserByID func(ctx context.Context, userID int) (*models.User, error)
	createUser  func(ctx context.Context, user *models.User) error
}

func (m *mockUserRepo) GetUser(ctx context.Context, login string) (*models.User, error) {
	if m.getUser != nil {
		return m.getUser(ctx, login)
	}
	return nil, nil
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	if m.createUser != nil {
		return m.createUser(ctx, user)
	}
	return nil
}

type mockInventoryRepo struct {
	getInventoryByID func(ctx context.Context, userID int) ([]models.InventoryItem, error)
}

func (m *mockInventoryRepo) GetInventoryByID(ctx context.Context, userID int) ([]models.InventoryItem, error) {
	if m.getInventoryByID != nil {
		return m.getInventoryByID(ctx, userID)
	}
	return nil, nil
}

type mockCoinTransferRepo struct {
	getTransferHistory func(ctx context.Context, userID int) ([]models.CoinTransferDetail, error)
}

func (m *mockCoinTransferRepo) GetTransferHistory(ctx context.Context, userID int) ([]models.CoinTransferDetail, error) {
	if m.getTransferHistory != nil {
		return m.getTransferHistory(ctx, userID)
	}
	return nil, nil
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return nil, app_errors.ErrUserNotFound
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, "")

	_, err := svc.Auth(context.Background(), "test_login", "test_password")

	if !errors.Is(err, app_errors.ErrUserNotFound) {
		t.Errorf("expected %v got %v", app_errors.ErrUserNotFound, err)
	}
}

func TestLogin_IncorrectPassword(t *testing.T) {
	password := "test_password"
	passwordWrong := "test_wrong_password"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userMock := models.User{
		ID:       1,
		Login:    "test_login",
		Password: string(hashedPassword),
		Balance:  1000,
	}

	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &userMock, nil
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, "")

	_, err := svc.Auth(context.Background(), userMock.Login, passwordWrong)
	if !errors.Is(err, app_errors.ErrIncorrectPassword) {
		t.Errorf("expected %v got %v", app_errors.ErrIncorrectPassword, err)
	}
}

func TestLogin_Success(t *testing.T) {
	secret := "secret_key"
	password := "test_password"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userMock := models.User{
		ID:       1,
		Login:    "test_login",
		Password: string(hashedPassword),
		Balance:  1000,
	}

	userRepoMock := &mockUserRepo{
		getUser: func(ctx context.Context, login string) (*models.User, error) {
			return &userMock, nil
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, secret)

	token, err := svc.Auth(context.Background(), userMock.Login, password)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !parsedToken.Valid {
		t.Fatalf("invalid token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims to MapClaims")
	}

	if float64(userMock.ID) != claims["userID"].(float64) {
		t.Errorf("expected userId %d got %v", userMock.ID, claims["userID"])
	}
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	login := "test_login"
	password := "test_password"

	userRepoMock := &mockUserRepo{
		createUser: func(ctx context.Context, user *models.User) error {
			return app_errors.ErrUserAlreadyExists
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, "")

	err := svc.Register(context.Background(), login, password)
	if !errors.Is(err, app_errors.ErrUserAlreadyExists) {
		t.Errorf("expected %v got %v", app_errors.ErrUserAlreadyExists, err)
	}
}

func TestRegister_Success(t *testing.T) {
	login := "test_login"
	password := "test_password"

	userRepoMock := &mockUserRepo{
		createUser: func(ctx context.Context, user *models.User) error {
			if user.Login != login {
				t.Errorf("expected login %s got %s", login, user.Login)
			}
			if user.Balance != 1000 {
				t.Errorf("expected balance 1000 got %d", user.Balance)
			}

			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
			if err != nil {
				t.Errorf("password in DB does not match original password: %v", err)
			}
			return nil
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, "secret")

	err := svc.Register(context.Background(), login, password)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
}

func TestGetInfo_UserNotFound(t *testing.T) {
	userRepoMock := &mockUserRepo{
		getUserByID: func(ctx context.Context, userID int) (*models.User, error) {
			return nil, app_errors.ErrUserNotFound
		},
	}

	svc := NewUserService(userRepoMock, nil, nil, "")

	_, err := svc.GetInfo(context.Background(), 1)

	if !errors.Is(err, app_errors.ErrUserNotFound) {
		t.Errorf("expected %v got %v", app_errors.ErrUserNotFound, err)
	}
}

func TestGetInfo_Success(t *testing.T) {
	userMock := models.User{
		ID:      1,
		Login:   "test_login1",
		Balance: 1000,
	}

	inventoryMock := []models.InventoryItem{
		{Quantity: 2, Item: models.Item{Name: "pen"}},
	}

	historyMock := []models.CoinTransferDetail{
		{FromUser: "test_login2", ToUser: "test_login1", Amount: 100},
		{FromUser: "test_login1", ToUser: "test_login3", Amount: 20},
	}

	userRepoMock := &mockUserRepo{
		getUserByID: func(ctx context.Context, userID int) (*models.User, error) {
			return &userMock, nil
		},
	}
	inventoryRepoMock := &mockInventoryRepo{
		getInventoryByID: func(ctx context.Context, userID int) ([]models.InventoryItem, error) {
			return inventoryMock, nil
		},
	}
	coinTransferRepoMock := &mockCoinTransferRepo{
		getTransferHistory: func(ctx context.Context, userID int) ([]models.CoinTransferDetail, error) {
			return historyMock, nil
		},
	}

	svc := NewUserService(userRepoMock, inventoryRepoMock, coinTransferRepoMock, "secret")
	info, err := svc.GetInfo(context.Background(), 1)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if info.Coins != 1000 {
		t.Errorf("expected coins 1000, got %d", info.Coins)
	}

	if len(info.Inventory) != 1 || info.Inventory[0].Type != "pen" || info.Inventory[0].Quantity != 2 {
		t.Errorf("unexpected inventory: %+v", info.Inventory)
	}

	if len(info.CoinHistory.Received) != 1 || info.CoinHistory.Received[0].FromUser != "test_login2" || info.CoinHistory.Received[0].Amount != 100 {
		t.Errorf("unexpected received history: %+v", info.CoinHistory.Received)
	}

	if len(info.CoinHistory.Sent) != 1 || info.CoinHistory.Sent[0].ToUser != "test_login3" || info.CoinHistory.Sent[0].Amount != 20 {
		t.Errorf("unexpected sent history: %+v", info.CoinHistory.Sent)
	}
}
