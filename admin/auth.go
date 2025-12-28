package admin

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "stencil_admin_session"
	SessionDuration   = 24 * time.Hour
)

// hashPassword creates a bcrypt hash of the password
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword compares a password with a bcrypt hash
func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// verifyCredentials checks if the provided username and password are valid
// Returns the username if valid, empty string if invalid
func (s *AdminServer) verifyCredentials(username, password string) string {
	// Check if it's the main admin user
	if username == "admin" {
		if verifyPassword(password, s.EnvConfig.Admin.Password) {
			return "admin"
		}
		return ""
	}

	// Check configured users
	for _, user := range s.EnvConfig.Admin.Users {
		if user.Username == username {
			if verifyPassword(password, user.PasswordHash) {
				return username
			}
			return ""
		}
	}

	return ""
}

// createSession creates a new session for the user
func (s *AdminServer) createSession(w http.ResponseWriter, r *http.Request, username string) error {
	session, err := s.SessionStore.Get(r, SessionCookieName)
	if err != nil {
		session, _ = s.SessionStore.New(r, SessionCookieName)
	}

	session.Values["username"] = username
	return session.Save(r, w)
}

// getSessionUsername returns the username for the session, or empty string if invalid
func (s *AdminServer) getSessionUsername(r *http.Request) string {
	session, err := s.SessionStore.Get(r, SessionCookieName)
	if err != nil {
		return ""
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		return ""
	}

	return username
}

// clearSession removes the session
func (s *AdminServer) clearSession(w http.ResponseWriter, r *http.Request) {
	session, err := s.SessionStore.Get(r, SessionCookieName)
	if err != nil {
		return
	}

	session.Options.MaxAge = -1
	session.Save(r, w)
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
		username := s.getSessionUsername(r)
		if username == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireSiteAccess middleware ensures the user has access to the site in the URL
func (s *AdminServer) requireSiteAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := s.getSessionUsername(r)
		if username == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

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
		username := s.getSessionUsername(r)
		if username == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !isAdmin(username) {
			http.Error(w, "Access denied: Superadmin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
