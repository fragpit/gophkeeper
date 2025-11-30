package config

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	cfg := &ServerConfig{
		Debug:        false,
		Address:      "https://locahost:8080",
		DatabaseURI:  "postgresql://localhost:5432",
		S3Endpoint:   "localhost:9091",
		S3AccessKey:  "asd",
		S3SecretKey:  "asd",
		S3BucketName: "gophkeeper",
		TLSCertFile:  "/tmp/cert.file",
		TLSKeyFile:   "/tmp/key.file",
		JWTTTL:       5 * time.Second,
		JWTSecret:    "YXNk",
		MasterKey:    "YXNk",
	}

	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "success",
			ttl:     2 * time.Hour,
			wantErr: false,
		},
		{
			name:    "error (less that 1 hour)",
			ttl:     2 * time.Minute,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run("test duration_min", func(t *testing.T) {
			cfg.JWTTTL = tc.ttl
			err := cfg.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
