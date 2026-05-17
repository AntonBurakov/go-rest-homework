package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-rest-homework/internal/store"
)

func TestTodoCRUD(t *testing.T) {
	handler := NewHandler(store.NewTodoStore(), slog.Default()).Routes()

	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(`{"title":"learn go"}`))
	handler.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}

	var created store.Todo
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created todo: %v", err)
	}
	if created.ID != 1 || created.Title != "learn go" || created.Done {
		t.Fatalf("created todo = %+v", created)
	}

	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPatch, "/todos/1", bytes.NewBufferString(`{"done":true}`))
	handler.ServeHTTP(updateResp, updateReq)

	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}

	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/todos/1", nil)
	handler.ServeHTTP(getResp, getReq)

	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body: %s", getResp.Code, http.StatusOK, getResp.Body.String())
	}

	var got store.Todo
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode got todo: %v", err)
	}
	if !got.Done {
		t.Fatalf("todo was not updated: %+v", got)
	}

	deleteResp := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
	handler.ServeHTTP(deleteResp, deleteReq)

	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResp.Code, http.StatusNoContent)
	}
}

func TestCreateTodoValidation(t *testing.T) {
	handler := NewHandler(store.NewTodoStore(), slog.Default()).Routes()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(`{"title":"   "}`))
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}
