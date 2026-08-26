package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nikita-onlyHD/merchandise-store/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) WithTx(tx DBTX) *userRepository {
	return &userRepository{
		db: tx,
	}
}

func (r *userRepository) GetUser(ctx context.Context, login string) (*models.User, error) {
	var user models.User

	err := r.db.QueryRow(ctx, "SELECT id, login, password, balance FROM users WHERE login = $1", login).Scan(
		&user.ID,
		&user.Login,
		&user.Password,
		&user.Balance,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	err := r.db.QueryRow(ctx, "INSERT INTO users (login, password, balance) VALUES ($1, $2, $3) RETURNING id",
		user.Login,
		user.Password,
		user.Balance,
	).Scan(&user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}
