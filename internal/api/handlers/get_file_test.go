package handlers

import (
	"fmt"
	"io"
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

type mockReadCloser struct {
	io.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func TestNewGetFileHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		item       *model.ItemEncrypted
		itemErr    error
		fileReader io.ReadCloser
		fileErr    error
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
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "myfile",
						Type:  model.ItemTypeFile,
					},
					Ciphertext: []byte("encrypted"),
				},
				itemErr:    nil,
				fileReader: &mockReadCloser{Reader: strings.NewReader("file content")},
				fileErr:    nil,
			},
			title:          "myfile",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: "file content",
		},
		{
			name: "no auth",
			mockData: &mockData{
				item:    nil,
				itemErr: nil,
			},
			title:    "myfile",
			withAuth: false,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "get item error",
			mockData: &mockData{
				item:    nil,
				itemErr: fmt.Errorf("db error"),
			},
			title:    "myfile",
			userID:   1,
			withAuth: true,
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "get file error",
			mockData: &mockData{
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "myfile",
						Type:  model.ItemTypeFile,
					},
				},
				itemErr:    nil,
				fileReader: nil,
				fileErr:    fmt.Errorf("s3 error"),
			},
			title:    "myfile",
			userID:   1,
			withAuth: true,
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockGetFileService(ctrl)

			if tc.mockData != nil {
				m.EXPECT().
					GetItemByTitle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.item, tc.mockData.itemErr).AnyTimes()

				if tc.mockData.itemErr == nil && tc.mockData.item != nil {
					m.EXPECT().
						GetFileByItem(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(tc.mockData.fileReader, tc.mockData.fileErr).AnyTimes()
				}
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
				e.GET("/", NewGetFileHandler(m))
			} else {
				e.GET("/", NewGetFileHandler(m))
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
