/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var ProdMode bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "stencil2",
	Short: "A brief description of your application",
	Long:  `Template Engine`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// flags and configuration settings.
	rootCmd.PersistentFlags().BoolVarP(&ProdMode, "prod-mode", "p", false, "production mode")
}
