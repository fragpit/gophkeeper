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
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.jwtToken)).
		SetQueryParam("title", title).
		SetError(respData).
		Delete("/api/delete/item")
	if err != nil {
		return fmt.Errorf("delete item request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "delete item", respData.Error)
	}

	fmt.Printf("Item '%s' successfully deleted\n", title)
	return nil
}
