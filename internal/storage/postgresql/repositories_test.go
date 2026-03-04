package postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositories(t *testing.T) {
	t.Run("creates repositories struct", func(t *testing.T) {
		repos := &Repositories{
			Users:       nil,
			Items:       nil,
			Healthcheck: nil,
		}
		assert.NotNil(t, repos)
	})
}
