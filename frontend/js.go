package frontend

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

type JSFiles map[string][]string

func (website *Website) LoadJS(outputFile string) (JSFiles, error) {

	log.Printf("Building JS files for: [%s]", website.WebsiteConfig.SiteName)
	jsMap := make(map[string][]string)

	for _, tpl := range *website.TemplateConfigs {

		// skip if there is no specified output file
		if tpl.JSFile == "" {
			continue
		}

		if outputFile != "" && tpl.JSFile != outputFile {
			continue
		}

		// Crawl the folders of tpl.Requires first
		for _, required := range tpl.Requires {
			requiredDir := filepath.Join(tpl.Directory, required)

			reqTpl := website.GetTemplate(required)
			err := crawlJSFiles(reqTpl.Directory, tpl.JSFile, jsMap)
			if err != nil {
				return jsMap, fmt.Errorf("failed to crawl JS files for %s: %v", requiredDir, err)
			}
		}

		// Then search for .js files in tpl.Directory
		err := crawlJSFiles(tpl.Directory, tpl.JSFile, jsMap)
		if err != nil {
			return jsMap, fmt.Errorf("failed to crawl JS files for %s: %v", tpl.Directory, err)
		}
	}

	for jsFile, filePaths := range jsMap {

		filePaths = removeDuplicateString(filePaths)

		destinationJSFolder := fmt.Sprintf("%s/public/js/", website.WebsiteConfig.Directory)
		err := MinifyAndCombineJS(filePaths, destinationJSFolder, jsFile)
		if err != nil {
			log.Println(err.Error())
		}

		fmt.Printf("			Building JS file: [%s]\n", jsFile)

		for _, filePath := range filePaths {
			fmt.Printf("			> Including: %s\n", filePath)
		}
	}

	log.Printf("Done Building JS files for: [%s]", website.WebsiteConfig.SiteName)

	return jsMap, nil
}

func crawlJSFiles(dir string, destinationJSFile string, jsMap JSFiles) error {
	var jsFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access directory: %v", err)
		}

		if !info.IsDir() && strings.HasSuffix(path, ".js") {
			jsFiles = append(jsFiles, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(jsFiles) > 0 {
		jsMap[destinationJSFile] = append(jsMap[destinationJSFile], jsFiles...)
	}

	return nil
}

func MinifyAndCombineJS(requiredJSFiles []string, destinationJSFolder string, destinationJSFile string) error {

	// Bail if we dont have any files
	if len(requiredJSFiles) < 1 {
		return nil
	}

	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)

	// Create js folder if it doesn't exist
	if _, err := os.Stat(destinationJSFolder); os.IsNotExist(err) {
		err = os.MkdirAll(destinationJSFolder, 0755)
		if err != nil {
			return err
		}
	}

	outputFile := destinationJSFolder + destinationJSFile

	// Open main.js to write to it.
	fo, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer fo.Close()

	// Create a minifier
	minifiedWriter := m.Writer("text/javascript", fo)

	for _, file := range requiredJSFiles {
		// Open the file
		fi, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fi.Close()

		_, err = io.Copy(minifiedWriter, fi)
		if err != nil {
			return err
		}

	}

	// Close the minified writer
	minifiedWriter.Close()

	return nil
}
