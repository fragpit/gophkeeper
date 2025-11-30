package client

import (
	"context"
	"fmt"
)

// RegisterResponse represents a registration response payload.
type RegisterResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
}

// Register registers a new user and saves the returned token.
func (c *Client) Register(
	ctx context.Context,
	login, password string,
) error {
	var respData RegisterResponse
	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{
			"login":    login,
			"password": password,
		}).
		SetResult(&respData).
		SetError(&respData).
		Post("/api/register")
	if err != nil {
		return fmt.Errorf("user register request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "user register", respData.Error)
	}

	if err := saveTokenToFile(respData.Token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}
