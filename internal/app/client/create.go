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

func (c *Client) CreateItem(
	ctx context.Context,
	item *model.ItemDecrypted,
) error {
	reqData := &createItemRequest{Item: item}

	respData := &createItemResponse{}
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.jwtToken)).
		SetBody(reqData).
		SetResult(respData).
		SetError(respData).
		Post("/api/create/item")
	if err != nil {
		return fmt.Errorf("create item request: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return handleErrorResponse(resp, "create item", respData.Error)
	}

	fmt.Printf("created item, type %s, id %d", item.Type, respData.ID)

	return nil
}
