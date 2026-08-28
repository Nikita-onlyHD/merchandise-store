package app_errors

import "errors"

var (
	ErrItemNotFound = errors.New("item not found")
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInsufficientFunds = errors.New("user does not have sufficient funds")
)

var ErrIncorrectPassword = errors.New("incorrect password")

var (
	ErrInvalidAmount = errors.New("amount must be positive")
	ErrSelfTransfer  = errors.New("cannot transfer coin to yourself")
)
