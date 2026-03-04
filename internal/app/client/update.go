package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fragpit/gophkeeper/internal/model"
)

type updateItemRequest struct {
	Title string               `json:"title"`
	Item  *model.ItemDecrypted `json:"item"`
}

type updateItemResponse struct {
	Error string `json:"error"`
}

// UpdateItem sends an update request to the server for an existing item.
func (c *Client) UpdateItem(
	ctx context.Context,
	title string,
	item *model.ItemDecrypted,
) error {
	slog.Debug("update item", "title", title, "type", item.Type)

	reqData := &updateItemRequest{
		Title: title,
		Item:  item,
	}

	respData := &updateItemResponse{}
	req := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.accessToken)).
		SetBody(reqData).
		SetResult(respData).
		SetError(respData)

	resp, err := c.requestWithAuth(ctx, "/api/update/item", req, req.Put)
	if err != nil {
		return fmt.Errorf("send update item request: %w", err)
	}
	if resp.IsError() {
		return handleErrorResponse(resp, "update item", respData.Error)
	}
	defer func() { _ = resp.Body.Close() }()

	slog.Info("item updated successfully", "title", item.Title)
	return nil
}
