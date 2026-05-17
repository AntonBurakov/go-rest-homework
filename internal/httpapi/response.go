package httpapi

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", joinMethods(methods))
	writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
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
