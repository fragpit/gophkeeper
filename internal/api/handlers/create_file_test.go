package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
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

func TestNewCreateFileHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	type mockData struct {
		id  int
		err error
	}

	tests := []struct {
		name           string
		mockData       *mockData
		item           *model.ItemDecrypted
		fileContent    string
		fileSize       int64 // для явного указания размера файла
		userID         int
		withAuth       bool
		noItem         bool
		noFile         bool
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "success",
			mockData: &mockData{
				id:  456,
				err: nil,
			},
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"test.txt"}`),
			},
			fileContent:    "file content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusOK,
			wantBodySubstr: `"id":456`,
		},
		{
			name: "no auth",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
			},
			fileContent: "file content",
			withAuth:    false,
			wantCode:    http.StatusUnauthorized,
		},
		{
			name: "item already exists",
			mockData: &mockData{
				id:  0,
				err: model.ErrAlreadyExists,
			},
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
			},
			fileContent:    "file content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusConflict,
			wantBodySubstr: "item already exists",
		},
		{
			name: "internal error",
			mockData: &mockData{
				id:  0,
				err: fmt.Errorf("s3 down"),
			},
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
			},
			fileContent:    "file content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusInternalServerError,
			wantBodySubstr: "failed to create file",
		},
		{
			name: "missing item",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			noItem:         true,
			fileContent:    "file content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "item is required",
		},
		{
			name: "missing file",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
			},
			noFile:         true,
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "file is required",
		},
		{
			name: "invalid item json",
			mockData: &mockData{
				id:  0,
				err: nil,
			},
			item:           nil,
			fileContent:    "file content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "invalid item format",
		},
		{
			name:     "file size exceeds limit",
			mockData: nil, // No mock setup - validation should fail before service is called
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "largefile",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"large.bin"}`),
			},
			fileContent:    "dummy",
			fileSize:       MaxFileSize + 1, // превышаем лимит
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusRequestEntityTooLarge,
			wantBodySubstr: "file size exceeds maximum limit",
		},
		{
			name:     "item title too long in file upload",
			mockData: nil, // No mock setup - validation should fail before service is called
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: strings.Repeat("a", MaxItemTitleLength+1),
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"test.txt"}`),
			},
			fileContent:    "content",
			userID:         1,
			withAuth:       true,
			wantCode:       http.StatusBadRequest,
			wantBodySubstr: "item title too long",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			e := echo.New()
			m := mock_handlers.NewMockCreateFileService(ctrl)

			if tc.mockData != nil && tc.mockData.err != nil ||
				(tc.mockData != nil && !tc.noItem && !tc.noFile) {
				m.EXPECT().
					CreateFileItem(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tc.mockData.id, tc.mockData.err).AnyTimes()
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			if !tc.noItem && tc.item != nil {
				itemJSON, _ := json.Marshal(tc.item)
				_ = writer.WriteField("item", string(itemJSON))
			} else if !tc.noItem && tc.item == nil {
				_ = writer.WriteField("item", "{invalid json")
			}

			if !tc.noFile {
				part, _ := writer.CreateFormFile("file", "test.txt")
				if tc.fileSize > 0 {
					// Write specified fileSize bytes
					_, _ = io.CopyN(
						part,
						io.LimitReader(
							bytes.NewReader(make([]byte, tc.fileSize)),
							tc.fileSize,
						),
						tc.fileSize,
					)
				} else {
					_, _ = io.WriteString(part, tc.fileContent)
				}
			}

			_ = writer.Close()

			req := httptest.NewRequest(
				http.MethodPost,
				"/",
				body,
			)
			req.Header.Set("Content-Type", writer.FormDataContentType())

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
				e.POST("/", NewCreateFileHandler(m))
			} else {
				e.POST("/", NewCreateFileHandler(m))
			}

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf(
					"unexpected status code: got %d want %d, body: %s",
					rec.Code,
					tc.wantCode,
					rec.Body.String(),
				)
			}

			if tc.wantBodySubstr != "" {
				respBody := rec.Body.String()
				if !strings.Contains(respBody, tc.wantBodySubstr) {
					t.Fatalf(
						"unexpected body: got %q want substring %q",
						respBody,
						tc.wantBodySubstr,
					)
				}
			}
		})
	}
}
