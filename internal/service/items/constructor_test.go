package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewItemsService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		svc := NewItemsService(nil)
		assert.NotNil(t, svc)
	})
}

func TestNewCreateItemService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		svc := NewCreateItemService(nil, nil, nil)
		assert.NotNil(t, svc)
	})
}

func TestNewCreateFileService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		svc := NewCreateFileService(nil, nil, nil, nil)
		assert.NotNil(t, svc)
	})
}

func TestNewGetItemService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		svc := NewGetItemService(nil, nil, nil)
		assert.NotNil(t, svc)
	})
}

func TestNewGetFileService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		svc := NewGetFileService(nil, nil, nil, nil)
		assert.NotNil(t, svc)
	})
}
