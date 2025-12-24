package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type EnvironmentConfig struct {
	ProdMode   bool
	HideErrors bool
	Database   struct {
		Host     string `json:"host"`
		User     string `json:"user"`
		Port     string `json:"port"`
		Password string `json:"password"`
		Name     string `json:"name"`
	} `json:"database"`
	HTTP struct {
		Port string `json:"port"`
	} `json:"http"`
	Admin struct {
		Enabled  bool   `json:"enabled"`
		Port     string `json:"port"`
		Password string `json:"password"`
		Database struct {
			Name string `json:"name"`
		} `json:"database"`
	} `json:"admin"`
	Email struct {
		Provider string `json:"provider"`
		SES      struct {
			Region          string `json:"region"`
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		} `json:"ses"`
		Admin struct {
			FromAddress string `json:"fromAddress"`
			FromName    string `json:"fromName"`
		} `json:"admin"`
	} `json:"email"`
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

	// default admin database name
	if envConfig.Admin.Database.Name == "" {
		envConfig.Admin.Database.Name = "stencil_admin"
	}

	return envConfig, nil
}
