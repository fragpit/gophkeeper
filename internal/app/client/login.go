package client

import (
	"context"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
)

// LoginResponse represents a login response payload.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

// Login authenticates a user and stores the received token.
func (c *Client) Login(
	ctx context.Context,
	login, password string,
) error {
	var respData LoginResponse
	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{
			"login":    login,
			"password": password,
		}).
		SetResult(&respData).
		SetError(&respData).
		Post("/api/login")
	if err != nil {
		return fmt.Errorf("user login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "user login", respData.Error)
	}

	if err := saveTokensToFile(model.TokenPair{
		AccessToken:  respData.AccessToken,
		RefreshToken: respData.RefreshToken,
	}); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}
