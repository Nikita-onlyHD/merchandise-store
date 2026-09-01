package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	apperr "github.com/Nikita-onlyHD/merchandise-store/internal/app_errors"
	"github.com/Nikita-onlyHD/merchandise-store/internal/dto"
	"github.com/Nikita-onlyHD/merchandise-store/internal/handler"
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
		handler.SendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	if req.Login == "" || req.Password == "" {
		handler.SendError(w, http.StatusBadRequest, "Неверный запрос.")
		return
	}

	token, err := h.userService.Auth(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, apperr.ErrIncorrectPassword) {
			handler.SendError(w, http.StatusUnauthorized, "Неавторизован.")
			return
		}
		handler.SendError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
		return
	}

	handler.SendJSON(w, http.StatusOK, dto.AuthResponse{Token: token})
}

func (h *UserHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		handler.SendError(w, http.StatusUnauthorized, "Неавторизован.")
		return
	}

	userInfo, err := h.userService.GetInfo(r.Context(), userID)
	if err != nil {
		handler.SendError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера.")
		return
	}

	handler.SendJSON(w, http.StatusOK, userInfo)
}
