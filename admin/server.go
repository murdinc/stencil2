package admin

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/email"
)

type AdminServer struct {
	Router       *chi.Mux
	EnvConfig    *configs.EnvironmentConfig
	DBConn       *database.DBConnection
	SessionStore *sessions.CookieStore
	CSRFKey      []byte
}

// NewAdminServer creates a new admin server instance
func NewAdminServer(envConfig configs.EnvironmentConfig) (*AdminServer, error) {
	// Note: Admin no longer uses a database - everything is filesystem-based
	// We only need DB connections for individual website databases

	// Create encrypted cookie session store
	sessionKey := []byte(envConfig.Admin.SessionKey)
	if len(sessionKey) != 32 {
		log.Fatal("Admin session key must be exactly 32 bytes")
	}

	store := sessions.NewCookieStore(sessionKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Secure:   envConfig.ProdMode,
		SameSite: http.SameSiteLaxMode,
	}

	// Validate CSRF key
	csrfKey := []byte(envConfig.Admin.CSRFKey)
	if len(csrfKey) != 32 {
		log.Fatal("Admin CSRF key must be exactly 32 bytes")
	}

	server := &AdminServer{
		Router:       chi.NewRouter(),
		EnvConfig:    &envConfig,
		DBConn:       nil, // Not needed anymore
		SessionStore: store,
		CSRFKey:      csrfKey,
	}

	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all admin routes
func (s *AdminServer) setupRoutes() {
	// Middleware
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.Compress(5))

	// CSRF protection - only enable in production
	if s.EnvConfig.ProdMode {
		csrfOptions := []csrf.Option{
			csrf.Secure(true),
			csrf.Path("/"),
			csrf.SameSite(csrf.SameSiteLaxMode),
			csrf.RequestHeader("X-CSRF-Token"),
		}

		csrfMiddleware := csrf.Protect(s.CSRFKey, csrfOptions...)
		s.Router.Use(csrfMiddleware)
	}

	// Public routes (login)
	s.Router.Get("/login", s.handleLoginPage)
	s.Router.Post("/login", s.handleLogin)

	// Protected routes (require authentication)
	s.Router.Group(func(r chi.Router) {
		r.Use(s.requireAuth)

		// Dashboard
		r.Get("/", s.handleDashboard)
		r.Get("/logout", s.handleLogout)

		// Website management (legacy - admin only)
		r.Group(func(r chi.Router) {
			r.Use(s.requireSuperadmin)
			r.Get("/websites/new", s.handleWebsiteNew)
			r.Post("/websites/new", s.handleWebsiteCreate)
		})

		// Superadmin console (admin only)
		r.Group(func(r chi.Router) {
			r.Use(s.requireSuperadmin)
			r.Get("/superadmin", s.handleSuperadmin)
			r.Get("/superadmin/checkup", s.handleSuperadminCheckup)
			r.Get("/superadmin/websites/new", s.handleSuperadminWebsiteNew)
			r.Post("/superadmin/websites/new", s.handleSuperadminWebsiteCreate)
			r.Get("/superadmin/users", s.handleSuperadminUsers)
			r.Post("/superadmin/users/create", s.handleSuperadminUserCreate)
			r.Post("/superadmin/users/update", s.handleSuperadminUserUpdate)
			r.Post("/superadmin/users/delete", s.handleSuperadminUserDelete)
		})

		// Site context routes (ID is database name) - requires site access
		r.Route("/site/{id}", func(r chi.Router) {
			r.Use(s.requireSiteAccess)

			// Site dashboard and settings
			r.Get("/", s.handleSiteDashboard)
			r.Get("/settings", s.handleSiteSettings)
			r.Post("/settings", s.handleSiteSettingsUpdate)
			r.Get("/webhooks", s.handleWebhooks)
			r.Post("/delete", s.handleWebsiteDelete)

			// Article management
			r.Get("/articles", s.handleArticlesList)
			r.Get("/articles/new", s.handleArticleNew)
			r.Post("/articles/new", s.handleArticleCreate)
			r.Get("/articles/{articleId}/edit", s.handleArticleEdit)
			r.Post("/articles/{articleId}/edit", s.handleArticleUpdate)
			r.Post("/articles/{articleId}/delete", s.handleArticleDelete)

			// Product management
			r.Get("/products", s.handleProductsList)
			r.Get("/products/new", s.handleProductNew)
			r.Post("/products/new", s.handleProductCreate)
			r.Get("/products/{productId}/edit", s.handleProductEdit)
			r.Post("/products/{productId}/edit", s.handleProductUpdate)
			r.Post("/products/{productId}/delete", s.handleProductDelete)
			r.Post("/products/{productId}/reorder/{direction}", s.handleProductReorder)
			r.Post("/products/{productId}/images/reorder", s.handleProductImageReorder)

			// Variant management
			r.Get("/products/{productId}/variants/new", s.handleVariantNew)
			r.Post("/products/{productId}/variants/create", s.handleVariantCreate)
			r.Get("/products/{productId}/variants/{variantId}/edit", s.handleVariantEdit)
			r.Post("/products/{productId}/variants/{variantId}/update", s.handleVariantUpdate)
			r.Post("/products/{productId}/variants/{variantId}/delete", s.handleVariantDelete)
			r.Post("/products/{productId}/variants/{variantId}/reorder/{direction}", s.handleVariantReorder)

			// Category management
			r.Get("/categories", s.handleCategoriesList)
			r.Post("/categories/new", s.handleCategoryCreate)
			r.Post("/categories/{categoryId}/delete", s.handleCategoryDelete)

			// Collection management
			r.Get("/collections", s.handleCollectionsList)
			r.Post("/collections/new", s.handleCollectionCreate)
			r.Get("/collections/{collectionId}/edit", s.handleCollectionEditForm)
			r.Post("/collections/{collectionId}/edit", s.handleCollectionUpdate)
			r.Post("/collections/{collectionId}/reorder/{direction}", s.handleCollectionReorder)
			r.Post("/collections/{collectionId}/delete", s.handleCollectionDelete)

			// Image management
			r.Get("/images", s.handleImagesList)
			r.Post("/images/upload", s.handleImageUpload)
			r.Post("/images/{imageId}/delete", s.handleImageDelete)

			// Order management
			r.Get("/orders", s.handleOrdersList)
			r.Get("/orders/{orderId}", s.handleOrderDetail)
			r.Get("/orders/{orderId}/packing-slip", s.handlePackingSlip)
			r.Post("/orders/{orderId}/fulfillment", s.handleOrderFulfillmentUpdate)
			r.Post("/orders/{orderId}/shipping/rates", s.handleShippingRates)
			r.Post("/orders/{orderId}/shipping/purchase", s.handleShippingLabelPurchase)

			// Customer management
			r.Get("/customers", s.handleCustomersList)
			r.Get("/customers/{customerId}", s.handleCustomerDetail)

			// Messages (Contact Form)
			r.Get("/messages", s.handleMessagesList)
			r.Get("/messages/{messageId}", s.handleMessageDetail)
			r.Post("/messages/{messageId}/reply", s.handleMessageReply)
			r.Post("/messages/{messageId}/toggle-read", s.handleMessageToggleRead)
			r.Post("/messages/{messageId}/delete", s.handleMessageDelete)

			// SMS Signups (Marketing)
			r.Get("/sms-signups", s.handleSMSSignupsList)
			r.Post("/sms-signups/{signupId}/delete", s.handleDeleteSMSSignup)
			r.Get("/sms-signups/export", s.handleExportSMSSignups)

			// SMS Campaigns (Marketing)
			r.Get("/sms-campaigns", s.handleSMSCampaignForm)
			r.Post("/sms-campaigns/send", s.handleSendBulkSMS)

			// Analytics
			r.Get("/analytics", s.handleAnalytics)
		})
	})

	// Static assets for admin UI
	s.Router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("admin/static"))))
}

// Start starts the admin server
func (s *AdminServer) Start() error {
	port := s.EnvConfig.Admin.Port
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting admin server on port %s", port)
	return http.ListenAndServe(":"+port, s.Router)
}

// StartEmailPolling starts background email polling for all websites with IMAP configured
func (s *AdminServer) StartEmailPolling() {
	// Poll every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)

	log.Println("Starting email polling service (checks every 5 minutes)")

	// Run immediately on start
	go s.pollAllWebsites()

	// Then run on schedule
	go func() {
		for range ticker.C {
			s.pollAllWebsites()
		}
	}()
}

// pollAllWebsites polls IMAP for all websites that have it configured
func (s *AdminServer) pollAllWebsites() {
	websites, err := s.GetAllWebsites()
	if err != nil {
		log.Printf("Error getting websites for email polling: %v", err)
		return
	}

	for _, website := range websites {
		// Skip if IMAP not configured
		if website.IMAPServer == "" || website.IMAPPort == 0 {
			continue
		}

		// Poll this website
		go s.pollWebsiteEmails(website)
	}
}

// pollWebsiteEmails polls IMAP for a single website
func (s *AdminServer) pollWebsiteEmails(website Website) {
	log.Printf("Polling emails for %s...", website.SiteName)

	// Build IMAP config
	imapConfig := email.IMAPConfig{
		Server:   website.IMAPServer,
		Port:     website.IMAPPort,
		Username: website.IMAPUsername,
		Password: website.IMAPPassword,
		UseTLS:   website.IMAPUseTLS,
	}

	// Get database connection
	db, err := s.GetWebsiteConnection(website.ID)
	if err != nil {
		log.Printf("Error connecting to database for %s: %v", website.SiteName, err)
		return
	}
	defer db.Close()

	// Create message matcher
	matcher := &email.DBMessageMatcher{DB: db}

	// Poll for emails
	result, err := email.PollIncomingEmails(imapConfig, matcher)
	if err != nil {
		log.Printf("Error polling emails for %s: %v", website.SiteName, err)
		return
	}

	// Log results
	if result.RepliesAdded > 0 {
		log.Printf("Added %d email replies for %s (checked %d emails)",
			result.RepliesAdded, website.SiteName, result.EmailsChecked)
	}

	if len(result.Errors) > 0 {
		log.Printf("Encountered %d errors while processing emails for %s",
			len(result.Errors), website.SiteName)
		for _, err := range result.Errors {
			log.Printf("  - %v", err)
		}
	}
}
