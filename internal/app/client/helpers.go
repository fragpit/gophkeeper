package client

import (
	"fmt"
	"os"
	"path/filepath"

	"resty.dev/v3"
)

func saveTokenToFile(token string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	fileName := filepath.Join(homeDir, ".gophkeeper", "auth.cfg")
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create auth directory: %w", err)
	}

	return os.WriteFile(fileName, []byte(token), 0o600)
}

func readTokenFromFile(file string) ([]byte, error) {
	return os.ReadFile(file)
}

func handleErrorResponse(resp *resty.Response, op string, msg string) error {
	if msg != "" {
		return fmt.Errorf("%s: %s", op, msg)
	}
	if resp == nil {
		return fmt.Errorf("%s: request failed: empty response", op)
	}
	if resp.StatusCode() == 401 {
		return fmt.Errorf("%s: unauthorized: status=%d", op, resp.StatusCode())
	}
	return fmt.Errorf(
		"%s: request failed: status=%d body=%s",
		op,
		resp.StatusCode(),
		resp.String(),
	)
}
