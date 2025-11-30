package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"normal password", "password123", false},
		{"empty password", "", false},
		{"long password", strings.Repeat("a", 70), false},
		{"special chars", "p@$$w0rd!#%", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				if hash == tt.password {
					t.Error("HashPassword() returned unhashed password")
				}
				if !ComparePasswordHash(tt.password, hash) {
					t.Error("HashPassword() hash doesn't match original password")
				}
			}
		})
	}
}

func TestComparePasswordHash(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "wrongpassword", hash, false},
		{"empty password", "", hash, false},
		{"invalid hash", password, "invalid_hash", false},
		{"empty hash", password, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComparePasswordHash(tt.password, tt.hash); got != tt.want {
				t.Errorf("ComparePasswordHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
