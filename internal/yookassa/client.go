package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/config"
	"github.com/google/uuid"
)

type IPaymentClient interface {
	CreatePayment(ctx context.Context, payload PaymentRequest) (map[string]any, error)
}

type authTransport struct {
	shopID    string
	secretKey string
	rt        http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.shopID, t.secretKey)
	req.Header.Set("Content-Type", "application/json")
	return t.rt.RoundTrip(req)
}

type YooKassaClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(cfg *config.Config, logger *slog.Logger) *YooKassaClient {
	logger.Info("YooKassa client created",
		"base_url", "https://api.yookassa.ru/v3/payments",
	)

	return &YooKassaClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &authTransport{
				shopID:    cfg.YooKassaShopID,
				secretKey: cfg.YooKassaSecretKey,
				rt:        http.DefaultTransport,
			},
		},
		baseURL: "https://api.yookassa.ru/v3/payments",
		logger:  logger,
	}
}

func (c *YooKassaClient) CreatePayment(ctx context.Context, payload PaymentRequest) (map[string]any, error) {
	idempotenceKey := uuid.NewString()

	c.logger.Info("YooKassaClient: CreatePayment started",
		"idempotence_key", idempotenceKey,
		"url", c.baseURL,
	)

	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("YooKassaClient: failed to marshal payload",
			"idempotence_key", idempotenceKey,
			"error", err,
		)
		return nil, err
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("YooKassaClient: failed to build request",
			"idempotence_key", idempotenceKey,
			"url", c.baseURL,
			"error", err,
		)
		return nil, err
	}

	reqHTTP.Header.Set("Idempotence-Key", idempotenceKey)

	resp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		c.logger.Error("YooKassaClient: request failed",
			"idempotence_key", idempotenceKey,
			"url", c.baseURL,
			"error", err,
		)
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("YooKassaClient: failed to read response body",
			"idempotence_key", idempotenceKey,
			"status_code", resp.StatusCode,
			"error", err,
		)
		return nil, err
	}

	if resp.StatusCode >= 300 {
		c.logger.Error("YooKassaClient: non-success status from yookassa",
			"idempotence_key", idempotenceKey,
			"status_code", resp.StatusCode,
			"body", string(rawBody),
		)
		return nil, fmt.Errorf("yookassa returned status %d: %s", resp.StatusCode, string(rawBody))
	}

	var result map[string]any
	if err := json.Unmarshal(rawBody, &result); err != nil {
		c.logger.Error("YooKassaClient: failed to unmarshal response",
			"idempotence_key", idempotenceKey,
			"status_code", resp.StatusCode,
			"error", err,
		)
		return nil, err
	}

	c.logger.Info("YooKassaClient: CreatePayment succeeded",
		"idempotence_key", idempotenceKey,
		"status_code", resp.StatusCode,
	)

	return result, nil
}
