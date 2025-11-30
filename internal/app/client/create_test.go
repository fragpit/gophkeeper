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

func TestClient_CreateItem(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		item           *model.ItemDecrypted
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name: "success create login item",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "gmail",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{"login":"user","password":"pass"}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/create/item", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				var body createItemRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "gmail", body.Item.Title)
				assert.Equal(t, model.ItemTypeLogin, body.Item.Type)

				resp := createItemResponse{ID: 1}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name: "success create note item",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "mynote",
					Type:  model.ItemTypeNote,
				},
				Data: []byte(`{"text":"secret note"}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body createItemRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "mynote", body.Item.Title)
				assert.Equal(t, model.ItemTypeNote, body.Item.Type)

				resp := createItemResponse{ID: 2}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name: "item already exists",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "duplicate",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createItemResponse{Error: "item already exists"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "item already exists",
		},
		{
			name: "unauthorized",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createItemResponse{Error: "invalid token"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "invalid token",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createItemResponse{ID: 1}
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
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				resp := createItemResponse{ID: 1}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name: "server error",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: []byte(`{}`),
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createItemResponse{Error: "database error"}
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
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token"),
				true,
			)
			require.NoError(t, err)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = client.CreateItem(tt.ctx, tt.item)
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

func TestClient_CreateFile(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		item           *model.ItemDecrypted
		fileContent    string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name: "success create file item",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "document",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{"filename":"test.pdf"}`),
			},
			fileContent: "test file content",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/create/file", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				err := r.ParseMultipartForm(10 << 20)
				require.NoError(t, err)

				itemJSON := r.FormValue("item")
				var item model.ItemDecrypted
				_ = json.Unmarshal([]byte(itemJSON), &item)
				assert.Equal(t, "document", item.Title)
				assert.Equal(t, model.ItemTypeFile, item.Type)

				file, _, err := r.FormFile("file")
				require.NoError(t, err)
				defer func() { _ = file.Close() }()

				content, err := io.ReadAll(file)
				require.NoError(t, err)
				assert.Equal(t, "test file content", string(content))

				resp := createFileResponse{ID: 1}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name: "file already exists",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "duplicate",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{}`),
			},
			fileContent: "content",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createFileResponse{Error: "item already exists"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "item already exists",
		},
		{
			name: "unauthorized",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{}`),
			},
			fileContent: "content",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createFileResponse{Error: "invalid token"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "unauthorized",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{}`),
			},
			fileContent: "content",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createFileResponse{ID: 1}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context canceled",
		},
		{
			name: "server error",
			ctx:  context.Background(),
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{}`),
			},
			fileContent: "content",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := createFileResponse{Error: "storage error"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "storage error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			tmpFile, err := os.CreateTemp("", "testfile-*.txt")
			require.NoError(t, err)
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			_, err = tmpFile.WriteString(tt.fileContent)
			require.NoError(t, err)
			_ = tmpFile.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token"),
				true,
			)
			require.NoError(t, err)
			client.SetDisableWarn(true)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = client.CreateFile(tt.ctx, tt.item, tmpFile.Name())

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

func TestClient_CreateFile_FileErrors(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "file not found",
			filePath:    "/nonexistent/file.txt",
			wantErr:     true,
			errContains: "open file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := createFileResponse{ID: 1}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(resp)
				}),
			)
			defer server.Close()

			client, err := NewClient(
				server.URL,
				createTempTokenFile(t, "test-token"),
				true,
			)
			require.NoError(t, err)
			client.SetDisableWarn(true)

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeFile,
				},
				Data: []byte(`{}`),
			}

			err = client.CreateFile(context.Background(), item, tt.filePath)

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
