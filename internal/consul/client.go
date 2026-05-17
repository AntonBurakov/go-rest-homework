package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-rest-homework/internal/config"
)

type Client struct {
	cfg    config.Config
	http   *http.Client
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) *Client {
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: 3 * time.Second},
		logger: logger,
	}
}

func (c *Client) Register(ctx context.Context) {
	if !c.cfg.ConsulEnabled {
		c.logger.Info("consul_registration_disabled")
		return
	}

	payload := map[string]any{
		"ID":      c.cfg.ServiceID,
		"Name":    c.cfg.ServiceName,
		"Address": c.cfg.ServiceAddress,
		"Port":    c.cfg.ServicePort,
		"Tags":    []string{"go", "auth", "observability"},
		"Check": map[string]string{
			"HTTP":                           c.cfg.ServiceHealthCheck,
			"Interval":                       "10s",
			"Timeout":                        "2s",
			"DeregisterCriticalServiceAfter": "1m",
		},
	}

	if err := c.request(ctx, http.MethodPut, "/v1/agent/service/register", payload); err != nil {
		c.logger.Warn("consul_service_registration_failed", "error", err)
		return
	}

	c.logger.Info(
		"consul_service_registered",
		"service_id", c.cfg.ServiceID,
		"service_name", c.cfg.ServiceName,
		"service_address", c.cfg.ServiceAddress,
		"service_port", c.cfg.ServicePort,
		"health_check_url", c.cfg.ServiceHealthCheck,
	)
}

func (c *Client) Deregister(ctx context.Context) {
	if !c.cfg.ConsulEnabled {
		return
	}

	path := "/v1/agent/service/deregister/" + c.cfg.ServiceID
	if err := c.request(ctx, http.MethodPut, path, nil); err != nil {
		c.logger.Warn("consul_service_deregistration_failed", "error", err)
		return
	}

	c.logger.Info("consul_service_deregistered", "service_id", c.cfg.ServiceID)
}

func (c *Client) request(ctx context.Context, method, path string, body any) error {
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.ConsulHTTPAddr, "/")+path, requestBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("consul returned status %d", resp.StatusCode)
	}
	return nil
}
