package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/email"
	"github.com/murdinc/stencil2/frontend"
	"github.com/murdinc/stencil2/twilio"
)

// Helper function to get website from URL parameter (db name)
func (s *AdminServer) getWebsiteFromURL(r *http.Request) (Website, error) {
	websiteID := chi.URLParam(r, "id")
	if websiteID == "" {
		return Website{}, fmt.Errorf("website ID not in URL")
	}
	return s.GetWebsite(websiteID)
}

// validateSlug ensures slug is valid: no leading/trailing slashes, only lowercase alphanumeric and hyphens
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if strings.HasPrefix(slug, "/") || strings.HasSuffix(slug, "/") {
		return fmt.Errorf("slug cannot start or end with a slash")
	}
	// Only allow lowercase letters, numbers, hyphens, and forward slashes (for nested paths)
	validSlug := regexp.MustCompile(`^[a-z0-9\-/]+$`)
	if !validSlug.MatchString(slug) {
		return fmt.Errorf("slug can only contain lowercase letters, numbers, hyphens, and forward slashes")
	}
	// Don't allow double slashes
	if strings.Contains(slug, "//") {
		return fmt.Errorf("slug cannot contain consecutive slashes")
	}
	return nil
}

// handleLoginPage renders the login page
func (s *AdminServer) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	sessionID := getSession(r)
	if isSessionValid(sessionID) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.renderTemplate(w, "login", nil)
}

// handleLogin processes the login form
func (s *AdminServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	if s.verifyPassword(password) {
		s.createSession(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.renderTemplate(w, "login", map[string]interface{}{
		"Error": "Invalid password",
	})
}

// handleLogout logs out the user
func (s *AdminServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := getSession(r)
	clearSession(w, sessionID)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleDashboard renders the main dashboard (no site selected)
func (s *AdminServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if user has a last selected site in cookie
	if cookie, err := r.Cookie("last_site"); err == nil && cookie.Value != "" {
		// Verify the site still exists
		if _, err := s.GetWebsite(cookie.Value); err == nil {
			// Redirect to the last selected site
			http.Redirect(w, r, "/site/"+cookie.Value, http.StatusFound)
			return
		}
	}

	s.renderWithLayout(w, r, "dashboard_content.html", map[string]interface{}{
		"Title":         "Dashboard",
		"ActiveSection": "",
	})
}

// handleSiteDashboard renders the dashboard for a specific site
func (s *AdminServer) handleSiteDashboard(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	if websiteID == "" {
		http.Error(w, "Invalid website ID", http.StatusBadRequest)
		return
	}

	site, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// Store this site as the last selected site in a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "last_site",
		Value:    websiteID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Get overview stats
	stats, err := s.GetOverviewStats(websiteID)
	if err != nil {
		log.Printf("Error fetching overview stats: %v", err)
		stats = &OverviewStats{} // Use empty stats on error
	}

	// Get active users count
	activeUsers, err := s.GetActiveUsers(websiteID, 5)
	if err != nil {
		log.Printf("Error fetching active users: %v", err)
		activeUsers = 0
	}

	// Get recent orders
	recentOrders, err := s.GetRecentOrders(websiteID, 5)
	if err != nil {
		log.Printf("Error fetching recent orders: %v", err)
		recentOrders = []RecentOrder{}
	}

	s.renderWithLayout(w, r, "overview_content.html", map[string]interface{}{
		"Title":         site.SiteName + " - Overview",
		"ActiveSection": "overview",
		"Website":       site,
		"Stats":         stats,
		"ActiveUsers":   activeUsers,
		"RecentOrders":  recentOrders,
	})
}

// handleSiteSettings renders the site settings page
func (s *AdminServer) handleSiteSettings(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")

	site, err := s.GetWebsite(siteID)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	s.renderWithLayout(w, r, "site_settings_content.html", map[string]interface{}{
		"Title":         site.SiteName + " - Settings",
		"ActiveSection": "settings",
		"Website":       site,
		"ProdMode":      s.EnvConfig.ProdMode,
	})
}

// handleSiteSettingsUpdate updates site settings
func (s *AdminServer) handleSiteSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get existing website to preserve directory
	existingWebsite, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Parse float values
	taxRate := 0.0
	if r.FormValue("taxRate") != "" {
		fmt.Sscanf(r.FormValue("taxRate"), "%f", &taxRate)
	}

	shippingCost := 0.0
	if r.FormValue("shippingCost") != "" {
		fmt.Sscanf(r.FormValue("shippingCost"), "%f", &shippingCost)
	}

	// Parse IMAP port
	imapPort := 0
	if r.FormValue("imapPort") != "" {
		fmt.Sscanf(r.FormValue("imapPort"), "%d", &imapPort)
	}

	// Parse SMTP port
	smtpPort := 0
	if r.FormValue("smtpPort") != "" {
		fmt.Sscanf(r.FormValue("smtpPort"), "%d", &smtpPort)
	}

	website := Website{
		ID:            websiteID,
		SiteName:      r.FormValue("siteName"),
		Directory:     existingWebsite.Directory,
		DatabaseName:  r.FormValue("databaseName"),
		HTTPAddress:   r.FormValue("httpAddress"),
		MediaProxyURL: r.FormValue("mediaProxyUrl"),
		APIVersion:    1,

		StripePublishableKey: r.FormValue("stripePublishableKey"),
		StripeSecretKey:      r.FormValue("stripeSecretKey"),

		ShippoAPIKey: r.FormValue("shippoApiKey"),
		LabelFormat:  r.FormValue("labelFormat"),

		TwilioAccountSID: r.FormValue("twilioAccountSid"),
		TwilioAuthToken:  r.FormValue("twilioAuthToken"),
		TwilioFromPhone:  r.FormValue("twilioFromPhone"),

		// Simplified email fields - use single email address for all
		EmailFromAddress: r.FormValue("emailAddress"),
		EmailFromName:    r.FormValue("emailFromName"),
		EmailReplyTo:     r.FormValue("emailAddress"), // Same as from address

		IMAPServer:   r.FormValue("imapServer"),
		IMAPPort:     imapPort,
		IMAPUsername: r.FormValue("emailAddress"), // Same as email address
		IMAPPassword: r.FormValue("emailPassword"),
		IMAPUseTLS:   r.FormValue("emailUseTLS") == "true",

		SMTPServer:   r.FormValue("smtpServer"),
		SMTPPort:     smtpPort,
		SMTPUsername: r.FormValue("emailAddress"), // Same as email address
		SMTPPassword: r.FormValue("emailPassword"), // Same as IMAP password
		SMTPUseTLS:   r.FormValue("emailUseTLS") == "true",

		TaxRate:      taxRate,
		ShippingCost: shippingCost,

		EarlyAccessEnabled:  r.FormValue("earlyAccessEnabled") == "on",
		EarlyAccessPassword: r.FormValue("earlyAccessPassword"),

		ShipFromName:    r.FormValue("shipFromName"),
		ShipFromStreet1: r.FormValue("shipFromStreet1"),
		ShipFromStreet2: r.FormValue("shipFromStreet2"),
		ShipFromCity:    r.FormValue("shipFromCity"),
		ShipFromState:   r.FormValue("shipFromState"),
		ShipFromZip:     r.FormValue("shipFromZip"),
		ShipFromCountry: r.FormValue("shipFromCountry"),

		RobotsTxt: r.FormValue("robots_txt"),
		Logo:      r.FormValue("logo"),
	}

	if err := s.UpdateWebsite(website); err != nil {
		http.Error(w, fmt.Sprintf("Error updating website: %v", err), http.StatusInternalServerError)
		return
	}

	// Reload the website configuration in the running frontend
	if frontendWebsite, exists := frontend.GetWebsite(websiteID); exists {
		if err := frontendWebsite.ReloadConfig(s.EnvConfig.ProdMode); err != nil {
			log.Printf("Warning: Failed to reload website config: %v", err)
		}
	}

	s.LogActivity("update", "website", 0, websiteID, website)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/settings", websiteID), http.StatusSeeOther)
}

// handleWebhooks renders the webhooks configuration page
func (s *AdminServer) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "id")

	site, err := s.GetWebsite(siteID)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// Construct base URL for webhooks
	baseURL := s.EnvConfig.BaseURL
	if baseURL == "" {
		// Fallback to request host if base URL not configured
		scheme := "https"
		if !s.EnvConfig.ProdMode {
			scheme = "http"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	s.renderWithLayout(w, r, "webhooks_content.html", map[string]interface{}{
		"Title":         site.SiteName + " - Webhooks",
		"ActiveSection": "webhooks",
		"Website":       site,
		"BaseURL":       baseURL,
	})
}

// handleWebsitesList renders the websites list
func (s *AdminServer) handleWebsitesList(w http.ResponseWriter, r *http.Request) {
	websites, err := s.GetAllWebsites()
	if err != nil {
		http.Error(w, "Error loading websites", http.StatusInternalServerError)
		return
	}

	s.renderTemplate(w, "websites_list", map[string]interface{}{
		"Websites": websites,
	})
}

// handleWebsiteNew renders the new website form
func (s *AdminServer) handleWebsiteNew(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "website_form", map[string]interface{}{
		"Title":  "Create New Website",
		"Action": "/websites/new",
	})
}

// handleWebsiteCreate creates a new website
func (s *AdminServer) handleWebsiteCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	website := Website{
		SiteName:      r.FormValue("siteName"),
		Directory:     r.FormValue("directory"),
		DatabaseName:  r.FormValue("databaseName"),
		HTTPAddress:   r.FormValue("httpAddress"),
		MediaProxyURL: r.FormValue("mediaProxyUrl"),
		APIVersion:    1,
	}

	// Create website directory structure
	websiteDir := filepath.Join("websites", website.Directory)
	if err := os.MkdirAll(websiteDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Error creating directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Create subdirectories
	for _, dir := range []string{"templates", "public", "sitemaps"} {
		if err := os.MkdirAll(filepath.Join(websiteDir, dir), 0755); err != nil {
			http.Error(w, fmt.Sprintf("Error creating directory: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Create config files
	devConfig := map[string]interface{}{
		"siteName":   website.SiteName,
		"apiVersion": website.APIVersion,
		"database": map[string]string{
			"name": website.DatabaseName,
		},
		"http": map[string]string{
			"address": website.HTTPAddress,
		},
	}

	if website.MediaProxyURL != "" {
		devConfig["mediaProxyUrl"] = website.MediaProxyURL
	}

	configData, _ := json.MarshalIndent(devConfig, "", "  ")
	if err := os.WriteFile(filepath.Join(websiteDir, "config-dev.json"), configData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Error creating config file: %v", err), http.StatusInternalServerError)
		return
	}

	// Save to admin database
	_, err := s.CreateWebsite(website)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving website: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("create", "website", 0, website.DatabaseName, website)

	http.Redirect(w, r, fmt.Sprintf("/site/%s", website.DatabaseName), http.StatusSeeOther)
}

// handleWebsiteDelete deletes a website
func (s *AdminServer) handleWebsiteDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := s.DeleteWebsite(websiteID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting website: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "website", 0, websiteID, nil)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleArticlesList renders the articles list for a website
func (s *AdminServer) handleArticlesList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	articles, err := s.GetArticles(websiteID, 100, 0)
	if err != nil {
		log.Printf("Error loading articles: %v", err)
		articles = []Article{}
	}

	s.renderWithLayout(w, r, "articles_list_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - Articles",
		"ActiveSection": "articles",
		"Website":       website,
		"Articles":      articles,
	})
}

// handleArticleNew renders the new article form
func (s *AdminServer) handleArticleNew(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	categories, err := s.GetCategories(websiteID)
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		categories = []Category{}
	}

	images, err := s.GetImages(websiteID, 100, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	s.renderWithLayout(w, r, "article_form_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - New Article",
		"ActiveSection": "articles",
		"FormTitle":     "Create New Article",
		"Website":       website,
		"Categories":    categories,
		"Images":        images,
		"Action":        fmt.Sprintf("/site/%s/articles/new", websiteID),
	})
}

// handleArticleCreate creates a new article
func (s *AdminServer) handleArticleCreate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Parse multipart form for file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	slug := r.FormValue("slug")

	// Validate slug
	if err := validateSlug(slug); err != nil {
		http.Error(w, fmt.Sprintf("Invalid slug: %v", err), http.StatusBadRequest)
		return
	}

	article := Article{
		Slug:        slug,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Content:     r.FormValue("content"),
		Excerpt:     r.FormValue("excerpt"),
		Type:        r.FormValue("type"),
		Status:      r.FormValue("status"),
	}

	// Parse published_date from form if provided
	publishedDateStr := r.FormValue("published_date")
	if publishedDateStr != "" {
		if parsedDate, err := time.Parse("2006-01-02T15:04", publishedDateStr); err == nil {
			article.PublishedDate = parsedDate
		}
	}

	// Set published date to now if status is published and no date was provided
	if article.Status == "published" && article.PublishedDate.IsZero() {
		article.PublishedDate = time.Now()
	}

	// Handle image upload or selection
	// Check if user uploaded a new image
	if file, header, err := r.FormFile("new_image"); err == nil {
		defer file.Close()

		// Get website info
		website, err := s.GetWebsite(websiteID)
		if err == nil {
			// Create uploads directory if it doesn't exist
			uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
			os.MkdirAll(uploadsDir, 0755)

			// Generate unique filename
			ext := filepath.Ext(header.Filename)
			filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(header.Filename[:len(header.Filename)-len(ext)], " ", "_"), ext)
			filePath := filepath.Join(uploadsDir, filename)

			// Save file
			if dst, err := os.Create(filePath); err == nil {
				defer dst.Close()
				dst.ReadFrom(file)
				fileInfo, _ := dst.Stat()

				// Create image record with protocol-relative URL
				imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
				image := Image{
					URL:      imageURL,
					AltText:  r.FormValue("image_alt"),
					Credit:   r.FormValue("image_credit"),
					Filename: header.Filename,
					Size:     fileInfo.Size(),
				}

				if imageID, err := s.CreateImage(websiteID, image); err == nil {
					article.ThumbnailID = int(imageID)
				}
			}
		}
	} else {
		// User selected an existing image
		imageIDStr := r.FormValue("image_id")
		if imageIDStr != "" {
			if imageID, err := strconv.Atoi(imageIDStr); err == nil {
				article.ThumbnailID = imageID
			}
		}
	}

	id, err := s.CreateArticle(websiteID, article)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating article: %v", err), http.StatusInternalServerError)
		return
	}

	// Save category relationships
	categoryIDStrs := r.Form["categories[]"]
	var categoryIDs []int
	for _, idStr := range categoryIDStrs {
		if catID, err := strconv.Atoi(idStr); err == nil {
			categoryIDs = append(categoryIDs, catID)
		}
	}
	if len(categoryIDs) > 0 {
		if err := s.SetArticleCategories(websiteID, int(id), categoryIDs); err != nil {
			log.Printf("Error setting article categories: %v", err)
		}
	}

	s.LogActivity("create", "article", int(id), websiteID, article)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/articles", websiteID), http.StatusSeeOther)
}

// handleArticleEdit renders the edit article form
func (s *AdminServer) handleArticleEdit(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	articleID, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	article, err := s.GetArticle(websiteID, articleID)
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	categories, err := s.GetCategories(websiteID)
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		categories = []Category{}
	}

	images, err := s.GetImages(websiteID, 100, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	// Load article's current categories
	articleCategories, err := s.GetArticleCategories(websiteID, articleID)
	if err != nil {
		log.Printf("Error loading article categories: %v", err)
		articleCategories = []Category{}
	}

	s.renderWithLayout(w, r, "article_form_content.html", map[string]interface{}{
		"Title":             website.SiteName + " - Edit Article",
		"ActiveSection":     "articles",
		"FormTitle":         "Edit Article",
		"Website":           website,
		"Article":           article,
		"Categories":        categories,
		"Images":            images,
		"ArticleCategories": articleCategories,
		"Action":            fmt.Sprintf("/site/%s/articles/%d/edit", websiteID, articleID),
	})
}

// handleArticleUpdate updates an existing article
func (s *AdminServer) handleArticleUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	articleID, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Parse multipart form for file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the existing article to check if we need to set published date
	existingArticle, err := s.GetArticle(websiteID, articleID)
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	slug := r.FormValue("slug")

	// Validate slug
	if err := validateSlug(slug); err != nil {
		http.Error(w, fmt.Sprintf("Invalid slug: %v", err), http.StatusBadRequest)
		return
	}

	article := Article{
		ID:            articleID,
		Slug:          slug,
		Title:         r.FormValue("title"),
		Description:   r.FormValue("description"),
		Content:       r.FormValue("content"),
		Excerpt:       r.FormValue("excerpt"),
		Type:          r.FormValue("type"),
		Status:        r.FormValue("status"),
		PublishedDate: existingArticle.PublishedDate,
		ThumbnailID:   existingArticle.ThumbnailID,
	}

	// Parse published_date from form if provided
	publishedDateStr := r.FormValue("published_date")
	if publishedDateStr != "" {
		if parsedDate, err := time.Parse("2006-01-02T15:04", publishedDateStr); err == nil {
			article.PublishedDate = parsedDate
		}
	}

	// Set published date to now if status is published and it wasn't published before
	if article.Status == "published" && article.PublishedDate.IsZero() {
		article.PublishedDate = time.Now()
	}

	// Handle image upload or selection
	// Check if user uploaded a new image
	if file, header, err := r.FormFile("new_image"); err == nil {
		defer file.Close()

		// Get website info
		website, err := s.GetWebsite(websiteID)
		if err == nil {
			// Create uploads directory if it doesn't exist
			uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
			os.MkdirAll(uploadsDir, 0755)

			// Generate unique filename
			ext := filepath.Ext(header.Filename)
			filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(header.Filename[:len(header.Filename)-len(ext)], " ", "_"), ext)
			filePath := filepath.Join(uploadsDir, filename)

			// Save file
			if dst, err := os.Create(filePath); err == nil {
				defer dst.Close()
				dst.ReadFrom(file)
				fileInfo, _ := dst.Stat()

				// Create image record with protocol-relative URL
				imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
				image := Image{
					URL:      imageURL,
					AltText:  r.FormValue("image_alt"),
					Credit:   r.FormValue("image_credit"),
					Filename: header.Filename,
					Size:     fileInfo.Size(),
				}

				if imageID, err := s.CreateImage(websiteID, image); err == nil {
					article.ThumbnailID = int(imageID)
				}
			}
		}
	} else {
		// User selected an existing image
		imageIDStr := r.FormValue("image_id")
		if imageIDStr != "" {
			if imageID, err := strconv.Atoi(imageIDStr); err == nil {
				article.ThumbnailID = imageID
			}
		} else {
			// No image selected, clear the thumbnail
			article.ThumbnailID = 0
		}
	}

	if err := s.UpdateArticle(websiteID, article); err != nil {
		http.Error(w, fmt.Sprintf("Error updating article: %v", err), http.StatusInternalServerError)
		return
	}

	// Save category relationships
	categoryIDStrs := r.Form["categories[]"]
	var categoryIDs []int
	for _, idStr := range categoryIDStrs {
		if catID, err := strconv.Atoi(idStr); err == nil {
			categoryIDs = append(categoryIDs, catID)
		}
	}
	// Always update categories (empty array will clear all associations)
	if err := s.SetArticleCategories(websiteID, articleID, categoryIDs); err != nil {
		log.Printf("Error setting article categories: %v", err)
	}

	s.LogActivity("update", "article", articleID, websiteID, article)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/articles", websiteID), http.StatusSeeOther)
}

// handleArticleDelete deletes an article
func (s *AdminServer) handleArticleDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	articleID, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	if err := s.DeleteArticle(websiteID, articleID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting article: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "article", articleID, websiteID, nil)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/articles", websiteID), http.StatusSeeOther)
}

// Product handlers (similar pattern)
func (s *AdminServer) handleProductsList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	products, err := s.GetProducts(websiteID, 100, 0)
	if err != nil {
		log.Printf("Error loading products: %v", err)
		products = []Product{}
	}

	s.renderWithLayout(w, r, "products_list_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - Products",
		"ActiveSection": "products",
		"Website":       website,
		"Products":      products,
	})
}

func (s *AdminServer) handleProductNew(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	collections, err := s.GetCollections(websiteID)
	if err != nil {
		log.Printf("Error loading collections: %v", err)
		collections = []Collection{}
	}

	s.renderWithLayout(w, r, "product_form_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - New Product",
		"ActiveSection": "products",
		"FormTitle":     "Create New Product",
		"Website":       website,
		"Collections":   collections,
		"Action":        fmt.Sprintf("/site/%s/products/new", websiteID),
	})
}

func (s *AdminServer) handleProductCreate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	compareAtPrice, _ := strconv.ParseFloat(r.FormValue("compareAtPrice"), 64)
	inventoryQuantity, _ := strconv.Atoi(r.FormValue("inventoryQuantity"))
	featured := r.FormValue("featured") == "on"

	product := Product{
		Name:              r.FormValue("name"),
		Slug:              r.FormValue("slug"),
		Description:       r.FormValue("description"),
		Price:             price,
		CompareAtPrice:    compareAtPrice,
		SKU:               r.FormValue("sku"),
		InventoryQuantity: inventoryQuantity,
		InventoryPolicy:   r.FormValue("inventoryPolicy"),
		Status:            r.FormValue("status"),
		Featured:          featured,
	}

	// Set released date to now if status is published
	if product.Status == "published" {
		product.ReleasedDate = time.Now()
	}

	id, err := s.CreateProduct(websiteID, product)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating product: %v", err), http.StatusInternalServerError)
		return
	}

	productID := int(id)

	// Save collection relationships
	collectionIDStrs := r.Form["collections[]"]
	var collectionIDs []int
	for _, idStr := range collectionIDStrs {
		if collID, err := strconv.Atoi(idStr); err == nil {
			collectionIDs = append(collectionIDs, collID)
		}
	}
	if len(collectionIDs) > 0 {
		if err := s.SetProductCollections(websiteID, productID, collectionIDs); err != nil {
			log.Printf("Error setting product collections: %v", err)
		}
	}

	// Handle product images - direct upload to product_images_data
	website, _ := s.GetWebsite(websiteID)
	if website.ID != "" {
		files := r.MultipartForm.File["product_images"]
		altText := r.FormValue("new_images_alt")
		credit := r.FormValue("new_images_credit")

		for position, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			defer file.Close()

			// Create uploads directory
			uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
			os.MkdirAll(uploadsDir, 0755)

			// Generate unique filename
			ext := filepath.Ext(fileHeader.Filename)
			filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), strings.ReplaceAll(fileHeader.Filename[:len(fileHeader.Filename)-len(ext)], " ", "_"), ext)
			filePath := filepath.Join(uploadsDir, filename)

			// Save file
			dst, err := os.Create(filePath)
			if err != nil {
				continue
			}
			dst.ReadFrom(file)
			fileInfo, _ := dst.Stat()
			dst.Close()

			// Create product image record directly (no shared image library)
			imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
			productImage := ProductImageData{
				ProductID: productID,
				URL:       imageURL,
				Filename:  fileHeader.Filename,
				Filepath:  filePath,
				AltText:   altText,
				Credit:    credit,
				Size:      fileInfo.Size(),
				Position:  position,
			}

			s.AddProductImageData(websiteID, productImage)
		}
	}

	s.LogActivity("create", "product", productID, websiteID, product)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleProductEdit(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	product, err := s.GetProduct(websiteID, productID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	collections, err := s.GetCollections(websiteID)
	if err != nil {
		log.Printf("Error loading collections: %v", err)
		collections = []Collection{}
	}

	// Load product's current collections
	productCollections, err := s.GetProductCollections(websiteID, productID)
	if err != nil {
		log.Printf("Error loading product collections: %v", err)
		productCollections = []Collection{}
	}

	// Load product images
	productImages, err := s.GetProductImagesData(websiteID, productID)
	if err != nil {
		log.Printf("Error loading product images: %v", err)
		productImages = []ProductImageData{}
	}

	s.renderWithLayout(w, r, "product_form_content.html", map[string]interface{}{
		"Title":              website.SiteName + " - Edit Product",
		"ActiveSection":      "products",
		"FormTitle":          "Edit Product",
		"Website":            website,
		"Product":            product,
		"Collections":        collections,
		"ProductCollections": productCollections,
		"ProductImages":      productImages,
		"Action":             fmt.Sprintf("/site/%s/products/%d/edit", websiteID, productID),
	})
}

func (s *AdminServer) handleProductUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the existing product to check if we need to set published date
	existingProduct, err := s.GetProduct(websiteID, productID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	compareAtPrice, _ := strconv.ParseFloat(r.FormValue("compareAtPrice"), 64)
	inventoryQuantity, _ := strconv.Atoi(r.FormValue("inventoryQuantity"))
	featured := r.FormValue("featured") == "on"

	product := Product{
		ID:                productID,
		Name:              r.FormValue("name"),
		Slug:              r.FormValue("slug"),
		Description:       r.FormValue("description"),
		Price:             price,
		CompareAtPrice:    compareAtPrice,
		SKU:               r.FormValue("sku"),
		InventoryQuantity: inventoryQuantity,
		InventoryPolicy:   r.FormValue("inventoryPolicy"),
		Status:            r.FormValue("status"),
		Featured:          featured,
		ReleasedDate:      existingProduct.ReleasedDate,
	}

	// Set released date to now if status is published and it wasn't published before
	if product.Status == "published" && existingProduct.ReleasedDate.IsZero() {
		product.ReleasedDate = time.Now()
	}

	if err := s.UpdateProduct(websiteID, product); err != nil {
		http.Error(w, fmt.Sprintf("Error updating product: %v", err), http.StatusInternalServerError)
		return
	}

	// Save collection relationships
	collectionIDStrs := r.Form["collections[]"]
	var collectionIDs []int
	for _, idStr := range collectionIDStrs {
		if collID, err := strconv.Atoi(idStr); err == nil {
			collectionIDs = append(collectionIDs, collID)
		}
	}
	// Always update collections (empty array will clear all associations)
	if err := s.SetProductCollections(websiteID, productID, collectionIDs); err != nil {
		log.Printf("Error setting product collections: %v", err)
	}

	// Handle image removals - deletes from DB and disk
	removeImageIDStrs := r.Form["remove_images[]"]
	for _, idStr := range removeImageIDStrs {
		if imageID, err := strconv.Atoi(idStr); err == nil {
			s.DeleteProductImageData(websiteID, imageID)
		}
	}

	// Get current max position for new uploads
	currentImages, _ := s.GetProductImagesData(websiteID, productID)
	position := len(currentImages)

	// Upload and add new images directly to product_images_data
	website, _ := s.GetWebsite(websiteID)
	if website.ID != "" {
		files := r.MultipartForm.File["product_images"]
		altText := r.FormValue("new_images_alt")
		credit := r.FormValue("new_images_credit")

		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			defer file.Close()

			// Create uploads directory
			uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
			os.MkdirAll(uploadsDir, 0755)

			// Generate unique filename
			ext := filepath.Ext(fileHeader.Filename)
			filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), strings.ReplaceAll(fileHeader.Filename[:len(fileHeader.Filename)-len(ext)], " ", "_"), ext)
			filePath := filepath.Join(uploadsDir, filename)

			// Save file
			dst, err := os.Create(filePath)
			if err != nil {
				continue
			}
			dst.ReadFrom(file)
			fileInfo, _ := dst.Stat()
			dst.Close()

			// Create product image record directly (no shared image library)
			imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
			productImage := ProductImageData{
				ProductID: productID,
				URL:       imageURL,
				Filename:  fileHeader.Filename,
				Filepath:  filePath,
				AltText:   altText,
				Credit:    credit,
				Size:      fileInfo.Size(),
				Position:  position,
			}

			s.AddProductImageData(websiteID, productImage)
			position++
		}
	}

	s.LogActivity("update", "product", productID, websiteID, product)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleProductDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := s.DeleteProduct(websiteID, productID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting product: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "product", productID, websiteID, nil)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleProductImageReorder(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid product ID",
		})
		return
	}

	// Parse JSON body with array of image IDs in new order
	var requestData struct {
		ImageIDs []int `json:"imageIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	// Update positions
	if err := s.UpdateProductImagePositions(websiteID, requestData.ImageIDs); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error updating image positions: %v", err),
		})
		return
	}

	s.LogActivity("reorder", "product_images", productID, websiteID, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *AdminServer) handleProductReorder(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	direction := chi.URLParam(r, "direction")
	if direction != "up" && direction != "down" {
		http.Error(w, "Invalid direction", http.StatusBadRequest)
		return
	}

	if err := s.ReorderProduct(websiteID, productID, direction); err != nil {
		http.Error(w, fmt.Sprintf("Error reordering product: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("reorder", "product", productID, websiteID, map[string]string{"direction": direction})

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products", websiteID), http.StatusSeeOther)
}

// Variant handlers
func (s *AdminServer) handleVariantNew(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	product, err := s.GetProduct(websiteID, productID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	s.renderWithLayout(w, r, "variant_form_content.html", map[string]interface{}{
		"Title":         "New Variant",
		"Website":       website,
		"Product":       product,
		"Action":        fmt.Sprintf("/site/%s/products/%d/variants/create", websiteID, productID),
		"ActiveSection": "products",
	})
}

func (s *AdminServer) handleVariantCreate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	priceModifier, _ := strconv.ParseFloat(r.FormValue("priceModifier"), 64)
	inventoryQuantity, _ := strconv.Atoi(r.FormValue("inventoryQuantity"))

	err = s.CreateVariant(websiteID, productID, map[string]interface{}{
		"title":             r.FormValue("title"),
		"priceModifier":     priceModifier,
		"sku":               r.FormValue("sku"),
		"inventoryQuantity": inventoryQuantity,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating variant: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("create", "variant", 0, websiteID, map[string]interface{}{
		"product_id": productID,
		"title":      r.FormValue("title"),
	})

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products/%d/edit", websiteID, productID), http.StatusSeeOther)
}

func (s *AdminServer) handleVariantEdit(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	variantID, err := strconv.Atoi(chi.URLParam(r, "variantId"))
	if err != nil {
		http.Error(w, "Invalid variant ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	product, err := s.GetProduct(websiteID, productID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	variant, err := s.GetVariant(websiteID, variantID)
	if err != nil {
		http.Error(w, "Variant not found", http.StatusNotFound)
		return
	}

	s.renderWithLayout(w, r, "variant_form_content.html", map[string]interface{}{
		"Title":         "Edit Variant",
		"Website":       website,
		"Product":       product,
		"Variant":       variant,
		"Action":        fmt.Sprintf("/site/%s/products/%d/variants/%d/update", websiteID, productID, variantID),
		"ActiveSection": "products",
	})
}

func (s *AdminServer) handleVariantUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	variantID, err := strconv.Atoi(chi.URLParam(r, "variantId"))
	if err != nil {
		http.Error(w, "Invalid variant ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	priceModifier, _ := strconv.ParseFloat(r.FormValue("priceModifier"), 64)
	inventoryQuantity, _ := strconv.Atoi(r.FormValue("inventoryQuantity"))

	err = s.UpdateVariant(websiteID, variantID, map[string]interface{}{
		"title":             r.FormValue("title"),
		"priceModifier":     priceModifier,
		"sku":               r.FormValue("sku"),
		"inventoryQuantity": inventoryQuantity,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating variant: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("update", "variant", variantID, websiteID, map[string]interface{}{
		"product_id": productID,
		"title":      r.FormValue("title"),
	})

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products/%d/edit", websiteID, productID), http.StatusSeeOther)
}

func (s *AdminServer) handleVariantDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	variantID, err := strconv.Atoi(chi.URLParam(r, "variantId"))
	if err != nil {
		http.Error(w, "Invalid variant ID", http.StatusBadRequest)
		return
	}

	err = s.DeleteVariant(websiteID, variantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting variant: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "variant", variantID, websiteID, map[string]interface{}{
		"product_id": productID,
	})

	w.WriteHeader(http.StatusOK)
}

func (s *AdminServer) handleVariantReorder(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(chi.URLParam(r, "productId"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	variantID, err := strconv.Atoi(chi.URLParam(r, "variantId"))
	if err != nil {
		http.Error(w, "Invalid variant ID", http.StatusBadRequest)
		return
	}

	direction := chi.URLParam(r, "direction")
	if direction != "up" && direction != "down" {
		http.Error(w, "Invalid direction", http.StatusBadRequest)
		return
	}

	if err := s.ReorderVariant(websiteID, variantID, direction); err != nil {
		http.Error(w, fmt.Sprintf("Error reordering variant: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("reorder", "variant", variantID, websiteID, map[string]string{"direction": direction, "product_id": fmt.Sprintf("%d", productID)})

	http.Redirect(w, r, fmt.Sprintf("/site/%s/products/%d/edit", websiteID, productID), http.StatusSeeOther)
}

// Category/Collection handlers (simplified - just list and create/delete)
func (s *AdminServer) handleCategoriesList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	categories, err := s.GetCategories(websiteID)
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		categories = []Category{}
	}

	s.renderWithLayout(w, r, "categories_list_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - Categories",
		"ActiveSection": "categories",
		"Website":       website,
		"Categories":    categories,
	})
}

func (s *AdminServer) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	category := Category{
		Name: r.FormValue("name"),
		Slug: strings.ToLower(strings.ReplaceAll(r.FormValue("name"), " ", "-")),
	}

	id, err := s.CreateCategory(websiteID, category)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating category: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("create", "category", int(id), websiteID, category)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/categories", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	categoryID, err := strconv.Atoi(chi.URLParam(r, "categoryId"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	if err := s.DeleteCategory(websiteID, categoryID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting category: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "category", categoryID, websiteID, nil)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/categories", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleCollectionsList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	collections, err := s.GetCollections(websiteID)
	if err != nil {
		log.Printf("Error loading collections: %v", err)
		collections = []Collection{}
	}

	s.renderWithLayout(w, r, "collections_list_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - Collections",
		"ActiveSection": "collections",
		"Website":       website,
		"Collections":   collections,
	})
}

func (s *AdminServer) handleCollectionCreate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	collection := Collection{
		Name:   r.FormValue("name"),
		Slug:   strings.ToLower(strings.ReplaceAll(r.FormValue("name"), " ", "-")),
		Status: "published",
	}

	id, err := s.CreateCollection(websiteID, collection)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating collection: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("create", "collection", int(id), websiteID, collection)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/collections", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleCollectionDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	collectionID, err := strconv.Atoi(chi.URLParam(r, "collectionId"))
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := s.DeleteCollection(websiteID, collectionID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting collection: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "collection", collectionID, websiteID, nil)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/collections", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleCollectionEditForm(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	collectionID, err := strconv.Atoi(chi.URLParam(r, "collectionId"))
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	collection, err := s.GetCollection(websiteID, collectionID)
	if err != nil {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	// Load all images
	images, err := s.GetImages(websiteID, 1000, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	// Load collection's current image if it has one
	var collectionImage *Image
	if collection.ImageID > 0 {
		for _, img := range images {
			if img.ID == collection.ImageID {
				collectionImage = &img
				break
			}
		}
	}

	s.renderWithLayout(w, r, "collection_form_content.html", map[string]interface{}{
		"Title":           "Edit Collection",
		"ActiveSection":   "collections",
		"Website":         website,
		"Collection":      collection,
		"Images":          images,
		"CollectionImage": collectionImage,
		"Action":          fmt.Sprintf("/site/%s/collections/%d/edit", websiteID, collectionID),
	})
}

func (s *AdminServer) handleCollectionUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	collectionID, err := strconv.Atoi(chi.URLParam(r, "collectionId"))
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get existing collection
	existingCollection, _ := s.GetCollection(websiteID, collectionID)

	sortOrder, _ := strconv.Atoi(r.FormValue("sortOrder"))

	collection := Collection{
		ID:          collectionID,
		Name:        r.FormValue("name"),
		Slug:        r.FormValue("slug"),
		Description: r.FormValue("description"),
		SortOrder:   sortOrder,
		Status:      r.FormValue("status"),
		ImageID:     existingCollection.ImageID,
	}

	// Handle image upload or selection
	if file, header, err := r.FormFile("new_image"); err == nil {
		defer file.Close()

		// Get website info
		website, _ := s.GetWebsite(websiteID)
		if website.ID != "" {
			// Create uploads directory
			uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
			os.MkdirAll(uploadsDir, 0755)

			// Generate unique filename
			ext := filepath.Ext(header.Filename)
			filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(header.Filename[:len(header.Filename)-len(ext)], " ", "_"), ext)
			filePath := filepath.Join(uploadsDir, filename)

			// Save file
			dst, err := os.Create(filePath)
			if err == nil {
				defer dst.Close()
				dst.ReadFrom(file)
				fileInfo, _ := dst.Stat()

				// Create image record
				imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
				image := Image{
					URL:      imageURL,
					AltText:  r.FormValue("image_alt"),
					Credit:   r.FormValue("image_credit"),
					Filename: header.Filename,
					Size:     fileInfo.Size(),
				}

				if imageID, err := s.CreateImage(websiteID, image); err == nil {
					collection.ImageID = int(imageID)
				}
			}
		}
	} else {
		// User selected an existing image or cleared it
		imageIDStr := r.FormValue("image_id")
		if imageIDStr != "" {
			if imageID, err := strconv.Atoi(imageIDStr); err == nil {
				collection.ImageID = imageID
			}
		} else {
			// No image selected, clear it
			collection.ImageID = 0
		}
	}

	if err := s.UpdateCollection(websiteID, collection); err != nil {
		http.Error(w, fmt.Sprintf("Error updating collection: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("update", "collection", collectionID, websiteID, collection)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/collections", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleCollectionReorder(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	collectionID, err := strconv.Atoi(chi.URLParam(r, "collectionId"))
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	direction := chi.URLParam(r, "direction")
	if direction != "up" && direction != "down" {
		http.Error(w, "Invalid direction", http.StatusBadRequest)
		return
	}

	if err := s.ReorderCollection(websiteID, collectionID, direction); err != nil {
		http.Error(w, fmt.Sprintf("Error reordering collection: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("reorder", "collection", collectionID, websiteID, map[string]string{"direction": direction})

	http.Redirect(w, r, fmt.Sprintf("/site/%s/collections", websiteID), http.StatusSeeOther)
}

// Image handlers (basic list and upload/delete)
func (s *AdminServer) handleImagesList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	images, err := s.GetImages(websiteID, 100, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	s.renderWithLayout(w, r, "images_list_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - Images",
		"ActiveSection": "images",
		"Website":       website,
		"Images":        images,
	})
}

func (s *AdminServer) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Parse multipart form for file uploads (32 MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get website info to find the directory
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join("websites", website.Directory, "public", "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Error creating uploads directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate unique filename with timestamp
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(header.Filename[:len(header.Filename)-len(ext)], " ", "_"), ext)
	filePath := filepath.Join(uploadsDir, filename)

	// Create the file on disk
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating file: %v", err), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	if _, err := dst.ReadFrom(file); err != nil {
		http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusInternalServerError)
		return
	}

	// Get file size
	fileInfo, _ := dst.Stat()

	// Create image record in database with protocol-relative URL
	imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
	image := Image{
		URL:      imageURL,
		AltText:  r.FormValue("alt"),
		Credit:   r.FormValue("credit"),
		Filename: header.Filename,
		Size:     fileInfo.Size(),
	}

	id, err := s.CreateImage(websiteID, image)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating image: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("create", "image", int(id), websiteID, image)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/images", websiteID), http.StatusSeeOther)
}

func (s *AdminServer) handleImageDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	imageID, err := strconv.Atoi(chi.URLParam(r, "imageId"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	if err := s.DeleteImage(websiteID, imageID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting image: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("delete", "image", imageID, websiteID, nil)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/images", websiteID), http.StatusSeeOther)
}

// handleOrdersList displays list of orders for a website
func (s *AdminServer) handleOrdersList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Parse filter parameters from query string
	filters := OrderFilters{
		PaymentStatus:     r.URL.Query().Get("payment_status"),
		FulfillmentStatus: r.URL.Query().Get("fulfillment_status"),
		Sort:              r.URL.Query().Get("sort"),
	}

	// Default sort
	if filters.Sort == "" {
		filters.Sort = "date_desc"
	}

	orders, err := s.GetOrdersFiltered(websiteID, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching orders: %v", err), http.StatusInternalServerError)
		return
	}

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":         "Orders",
		"Website":       website,
		"Orders":        orders,
		"AllSites":      allSites,
		"CurrentSite":   website,
		"ActiveSection": "orders",
		"Filters":       filters,
	}

	s.renderWithLayout(w, r, "orders_list_content.html", data)
}

// handleOrderDetail displays order detail
func (s *AdminServer) handleOrderDetail(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	order, err := s.GetOrder(websiteID, orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching order: %v", err), http.StatusInternalServerError)
		return
	}

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":       "Order Detail",
		"Website":     website,
		"Order":       order,
		"AllSites":    allSites,
		"CurrentSite": website,
		"ActiveSection": "orders",
	}

	s.renderWithLayout(w, r, "order_detail_content.html", data)
}

// handlePackingSlip renders the packing slip for an order
func (s *AdminServer) handlePackingSlip(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	order, err := s.GetOrder(websiteID, orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching order: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Website": website,
		"Order":   order,
	}

	// Render packing slip template without layout (for printing)
	tmpl, err := template.ParseFiles("admin/templates/packing_slip.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading template: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// handleOrderFulfillmentUpdate updates the fulfillment status of an order
func (s *AdminServer) handleOrderFulfillmentUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the order to check payment status
	order, err := s.GetOrder(websiteID, orderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching order: %v", err), http.StatusInternalServerError)
		return
	}

	// Only allow fulfillment updates if payment is completed
	if order.PaymentStatus != "paid" {
		http.Error(w, "Cannot update fulfillment status: payment has not been completed", http.StatusBadRequest)
		return
	}

	fulfillmentStatus := r.FormValue("fulfillment_status")
	if fulfillmentStatus == "" {
		http.Error(w, "Fulfillment status is required", http.StatusBadRequest)
		return
	}

	err = s.UpdateOrderFulfillmentStatus(websiteID, orderID, fulfillmentStatus)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating fulfillment status: %v", err), http.StatusInternalServerError)
		return
	}

	// Send shipping confirmation email if status changed to "shipped" and we have tracking info
	if fulfillmentStatus == "shipped" && order.TrackingNumber != "" && order.ShippingCarrier != "" {
		emailService, err := email.NewEmailService(s.EnvConfig)
		if err == nil {
			// Get website to build config
			website, err := s.GetWebsite(websiteID)
			if err == nil {
				websiteConfig := &configs.WebsiteConfig{
					SiteName: website.SiteName,
				}
				websiteConfig.Email.FromAddress = website.EmailFromAddress
				websiteConfig.Email.FromName = website.EmailFromName
				websiteConfig.Email.ReplyTo = website.EmailReplyTo

				err = emailService.SendShippingConfirmation(
					websiteConfig,
					order.OrderNumber,
					order.CustomerEmail,
					order.CustomerName,
					order.TrackingNumber,
					order.ShippingCarrier,
				)
				if err != nil {
					log.Printf("Failed to send shipping confirmation email: %v", err)
					// Continue even if email fails
				}
			} else {
				log.Printf("Failed to get website config: %v", err)
			}
		} else {
			log.Printf("Failed to create email service: %v", err)
		}
	}

	// Redirect back to order detail page
	http.Redirect(w, r, fmt.Sprintf("/site/%s/orders/%d", websiteID, orderID), http.StatusSeeOther)
}

// handleShippingRates gets shipping rates for an order
func (s *AdminServer) handleShippingRates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	websiteID := chi.URLParam(r, "id")
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid order ID",
		})
		return
	}

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	// Parse form data for package dimensions (multipart or regular form)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		// Try regular form parsing if multipart fails
		if err := r.ParseForm(); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid form data",
			})
			return
		}
	}

	// Get order
	order, err := s.GetOrder(websiteID, orderID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error fetching order: %v", err),
		})
		return
	}

	// Parse package dimensions from form
	length, _ := strconv.ParseFloat(r.FormValue("length"), 64)
	width, _ := strconv.ParseFloat(r.FormValue("width"), 64)
	height, _ := strconv.ParseFloat(r.FormValue("height"), 64)
	weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)

	if length <= 0 || width <= 0 || height <= 0 || weight <= 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid package dimensions",
		})
		return
	}

	// Get rates from Shippo
	rates, err := s.GetShippingRates(websiteID, order, length, width, height, weight)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error getting shipping rates: %v", err),
		})
		return
	}

	// Return rates as JSON
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"rates":   rates,
	})
}

// handleShippingLabelPurchase purchases a shipping label for an order
func (s *AdminServer) handleShippingLabelPurchase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	websiteID := chi.URLParam(r, "id")
	orderIDStr := chi.URLParam(r, "orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid order ID",
		})
		return
	}

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	// Parse form data (multipart or regular form)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		// Try regular form parsing if multipart fails
		if err := r.ParseForm(); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid form data",
			})
			return
		}
	}

	// Get the rate ID from form
	rateID := r.FormValue("rate_id")
	if rateID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Rate ID is required",
		})
		return
	}

	// Get order to verify payment status
	order, err := s.GetOrder(websiteID, orderID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error fetching order: %v", err),
		})
		return
	}

	// Only allow label purchase if payment is completed
	if order.PaymentStatus != "paid" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Cannot purchase label: payment has not been completed",
		})
		return
	}

	// Purchase label
	labelInfo, err := s.PurchaseShippingLabel(websiteID, orderID, rateID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error purchasing label: %v", err),
		})
		return
	}

	// Automatically update order status to "shipped" since we just bought a label
	err = s.UpdateOrderFulfillmentStatus(websiteID, orderID, "shipped")
	if err != nil {
		log.Printf("Warning: Failed to update order status to shipped: %v", err)
		// Continue anyway - label was purchased successfully
	}

	// Send shipping confirmation email to customer
	emailService, err := email.NewEmailService(s.EnvConfig)
	if err == nil {
		website, err := s.GetWebsite(websiteID)
		if err == nil {
			websiteConfig := &configs.WebsiteConfig{
				SiteName: website.SiteName,
			}
			websiteConfig.Email.FromAddress = website.EmailFromAddress
			websiteConfig.Email.FromName = website.EmailFromName
			websiteConfig.Email.ReplyTo = website.EmailReplyTo

			err = emailService.SendShippingConfirmation(
				websiteConfig,
				order.OrderNumber,
				order.CustomerEmail,
				order.CustomerName,
				labelInfo.TrackingNumber,
				labelInfo.Carrier,
			)
			if err != nil {
				log.Printf("Warning: Failed to send shipping confirmation email: %v", err)
				// Continue anyway - label was purchased successfully
			} else {
				log.Printf("Sent shipping confirmation email for order %s", order.OrderNumber)
			}
		} else {
			log.Printf("Warning: Failed to get website config: %v", err)
		}
	} else {
		log.Printf("Warning: Failed to create email service: %v", err)
	}

	// Return success with label info
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"trackingNumber": labelInfo.TrackingNumber,
		"labelUrl":       labelInfo.LabelURL,
		"carrier":        labelInfo.Carrier,
		"cost":           labelInfo.Cost,
	})
}

// handleCustomersList displays the list of customers with statistics
func (s *AdminServer) handleCustomersList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Get website
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Parse filters from query params
	filters := CustomerFilters{
		Sort: r.URL.Query().Get("sort"),
	}
	if filters.Sort == "" {
		filters.Sort = "total_desc" // Default sort
	}

	// Get customers
	customers, err := s.GetCustomers(websiteID, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching customers: %v", err), http.StatusInternalServerError)
		return
	}

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":         "Customers",
		"Website":       website,
		"Customers":     customers,
		"AllSites":      allSites,
		"CurrentSite":   website,
		"ActiveSection": "customers",
		"Filters":       filters,
	}

	s.renderWithLayout(w, r, "customers_list_content.html", data)
}

// handleCustomerDetail displays a single customer with order history
func (s *AdminServer) handleCustomerDetail(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	customerIDStr := chi.URLParam(r, "customerId")

	customerID, err := strconv.Atoi(customerIDStr)
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	// Get website
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Get customer
	customer, err := s.GetCustomer(websiteID, customerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching customer: %v", err), http.StatusInternalServerError)
		return
	}

	// Get customer orders
	orders, err := s.GetCustomerOrders(websiteID, customerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching customer orders: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate average order value
	avgOrderValue := 0.0
	if customer.OrderCount > 0 {
		avgOrderValue = customer.TotalSpent / float64(customer.OrderCount)
	}

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":          "Customer Details",
		"Website":        website,
		"Customer":       customer,
		"Orders":         orders,
		"AvgOrderValue":  avgOrderValue,
		"AllSites":       allSites,
		"CurrentSite":    website,
		"ActiveSection":  "customers",
	}

	s.renderWithLayout(w, r, "customer_detail_content.html", data)
}

// handleSMSSignupsList displays the list of SMS signups
func (s *AdminServer) handleSMSSignupsList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Get website
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Parse filter parameters from query string
	filters := SMSSignupFilters{
		CountryCode: r.URL.Query().Get("country_code"),
		Source:      r.URL.Query().Get("source"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
		Sort:        r.URL.Query().Get("sort"),
	}

	// Default sort
	if filters.Sort == "" {
		filters.Sort = "date_desc"
	}

	// Get all SMS signups with filters
	signups, err := s.GetSMSSignups(websiteID, filters)
	if err != nil {
		http.Error(w, "Failed to load SMS signups", http.StatusInternalServerError)
		return
	}

	// Get unique country codes and sources for filter dropdowns
	countryCodes, _ := s.GetUniqueCountryCodes(websiteID)
	sources, _ := s.GetUniqueSources(websiteID)

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":         "SMS Signups",
		"Website":       website,
		"Signups":       signups,
		"ActiveSection": "sms-signups",
		"AllSites":      allSites,
		"CurrentSite":   website,
		"Filters":       filters,
		"CountryCodes":  countryCodes,
		"Sources":       sources,
	}

	s.renderWithLayout(w, r, "sms_signups_list_content.html", data)
}

// handleDeleteSMSSignup deletes an SMS signup
func (s *AdminServer) handleDeleteSMSSignup(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	signupIDStr := chi.URLParam(r, "signupId")
	signupID, err := strconv.Atoi(signupIDStr)
	if err != nil {
		http.Error(w, "Invalid signup ID", http.StatusBadRequest)
		return
	}

	err = s.DeleteSMSSignup(websiteID, signupID)
	if err != nil {
		http.Error(w, "Failed to delete signup", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/site/%s/sms-signups", websiteID), http.StatusSeeOther)
}

// handleExportSMSSignups exports SMS signups to CSV
func (s *AdminServer) handleExportSMSSignups(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Parse filter parameters from query string
	filters := SMSSignupFilters{
		CountryCode: r.URL.Query().Get("country_code"),
		Source:      r.URL.Query().Get("source"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
		Sort:        r.URL.Query().Get("sort"),
	}

	// Default sort
	if filters.Sort == "" {
		filters.Sort = "date_desc"
	}

	// Get filtered SMS signups
	signups, err := s.GetSMSSignups(websiteID, filters)
	if err != nil {
		http.Error(w, "Failed to load SMS signups", http.StatusInternalServerError)
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=sms-signups-%s.csv", time.Now().Format("2006-01-02")))

	// Write CSV header
	fmt.Fprintf(w, "ID,Country Code,Phone,Email,Source,Created At\n")

	// Write data rows
	for _, signup := range signups {
		fmt.Fprintf(w, "%d,%s,%s,%s,%s,%s\n",
			signup.ID,
			signup.CountryCode,
			signup.Phone,
			signup.Email,
			signup.Source,
			signup.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}

// handleSMSCampaignForm displays the bulk SMS campaign form
func (s *AdminServer) handleSMSCampaignForm(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Get website
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	// Parse filter parameters from query string
	filters := SMSSignupFilters{
		CountryCode: r.URL.Query().Get("country_code"),
		Source:      r.URL.Query().Get("source"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
		Sort:        r.URL.Query().Get("sort"),
	}

	// Default sort
	if filters.Sort == "" {
		filters.Sort = "date_desc"
	}

	// Get verified SMS signups with filters to show count
	signups, err := s.GetVerifiedSMSSignups(websiteID, filters)
	if err != nil {
		http.Error(w, "Failed to load verified SMS signups", http.StatusInternalServerError)
		return
	}

	// Get unique country codes and sources for filter dropdowns
	countryCodes, _ := s.GetUniqueCountryCodes(websiteID)
	sources, _ := s.GetUniqueSources(websiteID)

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":          "SMS Campaign",
		"Website":        website,
		"RecipientCount": len(signups),
		"ActiveSection":  "sms-campaigns",
		"AllSites":       allSites,
		"CurrentSite":    website,
		"Filters":        filters,
		"CountryCodes":   countryCodes,
		"Sources":        sources,
	}

	s.renderWithLayout(w, r, "sms_campaign_content.html", data)
}

// handleSendBulkSMS processes the bulk SMS send request
func (s *AdminServer) handleSendBulkSMS(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	// Get website
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get message from form
	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Parse filter parameters
	filters := SMSSignupFilters{
		CountryCode: r.FormValue("country_code"),
		Source:      r.FormValue("source"),
		DateFrom:    r.FormValue("date_from"),
		DateTo:      r.FormValue("date_to"),
	}

	// Get verified SMS signups with filters
	signups, err := s.GetVerifiedSMSSignups(websiteID, filters)
	if err != nil {
		http.Error(w, "Failed to load verified SMS signups", http.StatusInternalServerError)
		return
	}

	if len(signups) == 0 {
		http.Error(w, "No verified recipients found with the selected filters", http.StatusBadRequest)
		return
	}

	// Load website config to get Twilio credentials
	configPath := filepath.Join("websites", website.Directory, "config-dev.json")
	if s.EnvConfig.ProdMode {
		configPath = filepath.Join("websites", website.Directory, "config-prod.json")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		http.Error(w, "Failed to read website config", http.StatusInternalServerError)
		return
	}

	var siteConfig struct {
		Twilio struct {
			AccountSID string `json:"accountSid"`
			AuthToken  string `json:"authToken"`
			FromPhone  string `json:"fromPhone"`
		} `json:"twilio"`
	}

	if err := json.Unmarshal(configData, &siteConfig); err != nil {
		http.Error(w, "Failed to parse website config", http.StatusInternalServerError)
		return
	}

	// Initialize Twilio client
	twilioClient := twilio.NewClient(
		siteConfig.Twilio.AccountSID,
		siteConfig.Twilio.AuthToken,
		siteConfig.Twilio.FromPhone,
	)

	// Build phone numbers list with E.164 formatting
	var phoneNumbers []string
	for _, signup := range signups {
		// Format phone: country code + phone number
		formattedPhone := twilio.FormatPhoneNumber(signup.CountryCode, signup.Phone)
		phoneNumbers = append(phoneNumbers, formattedPhone)
	}

	// Add opt-out message for compliance
	messageWithOptOut := message + "\n\nReply STOP to unsubscribe"

	// Send bulk SMS
	results, err := twilioClient.SendBulkSMS(phoneNumbers, messageWithOptOut)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send bulk SMS: %v", err), http.StatusInternalServerError)
		return
	}

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for _, status := range results {
		if status == "success" {
			successCount++
		} else {
			failureCount++
		}
	}

	// Get unique country codes and sources for filter dropdowns
	countryCodes, _ := s.GetUniqueCountryCodes(websiteID)
	sources, _ := s.GetUniqueSources(websiteID)

	allSites, _ := s.GetAllWebsites()

	// Render results page
	data := map[string]interface{}{
		"Title":          "SMS Campaign Results",
		"Website":        website,
		"Message":        message,
		"RecipientCount": len(signups),
		"SuccessCount":   successCount,
		"FailureCount":   failureCount,
		"Results":        results,
		"ActiveSection":  "sms-campaigns",
		"AllSites":       allSites,
		"CurrentSite":    website,
		"Filters":        filters,
		"CountryCodes":   countryCodes,
		"Sources":        sources,
	}

	s.renderWithLayout(w, r, "sms_campaign_content.html", data)
}

// ===============================
// Analytics
// ===============================

func (s *AdminServer) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Parse date range from query params (default to last 30 days)
	daysParam := r.URL.Query().Get("days")
	days := 30
	if daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
			days = d
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get analytics data
	stats, err := s.GetPageViewStats(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching analytics stats: %v", err)
		stats = make(map[string]interface{})
	}

	topPages, err := s.GetTopPages(websiteID, startDate, endDate, 20)
	if err != nil {
		log.Printf("Error fetching top pages: %v", err)
		topPages = []map[string]interface{}{}
	}

	topReferrers, err := s.GetTopReferrers(websiteID, startDate, endDate, 20)
	if err != nil {
		log.Printf("Error fetching top referrers: %v", err)
		topReferrers = []map[string]interface{}{}
	}

	eventStats, err := s.GetEventStats(websiteID, startDate, endDate, 20)
	if err != nil {
		log.Printf("Error fetching event stats: %v", err)
		eventStats = []map[string]interface{}{}
	}

	// Calculate average pages per visitor
	avgPages := 0.0
	if totalViews, ok := stats["total_views"].(int64); ok {
		if uniqueSessions, ok := stats["unique_sessions"].(int64); ok && uniqueSessions > 0 {
			avgPages = float64(totalViews) / float64(uniqueSessions)
		}
	}

	// Get real-time metrics (active in last 5 minutes)
	activeUsers, err := s.GetActiveUsers(websiteID, 5)
	if err != nil {
		log.Printf("Error fetching active users: %v", err)
		activeUsers = 0
	}

	currentPages, err := s.GetCurrentPages(websiteID, 5)
	if err != nil {
		log.Printf("Error fetching current pages: %v", err)
		currentPages = []map[string]interface{}{}
	}

	// Get engagement metrics
	bounceRate, err := s.GetBounceRate(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching bounce rate: %v", err)
		bounceRate = 0
	}

	avgSessionDuration, err := s.GetAverageSessionDuration(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching avg session duration: %v", err)
		avgSessionDuration = 0
	}

	// Format session duration for display
	var sessionDurationDisplay string
	if avgSessionDuration >= 60 {
		sessionDurationDisplay = fmt.Sprintf("%.0fm", avgSessionDuration/60)
	} else {
		sessionDurationDisplay = fmt.Sprintf("%.0fs", avgSessionDuration)
	}

	deviceBreakdown, err := s.GetDeviceBreakdown(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching device breakdown: %v", err)
		deviceBreakdown = make(map[string]int)
	}

	entryPages, err := s.GetEntryPages(websiteID, startDate, endDate, 10)
	if err != nil {
		log.Printf("Error fetching entry pages: %v", err)
		entryPages = []map[string]interface{}{}
	}

	exitPages, err := s.GetExitPages(websiteID, startDate, endDate, 10)
	if err != nil {
		log.Printf("Error fetching exit pages: %v", err)
		exitPages = []map[string]interface{}{}
	}

	// Get e-commerce metrics
	conversionRate, convertedSessions, totalSessions, err := s.GetConversionRate(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching conversion rate: %v", err)
		conversionRate, convertedSessions, totalSessions = 0, 0, 0
	}

	abandonmentRate, abandonedCarts, totalCarts, err := s.GetCartAbandonmentRate(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching cart abandonment: %v", err)
		abandonmentRate, abandonedCarts, totalCarts = 0, 0, 0
	}

	revenueMetrics, err := s.GetRevenueMetrics(websiteID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching revenue metrics: %v", err)
		revenueMetrics = make(map[string]interface{})
	}

	// Check if there are any orders for conditional display
	hasOrders := false
	if totalOrders, ok := revenueMetrics["total_orders"].(int); ok && totalOrders > 0 {
		hasOrders = true
	}

	allSites, _ := s.GetAllWebsites()

	data := map[string]interface{}{
		"Title":                   "Analytics",
		"Website":                 website,
		"AllSites":                allSites,
		"CurrentSite":             website,
		"ActiveSection":           "analytics",
		"Days":                    days,
		"StartDate":               startDate.Format("2006-01-02"),
		"EndDate":                 endDate.Format("2006-01-02"),
		"Stats":                   stats,
		"AvgPages":                avgPages,
		"TopPages":                topPages,
		"TopReferrers":            topReferrers,
		"EventStats":              eventStats,
		"ActiveUsers":             activeUsers,
		"CurrentPages":            currentPages,
		"BounceRate":              bounceRate,
		"AvgSessionDuration":      avgSessionDuration,
		"SessionDurationDisplay":  sessionDurationDisplay,
		"DeviceBreakdown":         deviceBreakdown,
		"EntryPages":              entryPages,
		"ExitPages":               exitPages,
		"ConversionRate":          conversionRate,
		"ConvertedSessions":       convertedSessions,
		"TotalSessions":           totalSessions,
		"AbandonmentRate":         abandonmentRate,
		"AbandonedCarts":          abandonedCarts,
		"TotalCarts":              totalCarts,
		"RevenueMetrics":          revenueMetrics,
		"HasOrders":               hasOrders,
	}

	s.renderWithLayout(w, r, "analytics_content.html", data)
}

// renderTemplate renders a template with data
func (s *AdminServer) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	tmplPath := filepath.Join("admin", "templates", tmpl+".html")

	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		// If template doesn't exist, render a simple placeholder
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>%s</h1><pre>%+v</pre><p><a href='/'>Back to Dashboard</a></p>", tmpl, data)
		return
	}

	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleMessagesList renders the messages inbox
func (s *AdminServer) handleMessagesList(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	messages, err := s.GetMessages(websiteID)
	if err != nil {
		log.Printf("Error fetching messages: %v", err)
		messages = []Message{}
	}

	data := map[string]interface{}{
		"Title":         "Messages",
		"Website":       website,
		"Messages":      messages,
		"ActiveSection": "messages",
	}

	s.renderWithLayout(w, r, "messages_list_content.html", data)
}

// handleMessageDetail renders a single message with replies
func (s *AdminServer) handleMessageDetail(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	messageIDStr := chi.URLParam(r, "messageId")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	website, err := s.GetWebsite(websiteID)
	if err != nil {
		http.Error(w, "Website not found", http.StatusNotFound)
		return
	}

	message, err := s.GetMessage(websiteID, messageID)
	if err != nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Mark as read when viewing
	db, err := s.GetWebsiteConnection(websiteID)
	if err == nil {
		dbConn := &database.DBConnection{Database: db, Connected: true}
		dbConn.MarkMessageAsRead(messageID)
		db.Close()
		// Update the message status in memory to reflect the change
		message.Status = "read"
	}

	// Look up customer by email if exists
	customer, err := s.GetCustomerByEmail(websiteID, message.Email)
	if err != nil {
		log.Printf("Error looking up customer by email: %v", err)
	}

	// Get status from query parameters (for reply feedback)
	status := r.URL.Query().Get("status")
	errorMsg := r.URL.Query().Get("error")
	preservedReply := r.URL.Query().Get("reply_text")

	data := map[string]interface{}{
		"Title":         "Message from " + message.Name,
		"Website":       website,
		"Message":       message,
		"Customer":      customer,
		"ActiveSection": "messages",
		"ReplyStatus":   status,
		"ReplyError":    errorMsg,
		"PreservedReply": preservedReply,
	}

	s.renderWithLayout(w, r, "message_detail_content.html", data)
}

// handleMessageReply handles reply submission and sends email
func (s *AdminServer) handleMessageReply(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	messageIDStr := chi.URLParam(r, "messageId")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	replyText := r.FormValue("reply_text")
	if replyText == "" {
		http.Error(w, "Reply text is required", http.StatusBadRequest)
		return
	}

	// Get the original message
	message, err := s.GetMessage(websiteID, messageID)
	if err != nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Send email to customer FIRST
	website, _ := s.GetWebsite(websiteID)
	err = s.SendReplyEmail(&website, message, replyText)

	redirectURL := fmt.Sprintf("/site/%s/messages/%d", websiteID, messageID)
	if err != nil {
		// Email failed - don't save reply, preserve text for retry
		log.Printf("Failed to send email: %v", err)
		redirectURL += "?status=error&error=" + url.QueryEscape(err.Error()) + "&reply_text=" + url.QueryEscape(replyText)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// Email sent successfully - now save reply to database
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	dbConn := &database.DBConnection{Database: db, Connected: true}
	err = dbConn.CreateReply(messageID, replyText, "admin")
	if err != nil {
		log.Printf("Error saving reply: %v", err)
		http.Error(w, "Failed to save reply", http.StatusInternalServerError)
		return
	}

	// Both emailed and saved successfully
	redirectURL += "?status=success"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// handleMessageToggleRead toggles message read status
func (s *AdminServer) handleMessageToggleRead(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	messageIDStr := chi.URLParam(r, "messageId")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get current status
	message, err := s.GetMessage(websiteID, messageID)
	if err != nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	dbConn := &database.DBConnection{Database: db, Connected: true}
	var redirectToList bool
	if message.Status == "read" {
		err = dbConn.MarkMessageAsUnread(messageID)
		redirectToList = true // Mark as unread goes back to list
	} else {
		err = dbConn.MarkMessageAsRead(messageID)
		redirectToList = false // Mark as read stays on detail page
	}

	if err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	// Redirect appropriately
	if redirectToList {
		http.Redirect(w, r, fmt.Sprintf("/site/%s/messages", websiteID), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/site/%s/messages/%d", websiteID, messageID), http.StatusSeeOther)
	}
}

// handleMessageDelete deletes a message and its replies
func (s *AdminServer) handleMessageDelete(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")
	messageIDStr := chi.URLParam(r, "messageId")

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	err = s.DeleteMessage(websiteID, messageID)
	if err != nil {
		log.Printf("Error deleting message: %v", err)
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	// Redirect back to messages list
	http.Redirect(w, r, fmt.Sprintf("/site/%s/messages", websiteID), http.StatusSeeOther)
}

// SendReplyEmail sends an email reply to the customer using SMTP
func (s *AdminServer) SendReplyEmail(website *Website, message *MessageWithReplies, replyText string) error {
	// Check if SMTP is configured
	if website.SMTPServer == "" || website.SMTPPort == 0 {
		return fmt.Errorf("SMTP not configured for this website")
	}

	// Build SMTP config
	smtpConfig := email.SMTPConfig{
		Server:   website.SMTPServer,
		Port:     website.SMTPPort,
		Username: website.SMTPUsername,
		Password: website.SMTPPassword,
		UseTLS:   website.SMTPUseTLS,
	}

	// Use configured from address or fall back to username
	fromAddress := website.EmailFromAddress
	if fromAddress == "" {
		fromAddress = website.SMTPUsername
	}

	fromName := website.EmailFromName
	if fromName == "" {
		fromName = website.SiteName
	}

	// For proper threading, we need the original message's Message-ID
	// Since contact form messages don't have a Message-ID initially,
	// we'll just send a simple reply for now
	// The IMAP polling will handle incoming threaded replies

	subject := "Re: Message from " + website.SiteName

	outgoingEmail := email.OutgoingEmail{
		From:     fromAddress,
		FromName: fromName,
		To:       message.Email,
		Subject:  subject,
		Body:     fmt.Sprintf("Hi %s,\n\n%s\n\nBest regards,\n%s", message.Name, replyText, fromName),
		ReplyTo:  fromAddress,
	}

	err := email.SendEmail(smtpConfig, outgoingEmail)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Email sent successfully to %s (%s)", message.Email, message.Name)
	return nil
}
