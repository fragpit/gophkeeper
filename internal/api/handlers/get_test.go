package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mock_handlers "github.com/fragpit/gophkeeper/internal/api/handlers/mocks"
	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"go.uber.org/mock/gomock"
)

func TestNewGetHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		item *model.ItemDecrypted
		err  error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		title          string
		userID         int
		withAuth       bool
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			mockData: &mockData{
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "mylogin",
						Type:  model.ItemTypeLogin,
					},
					Data: json.RawMessage(`{"username":"user","password":"pass"}`),
				},
				err: nil,
			},
			title:          "mylogin",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"title":"mylogin"`,
		},
		{
			name: "empty title",
			mockData: &mockData{
				item: nil,
				err:  fmt.Errorf("not found"),
			},
			title:          "",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "title parameter is required",
		},
		{
			name: "no auth",
			mockData: &mockData{
				item: nil,
				err:  nil,
			},
			title:    "mylogin",
			withAuth: false,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			mockData: &mockData{
				item: nil,
				err:  fmt.Errorf("db error"),
			},
			title:          "mylogin",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockGetService(ctrl)

			if tc.mockData != nil {
				m.EXPECT().
					GetItemByTitle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.item, tc.mockData.err).AnyTimes()
			}

			req := httptest.NewRequest(
				http.MethodGet,
				"/?title="+tc.title,
				nil,
			)

			if tc.withAuth {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.AccessClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
					UserID: tc.userID,
				})
				tokenString, _ := token.SignedString([]byte("secret"))
				req.Header.Set("Authorization", "Bearer "+tokenString)

				e.Use(echojwt.WithConfig(echojwt.Config{
					SigningKey: []byte("secret"),
					NewClaimsFunc: func(c *echo.Context) jwt.Claims {
						return &auth.AccessClaims{}
					},
				}))
				e.GET("/", NewGetHandler(m))
			} else {
				e.GET("/", NewGetHandler(m))
			}

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf(
					"unexpected status code: got %d want %d",
					rec.Code,
					tc.wantCode,
				)
			}

			if tc.wantBodySubstr != "" {
				body := rec.Body.String()
				if !strings.Contains(body, tc.wantBodySubstr) {
					t.Fatalf(
						"unexpected body: got %q want substring %q",
						body,
						tc.wantBodySubstr,
					)
				}
			}
		})
	}
}
