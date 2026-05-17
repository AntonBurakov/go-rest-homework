package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"go-rest-homework/internal/vault"
)

const tokenPrefix = "mock-token-"

type TokenSigner struct {
	vault *vault.Client
}

func NewTokenSigner(vaultClient *vault.Client) *TokenSigner {
	return &TokenSigner{vault: vaultClient}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(password, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

func (s *TokenSigner) Create(ctx context.Context, userID int64) (string, error) {
	payload := strconv.FormatInt(userID, 10)
	signature, err := s.sign(ctx, payload)
	if err != nil {
		return "", err
	}
	return tokenPrefix + payload + "." + signature, nil
}

func (s *TokenSigner) Parse(ctx context.Context, token string) (int64, bool) {
	raw, ok := strings.CutPrefix(token, tokenPrefix)
	if !ok {
		return 0, false
	}

	payload, signature, ok := strings.Cut(raw, ".")
	if !ok {
		return 0, false
	}

	expected, err := s.sign(ctx, payload)
	if err != nil || !hmac.Equal([]byte(signature), []byte(expected)) {
		return 0, false
	}

	userID, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func (s *TokenSigner) sign(ctx context.Context, payload string) (string, error) {
	secret, err := s.vault.TokenSecret(ctx)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
