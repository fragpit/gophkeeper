package client

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fragpit/gophkeeper/internal/model"
)

// ListResponse represents a list items response payload.
type ListResponse struct {
	Items []model.ItemMeta `json:"items"`
	Error string           `json:"error"`
}

// List retrieves and prints user items of a specific type.
func (c *Client) List(
	ctx context.Context,
	iType string,
) error {
	respData := &ListResponse{}
	resp, err := c.http.R().
		SetContext(ctx).
		SetAuthToken(string(c.jwtToken)).
		SetQueryParams(map[string]string{
			"type": iType,
		}).
		SetResult(respData).
		SetError(respData).
		Get("/api/list/items")
	if err != nil {
		return fmt.Errorf("list items request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.IsError() {
		return handleErrorResponse(resp, "list items", respData.Error)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "#\ttitle\ttype")
	for i, v := range respData.Items {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", i+1, v.Title, v.Type)
	}
	_ = w.Flush()

	return nil
}
