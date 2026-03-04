package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mock_handlers "github.com/fragpit/gophkeeper/internal/api/handlers/mocks"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewHealthHandler(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mock_handlers.NewMockHealthService(ctrl)

	tests := []struct {
		name        string
		returnError error
		wantCode    int
	}{
		{
			name:        "success",
			returnError: nil,
			wantCode:    http.StatusOK,
		},
		{
			name:        "fail",
			returnError: errors.New("test error"),
			wantCode:    http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			m.EXPECT().Check(gomock.Any()).Return(tc.returnError)

			e.GET("/", NewHealthHandler(m))
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantCode, rec.Code)
		})
	}
}
