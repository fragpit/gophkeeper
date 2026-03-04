package client

import (
	"context"
	"fmt"
)

type DeleteResponse struct {
	Error string `json:"error"`
}

func (c *Client) DeleteItem(ctx context.Context, title string) error {
	if title == "" {
		return fmt.Errorf("title not provided")
	}

	respData := &DeleteResponse{}
	req := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.accessToken)).
		SetQueryParam("title", title).
		SetError(respData)

	resp, err := c.requestWithAuth(ctx, "/api/delete/item", req, req.Delete)
	if err != nil {
		return fmt.Errorf("delete item request: %w", err)
	}
	if resp.IsError() {
		return handleErrorResponse(resp, "delete item", respData.Error)
	}
	defer func() { _ = resp.Body.Close() }()

	fmt.Printf("Item '%s' successfully deleted\n", title)
	return nil
}
