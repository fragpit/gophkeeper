package handlers

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
