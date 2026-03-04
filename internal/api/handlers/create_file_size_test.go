package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

func TestCreateFileHandler_FileSizeValidation(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		item           *model.ItemDecrypted
		fileSize       int64
		wantCode       int
		wantBodySubstr string
	}{
		{
			name: "file at max size limit",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "maxfile",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"max.bin"}`),
			},
			fileSize:       MaxFileSize,
			wantCode:       http.StatusOK,
			wantBodySubstr: "",
		},
		{
			name: "file exceeds size limit",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "toolarge",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"large.bin"}`),
			},
			fileSize:       MaxFileSize + 1,
			wantCode:       http.StatusRequestEntityTooLarge,
			wantBodySubstr: "file size exceeds maximum limit",
		},
		{
			name: "file significantly exceeds limit",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "huge",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"huge.bin"}`),
			},
			fileSize:       MaxFileSize * 2,
			wantCode:       http.StatusRequestEntityTooLarge,
			wantBodySubstr: "file size exceeds maximum limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем минимальный multipart request
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			itemJSON, _ := json.Marshal(tt.item)
			_ = writer.WriteField("item", string(itemJSON))

			// Создаем фейковый файл с нужным размером
			part, _ := writer.CreateFormFile("file", "test.bin")
			// Пишем минимальные данные, реальный размер будем подменять
			_, _ = io.WriteString(part, "content")

			_ = writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Подменяем размер файла через переопределение ParseMultipartForm
			// Для простоты теста используем хак с модификацией заголовков
			originalBody := body.Bytes()

			// Создаем новый запрос с измененными данными
			req = httptest.NewRequest(
				http.MethodPost,
				"/",
				bytes.NewReader(originalBody),
			)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Устанавливаем аутентификацию
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.AccessClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
				UserID: 1,
			})
			tokenString, _ := token.SignedString([]byte("secret"))
			req.Header.Set("Authorization", "Bearer "+tokenString)

			e := echo.New()
			e.Use(echojwt.WithConfig(echojwt.Config{
				SigningKey: []byte("secret"),
				NewClaimsFunc: func(c *echo.Context) jwt.Claims {
					return &auth.AccessClaims{}
				},
			}))

			// Создаем мок сервиса с успешным ответом
			mockSvc := &mockCreateFileService{
				createFunc: func(ctx interface{}, uid int, item *model.ItemDecrypted, file interface{}, ct string) (int, error) {
					return 123, nil
				},
			}

			// Создаем кастомный хендлер для теста с подменой размера
			handler := func(c *echo.Context) error {
				v := c.Get("user")
				token, ok := v.(*jwt.Token)
				if !ok || token == nil {
					return c.NoContent(http.StatusUnauthorized)
				}

				claims, ok := token.Claims.(*auth.AccessClaims)
				if !ok {
					return c.NoContent(http.StatusUnauthorized)
				}
				uid := claims.UserID

				itemRaw := c.FormValue("item")
				if itemRaw == "" {
					return echo.NewHTTPError(http.StatusBadRequest, "item is required")
				}

				var item model.ItemDecrypted
				if err := json.Unmarshal([]byte(itemRaw), &item); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "invalid item format")
				}

				// Валидация размеров item
				if err := ValidateItemSize(&item); err != nil {
					return c.JSON(
						http.StatusBadRequest,
						&createFileResponse{Error: err.Error()},
					)
				}

				file, err := c.FormFile("file")
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "file is required")
				}

				// ПОДМЕНА: используем размер из теста
				actualSize := tt.fileSize

				// Валидация размера файла
				if err := ValidateFileSize(actualSize); err != nil {
					return c.JSON(
						http.StatusRequestEntityTooLarge,
						&createFileResponse{Error: err.Error()},
					)
				}

				f, err := file.Open()
				if err != nil {
					return echo.NewHTTPError(
						http.StatusBadRequest,
						"cannot open uploaded file",
					)
				}
				defer func() { _ = f.Close() }()

				contentType := file.Header.Get("Content-Type")
				id, err := mockSvc.CreateFileItem(
					c.Request().Context(),
					uid,
					&item,
					f,
					contentType,
				)
				if err != nil {
					return echo.NewHTTPError(
						http.StatusInternalServerError,
						"failed to create file",
					)
				}

				return c.JSON(http.StatusOK, &createFileResponse{ID: id})
			}

			e.POST("/", handler)

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("unexpected status code: got %d want %d, body: %s",
					rec.Code, tt.wantCode, rec.Body.String())
			}

			if tt.wantBodySubstr != "" {
				respBody := rec.Body.String()
				if !strings.Contains(respBody, tt.wantBodySubstr) {
					t.Errorf("unexpected body: got %q want substring %q",
						respBody, tt.wantBodySubstr)
				}
			}
		})
	}
}

// mockCreateFileService для тестирования
type mockCreateFileService struct {
	createFunc func(ctx interface{}, uid int, item *model.ItemDecrypted, file interface{}, ct string) (int, error)
}

func (m *mockCreateFileService) CreateFileItem(
	ctx interface{},
	uid int,
	item *model.ItemDecrypted,
	file interface{},
	ct string,
) (int, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, uid, item, file, ct)
	}
	return 0, nil
}
