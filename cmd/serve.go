/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/spf13/cobra"

	"github.com/murdinc/stencil2/admin"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/frontend"
	"github.com/murdinc/stencil2/utils"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start server",
	Long:  `Starts the HTTP server`,
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

var HideErrors bool

func init() {
	rootCmd.AddCommand(serveCmd)
	// flags and configuration settings.
	serveCmd.Flags().BoolVarP(&HideErrors, "hide-errors", "", false, "Hide friendly dev errors (only applies to dev mode)")
}

func serve() {

	// Set production mode for cookie security
	utils.SetProductionMode(ProdMode)

	// Read in the env config
	envConfig, err := configs.ReadEnvironmentConfig(ProdMode, HideErrors)
	if err != nil {
		log.Fatalf("Failed to load the environment config: %v", err)
	}

	// Setup admin credentials and keys if needed
	if envConfig.Admin.Enabled {
		configModified := false

		// Check if admin password needs to be set
		if envConfig.Admin.Password == "" {
			configModified = true
			if err := setupAdminPassword(&envConfig); err != nil {
				log.Fatalf("Failed to setup admin password: %v", err)
			}
		}

		// Check if session key needs to be generated
		if envConfig.Admin.SessionKey == "" || len(envConfig.Admin.SessionKey) != 32 {
			configModified = true
			sessionKey, err := utils.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("Failed to generate session key: %v", err)
			}
			envConfig.Admin.SessionKey = sessionKey
			log.Println("Generated new session key")
		}

		// Check if CSRF key needs to be generated
		if envConfig.Admin.CSRFKey == "" || len(envConfig.Admin.CSRFKey) != 32 {
			configModified = true
			csrfKey, err := utils.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("Failed to generate CSRF key: %v", err)
			}
			envConfig.Admin.CSRFKey = csrfKey
			log.Println("Generated new CSRF key")
		}

		// Save config if modified
		if configModified {
			if err := configs.SaveEnvironmentConfig(&envConfig, ProdMode); err != nil {
				log.Fatalf("Failed to save environment config: %v", err)
			}
			log.Println("✓ Environment config updated and saved")
		}
	}

	// Read in the site configs
	websiteConfigs, err := configs.ReadWebsiteConfigs(ProdMode)
	if err != nil {
		log.Fatalf("Failed to load site configs: %v", err)
	}

	// Setup the Router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	//r.Use(middleware.CleanPath)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Recoverer)
	r.Use(EnvCtx)

	hr := hostrouter.New()

	// Use channels and sync to parallelize website initialization
	type websiteResult struct {
		website *frontend.Website
		config  configs.WebsiteConfig
		err     error
	}

	results := make(chan websiteResult, len(websiteConfigs))
	var wg sync.WaitGroup

	log.Printf("Initializing %d websites in parallel...", len(websiteConfigs))

	// Initialize all websites concurrently
	for _, websiteConfig := range websiteConfigs {
		wg.Add(1)
		go func(config configs.WebsiteConfig) {
			defer wg.Done()
			result := websiteResult{config: config}

			log.Printf("[%s] Starting initialization...", config.SiteName)

			// Create a new site instance
			website, err := frontend.NewWebsite(envConfig, config)
			if err != nil {
				result.err = err
				results <- result
				return
			}

			result.website = website
			results <- result

			log.Printf("[%s] ✓ Initialized successfully", config.SiteName)
		}(websiteConfig)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	websites := make([]*frontend.Website, 0, len(websiteConfigs))
	for result := range results {
		if result.err != nil {
			log.Fatalf("[%s] Failed to initialize: %v", result.config.SiteName, result.err)
		}

		websites = append(websites, result.website)

		// Register router for this website
		router := result.website.GetRouter()
		hr.Map(result.website.WebsiteConfig.HTTP.Address, router())

		// Defer database close
		if result.website.DBConn.Connected {
			defer result.website.DBConn.Database.Close()
		}

		// Start file watcher on dev
		if ProdMode == false {
			go result.website.StartWatcher()
		}
	}

	log.Printf("✓ Successfully initialized all %d websites", len(websites))

	// Start admin server if enabled
	if envConfig.Admin.Enabled {
		adminServer, err := admin.NewAdminServer(envConfig)
		if err != nil {
			log.Printf("Warning: Failed to start admin server: %v", err)
		} else {
			// Start email polling service
			adminServer.StartEmailPolling()

			// Start admin HTTP server
			go func() {
				if err := adminServer.Start(); err != nil {
					log.Printf("Admin server error: %v", err)
				}
			}()
		}
	}

	log.Println("starting http server...")
	r.Mount("/", hr) // Mount the host router

	// Health check endpoint (simple - just checks if server is running)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	// Legacy hello endpoint
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	portInt, err := strconv.Atoi(envConfig.HTTP.Port)
	if err != nil || portInt < 0 {
		portInt = 80
	}

	port := fmt.Sprintf(":%d", portInt)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("HTTP server listening on %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

func EnvCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "prodmode", ProdMode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// setupAdminPassword prompts for and sets up the admin password
func setupAdminPassword(envConfig *configs.EnvironmentConfig) error {
	fmt.Println("\n=== Admin Setup ===")
	fmt.Println("No admin password found. Let's set one up.")
	fmt.Print("Enter admin password: ")

	password, err := utils.ReadPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %v", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	fmt.Print("Confirm admin password: ")
	confirmPassword, err := utils.ReadPassword()
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %v", err)
	}

	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Hash the password with bcrypt
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	envConfig.Admin.Password = hashedPassword
	log.Println("✓ Admin password set successfully")

	return nil
}
