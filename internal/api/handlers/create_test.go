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

func TestNewCreateHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		id  int
		err error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		reqBody        interface{}
		rawBody        string
		userID         int
		withAuth       bool
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			mockData: &mockData{
				id:  123,
				err: nil,
			},
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "test",
						Type:  model.ItemTypeLogin,
					},
					Data: json.RawMessage(`{"username":"user"}`),
				},
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"id":123`,
		},
		{
			name: "invalid json",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			rawBody:        `{"item": invalid}`,
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: http.StatusText(http.StatusBadRequest),
		},
		{
			name: "missing item",
			mockData: &mockData{
				id:  123,
				err: nil,
			},
			reqBody: &createRequest{
				Item: nil,
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: `item is nil`,
		},
		{
			name: "no auth",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "test",
						Type:  model.ItemTypeLogin,
					},
				},
			},
			withAuth: false,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "item already exists",
			mockData: &mockData{
				id:  0,
				err: model.ErrAlreadyExists,
			},
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "test",
						Type:  model.ItemTypeLogin,
					},
				},
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusConflict,
			wantBodySubstr: "already exists",
		},
		{
			name: "internal error",
			mockData: &mockData{
				id:  0,
				err: fmt.Errorf("db down"),
			},
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "test",
						Type:  model.ItemTypeLogin,
					},
				},
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: http.StatusText(http.StatusInternalServerError),
		},
		{
			name:     "item title too long",
			mockData: nil, // No mock setup - validation should fail before service is called
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: strings.Repeat("a", MaxItemTitleLength+1),
						Type:  model.ItemTypeLogin,
					},
					Data: json.RawMessage(`{"username":"user"}`),
				},
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "item title too long",
		},
		{
			name:     "item data too large",
			mockData: nil, // No mock setup - validation should fail before service is called
			reqBody: &createRequest{
				Item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "test",
						Type:  model.ItemTypeNote,
					},
					Data: json.RawMessage(
						fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("x", MaxItemDataSize)),
					),
				},
			},
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "item data too large",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockCreateService(ctrl)

			if tc.mockData != nil && tc.mockData.err == nil {
				m.EXPECT().
					CreateItem(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.id, tc.mockData.err).AnyTimes()
			} else if tc.mockData != nil && tc.mockData.err != nil {
				m.EXPECT().
					CreateItem(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.id, tc.mockData.err).AnyTimes()
			}

			var body string
			if tc.rawBody != "" {
				body = tc.rawBody
			} else {
				b, _ := json.Marshal(tc.reqBody)
				body = string(b)
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			if tc.withAuth {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.Claims{
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
						return &auth.Claims{}
					},
				}))
				e.POST("/", NewCreateHandler(m))
			} else {
				e.POST("/", NewCreateHandler(m))
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
