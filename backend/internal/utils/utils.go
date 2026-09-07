package utils

import (
	"chat_app/internal/constants"
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func ParseJSONRequestBody(w http.ResponseWriter, r *http.Request, data any) error {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return err
	}
	return nil
}

func ValidateRequestBody(w http.ResponseWriter, r *http.Request, data any) error {
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return err
	}
	return nil
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(constants.UserIDContextKey).(uuid.UUID)
	return userID, ok
}
