package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Login(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		login          string
		password       string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:     "success login",
			ctx:      context.Background(),
			login:    "testuser",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/login", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "testuser", body["login"])
				assert.Equal(t, "testpass", body["password"])

				resp := LoginResponse{
					AccessToken:  "jwt-token-123",
					RefreshToken: "refresh-123",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:     "invalid credentials",
			ctx:      context.Background(),
			login:    "testuser",
			password: "wrongpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := LoginResponse{Error: "invalid credentials"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "invalid credentials",
		},
		{
			name:     "server error",
			ctx:      context.Background(),
			login:    "testuser",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := LoginResponse{Error: "internal server error"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "internal server error",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			login:    "testuser",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := LoginResponse{
					AccessToken:  "jwt-token-123",
					RefreshToken: "refresh-123",
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
			login:    "testuser",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				resp := LoginResponse{
					AccessToken:  "jwt-token-123",
					RefreshToken: "refresh-123",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context deadline exceeded",
		},
		{
			name:     "empty login",
			ctx:      context.Background(),
			login:    "",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "", body["login"])

				resp := LoginResponse{
					AccessToken:  "jwt-token-123",
					RefreshToken: "refresh-123",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:     "empty password",
			ctx:      context.Background(),
			login:    "testuser",
			password: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "", body["password"])

				resp := LoginResponse{
					AccessToken:  "jwt-token-123",
					RefreshToken: "refresh-123",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
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
			client.SetDisableWarn(true)

			err = client.Login(tt.ctx, tt.login, tt.password)

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

func TestClient_Register(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		login          string
		password       string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:     "success register",
			ctx:      context.Background(),
			login:    "newuser",
			password: "newpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/register", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "newuser", body["login"])
				assert.Equal(t, "newpass", body["password"])

				resp := RegisterResponse{
					AccessToken:  "jwt-token-456",
					RefreshToken: "refresh-456",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr: false,
		},
		{
			name:     "user already exists",
			ctx:      context.Background(),
			login:    "existinguser",
			password: "testpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := RegisterResponse{Error: "user already exists"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "user already exists",
		},
		{
			name:     "invalid input",
			ctx:      context.Background(),
			login:    "ab",
			password: "12",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := RegisterResponse{Error: "validation failed"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			login:    "newuser",
			password: "newpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := RegisterResponse{
					AccessToken:  "jwt-token-456",
					RefreshToken: "refresh-456",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			},
			wantErr:     true,
			errContains: "context canceled",
		},
		{
			name:     "server error",
			ctx:      context.Background(),
			login:    "newuser",
			password: "newpass",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				resp := RegisterResponse{Error: "database error"}
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
				createTempTokenFile(t, "test-token", "refresh-token"),
				true,
			)
			require.NoError(t, err)
			client.SetDisableWarn(true)

			err = client.Register(tt.ctx, tt.login, tt.password)

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
