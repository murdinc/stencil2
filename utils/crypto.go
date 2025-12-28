package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
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
func GenerateVerificationCode() (string, error) {
	// Generate a random number between 100000 and 999999
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	code := n.Int64() + 100000
	return string([]byte{
		byte('0' + (code/100000)%10),
		byte('0' + (code/10000)%10),
		byte('0' + (code/1000)%10),
		byte('0' + (code/100)%10),
		byte('0' + (code/10)%10),
		byte('0' + code%10),
	}), nil
}

// GenerateRandomKey generates a random key of the specified length
func GenerateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return string(bytes), nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ReadPassword reads a password from stdin without echoing
func ReadPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}
