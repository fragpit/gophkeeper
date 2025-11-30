package client

import (
	"context"
	"fmt"
)

type RegisterResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
}

func (c *Client) Register(
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
		Post("/api/register")
	if err != nil {
		return fmt.Errorf("user register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return handleErrorResponse(resp, "user register", respData.Error)
	}

	if err := saveToken(respData.Token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}
