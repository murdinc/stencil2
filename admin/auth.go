package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	SessionCookieName = "stencil_admin_session"
	SessionDuration   = 24 * time.Hour
)

// Session storage (in-memory for now, could be moved to database)
var sessions = make(map[string]time.Time)

// generateSessionKey generates a random session key
func generateSessionKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSessionID generates a new session ID
func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// hashPassword creates a SHA256 hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// verifyPassword checks if the provided password matches the configured password
func (s *AdminServer) verifyPassword(password string) bool {
	expectedHash := hashPassword(s.EnvConfig.Admin.Password)
	providedHash := hashPassword(password)
	return expectedHash == providedHash
}

// createSession creates a new session for the user
func (s *AdminServer) createSession(w http.ResponseWriter) string {
	sessionID := generateSessionID()
	sessions[sessionID] = time.Now().Add(SessionDuration)

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true in production with HTTPS
	})

	return sessionID
}

// getSession retrieves the session ID from the cookie
func getSession(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// isSessionValid checks if the session is valid and not expired
func isSessionValid(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	expiry, exists := sessions[sessionID]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		delete(sessions, sessionID)
		return false
	}

	return true
}

// clearSession removes the session
func clearSession(w http.ResponseWriter, sessionID string) {
	delete(sessions, sessionID)
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// requireAuth middleware ensures the user is authenticated
func (s *AdminServer) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSession(r)
		if !isSessionValid(sessionID) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extend session expiry
		sessions[sessionID] = time.Now().Add(SessionDuration)

		next.ServeHTTP(w, r)
	})
}
