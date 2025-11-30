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
	ServerURL          string `mapstructure:"server_url" validate:"required"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

type Option func(*ClientConfig) error

func WithGlobalViper() Option {
	return func(cfg *ClientConfig) error {
		if err := viper.Unmarshal(cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		return nil
	}
}

// NewClientConfig loads and validates client configuration.
func NewClientConfig(opts ...Option) (*ClientConfig, error) {
	cfg := &ClientConfig{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Validate checks if the config struct fields are valid according to validation rules.
func (c *ClientConfig) Validate() error {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(c); err != nil {
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

		return fmt.Errorf("validate config for keys: %s", strings.Join(keys, ", "))
	}

	return nil
}
