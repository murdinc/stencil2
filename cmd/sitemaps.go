/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/frontend"
	"github.com/spf13/cobra"
)

// sitemapsCmd represents the sitemaps command
var sitemapsCmd = &cobra.Command{
	Use:   "sitemaps",
	Short: "Build sitemaps",
	Long:  `Build sitemaps`,
	Run: func(cmd *cobra.Command, args []string) {
		init, _ := cmd.Flags().GetBool("init")
		sitemaps(init)
	},
}

var InitSitemaps bool

func init() {
	rootCmd.AddCommand(sitemapsCmd)
	// flags and configuration settings.
	sitemapsCmd.Flags().BoolVarP(&InitSitemaps, "init", "i", false, "Initialize sitemaps table")
}

func sitemaps(init bool) {

	// Read in the env config
	envConfig, err := configs.ReadEnvironmentConfig(ProdMode, false)
	if err != nil {
		log.Fatalf("Failed to load the environment config: %v", err)
	}

	// Read in the site configs
	websiteConfigs, err := configs.ReadWebsiteConfigs(ProdMode)
	if err != nil {
		log.Fatalf("Failed to load site configs: %v", err)
	}

	if init {
		log.Println("initializing sitemaps...")
		for _, websiteConfig := range websiteConfigs {
			frontend.InitSitemaps(envConfig, websiteConfig)
		}
		return
	}

	log.Println("building sitemaps...")
	for _, websiteConfig := range websiteConfigs {
		frontend.BuildSitemaps(envConfig, websiteConfig)
	}

}
