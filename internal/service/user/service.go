package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperr "github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/dto"
	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

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

func (s *Service) Auth(ctx context.Context, login string, password string) (string, error) {
	user, err := s.userRepo.GetUser(ctx, login)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			if regErr := s.Register(ctx, login, password); regErr != nil {
				return "", regErr
			}
			user, err = s.userRepo.GetUser(ctx, login)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	} else {
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			return "", apperr.ErrIncorrectPassword
		}
	}

	claims := jwt.MapClaims{
		"userID":   user.ID,
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

func (s *Service) GetInfo(ctx context.Context, userID int) (*dto.UserInfo, error) {
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

	transformedInventory := make([]dto.Item, 0, len(inventory))
	for _, i := range inventory {
		item := dto.Item{
			Type:     i.Item.Name,
			Quantity: int(i.Quantity),
		}

		transformedInventory = append(transformedInventory, item)
	}

	received := make([]dto.Received, 0)
	sent := make([]dto.Sent, 0)
	for _, ct := range coinTransferHistory {
		if user.Login == ct.ToUser {
			transfer := dto.Received{
				FromUser: ct.FromUser,
				Amount:   int(ct.Amount),
			}
			received = append(received, transfer)
		} else {
			transfer := dto.Sent{
				ToUser: ct.ToUser,
				Amount: int(ct.Amount),
			}
			sent = append(sent, transfer)
		}
	}

	coinHistory := dto.CoinHistory{
		Received: received,
		Sent:     sent,
	}

	return &dto.UserInfo{
		Coins:       int(user.Balance),
		Inventory:   transformedInventory,
		CoinHistory: coinHistory,
	}, nil
}
