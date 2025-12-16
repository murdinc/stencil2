/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/spf13/cobra"
)

// localdbCmd represents the localdb command
var localdbCmd = &cobra.Command{
	Use:   "localdb",
	Short: "Start localdb",
	Long:  `Starts a localdb and loads any .sql files found in configs`,
	Run: func(cmd *cobra.Command, args []string) {
		localdb()
	},
}

func init() {
	rootCmd.AddCommand(localdbCmd)
}

func localdb() {

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

	dbConn := &database.DBConnection{}
	if !ProdMode && envConfig.Database.Host == "localhost" {
		// Create in-memory mysql engine
		dbConn = database.SetupLocalDB(envConfig.Database.Host, envConfig.Database.Port, envConfig.Database.Name)

		// Connect to the root database
		err = dbConn.Connect(envConfig.Database.User, envConfig.Database.Password, envConfig.Database.Host, envConfig.Database.Port, envConfig.Database.Name, 1000)
		if err != nil {
			log.Fatalf("Failed to connect to the database: %v", err)
		}
		if dbConn.Connected {
			defer dbConn.Database.Close()
		}

	}

	for _, websiteConfig := range websiteConfigs {
		if websiteConfig.Database.Name != "" {
			dbConn.LoadDB(websiteConfig.Database.Name, websiteConfig.Directory)
		}
	}

	log.Println("Database Ready!")
	blockForever()
}

func blockForever() {
	select {}
}
