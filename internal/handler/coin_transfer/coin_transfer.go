package transfer

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Nikita-onlyHD/merchandise-store/internal/dto"
	"github.com/Nikita-onlyHD/merchandise-store/internal/handler"
	"github.com/Nikita-onlyHD/merchandise-store/internal/middleware"
)

type CoinTransferService interface {
	SendCoins(ctx context.Context, fromUserID int, toUser string, amount int) error
}

type CoinTransferHandler struct {
	coinTransferService CoinTransferService
}

func NewCoinTransferHandler(coinTransferService CoinTransferService) *CoinTransferHandler {
	return &CoinTransferHandler{coinTransferService: coinTransferService}
}

func (h *CoinTransferHandler) SendCoins(w http.ResponseWriter, r *http.Request) {
	fromUserID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		handler.SendError(w, http.StatusUnauthorized, "Неавторизован.")
		return
	}

	var req dto.SendCoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.SendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	if req.ToUser == "" || req.Amount <= 0 {
		handler.SendError(w, http.StatusBadRequest, "Неверное имя получателя или сумма.")
		return
	}

	err := h.coinTransferService.SendCoins(r.Context(), fromUserID, req.ToUser, req.Amount)
	if err != nil {
		handler.SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}
