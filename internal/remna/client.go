package remna

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
)

type IRemnaClient interface {
	GetAllUser(ctx context.Context) (*AllUserResponse, error)
	GetUserByUUID(ctx context.Context, uuid string) (*RemnaUserResponse, error)
	GetUserByEmail(ctx context.Context, email string) (*RemnaUserResponse, error)
	GetUserByTelegramID(ctx context.Context, tgID string) (*RemnaUserResponse, error)
	CreateNewUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error)
	UpdateUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error)
}

type authTransport struct {
	token string
	rt    http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	req.Header.Set("Content-Type", "application/json")
	return t.rt.RoundTrip(req)
}

type RemnaClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(cfg *config.Config, logger *slog.Logger) *RemnaClient {
	logger.Info("Remna client created",
		"base_url", "https://panel.nevpn.shop/api",
	)

	return &RemnaClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &authTransport{
				token: cfg.RemnaToken,
				rt:    http.DefaultTransport,
			},
		},
		baseURL: "https://panel.nevpn.shop/api",
		logger:  logger,
	}
}

func (c *RemnaClient) newRemnaRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	reqHTTP, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		c.logger.Error("RemnaClient: failed to build request",
			"method", method,
			"url", url,
			"error", err,
		)
		return nil, err
	}

	resp, err := c.httpClient.Do(reqHTTP)
	if err != nil {
		c.logger.Error("RemnaClient: request failed",
			"method", method,
			"url", url,
			"error", err,
		)
		return nil, err
	}

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.Error("RemnaClient: non-success status from remna",
			"method", method,
			"url", url,
			"status_code", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("remna returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

func (c *RemnaClient) GetAllUser(ctx context.Context) (*AllUserResponse, error) {
	c.logger.Info("RemnaClient: GetAllUser started")

	resp, err := c.newRemnaRequest(ctx, http.MethodGet, c.baseURL+"/users", nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var out AllUserResponseRoot
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "GetAllUser",
			"error", err,
		)
		return nil, err
	}

	c.logger.Info("RemnaClient: GetAllUser succeeded")
	return &out.Response, nil
}

func (c *RemnaClient) GetUserByUUID(ctx context.Context, uuid string) (*RemnaUserResponse, error) {
	c.logger.Info("RemnaClient: GetUserByUUID started",
		"uuid", uuid,
	)

	resp, err := c.newRemnaRequest(ctx, http.MethodGet, fmt.Sprintf("%s/users/%s", c.baseURL, uuid), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Response RemnaUserResponse
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "GetUserByUUID",
			"uuid", uuid,
			"error", err,
		)
		return nil, err
	}

	c.logger.Info("RemnaClient: GetUserByUUID succeeded",
		"uuid", uuid,
	)
	return &out.Response, nil
}

func (c *RemnaClient) GetUserByEmail(ctx context.Context, email string) (*RemnaUserResponse, error) {
	c.logger.Info("RemnaClient: GetUserByEmail started",
		"email", email,
	)

	resp, err := c.newRemnaRequest(ctx, http.MethodGet, fmt.Sprintf("%s/users/by-email/%s", c.baseURL, email), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Response []RemnaUserResponse
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "GetUserByEmail",
			"email", email,
			"error", err,
		)
		return nil, err
	}

	if len(out.Response) == 0 {
		c.logger.Warn("RemnaClient: user not found by email",
			"email", email,
		)
		return nil, fmt.Errorf("not found user by this email")
	}

	c.logger.Info("RemnaClient: GetUserByEmail succeeded",
		"email", email,
	)
	return &out.Response[0], nil
}

func (c *RemnaClient) GetUserByTelegramID(ctx context.Context, tgID string) (*RemnaUserResponse, error) {
	c.logger.Info("RemnaClient: GetUserByTelegramID started",
		"telegram_id", tgID,
	)

	resp, err := c.newRemnaRequest(ctx, http.MethodGet, fmt.Sprintf("%s/users/by-telegram/%s", c.baseURL, tgID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Response []RemnaUserResponse
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "GetUserByTelegramID",
			"telegram_id", tgID,
			"error", err,
		)
		return nil, err
	}

	if len(out.Response) == 0 {
		c.logger.Warn("RemnaClient: user not found by telegram id",
			"telegram_id", tgID,
		)
		return nil, fmt.Errorf("not found user by this telegramID")
	}

	c.logger.Info("RemnaClient: GetUserByTelegramID succeeded",
		"telegram_id", tgID,
	)
	return &out.Response[0], nil
}

func (c *RemnaClient) CreateNewUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error) {
	c.logger.Info("RemnaClient: CreateNewUser started",
		"email", user.Email,
		"uuid", user.UUID,
	)

	body, _ := json.Marshal(user)

	resp, err := c.newRemnaRequest(ctx, http.MethodPost, fmt.Sprintf("%s/users", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Response RemnaUserResponse
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "CreateNewUser",
			"error", err,
		)
		return nil, err
	}

	c.logger.Info("RemnaClient: CreateNewUser succeeded",
		"email", out.Response.Email,
		"uuid", out.Response.UUID,
	)
	return &out.Response, nil
}

func (c *RemnaClient) UpdateUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error) {
	c.logger.Info("RemnaClient: UpdateUser started",
		"email", user.Email,
		"uuid", user.UUID,
	)

	body, _ := json.Marshal(user)

	resp, err := c.newRemnaRequest(ctx, http.MethodPatch, fmt.Sprintf("%s/users", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Response RemnaUserResponse
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.logger.Error("RemnaClient: failed to decode response",
			"operation", "UpdateUser",
			"error", err,
		)
		return nil, err
	}

	c.logger.Info("RemnaClient: UpdateUser succeeded",
		"email", out.Response.Email,
		"uuid", out.Response.UUID,
	)
	return &out.Response, nil
}
