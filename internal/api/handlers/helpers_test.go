package handlers

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func TestValidateParseJSONRequest(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type reqParams struct {
		ContentType string
		JSONData    string
	}

	validJSON := `{"test":"test"}`
	invalidJSON := `{"test":"test`
	largeJSON := `{"data":"` + strings.Repeat("a", 1<<20) + `"}`

	tests := []struct {
		name            string
		params          reqParams
		wantErr         bool
		wantHTTPCode    int
		wantEchoHTTPErr bool
	}{
		{
			name: "success",
			params: reqParams{
				ContentType: "application/json",
				JSONData:    validJSON,
			},
			wantErr: false,
		},
		{
			name: "body too large",
			params: reqParams{
				ContentType: "application/json",
				JSONData:    largeJSON,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusRequestEntityTooLarge,
		},
		{
			name: "wrong content type",
			params: reqParams{
				ContentType: "unknown",
			},
			wantErr:         true,
			wantHTTPCode:    http.StatusUnsupportedMediaType,
			wantEchoHTTPErr: true,
		},
		{
			name: "invalid json",
			params: reqParams{
				ContentType: "application/json",
				JSONData:    invalidJSON,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()

			inData := []byte(tc.params.JSONData)
			r := bytes.NewReader(inData)

			req := httptest.NewRequest(http.MethodPost, "/", r)
			req.Header.Add("Content-Type", tc.params.ContentType)

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var outData any
			err := ValidateParseJSONRequest(c, &outData)

			if tc.wantErr {
				if err == nil && rec.Code == 0 {
					t.Fatalf("expected error but got nil")
				}
				if tc.wantHTTPCode > 0 {
					if tc.wantEchoHTTPErr {
						var httpErr *echo.HTTPError
						if !errors.As(err, &httpErr) {
							t.Fatalf("expected *echo.HTTPError but got %T", err)
						}
						if httpErr.Code != tc.wantHTTPCode {
							t.Fatalf(
								"expected HTTP code %d but got %d",
								tc.wantHTTPCode,
								httpErr.Code,
							)
						}
					} else {
						if rec.Code != tc.wantHTTPCode {
							t.Fatalf(
								"expected HTTP code %d but got %d",
								tc.wantHTTPCode,
								rec.Code,
							)
						}
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExtractClaims(t *testing.T) {
	newContext := func() *echo.Context {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("valid AccessClaims", func(t *testing.T) {
		c := newContext()
		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			&auth.AccessClaims{UserID: 42},
		)
		c.Set("user", token)

		claims, err := extractClaims[*auth.AccessClaims](c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != 42 {
			t.Fatalf("expected UserID 42, got %d", claims.UserID)
		}
	})

	t.Run("valid RefreshClaims", func(t *testing.T) {
		c := newContext()
		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			&auth.RefreshClaims{UserID: 7, TokenID: "tok"},
		)
		c.Set("user", token)

		claims, err := extractClaims[*auth.RefreshClaims](c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != 7 || claims.TokenID != "tok" {
			t.Fatalf("unexpected claims: %+v", claims)
		}
	})

	t.Run("no token in context", func(t *testing.T) {
		c := newContext()
		// "user" key not set at all

		_, err := extractClaims[*auth.AccessClaims](c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 HTTPError, got %v", err)
		}
	})

	t.Run("nil token in context", func(t *testing.T) {
		c := newContext()
		c.Set("user", nil)

		_, err := extractClaims[*auth.AccessClaims](c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 HTTPError, got %v", err)
		}
	})

	t.Run("wrong claims type", func(t *testing.T) {
		c := newContext()
		// AccessClaims token, but extracting as RefreshClaims
		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			&auth.AccessClaims{UserID: 1},
		)
		c.Set("user", token)

		_, err := extractClaims[*auth.RefreshClaims](c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 HTTPError, got %v", err)
		}
	})

	t.Run("non-token value in context", func(t *testing.T) {
		c := newContext()
		c.Set("user", "not-a-jwt-token")

		_, err := extractClaims[*auth.AccessClaims](c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 HTTPError, got %v", err)
		}
	})
}
