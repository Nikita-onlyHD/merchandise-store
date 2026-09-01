package item

import (
	"context"
	"errors"
	"net/http"

	apperr "github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/handler"
	"github.com/Nikita-onlyHD/merchandise-store/internal/middleware"
)

type ItemService interface {
	BuyItem(ctx context.Context, itemName string, userID int) error
}

type ItemHandler struct {
	itemService ItemService
}

func NewItemHandler(itemService ItemService) *ItemHandler {
	return &ItemHandler{itemService: itemService}
}

func (h *ItemHandler) BuyItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		handler.SendError(w, http.StatusUnauthorized, "Неавторизован.")
		return
	}

	itemName := r.PathValue("item")
	if itemName == "" {
		handler.SendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	err := h.itemService.BuyItem(r.Context(), itemName, userID)
	if err != nil {
		if errors.Is(err, apperr.ErrInsufficientFunds) || errors.Is(err, apperr.ErrItemNotFound) {
			handler.SendError(w, http.StatusBadRequest, "Неверный запрос.")
			return
		}
		handler.SendError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
		return
	}

	w.WriteHeader(http.StatusOK)
}
