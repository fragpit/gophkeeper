//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var s3Storage *S3Storage

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start SeaweedFS container
	req := testcontainers.ContainerRequest{
		Image:        "chrislusf/seaweedfs",
		ExposedPorts: []string{"8333/tcp"},
		Cmd:          []string{"server", "-s3"},
		WaitingFor:   wait.ForLog("Start Seaweed S3 API Server"),
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		log.Fatalf("Failed to start SeaweedFS container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}()

	// Get the container endpoint
	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "8333")
	if err != nil {
		log.Fatalf("Failed to get container port: %v", err)
	}

	endpoint := host + ":" + port.Port()

	// Initialize S3 storage with empty credentials (no auth)
	s3Storage, err = NewS3Storage(ctx, endpoint, "", "", "gophkeeper")
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestNewS3Storage(t *testing.T) {
	t.Run("invalid endpoint", func(t *testing.T) {
		ctx := context.Background()

		_, err := NewS3Storage(ctx, "nonexistent-host:99999", "", "", "gophkeeper")

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrS3ServerNotAvailable)
	})
}

func TestPut(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		content := []byte("test file content")

		err := s3Storage.Put(ctx, "testfile.txt", bytes.NewReader(content))

		require.NoError(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		ctx := t.Context()

		err := s3Storage.Put(ctx, "empty.txt", bytes.NewReader([]byte{}))

		require.NoError(t, err)
	})

	t.Run("large file", func(t *testing.T) {
		ctx := t.Context()
		// Create 1MB file
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		err := s3Storage.Put(ctx, "largefile.bin", bytes.NewReader(largeContent))

		require.NoError(t, err)
	})
}

func TestGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		expectedContent := []byte("test content for retrieval")

		// Upload file first
		err := s3Storage.Put(ctx, "gettest.txt", bytes.NewReader(expectedContent))
		require.NoError(t, err)

		// Retrieve the file
		reader, err := s3Storage.Get(ctx, "gophkeeper", "gettest.txt")
		require.NoError(t, err)
		defer reader.Close()

		// Read and verify content
		actualContent, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, actualContent)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := t.Context()

		reader, err := s3Storage.Get(ctx, "gophkeeper", "nonexistent.txt")

		if err == nil {
			defer reader.Close()
			// Try to read - should fail
			_, readErr := io.ReadAll(reader)
			assert.Error(t, readErr)
		} else {
			// Connection might fail before read
			assert.Error(t, err)
		}
	})

	t.Run("overwrite file", func(t *testing.T) {
		ctx := t.Context()
		firstContent := []byte("first version")
		secondContent := []byte("second version updated")

		// Upload first version
		err := s3Storage.Put(ctx, "overwrite.txt", bytes.NewReader(firstContent))
		require.NoError(t, err)

		// Upload second version
		err = s3Storage.Put(ctx, "overwrite.txt", bytes.NewReader(secondContent))
		require.NoError(t, err)

		// Retrieve and verify it's the second version
		reader, err := s3Storage.Get(ctx, "gophkeeper", "overwrite.txt")
		require.NoError(t, err)
		defer reader.Close()

		actualContent, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, secondContent, actualContent)
	})

	t.Run("wrong object name", func(t *testing.T) {
		ctx := t.Context()

		err := s3Storage.Put(ctx, "test.txt", bytes.NewReader([]byte("data")))
		require.NoError(t, err)

		reader, err := s3Storage.Get(ctx, "gophkeeper", "wrong-name.txt")

		if err == nil {
			defer reader.Close()
			_, readErr := io.ReadAll(reader)
			assert.Error(t, readErr)
		}
	})
}
