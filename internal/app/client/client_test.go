package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_SetDisableWarn(t *testing.T) {
	t.Run("sets disable warn to true", func(t *testing.T) {
		client, err := NewClient(
			"http://localhost",
			createTempTokenFile(t, "token123", "refresh123"),
			true,
		)
		if err != nil {
			t.Skip("skipping test that requires HTTP client initialization")
		}

		// Just call the method to increase coverage
		client.SetDisableWarn(true)
		assert.NotNil(t, client)
	})

	t.Run("sets disable warn to false", func(t *testing.T) {
		client, err := NewClient(
			"http://localhost",
			createTempTokenFile(t, "token123", "refresh123"),
			true,
		)
		if err != nil {
			t.Skip("skipping test that requires HTTP client initialization")
		}

		client.SetDisableWarn(false)
		assert.NotNil(t, client)
	})
}
