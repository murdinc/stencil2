package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/murdinc/stencil2/utils"
)

const (
	SessionCookieName = "stencil_admin_session"
	SessionDuration   = 24 * time.Hour
)

// SessionData stores session information including username
type SessionData struct {
	Username  string
	ExpiresAt time.Time
}

// Session storage (in-memory for now, could be moved to database)
var sessions = make(map[string]SessionData)

// hashPassword creates a SHA256 hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// verifyCredentials checks if the provided username and password are valid
// Returns the username if valid, empty string if invalid
func (s *AdminServer) verifyCredentials(username, password string) string {
	// Check if it's the main admin user
	if username == "admin" {
		expectedHash := hashPassword(s.EnvConfig.Admin.Password)
		providedHash := hashPassword(password)
		if expectedHash == providedHash {
			return "admin"
		}
		return ""
	}

	// Check configured users
	for _, user := range s.EnvConfig.Admin.Users {
		if user.Username == username {
			providedHash := hashPassword(password)
			if user.PasswordHash == providedHash {
				return username
			}
			return ""
		}
	}

	return ""
}

// createSession creates a new session for the user
func (s *AdminServer) createSession(w http.ResponseWriter, username string) string {
	sessionID := utils.GenerateSessionID()
	sessions[sessionID] = SessionData{
		Username:  username,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	utils.SetCookie(w, SessionCookieName, sessionID, "/", int(SessionDuration.Seconds()))

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

	sessionData, exists := sessions[sessionID]
	if !exists {
		return false
	}

	if time.Now().After(sessionData.ExpiresAt) {
		delete(sessions, sessionID)
		return false
	}

	return true
}

// getSessionUsername returns the username for the session, or empty string if invalid
func getSessionUsername(sessionID string) string {
	if sessionID == "" {
		return ""
	}

	sessionData, exists := sessions[sessionID]
	if !exists {
		return ""
	}

	if time.Now().After(sessionData.ExpiresAt) {
		delete(sessions, sessionID)
		return ""
	}

	return sessionData.Username
}

// clearSession removes the session
func clearSession(w http.ResponseWriter, sessionID string) {
	delete(sessions, sessionID)
	utils.ClearCookie(w, SessionCookieName, "/")
}

// isAdmin checks if the username is the main admin user
func isAdmin(username string) bool {
	return username == "admin"
}

// getUserSiteAccess returns the list of site IDs the user can access
// Returns nil if user has access to all sites
func (s *AdminServer) getUserSiteAccess(username string) []string {
	if username == "admin" {
		return nil // Admin has access to all sites
	}

	for _, user := range s.EnvConfig.Admin.Users {
		if user.Username == username {
			if user.AllSites {
				return nil // User has access to all sites
			}
			return user.SiteIDs
		}
	}

	return []string{} // No access
}

// canAccessSite checks if the user has permission to access a specific site
func (s *AdminServer) canAccessSite(username, siteID string) bool {
	// Admin can access everything
	if isAdmin(username) {
		return true
	}

	// Get user's allowed sites
	allowedSiteIDs := s.getUserSiteAccess(username)

	// nil means all sites
	if allowedSiteIDs == nil {
		return true
	}

	// Check if site is in allowed list
	for _, allowedID := range allowedSiteIDs {
		if allowedID == siteID {
			return true
		}
	}

	return false
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
		sessionData := sessions[sessionID]
		sessionData.ExpiresAt = time.Now().Add(SessionDuration)
		sessions[sessionID] = sessionData

		next.ServeHTTP(w, r)
	})
}

// requireSiteAccess middleware ensures the user has access to the site in the URL
func (s *AdminServer) requireSiteAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSession(r)
		username := getSessionUsername(sessionID)

		// Get site ID from URL
		siteID := chi.URLParam(r, "id")

		// Check if user has access
		if !s.canAccessSite(username, siteID) {
			http.Error(w, "Access denied: You don't have permission to access this website", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireSuperadmin middleware ensures only admin user can access
func (s *AdminServer) requireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSession(r)
		username := getSessionUsername(sessionID)

		if !isAdmin(username) {
			http.Error(w, "Access denied: Superadmin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
