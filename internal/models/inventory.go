package models

type InventoryItem struct {
	ID       int
	Quantity int
	UserID   int
	Item     Item
}
