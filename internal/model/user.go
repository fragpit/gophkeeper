package model

import (
	"context"
	"errors"
	"unicode/utf8"
)

var (
	ErrUserExists             = errors.New("user already exists")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrPasswordPolicyViolated = errors.New("password policy violated")
)

const (
	minPasswordLength = 12
	maxPasswordLength = 64
)

//go:generate mockgen -destination ../service/auth/mocks/users_repo_gen.go . UsersRepository
type UsersRepository interface {
	CreateWithDEK(ctx context.Context, u *User, dek *EncryptedDEK) (*User, error)
	GetUserDEK(ctx context.Context, userID int) (*EncryptedDEK, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
}

type EncryptedDEK struct {
	EncryptedKey []byte
	Nonce        []byte
}

type User struct {
	ID           int
	Login        string
	PasswordHash string
}

func NewUser(login string) *User {
	return &User{Login: login}
}

func ValidatePassword(password string) error {
	passLength := utf8.RuneCountInString(password)
	if passLength < minPasswordLength || passLength > maxPasswordLength {
		return ErrPasswordPolicyViolated
	}

	return nil
}
