package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

var ErrIncorrectPassword = errors.New("incorrect password")

type UserRepository interface {
	GetUser(ctx context.Context, login string) (*models.User, error)
	GetUserByID(ctx context.Context, userID int) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
}

type InventoryRepository interface {
	GetInventoryByID(ctx context.Context, userID int) ([]models.InventoryItem, error)
}

type CoinTransferRepository interface {
	GetTransferHistory(ctx context.Context, userID int) ([]models.CoinTransferDetail, error)
}

type Service struct {
	userRepo         UserRepository
	inventoryRepo    InventoryRepository
	coinTransferRepo CoinTransferRepository
	jwtSecret        string
}

func NewUserService(
	userRepo UserRepository,
	inventoryRepo InventoryRepository,
	coinTransferRepo CoinTransferRepository,
	jwtSecret string,
) *Service {
	return &Service{
		userRepo:         userRepo,
		inventoryRepo:    inventoryRepo,
		coinTransferRepo: coinTransferRepo,
		jwtSecret:        jwtSecret,
	}
}

func (s *Service) Login(ctx context.Context, login string, password string) (string, error) {
	user, err := s.userRepo.GetUser(ctx, login)
	if err != nil {
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", ErrIncorrectPassword
	}

	claims := jwt.MapClaims{
		"userId":   user.ID,
		"username": user.Login,
		"exp":      time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to get token string: %w", err)
	}

	return tokenString, nil
}

func (s *Service) Register(ctx context.Context, login string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate hashed password: %w", err)
	}

	user := models.User{
		Login:    login,
		Password: string(hashedPassword),
		Balance:  1000,
	}

	err = s.userRepo.CreateUser(ctx, &user)
	if err != nil {
		return err
	}

	return nil
}

type UserInfoDTO struct {
	Coins       int         `json:"coins"`
	Inventory   []Item      `json:"inventory"`
	CoinHistory CoinHistory `json:"coinHistory"`
}

type Item struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []Received `json:"received"`
	Sent     []Sent     `json:"sent"`
}

type Received struct {
	FromUser string `json:"fromUser"`
	Amount   int    `json:"amount"`
}

type Sent struct {
	ToUser string `json:"toUser"`
	Amount int    `json:"amount"`
}

func (s *Service) GetInfo(ctx context.Context, userID int) (*UserInfoDTO, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	inventory, err := s.inventoryRepo.GetInventoryByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	coinTransferHistory, err := s.coinTransferRepo.GetTransferHistory(ctx, userID)
	if err != nil {
		return nil, err
	}

	transformedInventory := make([]Item, 0, len(inventory))
	for _, i := range inventory {
		item := Item{
			Type:     i.Item.Name,
			Quantity: int(i.Quantity),
		}

		transformedInventory = append(transformedInventory, item)
	}

	received := make([]Received, 0)
	sent := make([]Sent, 0)
	for _, ct := range coinTransferHistory {
		if user.Login == ct.ToUser {
			transfer := Received{
				FromUser: ct.FromUser,
				Amount:   int(ct.Amount),
			}
			received = append(received, transfer)
		} else {
			transfer := Sent{
				ToUser: ct.ToUser,
				Amount: int(ct.Amount),
			}
			sent = append(sent, transfer)
		}
	}

	coinHistory := CoinHistory{
		Received: received,
		Sent:     sent,
	}

	return &UserInfoDTO{
		Coins:       int(user.Balance),
		Inventory:   transformedInventory,
		CoinHistory: coinHistory,
	}, nil
}
