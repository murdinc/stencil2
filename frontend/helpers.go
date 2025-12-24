package frontend

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/murdinc/stencil2/configs"
	"github.com/radovskyb/watcher"
)

func removeDuplicateString(strSlice []string) []string {
	// map to store unique keys
	keys := make(map[string]bool)
	returnSlice := []string{}
	for _, item := range strSlice {
		if _, value := keys[item]; !value {
			keys[item] = true
			returnSlice = append(returnSlice, item)
		}
	}
	return returnSlice
}

// Watches and reloads files on edit (dev only)
func (website *Website) StartWatcher() {

	log.Printf("Starting watcher for: [%s]", website.WebsiteConfig.SiteName)

	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(watcher.Write)

	r := regexp.MustCompile(`.*\.(css|js|json)$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go func() {

		pwd, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}

		for {
			select {
			case event := <-w.Event:

				fileChanged := strings.TrimPrefix(event.Path, pwd+"/")

				log.Printf("File changed: [%s]", fileChanged)

				switch filepath.Ext(fileChanged) {

				case ".css":
				cssLoop:
					for outputFile, files := range website.CSSFiles {
						for _, file := range files {
							if file == fileChanged {
								website.LoadCSS(outputFile)
								break cssLoop
							}
						}
					}
				case ".js":
				jsLoop:
					for outputFile, files := range website.JSFiles {
						for _, file := range files {
							if file == fileChanged {
								website.LoadJS(outputFile)
								break jsLoop
							}
						}
					}
				case ".json":
					// Check if it's a config file (config-dev.json or config-prod.json)
					if strings.HasPrefix(filepath.Base(fileChanged), "config-") {
						log.Printf("Config file changed, reloading website config...")
						if err := website.ReloadConfig(website.EnvironmentConfig.ProdMode); err != nil {
							log.Printf("Error reloading config: %v", err)
						}
					} else {
						// Template config changed
						templateConfigs, err := configs.ReadTemplateConfigs(website.WebsiteConfig.Directory)
						website.TemplateConfigs = &templateConfigs
						if err != nil {
							log.Println(err)
						}
					}
				default:

				}

			case err := <-w.Error:
				log.Fatalln(err.Error())
			case <-w.Closed:
				return
			}
		}
	}()

	// Watch the templates folder recursively for changes.
	if err := w.AddRecursive(website.WebsiteConfig.Directory + "/templates"); err != nil {
		log.Fatalln(err)
	}

	// Also watch the website root directory for config file changes
	if err := w.Add(website.WebsiteConfig.Directory); err != nil {
		log.Fatalln(err)
	}

	for file, _ := range w.WatchedFiles() {
		fmt.Printf("			> Including: %s\n", file)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func ParseQueryParams(queryParam string, data interface{}) string {
	t, _ := template.New("queryParam").Parse(queryParam)

	var tmpParam bytes.Buffer
	t.Execute(&tmpParam, data)

	return tmpParam.String()
}
