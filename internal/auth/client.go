package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
)

type AuthClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(cfg *config.Config, logger *slog.Logger) *AuthClient {
	logger.Info("SMTP client created",
		"base_url", "http://localhost:4444/api/v1/",
	)

	return &AuthClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "http://localhost:4444/api/v1/",
		logger:  logger,
	}
}

func (c *AuthClient) newRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	reqHTTP, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		c.logger.Error("AuthClient: failed to build request",
			"method", method,
			"url", url,
			"error", err,
		)
		return nil, err
	}

	resp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		c.logger.Error("AuthClient: request failed",
			"method", method,
			"url", url,
			"error", err,
		)
		return nil, err
	}

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.Error("SMTP: non-success status",
			"method", method,
			"url", url,
			"status_code", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("SMTP returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}
