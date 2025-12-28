package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type AdminUser struct {
	Username     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	AllSites     bool     `json:"allSites"`     // If true, user has access to all sites
	SiteIDs      []string `json:"siteIds"`      // If AllSites is false, this lists the specific site IDs (database names) they can access
}

type EnvironmentConfig struct {
	ProdMode   bool
	HideErrors bool
	BaseURL    string `json:"baseUrl"` // Base URL for webhooks (e.g., "https://example.com")
	Database   struct {
		Host     string `json:"host"`
		User     string `json:"user"`
		Port     string `json:"port"`
		Password string `json:"password"`
	} `json:"database"`
	HTTP struct {
		Port string `json:"port"`
	} `json:"http"`
	Admin struct {
		Enabled    bool   `json:"enabled"`
		Port       string `json:"port"`
		Password   string `json:"password"`   // Password for the main "admin" user
		SessionKey string `json:"sessionKey"` // 32-byte key for encrypting session cookies
		CSRFKey    string `json:"csrfKey"`    // 32-byte key for CSRF token encryption
		Users      []AdminUser `json:"users"` // Additional users with limited permissions
	} `json:"admin"`
}

func ReadEnvironmentConfig(prodMode bool, hideErrors bool) (EnvironmentConfig, error) {
	configName := "env-dev.json"
	if prodMode {
		configName = "env-prod.json"
	}

	configPath := filepath.Join("websites", configName)
	configFile, err := os.Open(configPath)
	if err != nil {
		return EnvironmentConfig{}, fmt.Errorf("failed to open config file %s: %v", configName, err)
	}
	defer configFile.Close()

	configData, err := ioutil.ReadAll(configFile)
	if err != nil {
		return EnvironmentConfig{}, fmt.Errorf("failed to read config file %s: %v", configName, err)
	}

	var envConfig EnvironmentConfig
	err = json.Unmarshal(configData, &envConfig)
	if err != nil {
		return EnvironmentConfig{}, fmt.Errorf("failed to parse config file %s: %v", configName, err)
	}

	envConfig.ProdMode = prodMode
	envConfig.HideErrors = hideErrors

	// default port
	if envConfig.HTTP.Port == "" {
		envConfig.HTTP.Port = "80"
	}

	// default admin port
	if envConfig.Admin.Port == "" {
		envConfig.Admin.Port = "8081"
	}

	return envConfig, nil
}

// SaveEnvironmentConfig saves the environment config to disk
func SaveEnvironmentConfig(envConfig *EnvironmentConfig, prodMode bool) error {
	configName := "env-dev.json"
	if prodMode {
		configName = "env-prod.json"
	}

	configPath := filepath.Join("websites", configName)

	// Create a map to exclude runtime-only fields (ProdMode, HideErrors)
	configMap := map[string]interface{}{
		"baseUrl":  envConfig.BaseURL,
		"database": envConfig.Database,
		"http":     envConfig.HTTP,
		"admin":    envConfig.Admin,
	}

	// Marshal config to JSON with indentation
	configData, err := json.MarshalIndent(configMap, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file with proper permissions
	err = ioutil.WriteFile(configPath, configData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %v", configName, err)
	}

	return nil
}
