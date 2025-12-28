/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

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

	// for server health checks
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	portInt, err := strconv.Atoi(envConfig.HTTP.Port)
	if err != nil || portInt < 0 {
		portInt = 80
	}

	port := fmt.Sprintf(":%d", portInt)
	log.Fatal(http.ListenAndServe(port, r))

}

func EnvCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "prodmode", ProdMode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
