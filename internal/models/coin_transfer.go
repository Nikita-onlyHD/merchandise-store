package models

import "time"

type CoinTransfer struct {
	ID         int
	FromUserID int
	ToUserID   int
	Amount     uint
	CreatedAt  time.Time
}
