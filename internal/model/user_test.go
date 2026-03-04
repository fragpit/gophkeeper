package model

import (
	"errors"
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name  string
		login string
	}{
		{"with valid login", "testuser"},
		{"with email", "user@example.com"},
		{"with empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUser(tt.login)
			if got.Login != tt.login {
				t.Errorf("NewUser().Login = %v, want %v", got.Login, tt.login)
			}
			if got.ID != 0 {
				t.Errorf("NewUser().ID = %v, want 0", got.ID)
			}
			if got.PasswordHash != "" {
				t.Errorf("NewUser().PasswordHash = %v, want empty", got.PasswordHash)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"min length", "123456789012", false},
		{"medium length", "this_is_a_valid_password", false},
		{"max length", strings.Repeat("a", 64), false},
		{"too short", "short", true},
		{"one less", "12345678901", true},
		{"too long", strings.Repeat("a", 65), true},
		{"empty", "", true},
		{"unicode valid", "пароль123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrPasswordPolicyViolated) {
				t.Errorf(
					"ValidatePassword() error = %v, want %v",
					err,
					ErrPasswordPolicyViolated,
				)
			}
		})
	}
}
