package model

import "errors"

var (
	// ErrTokenNotFound is returned when a refresh token is not found in the repository.
	ErrTokenNotFound = errors.New("token not found")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
