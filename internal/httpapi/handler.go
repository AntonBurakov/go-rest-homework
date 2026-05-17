package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-rest-homework/internal/store"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	store  *store.TodoStore
	logger *slog.Logger
}

func NewHandler(todoStore *store.TodoStore, logger *slog.Logger) *Handler {
	return &Handler{
		store:  todoStore,
		logger: logger,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/todos", h.handleTodos)
	mux.HandleFunc("/todos/", h.handleTodoByID)

	return h.recoverPanic(h.withRequestTimeout(mux))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTodos(w, r)
	case http.MethodPost:
		h.createTodo(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	id, ok := todoIDFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTodo(w, r, id)
	case http.MethodPatch:
		h.updateTodo(w, r, id)
	case http.MethodDelete:
		h.deleteTodo(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func (h *Handler) listTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := h.store.List(r.Context())
	if err != nil {
		writeContextError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": todos})
}

func (h *Handler) createTodo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	todo, err := h.store.Create(r.Context(), title)
	if err != nil {
		writeContextError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, todo)
}

func (h *Handler) getTodo(w http.ResponseWriter, r *http.Request, id int64) {
	todo, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (h *Handler) updateTodo(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Title *string `json:"title"`
		Done  *bool   `json:"done"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title must not be empty")
			return
		}
		req.Title = &title
	}
	if req.Title == nil && req.Done == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	todo, err := h.store.Update(r.Context(), id, req.Title, req.Done)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (h *Handler) deleteTodo(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) withRequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("request panic", "path", r.URL.Path, "error", recovered)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid json body")
	}
	return nil
}

func todoIDFromPath(path string) (int64, bool) {
	rawID, ok := strings.CutPrefix(path, "/todos/")
	if !ok || rawID == "" || strings.Contains(rawID, "/") {
		return 0, false
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}
