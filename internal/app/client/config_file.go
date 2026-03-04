package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fragpit/gophkeeper/internal/model"
)

type clientConfig struct {
	Auth authConfig `json:"auth"`
}

type authConfig struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func saveTokensToFile(fileName string, tokens model.TokenPair) error {
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create auth directory: %w", err)
	}

	cfg := &clientConfig{
		Auth: authConfig{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
	}

	cfgJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal client config: %w", err)
	}

	return os.WriteFile(fileName, cfgJSON, 0o600)
}

func readTokensFromFile(file string) (*model.TokenPair, error) {
	cfgJSON, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read client config: %w", err)
	}

	cfg := &clientConfig{}
	if err := json.Unmarshal(cfgJSON, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal client config: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  cfg.Auth.AccessToken,
		RefreshToken: cfg.Auth.RefreshToken,
	}, nil
}
