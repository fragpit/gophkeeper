package client

import (
	"context"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
)

// RegisterResponse represents a registration response payload.
type RegisterResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
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

	if err := saveTokensToFile(model.TokenPair{
		AccessToken:  respData.AccessToken,
		RefreshToken: respData.RefreshToken,
	}); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}
