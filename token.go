package main

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateToken generates a cryptographically strong token that uses only base64 characters
func GenerateToken() (string, error) {
	bytes := make([]byte, 33)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}
