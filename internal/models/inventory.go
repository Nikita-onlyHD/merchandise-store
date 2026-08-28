package models

type InventoryItem struct {
	ID       int
	Quantity uint
	UserID   int
	Item     Item
}
