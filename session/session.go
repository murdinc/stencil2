package session

import (
	"net/http"

	"github.com/murdinc/stencil2/utils"
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
	cartID := utils.GenerateSessionID()

	// Set cookie
	utils.SetCookie(w, CartCookieName, cartID, CartCookiePath, CartCookieMaxAge)

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
	utils.ClearCookie(w, CartCookieName, CartCookiePath)
}

// SetEarlyAccessSession sets the early access unlocked cookie
func SetEarlyAccessSession(w http.ResponseWriter, value string) {
	utils.SetCookie(w, EarlyAccessCookieName, value, EarlyAccessCookiePath, EarlyAccessCookieMaxAge)
}

// GetEarlyAccessSession retrieves the early access session value if it exists
func GetEarlyAccessSession(r *http.Request) string {
	cookie, err := r.Cookie(EarlyAccessCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
