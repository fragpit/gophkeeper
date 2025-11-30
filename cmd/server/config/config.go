package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Debug       bool          `mapstructure:"debug" validate:"boolean"`
	Address     string        `mapstructure:"address" validate:"required"`
	DatabaseURI string        `mapstructure:"database_uri" validate:"required,uri"`
	S3Endpoint  string        `mapstructure:"s3_endpoint" validate:"required,hostname_port"`
	S3AccessKey string        `mapstructure:"s3_access_key" validate:"required"`
	S3SecretKey string        `mapstructure:"s3_secret_key" validate:"required"`
	TLSCertFile string        `mapstructure:"tls_cert_file" validate:"filepath"`
	TLSKeyFile  string        `mapstructure:"tls_key_file" validate:"filepath"`
	JWTTTL      time.Duration `mapstructure:"jwt_ttl" validate:"duration_min=1h"`
	JWTSecret   string        `mapstructure:"jwt_secret" validate:"required,base64"`
	MasterKey   string        `mapstructure:"master_key" validate:"required,base64"`
}

func NewServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	v := validator.New(validator.WithRequiredStructEnabled())

	err := v.RegisterValidation(
		"duration_min",
		func(fl validator.FieldLevel) bool {
			param := fl.Param()
			min := time.Duration(0)
			if param != "" {
				d, err := time.ParseDuration(param)
				if err != nil {
					return false
				}
				min = d
			}
			d, ok := fl.Field().Interface().(time.Duration)
			if !ok {
				return false
			}
			return d >= min
		},
	)
	if err != nil {
		return nil, fmt.Errorf("register custom validator: %w", err)
	}

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
