package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// Generate Random String (for password reset token, email verification)
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Generate Numeric code (for email verification 6 digit code)
func GenerateNumericCode(length int) (string, error) {
	max := new(big.Int)
	max.Exp(big.NewInt(10), big.NewInt(int64(length)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", length, n), nil
}

func GenerateUniqueID() (string, error) {
	timestamp := time.Now().Unix()
	randomPart, err := GenerateRandomToken(8)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d_%s", timestamp, randomPart), nil
}

func GenerateSessionID() (string, error) {
	return GenerateRandomToken(32)
}
