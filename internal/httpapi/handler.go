package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-rest-homework/internal/metrics"
	"go-rest-homework/internal/service"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	auth    *service.AuthService
	metrics *metrics.Registry
	logger  *slog.Logger
}

func NewHandler(auth *service.AuthService, registry *metrics.Registry, logger *slog.Logger) *Handler {
	return &Handler{
		auth:    auth,
		metrics: registry,
		logger:  logger,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/metrics", h.handleMetrics)
	mux.HandleFunc("/auth/register", h.handleRegister)
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/users/me", h.handleMe)

	return h.recoverPanic(h.withRequestTimeout(h.withObservability(mux)))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(h.metrics.Render()))
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validEmail(req.Email) || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	user, err := h.auth.Register(r.Context(), req.Email, req.Password, traceIDFromContext(r.Context()))
	if errors.Is(err, service.ErrConflict) {
		writeError(w, http.StatusConflict, "User with this email already exists")
		return
	}
	if err != nil {
		h.logger.Error("register_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if err != nil {
		h.logger.Error("login_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": token,
		"token_type":   "bearer",
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	user, err := h.auth.CurrentUser(r.Context(), token)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "Invalid token")
		return
	}
	if err != nil {
		h.logger.Error("current_user_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

func (h *Handler) withRequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = newTraceID()
		}

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		ctx := context.WithValue(r.Context(), traceIDKey{}, traceID)

		recorder.Header().Set("X-Trace-Id", traceID)
		h.logger.Info("http_request_started", "trace_id", traceID, "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(recorder, r.WithContext(ctx))

		duration := time.Since(start)
		h.metrics.Observe(r.Method, r.URL.Path, recorder.statusCode, duration.Seconds())
		h.logger.Info(
			"http_request_finished",
			"trace_id", traceID,
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", recorder.statusCode,
			"duration_ms", float64(duration.Microseconds())/1000,
		)
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("request_panic", "path", r.URL.Path, "error", recovered)
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
	return decoder.Decode(dst)
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != ""
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func newTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "trace-id-unavailable"
	}
	return hex.EncodeToString(b[:])
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
