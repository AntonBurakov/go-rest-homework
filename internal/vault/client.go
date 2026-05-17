package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-rest-homework/internal/config"
)

type Client struct {
	cfg    config.Config
	http   *http.Client
	logger *slog.Logger
	mu     sync.Mutex
	secret string
}

func New(cfg config.Config, logger *slog.Logger) *Client {
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: 3 * time.Second},
		logger: logger,
	}
}

func (c *Client) TokenSecret(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.secret != "" {
		return c.secret, nil
	}

	secret, err := c.loadTokenSecret(ctx)
	if err != nil {
		return "", err
	}

	c.secret = secret
	return secret, nil
}

func (c *Client) loadTokenSecret(ctx context.Context) (string, error) {
	if !c.cfg.VaultEnabled {
		if c.cfg.AppTokenSecret == "" {
			return "", errors.New("APP_TOKEN_SECRET is required when VAULT_ENABLED=false")
		}
		return c.cfg.AppTokenSecret, nil
	}

	if c.cfg.VaultToken == "" {
		return "", errors.New("VAULT_TOKEN is required to read secrets from Vault")
	}

	url := strings.TrimRight(c.cfg.VaultAddr, "/") + "/v1/" + strings.TrimLeft(c.cfg.VaultTokenSecretPath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", c.cfg.VaultToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("read token secret from Vault: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("read token secret from Vault: status %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Vault response: %w", err)
	}

	secret := payload.Data.Data["token_secret"]
	if secret == "" {
		return "", fmt.Errorf("token_secret is missing in Vault path %s", c.cfg.VaultTokenSecretPath)
	}

	c.logger.Info("vault_secret_loaded", "secret_path", c.cfg.VaultTokenSecretPath, "secret_key", "token_secret")
	return secret, nil
}
