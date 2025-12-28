package admin

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
	"github.com/go-chi/chi/v5"
)

// Template functions - merge Sprig functions with custom ones
var templateFuncs = func() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["formatBytes"] = func(bytes int64) string {
		if bytes == 0 {
			return "—"
		}
		kb := float64(bytes) / 1024.0
		if kb < 1024 {
			return fmt.Sprintf("%.1f KB", kb)
		}
		mb := kb / 1024.0
		return fmt.Sprintf("%.1f MB", mb)
	}
	return funcs
}()

// LayoutData holds common data for all pages
type LayoutData struct {
	Title         string
	AllSites      []Website
	CurrentSite   *Website
	ActiveSection string
	Data          interface{}
}

// renderWithLayout renders a page using the layout template
func (s *AdminServer) renderWithLayout(w http.ResponseWriter, r *http.Request, contentTemplate string, data map[string]interface{}) {
	// Get current user from session
	sessionID := getSession(r)
	username := getSessionUsername(sessionID)
	isAdminUser := isAdmin(username)

	// Get all sites
	allSites, err := s.GetAllWebsites()
	if err != nil {
		log.Printf("Error loading websites: %v", err)
		allSites = []Website{}
	}

	// Filter sites based on user permissions
	var sites []Website
	if isAdminUser {
		sites = allSites
	} else {
		allowedSiteIDs := s.getUserSiteAccess(username)
		if allowedSiteIDs == nil {
			// User has access to all sites
			sites = allSites
		} else {
			// Filter to only allowed sites
			for _, site := range allSites {
				for _, allowedID := range allowedSiteIDs {
					if site.DatabaseName == allowedID || site.ID == allowedID {
						sites = append(sites, site)
						break
					}
				}
			}
		}
	}

	// Get current site if ID is in URL
	var currentSite *Website
	siteID := chi.URLParam(r, "id")
	if siteID != "" {
		site, err := s.GetWebsite(siteID)
		if err == nil {
			currentSite = &site
		}
	}

	// Build layout data
	layoutData := LayoutData{
		Title:         data["Title"].(string),
		AllSites:      sites,
		CurrentSite:   currentSite,
		ActiveSection: data["ActiveSection"].(string),
		Data:          data,
	}

	// Merge data into layout data for template access
	finalData := map[string]interface{}{
		"Title":         layoutData.Title,
		"AllSites":      layoutData.AllSites,
		"CurrentSite":   layoutData.CurrentSite,
		"ActiveSection": layoutData.ActiveSection,
		"IsAdmin":       isAdminUser,
		"Username":      username,
	}
	for k, v := range data {
		finalData[k] = v
	}

	// Parse templates with custom functions
	tmpl, err := template.New("layout.html").Funcs(templateFuncs).ParseFiles(
		filepath.Join("admin", "templates", "layout.html"),
		filepath.Join("admin", "templates", contentTemplate),
	)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute
	if err := tmpl.ExecuteTemplate(w, "layout.html", finalData); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderSimpleTemplate renders without the layout (for login, etc)
func (s *AdminServer) renderSimpleTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	tmplPath := filepath.Join("admin", "templates", tmplName)

	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
