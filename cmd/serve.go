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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/spf13/cobra"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/frontend"
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

	for _, websiteConfig := range websiteConfigs {

		// Create a new site instance
		website, err := frontend.NewWebsite(envConfig, websiteConfig)
		if err != nil {
			log.Fatal(err)
		}
		if website.DBConn.Connected {
			defer website.DBConn.Database.Close()
		}

		router := website.GetRouter()

		hr.Map(website.WebsiteConfig.HTTP.Address, router())

		// start file watcher on dev
		if ProdMode == false {
			go website.StartWatcher()
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
