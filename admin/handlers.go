package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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

	s.renderWithLayout(w, r, "site_dashboard_content.html", map[string]interface{}{
		"Title":         site.SiteName + " - Dashboard",
		"ActiveSection": "",
		"Website":       site,
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
	})
}

// handleSiteSettingsUpdate updates site settings
func (s *AdminServer) handleSiteSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	websiteID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	website := Website{
		ID:            websiteID,
		SiteName:      r.FormValue("siteName"),
		Directory:     r.FormValue("directory"),
		DatabaseName:  r.FormValue("databaseName"),
		HTTPAddress:   r.FormValue("httpAddress"),
		MediaProxyURL: r.FormValue("mediaProxyUrl"),
		APIVersion:    1,
	}

	if err := s.UpdateWebsite(website); err != nil {
		http.Error(w, fmt.Sprintf("Error updating website: %v", err), http.StatusInternalServerError)
		return
	}

	s.LogActivity("update", "website", 0, websiteID, website)

	http.Redirect(w, r, fmt.Sprintf("/site/%s/settings", websiteID), http.StatusSeeOther)
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

	// Load all images
	images, err := s.GetImages(websiteID, 1000, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	s.renderWithLayout(w, r, "product_form_content.html", map[string]interface{}{
		"Title":         website.SiteName + " - New Product",
		"ActiveSection": "products",
		"FormTitle":     "Create New Product",
		"Website":       website,
		"Collections":   collections,
		"Images":        images,
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

	// Handle product images
	position := 0

	// Add existing images
	existingImageIDStrs := r.Form["existing_images[]"]
	for _, idStr := range existingImageIDStrs {
		if imageID, err := strconv.Atoi(idStr); err == nil {
			s.AddProductImage(websiteID, productID, imageID, position)
			position++
		}
	}

	// Upload and add new images
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

			// Create image record
			imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
			image := Image{
				URL:      imageURL,
				AltText:  altText,
				Credit:   credit,
				Filename: fileHeader.Filename,
				Size:     fileInfo.Size(),
			}

			if imageID, err := s.CreateImage(websiteID, image); err == nil {
				s.AddProductImage(websiteID, productID, int(imageID), position)
				position++
			}
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

	// Load all images
	images, err := s.GetImages(websiteID, 1000, 0)
	if err != nil {
		log.Printf("Error loading images: %v", err)
		images = []Image{}
	}

	// Load product images
	productImages, err := s.GetProductImages(websiteID, productID)
	if err != nil {
		log.Printf("Error loading product images: %v", err)
		productImages = []ProductImage{}
	}

	s.renderWithLayout(w, r, "product_form_content.html", map[string]interface{}{
		"Title":              website.SiteName + " - Edit Product",
		"ActiveSection":      "products",
		"FormTitle":          "Edit Product",
		"Website":            website,
		"Product":            product,
		"Collections":        collections,
		"ProductCollections": productCollections,
		"Images":             images,
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

	// Handle image removals
	removeImageIDStrs := r.Form["remove_images[]"]
	for _, idStr := range removeImageIDStrs {
		if piID, err := strconv.Atoi(idStr); err == nil {
			s.RemoveProductImage(websiteID, piID)
		}
	}

	// Get current max position
	currentImages, _ := s.GetProductImages(websiteID, productID)
	position := len(currentImages)

	// Add existing images
	existingImageIDStrs := r.Form["existing_images[]"]
	for _, idStr := range existingImageIDStrs {
		if imageID, err := strconv.Atoi(idStr); err == nil {
			s.AddProductImage(websiteID, productID, imageID, position)
			position++
		}
	}

	// Upload and add new images
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

			// Create image record
			imageURL := fmt.Sprintf("//%s/public/uploads/%s", website.HTTPAddress, filename)
			image := Image{
				URL:      imageURL,
				AltText:  altText,
				Credit:   credit,
				Filename: fileHeader.Filename,
				Size:     fileInfo.Size(),
			}

			if imageID, err := s.CreateImage(websiteID, image); err == nil {
				s.AddProductImage(websiteID, productID, int(imageID), position)
				position++
			}
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

	// Redirect back to order detail page
	http.Redirect(w, r, fmt.Sprintf("/site/%s/orders/%d", websiteID, orderID), http.StatusSeeOther)
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
