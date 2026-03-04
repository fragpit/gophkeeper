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

func TestClient_List(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		itemType       string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		checkOutput    bool
		wantErr        bool
		errContains    string
	}{
		{
			name:     "success with items",
			ctx:      context.Background(),
			itemType: "login",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/list/items", r.URL.Path)
				assert.Equal(t, "login", r.URL.Query().Get("type"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				resp := ListResponse{
					Items: []model.ItemMeta{
						{Title: "gmail", Type: model.ItemTypeLogin},
						{Title: "github", Type: model.ItemTypeLogin},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
		},
		{
			name:     "success with empty list",
			ctx:      context.Background(),
			itemType: "note",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "note", r.URL.Query().Get("type"))

				resp := ListResponse{Items: []model.ItemMeta{}}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
		},
		{
			name:     "server error",
			ctx:      context.Background(),
			itemType: "file",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := ListResponse{Error: "database connection failed"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "database connection failed",
		},
		{
			name:     "server error without message",
			ctx:      context.Background(),
			itemType: "login",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:     true,
			errContains: "status=500",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			itemType: "login",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := ListResponse{Items: []model.ItemMeta{}}
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
			itemType: "login",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				resp := ListResponse{Items: []model.ItemMeta{}}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:     "empty item type",
			ctx:      context.Background(),
			itemType: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "", r.URL.Query().Get("type"))
				resp := ListResponse{Items: []model.ItemMeta{}}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
		},
		{
			name:     "item type note",
			ctx:      context.Background(),
			itemType: "note",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "note", r.URL.Query().Get("type"))
				resp := ListResponse{
					Items: []model.ItemMeta{
						{Title: "note1", Type: model.ItemTypeNote},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
		},
		{
			name:     "item type file",
			ctx:      context.Background(),
			itemType: "file",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "file", r.URL.Query().Get("type"))
				resp := ListResponse{
					Items: []model.ItemMeta{
						{Title: "doc.pdf", Type: model.ItemTypeFile},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
		},
		{
			name:     "invalid item type",
			ctx:      context.Background(),
			itemType: "invalid_type",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "invalid_type", r.URL.Query().Get("type"))
				resp := ListResponse{Items: []model.ItemMeta{}}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			checkOutput: true,
			wantErr:     false,
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
			if tt.checkOutput {
				oldStdout := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w

				err = client.List(tt.ctx, tt.itemType)

				_ = w.Close()
				os.Stdout = oldStdout

				var buf bytes.Buffer
				_, copyErr := io.Copy(&buf, r)
				require.NoError(t, copyErr)

				if tt.wantErr {
					require.Error(t, err)
					if tt.errContains != "" {
						assert.Contains(t, err.Error(), tt.errContains)
					}
				} else {
					require.NoError(t, err)
				}
			} else {
				err = client.List(tt.ctx, tt.itemType)

				if tt.wantErr {
					require.Error(t, err)
					if tt.errContains != "" {
						assert.Contains(t, err.Error(), tt.errContains)
					}
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestClient_List_RefreshFailure(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"/api/list/items",
		func(w http.ResponseWriter, r *http.Request) {
			resp := ListResponse{Error: "invalid token"}
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

	err = client.List(context.Background(), "note")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token expired")
}

func TestClient_List_RefreshSuccess(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()

	mux.HandleFunc(
		"/api/list/items",
		func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))
				resp := ListResponse{Error: "invalid token"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			assert.Equal(t, "Bearer access-2", r.Header.Get("Authorization"))
			resp := ListResponse{
				Items: []model.ItemMeta{{Title: "note1", Type: model.ItemTypeNote}},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		},
	)

	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer refresh-1", r.Header.Get("Authorization"))
		resp := refreshTokenResponse{
			AccessToken:  "access-2",
			RefreshToken: "refresh-2",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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

	err = client.List(context.Background(), "note")
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func createTempTokenFile(
	t *testing.T,
	accessToken string,
	refreshToken string,
) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "token-*.txt")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	cfg := clientConfig{
		Auth: authConfig{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	_ = tmpFile.Close()

	return tmpFile.Name()
}
