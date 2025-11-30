//go:build integration

package postgresql

import (
	"testing"

	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckRepo_Ping(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := repos.Healthcheck.Ping(t.Context())
		assert.NoError(t, err)
	})

	t.Run("nil connection", func(t *testing.T) {
		isRetriable := func(err error) bool {
			return false
		}
		retrier := retry.New(isRetriable)
		hc := NewHealthCheckRepo(nil, retrier)

		err := hc.Ping(t.Context())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConnectionNotInitialized)
	})
}
