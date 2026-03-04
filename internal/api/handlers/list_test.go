package handlers

import (
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

func TestNewListHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		items []model.ItemMeta
		err   error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		itemType       string
		userID         int
		withAuth       bool
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			mockData: &mockData{
				items: []model.ItemMeta{
					{
						Title: "login1",
						Type:  model.ItemTypeLogin,
					},
					{
						Title: "login2",
						Type:  model.ItemTypeLogin,
					},
				},
				err: nil,
			},
			itemType:       "login",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"title":"login1"`,
		},
		{
			name: "success empty list",
			mockData: &mockData{
				items: []model.ItemMeta{},
				err:   nil,
			},
			itemType:       "note",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"items":[]`,
		},
		{
			name: "no auth",
			mockData: &mockData{
				items: nil,
				err:   nil,
			},
			itemType: "login",
			withAuth: false,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			mockData: &mockData{
				items: nil,
				err:   fmt.Errorf("db error"),
			},
			itemType:       "login",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
		{
			name:           "invalid item type",
			mockData:       nil,
			itemType:       "invalid_type",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "invalid item type",
		},
		{
			name: "empty item type",
			mockData: &mockData{
				items: []model.ItemMeta{
					{
						Title: "login1",
						Type:  model.ItemTypeLogin,
					},
					{
						Title: "note1",
						Type:  model.ItemTypeNote,
					},
				},
				err: nil,
			},
			itemType:       "",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"title":"login1"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockItemsService(ctrl)

			if tc.mockData != nil {
				m.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.items, tc.mockData.err).AnyTimes()
			}

			req := httptest.NewRequest(
				http.MethodGet,
				"/?type="+tc.itemType,
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
				e.GET("/", NewListHandler(m))
			} else {
				e.GET("/", NewListHandler(m))
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
