package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObjectKey(t *testing.T) {
	t.Run("generates object key", func(t *testing.T) {
		key, err := newObjectKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 32) // 16 bytes hex encoded = 32 chars
	})

	t.Run("generates different keys", func(t *testing.T) {
		key1, err := newObjectKey()
		require.NoError(t, err)

		key2, err := newObjectKey()
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})
}
