package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mock_handlers "github.com/fragpit/gophkeeper/internal/api/handlers/mocks"
	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"go.uber.org/mock/gomock"
)

func TestNewAuthRegisterHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		tokens *model.TokenPair
		err    error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		reqBody        *authRequest
		contentType    string
		wantCode       int
		wantBodySubstr string
		wantErr        string
	}{
		{
			name: "success",
			mockData: &mockData{
				tokens: &model.TokenPair{
					AccessToken:  "access-1",
					RefreshToken: "refresh-1",
				},
				err: nil,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusOK,
			wantBodySubstr: `{"access_token":"access-1","refresh_token":"refresh-1","error":""}`,
		},
		{
			name: "user exists",
			mockData: &mockData{
				tokens: nil,
				err:    model.ErrUserExists,
			},
			reqBody: &authRequest{
				Login:    "",
				Password: "p",
			},
			wantCode:       http.StatusConflict,
			wantBodySubstr: http.StatusText(http.StatusConflict),
		},
		{
			name: "password policy violated",
			mockData: &mockData{
				tokens: nil,
				err:    model.ErrPasswordPolicyViolated,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "password policy violated",
		},
		{
			name: "internal error",
			mockData: &mockData{
				tokens: nil,
				err:    fmt.Errorf("db down"),
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
		{
			name: "wrong content type",
			mockData: &mockData{
				tokens: nil,
				err:    nil,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			contentType:    "wrong/type",
			wantCode:       http.StatusUnsupportedMediaType,
			wantBodySubstr: "empty or unsupported content type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockAuthService(ctrl)

			m.EXPECT().
				Register(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.mockData.tokens, tc.mockData.err).AnyTimes()

			rec := httptest.NewRecorder()

			if tc.contentType == "" {
				tc.contentType = "application/json"
			}

			b, _ := json.Marshal(tc.reqBody)
			req := httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(string(b)),
			)
			req.Header.Set("Content-Type", tc.contentType)

			e.POST("/", NewAuthRegisterHandler(m))
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf(
					"unexpected status code: got %d want %d",
					rec.Code,
					tc.wantCode,
				)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tc.wantBodySubstr) {
				t.Fatalf(
					"unexpected body: got %q want substring %q",
					body,
					tc.wantBodySubstr,
				)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf(
					"unexpected Content-Type: got %q want %q",
					ct,
					"application/json",
				)
			}
		})
	}
}

func TestNewAuthLoginHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		tokens *model.TokenPair
		err    error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		reqBody        *authRequest
		contentType    string
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			mockData: &mockData{
				tokens: &model.TokenPair{
					AccessToken:  "access-1",
					RefreshToken: "refresh-1",
				},
				err: nil,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusOK,
			wantBodySubstr: `{"access_token":"access-1","refresh_token":"refresh-1","error":""}`,
		},
		{
			name: "wrong username or password",
			mockData: &mockData{
				tokens: nil,
				err:    model.ErrInvalidCredentials,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusUnauthorized,
			wantBodySubstr: "invalid credentials",
		},
		{
			name: "internal error",
			mockData: &mockData{
				tokens: nil,
				err:    fmt.Errorf("db down"),
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
		{
			name: "wrong content type",
			mockData: &mockData{
				tokens: nil,
				err:    nil,
			},
			reqBody: &authRequest{
				Login:    "u",
				Password: "p",
			},
			contentType:    "wrong/type",
			wantCode:       http.StatusUnsupportedMediaType,
			wantBodySubstr: "empty or unsupported content type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock_handlers.NewMockAuthService(ctrl)

			m.EXPECT().
				Login(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.mockData.tokens, tc.mockData.err).AnyTimes()
			rec := httptest.NewRecorder()

			if tc.contentType == "" {
				tc.contentType = "application/json"
			}

			b, _ := json.Marshal(tc.reqBody)
			req := httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(string(b)),
			)
			req.Header.Set("Content-Type", tc.contentType)

			e := echo.New()
			e.POST("/", NewAuthLoginHandler(m))
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf(
					"unexpected status code: got %d want %d",
					rec.Code,
					tc.wantCode,
				)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tc.wantBodySubstr) {
				t.Fatalf(
					"unexpected body: got %q want substring %q",
					body,
					tc.wantBodySubstr,
				)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf(
					"unexpected Content-Type: got %q want %q",
					ct,
					"application/json",
				)
			}
		})
	}
}

func TestNewAuthRefreshHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		setupContext   func(*echo.Context)
		mockSetup      func(*mock_handlers.MockAuthService)
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.RefreshClaims{UserID: 1, TokenID: "token-1"},
				)
				c.Set("user", token)
			},
			mockSetup: func(m *mock_handlers.MockAuthService) {
				m.EXPECT().
					Refresh(gomock.Any(), 1, "token-1").
					Return(&model.TokenPair{AccessToken: "access", RefreshToken: "refresh"}, nil)
			},
			wantCode:       http.StatusOK,
			wantBodySubstr: `{"access_token":"access","refresh_token":"refresh","error":""}`,
		},
		{
			name: "missing token",
			setupContext: func(c *echo.Context) {
				c.Set("user", nil)
			},
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "invalid refresh token",
		},
		{
			name: "wrong claims",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.AccessClaims{UserID: 1},
				)
				c.Set("user", token)
			},
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "invalid refresh token claims",
		},
		{
			name: "missing claims data",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.RefreshClaims{UserID: 0, TokenID: ""},
				)
				c.Set("user", token)
			},
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "missing token ID or user ID in claims",
		},
		{
			name: "token not found",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.RefreshClaims{UserID: 2, TokenID: "token-2"},
				)
				c.Set("user", token)
			},
			mockSetup: func(m *mock_handlers.MockAuthService) {
				m.EXPECT().
					Refresh(gomock.Any(), 2, "token-2").
					Return(nil, model.ErrTokenNotFound)
			},
			wantCode:       http.StatusUnauthorized,
			wantBodySubstr: "refresh token not found or expired",
		},
		{
			name: "user not found",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.RefreshClaims{UserID: 3, TokenID: "token-3"},
				)
				c.Set("user", token)
			},
			mockSetup: func(m *mock_handlers.MockAuthService) {
				m.EXPECT().
					Refresh(gomock.Any(), 3, "token-3").
					Return(nil, model.ErrUserNotFound)
			},
			wantCode:       http.StatusUnauthorized,
			wantBodySubstr: model.ErrUserNotFound.Error(),
		},
		{
			name: "internal error",
			setupContext: func(c *echo.Context) {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					&auth.RefreshClaims{UserID: 4, TokenID: "token-4"},
				)
				c.Set("user", token)
			},
			mockSetup: func(m *mock_handlers.MockAuthService) {
				m.EXPECT().
					Refresh(gomock.Any(), 4, "token-4").
					Return(nil, fmt.Errorf("db down"))
			},
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockAuthService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(m)
			}

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tc.setupContext != nil {
				tc.setupContext(c)
			}

			h := NewAuthRefreshHandler(m)
			_ = h(c)

			if rec.Code != tc.wantCode {
				t.Fatalf(
					"unexpected status code: got %d want %d",
					rec.Code,
					tc.wantCode,
				)
			}

			if !strings.Contains(rec.Body.String(), tc.wantBodySubstr) {
				t.Fatalf(
					"unexpected body: got %q want substring %q",
					rec.Body.String(),
					tc.wantBodySubstr,
				)
			}
		})
	}
}
