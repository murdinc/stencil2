package utils

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"time"
)

// GenerateSessionID generates a random session ID
func GenerateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// GenerateVerificationCode generates a 6-digit verification code
func GenerateVerificationCode() string {
	// Generate a random number between 100000 and 999999
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to timestamp-based code if random fails
		return "123456"
	}
	code := n.Int64() + 100000
	return string([]byte{
		byte('0' + (code/100000)%10),
		byte('0' + (code/10000)%10),
		byte('0' + (code/1000)%10),
		byte('0' + (code/100)%10),
		byte('0' + (code/10)%10),
		byte('0' + code%10),
	})
}
