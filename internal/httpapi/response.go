package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go-rest-homework/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", joinMethods(methods))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrTodoNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	writeContextError(w, err)
}

func writeContextError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "request deadline exceeded")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func joinMethods(methods []string) string {
	if len(methods) == 0 {
		return ""
	}

	joined := methods[0]
	for _, method := range methods[1:] {
		joined += ", " + method
	}
	return joined
}
