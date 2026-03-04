package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_UpdateItem(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		title          string
		item           *model.ItemDecrypted
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:  "success update login item",
			ctx:   context.Background(),
			title: "gmail",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "gmail",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{"login":"newuser","password":"newpass"}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/update/item", r.URL.Path)
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				var body updateItemRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "gmail", body.Title)
				assert.Equal(t, "gmail", body.Item.Title)
				assert.Equal(t, model.ItemTypeLogin, body.Item.Type)

				resp := updateItemResponse{}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:  "success update note item",
			ctx:   context.Background(),
			title: "mynote",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "mynote",
					Type:  model.ItemTypeNote,
				},
				Data: []byte(`{"text":"updated secret note"}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body updateItemRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "mynote", body.Title)
				assert.Equal(t, "mynote", body.Item.Title)
				assert.Equal(t, model.ItemTypeNote, body.Item.Type)

				resp := updateItemResponse{}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:  "item conflict",
			ctx:   context.Background(),
			title: "duplicate",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "duplicate",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := updateItemResponse{Error: "item already exists"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "item already exists",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			title: "test",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := updateItemResponse{}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context canceled",
		},
		{
			name: "context timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(
					context.Background(),
					1*time.Nanosecond,
				)
				defer cancel()
				time.Sleep(10 * time.Millisecond)
				return ctx
			}(),
			title: "test",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				resp := updateItemResponse{}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:  "server error",
			ctx:   context.Background(),
			title: "test",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := updateItemResponse{Error: "database error"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "database error",
		},
		{
			name:  "bad request - empty title",
			ctx:   context.Background(),
			title: "",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := updateItemResponse{Error: "title is required"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token", "refresh-token"),
				true,
			)
			require.NoError(t, err)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = client.UpdateItem(tt.ctx, tt.title, tt.item)
			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_UpdateItem_RefreshFailure(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"/api/update/item",
		func(w http.ResponseWriter, r *http.Request) {
			resp := updateItemResponse{Error: "invalid token"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(resp)
		},
	)

	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		resp := refreshTokenResponse{Error: "refresh token expired"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(
		server.URL,
		createTempTokenFile(t, "access-1", "refresh-1"),
		true,
	)
	require.NoError(t, err)

	item := &model.ItemDecrypted{
		ItemMeta: &model.ItemMeta{
			Title: "test",
			Type:  model.ItemTypeLogin,
		},
		Data: []byte(`{}`),
	}

	err = client.UpdateItem(context.Background(), "test", item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token expired")
}
