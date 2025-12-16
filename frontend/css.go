package frontend

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

type CSSFiles map[string][]string

func (website *Website) LoadCSS(outputFile string) (CSSFiles, error) {

	log.Printf("Building CSS files for: [%s]", website.WebsiteConfig.SiteName)
	cssMap := make(map[string][]string)

	for _, tpl := range *website.TemplateConfigs {

		// skip if there is no specified output file
		if tpl.CSSFile == "" {
			continue
		}

		if outputFile != "" && tpl.CSSFile != outputFile {
			continue
		}

		// Crawl the folders of tpl.Requires first
		for _, required := range tpl.Requires {
			requiredDir := filepath.Join(tpl.Directory, required)

			reqTpl := website.GetTemplate(required)
			err := crawlCSSFiles(reqTpl.Directory, tpl.CSSFile, cssMap)
			if err != nil {
				return cssMap, fmt.Errorf("failed to crawl CSS files for %s: %v", requiredDir, err)
			}
		}

		// Then search for .css files in tpl.Directory
		err := crawlCSSFiles(tpl.Directory, tpl.CSSFile, cssMap)
		if err != nil {
			return cssMap, fmt.Errorf("failed to crawl CSS files for %s: %v", tpl.Directory, err)
		}
	}

	for cssFile, filePaths := range cssMap {

		filePaths = removeDuplicateString(filePaths)

		destinationCSSFolder := fmt.Sprintf("%s/public/css/", website.WebsiteConfig.Directory)
		err := MinifyAndCombineCSS(filePaths, destinationCSSFolder, cssFile)
		if err != nil {
			log.Println(err.Error())
		}

		fmt.Printf("			Building CSS file: [%s]\n", cssFile)

		for _, filePath := range filePaths {
			fmt.Printf("			> Including: %s\n", filePath)
		}
	}

	log.Printf("Done Building CSS files for: [%s]", website.WebsiteConfig.SiteName)

	return cssMap, nil
}

func crawlCSSFiles(dir string, destinationCSSFile string, cssMap CSSFiles) error {
	var cssFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access directory: %v", err)
		}

		if !info.IsDir() && strings.HasSuffix(path, ".css") {
			cssFiles = append(cssFiles, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(cssFiles) > 0 {
		cssMap[destinationCSSFile] = append(cssMap[destinationCSSFile], cssFiles...)
	}

	return nil
}

func MinifyAndCombineCSS(requiredCSSFiles []string, destinationCSSFolder string, destinationCSSFile string) error {

	// Bail if we dont have any files
	if len(requiredCSSFiles) < 1 {
		return nil
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	// Create css folder if it doesn't exist
	if _, err := os.Stat(destinationCSSFolder); os.IsNotExist(err) {
		err = os.MkdirAll(destinationCSSFolder, 0755)
		if err != nil {
			return err
		}
	}

	outputFile := destinationCSSFolder + destinationCSSFile

	// Open outputFile to write to it.
	fo, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer fo.Close()

	// Create a minifier
	minifiedWriter := m.Writer("text/css", fo)

	for _, file := range requiredCSSFiles {
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
