package admin

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
)

type AdminServer struct {
	Router     *chi.Mux
	EnvConfig  *configs.EnvironmentConfig
	DBConn     *database.DBConnection
	SessionKey string
}

// NewAdminServer creates a new admin server instance
func NewAdminServer(envConfig configs.EnvironmentConfig) (*AdminServer, error) {
	// Note: Admin no longer uses a database - everything is filesystem-based
	// We only need DB connections for individual website databases
	server := &AdminServer{
		Router:     chi.NewRouter(),
		EnvConfig:  &envConfig,
		DBConn:     nil, // Not needed anymore
		SessionKey: generateSessionKey(),
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

		// Category management
		r.Get("/site/{id}/categories", s.handleCategoriesList)
		r.Post("/site/{id}/categories/new", s.handleCategoryCreate)
		r.Post("/site/{id}/categories/{categoryId}/delete", s.handleCategoryDelete)

		// Collection management
		r.Get("/site/{id}/collections", s.handleCollectionsList)
		r.Post("/site/{id}/collections/new", s.handleCollectionCreate)
		r.Post("/site/{id}/collections/{collectionId}/delete", s.handleCollectionDelete)

		// Image management
		r.Get("/site/{id}/images", s.handleImagesList)
		r.Post("/site/{id}/images/upload", s.handleImageUpload)
		r.Post("/site/{id}/images/{imageId}/delete", s.handleImageDelete)
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
