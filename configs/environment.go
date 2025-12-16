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

	return envConfig, nil
}
