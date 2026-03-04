package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"resty.dev/v3"
)

func handleErrorResponse(resp *resty.Response, op string, msg string) error {
	if msg != "" {
		return fmt.Errorf("%s: %s", op, msg)
	}
	if resp == nil {
		return fmt.Errorf("%s: request failed: empty response", op)
	}
	if resp.StatusCode() == 401 {
		return fmt.Errorf("%s: unauthorized: status=%d", op, resp.StatusCode())
	}
	return fmt.Errorf(
		"%s: request failed: status=%d body=%s",
		op,
		resp.StatusCode(),
		resp.String(),
	)
}

type refreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

func (c *Client) refreshAccessToken(ctx context.Context) error {
	respData := &refreshTokenResponse{}
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.refreshToken)).
		SetResult(respData).
		SetError(respData).
		Post("/api/refresh")
	if err != nil {
		return fmt.Errorf("refresh token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "refresh access token", respData.Error)
	}

	c.accessToken = []byte(respData.AccessToken)
	c.refreshToken = []byte(respData.RefreshToken)

	if err := saveTokensToFile(c.configFile, model.TokenPair{
		AccessToken:  respData.AccessToken,
		RefreshToken: respData.RefreshToken,
	}); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}

func (c *Client) requestWithAuth(
	ctx context.Context,
	url string,
	req *resty.Request,
	reqFunc func(url string) (*resty.Response, error),
) (*resty.Response, error) {
	resp, err := reqFunc(url)
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", url, err)
	}

	if resp != nil && resp.StatusCode() == http.StatusUnauthorized {
		if err := c.refreshAccessToken(ctx); err != nil {
			return nil, fmt.Errorf("refresh access token: %w", err)
		}

		req.SetAuthToken(string(c.accessToken))
		resp, err = reqFunc(url)
		if err != nil {
			return nil, fmt.Errorf("request after token refresh: %w", err)
		}
	}

	return resp, nil
}
