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
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.jwtToken)).
		SetQueryParam("title", title).
		SetResult(respData).
		SetError(respData).
		Get("/api/get/item")
	if err != nil {
		return fmt.Errorf("get item request: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return handleErrorResponse(resp, "get item", respData.Error)
	}

	itemJSON, _ := json.MarshalIndent(&respData.Item, "", "  ")
	fmt.Println(string(itemJSON))

	if respData.Item.Type == model.ItemTypeFile {
		if filePath == "" {
			return fmt.Errorf("file path not provided")
		}
		resp, err := c.http.R().
			SetContext(ctx).
			SetAuthToken(string(c.jwtToken)).
			SetQueryParam("title", title).
			SetDoNotParseResponse(true).
			Get("/api/get/file")
		if err != nil {
			return fmt.Errorf("get file request: %w", err)
		}
		defer resp.Body.Close()

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
		defer f.Close()

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
