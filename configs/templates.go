package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// TemplateConfig represents the configuration for a template
type TemplateConfig struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	PaginateType int      `json:"paginateType"`
	Requires     []string `json:"requires"`
	JSFile       string   `json:"jsFile"`
	CSSFile      string   `json:"cssFile"`
	QueryRow     string   `json:"queryRow"`
	APIEndpoint  string   `json:"apiEndpoint"`
	APITaxonomy  string   `json:"apiTaxonomy"`
	APISlug      string   `json:"apiSlug"`
	APICount     int      `json:"apiCount"`
	APIOffset    int      `json:"apiOffset"`
	MimeType     string   `json:"mimeType"`
	NoCache      bool     `json:"noCache"`
	CacheTime    int      `json:"cacheTime"`
	Directory    string
}

// ReadTemplateConfigs reads template configurations from the website configs
func ReadTemplateConfigs(baseDir string) ([]TemplateConfig, error) {
	var templateConfigs []TemplateConfig

	templatesDir := filepath.Join(baseDir, "templates")

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access directory: %v", err)
		}

		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read template config file [%s]: %v", path, err)
			}

			var templateConfig TemplateConfig
			err = json.Unmarshal(data, &templateConfig)
			if err != nil {
				return fmt.Errorf("failed to parse template config file [%s]: %v", path, err)
			}

			templateConfig.Directory = filepath.Dir(path)
			templateConfigs = append(templateConfigs, templateConfig)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return templateConfigs, nil
}
