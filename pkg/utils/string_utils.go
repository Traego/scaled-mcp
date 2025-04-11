package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateSecureID generates a cryptographically secure random string of the specified length
// using characters from a-z, A-Z, and 0-9.
func GenerateSecureID(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLength := big.NewInt(int64(len(charset)))

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// Generate a cryptographically secure random number
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate secure random number: %w", err)
		}

		// Use the random number as an index into the charset
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result), nil
}

// MustGenerateSecureID is like GenerateSecureID but panics on error.
// This is useful when you're confident the ID generation won't fail
// and don't want to handle the error case.
func MustGenerateSecureID(length int) string {
	id, err := GenerateSecureID(length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate secure ID: %v", err))
	}
	return id
}
