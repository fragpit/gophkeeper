package client

import (
	"context"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
)

type createItemRequest struct {
	Item *model.ItemDecrypted `json:"item"`
}

type createItemResponse struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

// CreateItem sends a request to create a new item for the authenticated user.
func (c *Client) CreateItem(
	ctx context.Context,
	item *model.ItemDecrypted,
) error {
	reqData := &createItemRequest{Item: item}

	respData := &createItemResponse{}
	req := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.accessToken)).
		SetBody(reqData).
		SetResult(respData).
		SetError(respData)

	resp, err := c.requestWithAuth(ctx, "/api/create/item", req, req.Post)
	if err != nil {
		return fmt.Errorf("create item request: %w", err)
	}
	if resp.IsError() {
		return handleErrorResponse(resp, "create item", respData.Error)
	}
	defer func() { _ = resp.Body.Close() }()

	fmt.Printf("created item, type %s, id %d", item.Type, respData.ID)

	return nil
}
