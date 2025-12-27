package admin

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/email"
)

type AdminServer struct {
	Router    *chi.Mux
	EnvConfig *configs.EnvironmentConfig
	DBConn    *database.DBConnection
}

// NewAdminServer creates a new admin server instance
func NewAdminServer(envConfig configs.EnvironmentConfig) (*AdminServer, error) {
	// Note: Admin no longer uses a database - everything is filesystem-based
	// We only need DB connections for individual website databases
	server := &AdminServer{
		Router:    chi.NewRouter(),
		EnvConfig: &envConfig,
		DBConn:    nil, // Not needed anymore
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

	// Public routes (login)
	s.Router.Get("/login", s.handleLoginPage)
	s.Router.Post("/login", s.handleLogin)

	// Protected routes (require authentication)
	s.Router.Group(func(r chi.Router) {
		r.Use(s.requireAuth)

		// Dashboard
		r.Get("/", s.handleDashboard)
		r.Get("/logout", s.handleLogout)

		// Website management
		r.Get("/websites/new", s.handleWebsiteNew)
		r.Post("/websites/new", s.handleWebsiteCreate)

		// Site context routes (ID is database name)
		r.Get("/site/{id}", s.handleSiteDashboard)
		r.Get("/site/{id}/settings", s.handleSiteSettings)
		r.Post("/site/{id}/settings", s.handleSiteSettingsUpdate)
		r.Get("/site/{id}/webhooks", s.handleWebhooks)
		r.Post("/site/{id}/delete", s.handleWebsiteDelete)

		// Article management
		r.Get("/site/{id}/articles", s.handleArticlesList)
		r.Get("/site/{id}/articles/new", s.handleArticleNew)
		r.Post("/site/{id}/articles/new", s.handleArticleCreate)
		r.Get("/site/{id}/articles/{articleId}/edit", s.handleArticleEdit)
		r.Post("/site/{id}/articles/{articleId}/edit", s.handleArticleUpdate)
		r.Post("/site/{id}/articles/{articleId}/delete", s.handleArticleDelete)

		// Product management
		r.Get("/site/{id}/products", s.handleProductsList)
		r.Get("/site/{id}/products/new", s.handleProductNew)
		r.Post("/site/{id}/products/new", s.handleProductCreate)
		r.Get("/site/{id}/products/{productId}/edit", s.handleProductEdit)
		r.Post("/site/{id}/products/{productId}/edit", s.handleProductUpdate)
		r.Post("/site/{id}/products/{productId}/delete", s.handleProductDelete)
		r.Post("/site/{id}/products/{productId}/reorder/{direction}", s.handleProductReorder)
		r.Post("/site/{id}/products/{productId}/images/reorder", s.handleProductImageReorder)

		// Variant management
		r.Get("/site/{id}/products/{productId}/variants/new", s.handleVariantNew)
		r.Post("/site/{id}/products/{productId}/variants/create", s.handleVariantCreate)
		r.Get("/site/{id}/products/{productId}/variants/{variantId}/edit", s.handleVariantEdit)
		r.Post("/site/{id}/products/{productId}/variants/{variantId}/update", s.handleVariantUpdate)
		r.Post("/site/{id}/products/{productId}/variants/{variantId}/delete", s.handleVariantDelete)
		r.Post("/site/{id}/products/{productId}/variants/{variantId}/reorder/{direction}", s.handleVariantReorder)

		// Category management
		r.Get("/site/{id}/categories", s.handleCategoriesList)
		r.Post("/site/{id}/categories/new", s.handleCategoryCreate)
		r.Post("/site/{id}/categories/{categoryId}/delete", s.handleCategoryDelete)

		// Collection management
		r.Get("/site/{id}/collections", s.handleCollectionsList)
		r.Post("/site/{id}/collections/new", s.handleCollectionCreate)
		r.Get("/site/{id}/collections/{collectionId}/edit", s.handleCollectionEditForm)
		r.Post("/site/{id}/collections/{collectionId}/edit", s.handleCollectionUpdate)
		r.Post("/site/{id}/collections/{collectionId}/reorder/{direction}", s.handleCollectionReorder)
		r.Post("/site/{id}/collections/{collectionId}/delete", s.handleCollectionDelete)

		// Image management
		r.Get("/site/{id}/images", s.handleImagesList)
		r.Post("/site/{id}/images/upload", s.handleImageUpload)
		r.Post("/site/{id}/images/{imageId}/delete", s.handleImageDelete)

		// Order management
		r.Get("/site/{id}/orders", s.handleOrdersList)
		r.Get("/site/{id}/orders/{orderId}", s.handleOrderDetail)
		r.Get("/site/{id}/orders/{orderId}/packing-slip", s.handlePackingSlip)
		r.Post("/site/{id}/orders/{orderId}/fulfillment", s.handleOrderFulfillmentUpdate)
		r.Post("/site/{id}/orders/{orderId}/shipping/rates", s.handleShippingRates)
		r.Post("/site/{id}/orders/{orderId}/shipping/purchase", s.handleShippingLabelPurchase)

		// Customer management
		r.Get("/site/{id}/customers", s.handleCustomersList)
		r.Get("/site/{id}/customers/{customerId}", s.handleCustomerDetail)


	// Messages (Contact Form)
	r.Get("/site/{id}/messages", s.handleMessagesList)
	r.Get("/site/{id}/messages/{messageId}", s.handleMessageDetail)
	r.Post("/site/{id}/messages/{messageId}/reply", s.handleMessageReply)
	r.Post("/site/{id}/messages/{messageId}/toggle-read", s.handleMessageToggleRead)
	r.Post("/site/{id}/messages/{messageId}/delete", s.handleMessageDelete)
		// SMS Signups (Marketing)
		r.Get("/site/{id}/sms-signups", s.handleSMSSignupsList)
		r.Post("/site/{id}/sms-signups/{signupId}/delete", s.handleDeleteSMSSignup)
		r.Get("/site/{id}/sms-signups/export", s.handleExportSMSSignups)

		// SMS Campaigns (Marketing)
		r.Get("/site/{id}/sms-campaigns", s.handleSMSCampaignForm)
		r.Post("/site/{id}/sms-campaigns/send", s.handleSendBulkSMS)

		// Analytics
		r.Get("/site/{id}/analytics", s.handleAnalytics)
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
