package frontend

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/murdinc/stencil2/api"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
)

// Global registry of websites for live configuration updates
var (
	websiteRegistry = make(map[string]*Website)
	registryMutex   sync.RWMutex
)

// Website represents an individual site
type Website struct {
	EnvironmentConfig *configs.EnvironmentConfig
	WebsiteConfig     *configs.WebsiteConfig
	TemplateConfigs   *[]configs.TemplateConfig
	APIHandler        *api.APIHandler
	DBConn            *database.DBConnection
	JSFiles           JSFiles
	CSSFiles          CSSFiles
	Hash              string
}

type UrlVars struct {
	Slug string
}

// NewWebsite creates a new Website instance
func NewWebsite(envConfig configs.EnvironmentConfig, websiteConfig configs.WebsiteConfig) (*Website, error) {

	dbConn := &database.DBConnection{}

	// Open a connection to the MySQL database
	err := dbConn.Connect(envConfig.Database.User, envConfig.Database.Password, envConfig.Database.Host, envConfig.Database.Port, websiteConfig.Database.Name, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Verify the database connection
	if dbConn.Connected {
		err = dbConn.Database.Ping()
		if err != nil {
			return nil, err
		}

		// Initialize article/content tables if they don't exist
		err = dbConn.InitArticleTables()
		if err != nil {
			log.Printf("Warning: Failed to initialize article tables: %v", err)
		}

		// Initialize e-commerce tables if they don't exist
		err = dbConn.InitEcommerceTables()
		if err != nil {
			log.Printf("Warning: Failed to initialize e-commerce tables: %v", err)
		}
	}

	// Read in the template configs
	templateConfigs, err := configs.ReadTemplateConfigs(websiteConfig.Directory)
	if err != nil {
		log.Fatalf("Failed to load template configs: %v", err)
	}

	website := &Website{
		EnvironmentConfig: &envConfig,
		WebsiteConfig:     &websiteConfig,
		TemplateConfigs:   &templateConfigs,
		DBConn:            dbConn,
	}

	// Load the JS Files
	website.JSFiles, err = website.LoadJS("")
	if err != nil {
		log.Fatalf("Error loading JS files %v", err)
	}

	// Load the CSS Files
	website.CSSFiles, err = website.LoadCSS("")
	if err != nil {
		log.Fatalf("Error loading CSS files %v", err)
	}

	// Store the hash of the website public directory
	website.Hash, err = MD5All(fmt.Sprintf("%s/public/", websiteConfig.Directory))
	if err != nil {
		log.Fatalf("Error generating hash of public directory %v", err)
	}

	// Register website in global registry
	RegisterWebsite(websiteConfig.Database.Name, website)

	return website, nil
}

// RegisterWebsite adds a website to the global registry
func RegisterWebsite(dbName string, website *Website) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	websiteRegistry[dbName] = website
}

// GetWebsite retrieves a website from the global registry
func GetWebsite(dbName string) (*Website, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	website, exists := websiteRegistry[dbName]
	return website, exists
}

// ReloadConfig reloads the website configuration from disk
func (website *Website) ReloadConfig(prodMode bool) error {
	// Read the fresh config from disk
	websiteConfigs, err := configs.ReadWebsiteConfigs(prodMode)
	if err != nil {
		return fmt.Errorf("failed to read website configs: %v", err)
	}

	// Find this website's config
	var newConfig configs.WebsiteConfig
	found := false
	for _, cfg := range websiteConfigs {
		if cfg.Database.Name == website.WebsiteConfig.Database.Name {
			newConfig = cfg
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("website config not found for database: %s", website.WebsiteConfig.Database.Name)
	}

	// Update the website's configuration
	registryMutex.Lock()
	website.WebsiteConfig = &newConfig
	registryMutex.Unlock()

	log.Printf("Reloaded configuration for website: %s", newConfig.SiteName)
	return nil
}

func MD5All(root string) (string, error) {
	var combinedHash [md5.Size]byte

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		fileHash := md5.Sum(data)
		// Combine the current file's hash with the accumulated hash
		for i := 0; i < md5.Size; i++ {
			combinedHash[i] += fileHash[i]
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	// Convert the combined hash to a hexadecimal string
	hashString := hex.EncodeToString(combinedHash[:])

	return hashString, nil
}
