package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() (string, error) {
	// 32 bytes = 256 bits
	bytes := make([]byte, 32)

	// Fill slice with secure random bytes
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Convert to hex string
	token := hex.EncodeToString(bytes)

	return token, nil
}
