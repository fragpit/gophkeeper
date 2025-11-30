//go:build integration

package postgresql

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	repos *Repositories
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Catch panic from testcontainers if Docker is not available
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Docker is not available or not running properly.")
			fmt.Println("Please start Docker daemon to run integration tests.")
			fmt.Printf("Error: %v\n", r)
			os.Exit(0)
		}
	}()

	pgContainter, err := testcontainers.Run(
		ctx,
		"postgres:18.1-alpine3.23",
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "metricstest",
		}),
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create pg container: %s", err)
	}
	defer func() {
		if err := pgContainter.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	pgEndpoint, err := pgContainter.Endpoint(ctx, "tcp")
	if err != nil {
		log.Fatalf("failed to get pg endpoint: %s", err)
	}

	pgEndpoint = strings.TrimPrefix(pgEndpoint, "tcp://")
	pgHost, pgPort, err := net.SplitHostPort(pgEndpoint)
	if err != nil {
		log.Fatalf("failed to get pg port number: %s", err)
	}

	pgDSN := fmt.Sprintf(
		"postgresql://postgres:postgres@%s:%s/metricstest?sslmode=disable",
		pgHost,
		pgPort,
	)

	repos, err = NewStorage(ctx, pgDSN)
	if err != nil {
		log.Fatalf("failed to create pg storage: %s", err)
	}

	os.Exit(m.Run())
}

func TestNewStorage(t *testing.T) {
	t.Run("invalid dsn", func(t *testing.T) {
		ctx := context.Background()

		_, err := NewStorage(ctx, "invalid dsn")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "create pgxpool")
	})

	t.Run("unreachable database", func(t *testing.T) {
		ctx := context.Background()

		_, err := NewStorage(
			ctx,
			"postgresql://user:pass@localhost:9999/db?sslmode=disable",
		)

		require.Error(t, err)
	})
}
