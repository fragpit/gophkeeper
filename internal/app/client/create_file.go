package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fragpit/gophkeeper/internal/model"
)

type createFileResponse struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

func (c *Client) CreateFile(
	ctx context.Context,
	item *model.ItemDecrypted,
	filePath string,
) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal item: %w", err)
	}

	filename := filepath.Base(filePath)
	respData := &createFileResponse{}
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.jwtToken)).
		SetMultipartFormData(map[string]string{
			"item": string(itemJSON),
		}).
		SetMultipartField("file", filename, "application/octet-stream", f).
		SetResult(respData).
		Post("/api/create/file")
	if err != nil {
		return fmt.Errorf("create file request: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return handleErrorResponse(resp, "create file", respData.Error)
	}

	fmt.Printf("created item, type %s, id %d", item.Type, respData.ID)

	return nil
}
