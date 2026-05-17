package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-rest-homework/internal/metrics"
)

func TestHealth(t *testing.T) {
	handler := NewHandler(nil, metrics.NewRegistry(), slog.Default()).Routes()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}
}

func TestRegisterValidation(t *testing.T) {
	handler := NewHandler(nil, metrics.NewRegistry(), slog.Default()).Routes()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"email":"bad","password":""}`))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}
