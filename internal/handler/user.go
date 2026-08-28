package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/dto"
	"github.com/Nikita-onlyHD/merchandise-store/internal/middleware"
)

type UserService interface {
	Auth(ctx context.Context, login string, password string) (string, error)
	GetInfo(ctx context.Context, userID int) (*dto.UserInfo, error)
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	if req.Login == "" || req.Password == "" {
		sendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	token, err := h.userService.Auth(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, app_errors.ErrIncorrectPassword) {
			sendError(w, http.StatusUnauthorized, "Неавторизован.")
			return
		}
		sendError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
		return
	}

	sendJSON(w, http.StatusOK, dto.AuthResponse{Token: token})
}

func (h *UserHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Неавторизован.")
		return
	}

	userInfo, err := h.userService.GetInfo(r.Context(), userID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
		return
	}

	sendJSON(w, http.StatusOK, userInfo)
}

func sendJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type ErrorResponse struct {
	Errors string
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, ErrorResponse{Errors: message})
}
