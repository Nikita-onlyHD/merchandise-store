package handler

import (
	"encoding/json"
	"net/http"
)

func SendJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type ErrorResponse struct {
	Errors string
}

func SendError(w http.ResponseWriter, status int, message string) {
	SendJSON(w, status, ErrorResponse{Errors: message})
}
