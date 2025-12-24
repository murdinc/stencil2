package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type WebsiteConfig struct {
	SiteName   string `json:"siteName"`
	APIVersion int    `json:"apiVersion"`
	Database   struct {
		Name string `json:"name"`
	} `json:"database"`
	MediaProxyURL string `json:"mediaProxyUrl"`
	HTTP          struct {
		Address string `json:"address"`
	} `json:"http"`
	Stripe struct {
		PublishableKey string `json:"publishableKey"`
		SecretKey      string `json:"secretKey"`
	} `json:"stripe"`
	Shippo struct {
		APIKey      string `json:"apiKey"`
		LabelFormat string `json:"labelFormat"` // PDF, PDF_4x6, ZPLII, PNG
	} `json:"shippo"`
	Email struct {
		FromAddress string `json:"fromAddress"`
		FromName    string `json:"fromName"`
		ReplyTo     string `json:"replyTo"`
	} `json:"email"`
	Ecommerce struct {
		TaxRate      float64 `json:"taxRate"`      // e.g., 0.08 for 8%
		ShippingCost float64 `json:"shippingCost"` // flat rate shipping cost
	} `json:"ecommerce"`
	EarlyAccess struct {
		Enabled  bool   `json:"enabled"`
		Password string `json:"password"`
	} `json:"earlyAccess"`
	ShipFrom struct {
		Name    string `json:"name"`
		Street1 string `json:"street1"`
		Street2 string `json:"street2"`
		City    string `json:"city"`
		State   string `json:"state"`
		Zip     string `json:"zip"`
		Country string `json:"country"`
	} `json:"shipFrom"`
	Directory string
}

func ReadWebsiteConfigs(prodMode bool) ([]WebsiteConfig, error) {
	var websiteConfigs []WebsiteConfig

	configName := "config-dev.json"
	if prodMode {
		configName = "config-prod.json"
	}

	baseDir := "websites"
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			configPath := filepath.Join(path, configName)
			configFile, err := os.Open(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil // Continue searching
				}
				return fmt.Errorf("failed to open config file in directory %s: %v", path, err)
			}

			configData, err := ioutil.ReadAll(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file in directory %s: %v", path, err)
			}

			var websiteConfig WebsiteConfig
			err = json.Unmarshal(configData, &websiteConfig)
			if err != nil {
				return fmt.Errorf("failed to parse config file in directory %s: %v", path, err)
			}

			// default API version (1)
			if websiteConfig.APIVersion == 0 {
				websiteConfig.APIVersion = 1
			}

			websiteConfig.Directory = path
			websiteConfigs = append(websiteConfigs, websiteConfig)

			log.Printf("Found website config file in: [%s]", websiteConfig.Directory)

			configFile.Close()
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk through directories: %v", err)
	}

	return websiteConfigs, nil
}
