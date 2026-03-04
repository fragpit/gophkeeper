package model

import (
	"context"
	"errors"
	"unicode/utf8"
)

var (
	// ErrUserExists indicates that a user with the same login already exists.
	ErrUserExists = errors.New("user already exists")
	// ErrUserNotFound indicates the user could not be located.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials signals that provided credentials are incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrPasswordPolicyViolated is returned when password requirements are not met.
	ErrPasswordPolicyViolated = errors.New("password policy violated")
)

const (
	minPasswordLength = 12
	maxPasswordLength = 64
)

// UsersRepository defines storage operations for users and their encryption keys.
//
//go:generate mockgen -destination ../service/auth/mocks/users_repo_gen.go . UsersRepository
type UsersRepository interface {
	CreateWithDEK(ctx context.Context, u *User, dek *EncryptedDEK) (*User, error)
	GetUserDEK(ctx context.Context, userID int) (*EncryptedDEK, error)
	GetByID(ctx context.Context, userID int) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
}

// EncryptedDEK stores an encrypted data encryption key and its nonce.
type EncryptedDEK struct {
	EncryptedKey []byte
	Nonce        []byte
}

// User represents an application user entity.
type User struct {
	ID           int
	Login        string
	PasswordHash string
}

// NewUser constructs a new User with the given login.
func NewUser(login string) *User {
	return &User{Login: login}
}

// ValidatePassword checks whether the password meets length requirements.
func ValidatePassword(password string) error {
	passLength := utf8.RuneCountInString(password)
	if passLength < minPasswordLength || passLength > maxPasswordLength {
		return ErrPasswordPolicyViolated
	}

	return nil
}
