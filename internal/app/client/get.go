package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fragpit/gophkeeper/internal/model"
)

// GetResponse represents a response containing a decrypted item.
type GetResponse struct {
	Item  *model.ItemDecrypted `json:"item"`
	Error string               `json:"error"`
}

// GetItemMetadata fetches only the item metadata without downloading file content.
func (c *Client) GetItemMetadata(
	ctx context.Context,
	title string,
) (*model.ItemDecrypted, error) {
	if title == "" {
		return nil, fmt.Errorf("title not provided")
	}

	respData := &GetResponse{}
	req := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.accessToken)).
		SetQueryParam("title", title).
		SetResult(respData).
		SetError(respData)

	resp, err := c.requestWithAuth(ctx, "/api/get/item", req, req.Get)
	if err != nil {
		return nil, fmt.Errorf("get item request: %w", err)
	}
	if resp.IsError() {
		return nil, handleErrorResponse(resp, "get item", respData.Error)
	}
	defer func() { _ = resp.Body.Close() }()

	return respData.Item, nil
}

// GetItem fetches an item by title and optionally downloads its file content.
func (c *Client) GetItem(
	ctx context.Context,
	title string,
	filePath string,
) error {
	if title == "" {
		return fmt.Errorf("title not provided")
	}

	respData := &GetResponse{}
	req := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.accessToken)).
		SetQueryParam("title", title).
		SetResult(respData).
		SetError(respData)

	resp, err := c.requestWithAuth(ctx, "/api/get/item", req, req.Get)
	if err != nil {
		return fmt.Errorf("get item request: %w", err)
	}
	if resp.IsError() {
		return handleErrorResponse(resp, "get item", respData.Error)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "get item", respData.Error)
	}

	itemJSON, _ := json.MarshalIndent(&respData.Item, "", "  ")
	fmt.Println(string(itemJSON))

	if respData.Item.Type == model.ItemTypeFile {
		if filePath == "" {
			return fmt.Errorf("file path not provided")
		}
		req := c.http.R().
			SetContext(ctx).
			SetAuthToken(string(c.accessToken)).
			SetQueryParam("title", title).
			SetDoNotParseResponse(true)

		resp, err := c.requestWithAuth(ctx, "/api/get/file", req, req.Get)
		if err != nil {
			return fmt.Errorf("get file request: %w", err)
		}
		if resp.IsError() {
			return handleErrorResponse(resp, "get file", respData.Error)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.IsError() {
			return handleErrorResponse(resp, "get file", respData.Error)
		}

		var fileExist bool
		info, err := os.Stat(filePath)
		if err == nil {
			fileExist = true
			if info.IsDir() {
				return fmt.Errorf("path %s is a directory", filePath)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat file %s: %w", filePath, err)
		}

		if fileExist && !userConfirmOverwrite(filePath) {
			return fmt.Errorf("file %s exist and can't be overwritten", filePath)
		}

		f, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("get file: %w", err)
		}
		defer func() { _ = f.Close() }()

		if _, err := io.Copy(f, resp.Body); err != nil {
			return fmt.Errorf("get file: %w", err)
		}

		fmt.Printf("file saved to: %s\n", filePath)
	}

	return nil
}

func userConfirmOverwrite(name string) bool {
	fmt.Printf("file %s already exist, do you want to overwrite?[y/n]: ", name)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if scanner.Err() != nil {
		return false
	}

	input := scanner.Text()
	return input == "y"
}
