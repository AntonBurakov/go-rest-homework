package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"go-rest-homework/internal/kafka"
	"go-rest-homework/internal/security"
	"go-rest-homework/internal/storage"
)

var (
	ErrConflict           = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AuthService struct {
	users     *storage.UserStore
	publisher *kafka.Publisher
	signer    *security.TokenSigner
	logger    *slog.Logger
}

func NewAuthService(users *storage.UserStore, publisher *kafka.Publisher, signer *security.TokenSigner, logger *slog.Logger) *AuthService {
	return &AuthService{
		users:     users,
		publisher: publisher,
		signer:    signer,
		logger:    logger,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, traceID string) (storage.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return storage.User{}, ErrConflict
	} else if !errors.Is(err, storage.ErrUserNotFound) {
		return storage.User{}, err
	}

	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return storage.User{}, err
	}

	user, err := s.users.Create(ctx, email, hashedPassword)
	if err != nil {
		return storage.User{}, err
	}

	if s.publisher != nil {
		if published := s.publisher.PublishUserRegistered(ctx, user, traceID); !published {
			s.logger.Warn("user_registered_event_not_published", "user_id", user.ID)
		}
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil || !security.VerifyPassword(password, user.HashedPassword) {
		return "", ErrInvalidCredentials
	}

	return s.signer.Create(ctx, user.ID)
}

func (s *AuthService) CurrentUser(ctx context.Context, token string) (storage.User, error) {
	userID, ok := s.signer.Parse(ctx, token)
	if !ok {
		return storage.User{}, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return storage.User{}, ErrInvalidCredentials
	}
	return user, nil
}
