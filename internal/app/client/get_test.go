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

func TestClient_GetItem(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		title          string
		filePath       string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:     "success get login item",
			ctx:      context.Background(),
			title:    "gmail",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/get/item", r.URL.Path)
				assert.Equal(t, "gmail", r.URL.Query().Get("title"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				resp := GetResponse{
					Item: &model.ItemDecrypted{
						ItemMeta: &model.ItemMeta{
							Title: "gmail",
							Type:  model.ItemTypeLogin,
						},
						Data: []byte(`{"login":"user@gmail.com","password":"pass123"}`),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:     "success get note item",
			ctx:      context.Background(),
			title:    "mynote",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "mynote", r.URL.Query().Get("title"))

				resp := GetResponse{
					Item: &model.ItemDecrypted{
						ItemMeta: &model.ItemMeta{
							Title: "mynote",
							Type:  model.ItemTypeNote,
						},
						Data: []byte(`{"text":"my secret note"}`),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:     "item not found",
			ctx:      context.Background(),
			title:    "nonexistent",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := GetResponse{Error: "item not found"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "item not found",
		},
		{
			name:     "empty title",
			ctx:      context.Background(),
			title:    "",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:     true,
			errContains: "title not provided",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			title:    "test",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := GetResponse{
					Item: &model.ItemDecrypted{
						ItemMeta: &model.ItemMeta{
							Title: "test",
							Type:  model.ItemTypeLogin,
						},
						Data: []byte(`{}`),
					},
				}
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
			title:    "test",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				resp := GetResponse{
					Item: &model.ItemDecrypted{
						ItemMeta: &model.ItemMeta{
							Title: "test",
							Type:  model.ItemTypeLogin,
						},
						Data: []byte(`{}`),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:     "server error",
			ctx:      context.Background(),
			title:    "test",
			filePath: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := GetResponse{Error: "database error"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/get/item", tt.serverResponse)

			server := httptest.NewServer(mux)
			defer server.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token", "refresh-token"),
				true,
			)
			require.NoError(t, err)
			client.SetDisableWarn(true)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = client.GetItem(tt.ctx, tt.title, tt.filePath)

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

func TestClient_GetItem_RefreshFailure(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/get/item", func(w http.ResponseWriter, r *http.Request) {
		resp := GetResponse{Error: "invalid token"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
	})

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

	err = client.GetItem(context.Background(), "test", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token expired")
}

func TestClient_GetItem_FileDownload(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		title        string
		fileContent  string
		wantErr      bool
		errContains  string
		skipFilePath bool
	}{
		{
			name:        "success download file",
			ctx:         context.Background(),
			title:       "document",
			fileContent: "test file content",
			wantErr:     false,
		},
		{
			name:         "file path not provided",
			ctx:          context.Background(),
			title:        "document",
			fileContent:  "content",
			skipFilePath: true,
			wantErr:      true,
			errContains:  "file path not provided",
		},
		{
			name:        "file download error",
			ctx:         context.Background(),
			title:       "document",
			fileContent: "content",
			wantErr:     true,
			errContains: "status=500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if !tt.skipFilePath {
				tmpFile, err := os.CreateTemp("", "download-*.txt")
				require.NoError(t, err)
				_ = tmpFile.Close()
				filePath = tmpFile.Name()
				defer func() { _ = os.Remove(filePath) }()
			}

			mux := http.NewServeMux()
			mux.HandleFunc(
				"/api/get/item",
				func(w http.ResponseWriter, r *http.Request) {
					resp := GetResponse{
						Item: &model.ItemDecrypted{
							ItemMeta: &model.ItemMeta{
								Title: "document",
								Type:  model.ItemTypeFile,
							},
							Data: []byte(`{"filename":"doc.pdf"}`),
						},
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				},
			)

			if tt.name == "file download error" {
				mux.HandleFunc(
					"/api/get/file",
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
					},
				)
			} else if !tt.skipFilePath {
				mux.HandleFunc(
					"/api/get/file",
					func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "/api/get/file", r.URL.Path)
						assert.Equal(t, "document", r.URL.Query().Get("title"))
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(tt.fileContent))
					},
				)
			}

			server := httptest.NewServer(mux)
			defer server.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token", "refresh-token"),
				true,
			)
			require.NoError(t, err)
			client.SetDisableWarn(true)

			oldStdout := os.Stdout
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdout = w

			stdinR, stdinW, _ := os.Pipe()
			os.Stdin = stdinR
			_, _ = stdinW.WriteString("y\n")
			_ = stdinW.Close()

			err = client.GetItem(tt.ctx, tt.title, filePath)

			_ = w.Close()
			os.Stdout = oldStdout
			os.Stdin = oldStdin

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.Equal(t, tt.fileContent, string(content))
			}
		})
	}
}
