package dto

type AuthRequest struct {
	Login    string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type UserInfo struct {
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
