package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ClientConfig holds client application settings.
type ClientConfig struct {
	ServerURL string `mapstructure:"server_url" validate:"required"`
}

// NewClientConfig loads and validates client configuration.
func NewClientConfig() (*ClientConfig, error) {
	cfg := &ClientConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		var keys []string
		validationErrors := err.(validator.ValidationErrors)
		for _, vErr := range validationErrors {
			slog.Error(
				"validate config",
				"value",
				vErr.Value(),
				"error",
				vErr.Error(),
			)
			keys = append(keys, vErr.Field())
		}

		return nil, fmt.Errorf(
			"validate config for keys: %s",
			strings.Join(keys, ", "),
		)
	}

	return cfg, nil
}
