package client

import (
	"context"
	"fmt"
)

// LoginResponse represents a login response payload.
type LoginResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
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

	if err := saveTokenToFile(respData.Token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}
