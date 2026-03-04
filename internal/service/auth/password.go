package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func comparePasswordHash(password, hash string) bool {
	if err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	); err != nil {
		return false
	}

	return true
}
