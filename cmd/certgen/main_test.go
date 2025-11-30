package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKeys(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name: "success",
			path: "/tmp",
		},
		{
			name: "success relative path",
			path: "../keygen/",
		},
		{
			name:    "fail non existent path",
			path:    "/tmp/asd123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generateCert(tt.path)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			cert := filepath.Join(tt.path, certName)
			key := filepath.Join(tt.path, keyName)

			_, err = os.Stat(cert)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			_, err = os.Stat(key)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			err = os.Remove(cert)
			if !tt.wantErr {
				assert.NoError(t, err)
			}

			err = os.Remove(key)
			if !tt.wantErr {
				assert.NoError(t, err)
			}
		})
	}
}
