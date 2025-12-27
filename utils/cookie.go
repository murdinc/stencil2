package utils

import (
	"net/http"
)

// prodMode tracks whether the application is running in production mode
var prodMode = false

// SetProductionMode sets the production mode flag for cookie security
// Call this at application startup with the --prod-mode flag value
func SetProductionMode(isProduction bool) {
	prodMode = isProduction
}

// CookieOptions represents options for creating an HTTP cookie
type CookieOptions struct {
	Name     string
	Value    string
	Path     string
	MaxAge   int
	HttpOnly bool
	SameSite http.SameSite
	Secure   bool
}

// NewCookie creates an HTTP cookie with the specified options
func NewCookie(opts CookieOptions) *http.Cookie {
	return &http.Cookie{
		Name:     opts.Name,
		Value:    opts.Value,
		Path:     opts.Path,
		MaxAge:   opts.MaxAge,
		HttpOnly: opts.HttpOnly,
		SameSite: opts.SameSite,
		Secure:   opts.Secure,
	}
}

// SetCookie is a convenience function to set a cookie with common defaults
// Automatically sets Secure flag based on production mode
func SetCookie(w http.ResponseWriter, name, value, path string, maxAge int) {
	cookie := NewCookie(CookieOptions{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   prodMode, // Automatically secure in production mode
	})
	http.SetCookie(w, cookie)
}

// ClearCookie removes a cookie by setting its MaxAge to -1
func ClearCookie(w http.ResponseWriter, name, path string) {
	cookie := NewCookie(CookieOptions{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, cookie)
}
