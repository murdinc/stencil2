package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	CartCookieName = "stencil_cart_id"
	CartCookiePath = "/"
	CartCookieMaxAge = 60 * 60 * 24 * 7 // 7 days in seconds

	EarlyAccessCookieName = "stencil_early_access"
	EarlyAccessCookiePath = "/"
	EarlyAccessCookieMaxAge = 60 * 60 * 24 * 30 // 30 days in seconds
)

// GetOrCreateCartSession retrieves or creates a cart session ID
func GetOrCreateCartSession(r *http.Request, w http.ResponseWriter) string {
	// Try to get existing cart ID from cookie
	cookie, err := r.Cookie(CartCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Generate new cart ID
	cartID := generateSessionID()

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     CartCookieName,
		Value:    cartID,
		Path:     CartCookiePath,
		MaxAge:   CartCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true in production with HTTPS
	})

	return cartID
}

// GetCartSession retrieves the cart session ID if it exists
func GetCartSession(r *http.Request) string {
	cookie, err := r.Cookie(CartCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// ClearCartSession removes the cart session cookie
func ClearCartSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CartCookieName,
		Value:    "",
		Path:     CartCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// SetEarlyAccessSession sets the early access unlocked cookie
func SetEarlyAccessSession(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     EarlyAccessCookieName,
		Value:    value,
		Path:     EarlyAccessCookiePath,
		MaxAge:   EarlyAccessCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true in production with HTTPS
	})
}

// GetEarlyAccessSession retrieves the early access session value if it exists
func GetEarlyAccessSession(r *http.Request) string {
	cookie, err := r.Cookie(EarlyAccessCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
