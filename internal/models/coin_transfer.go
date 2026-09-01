package models

import "time"

type CoinTransfer struct {
	ID         int
	FromUserID int
	ToUserID   int
	Amount     int
	CreatedAt  time.Time
}

type CoinTransferDetail struct {
	ID        int
	FromUser  string
	ToUser    string
	Amount    int
	CreatedAt time.Time
}
