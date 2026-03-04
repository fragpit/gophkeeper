package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthRegisterHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewAuthRegisterHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewAuthLoginHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewAuthLoginHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewCreateHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewCreateHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewCreateFileHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewCreateFileHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewGetHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewGetHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewGetFileHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewGetFileHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestNewListHandler_Constructor(t *testing.T) {
	t.Run("creates handler", func(t *testing.T) {
		handler := NewListHandler(nil)
		assert.NotNil(t, handler)
	})
}
