package handler

import (
	"net/http"

	"github.com/Nikita-onlyHD/merchandise-store/internal/middleware"
)

type UserHandler interface {
	Auth(w http.ResponseWriter, r *http.Request)
	GetInfo(w http.ResponseWriter, r *http.Request)
}

type ItemHandler interface {
	BuyItem(w http.ResponseWriter, r *http.Request)
}

type CoinTransferHandler interface {
	SendCoins(w http.ResponseWriter, r *http.Request)
}

func RegisterRoutes(
	mux *http.ServeMux,
	userHandler UserHandler,
	itemHandler ItemHandler,
	coinTransferHandler CoinTransferHandler,
	jwtSecret string,
) http.Handler {
	authMiddleware := middleware.AuthMiddleware(jwtSecret)

	protected := func(h http.HandlerFunc) http.Handler {
		return authMiddleware(h)
	}

	mux.HandleFunc("POST /api/auth", userHandler.Auth)

	mux.Handle("GET /api/info", protected(userHandler.GetInfo))
	mux.Handle("POST /api/sendCoin", protected(coinTransferHandler.SendCoins))
	mux.Handle("GET /api/buy/{item}", protected(itemHandler.BuyItem))

	handler := middleware.Recovery(
		middleware.RequestID(
			middleware.Logger(mux),
		),
	)

	return handler
}
