package frontend

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/structs"
)

type SEOData struct {
	Title          string // Page title for <title> and og:title
	Description    string // Meta description and og:description
	Keywords       string // Meta keywords (optional, for legacy)
	Canonical      string // Canonical URL (full absolute URL)
	Image          string // og:image and twitter:image URL
	Type           string // og:type: "website", "article", "product"
	StructuredData string // JSON-LD structured data for rich snippets
}

type PageData struct {
	ProdMode         bool
	HideErrors       bool
	Slug             string
	Page             string
	Category         structs.Category
	Categories       []structs.Category
	Post             structs.Post
	Posts            []structs.Post
	Product          structs.Product
	Products         []structs.Product
	Collection       structs.Collection
	Collections      []structs.Collection
	ErrorString      string
	StatusCode       int
	Template         configs.TemplateConfig
	Preview          bool
	ErrorDescription string
	CartItemCount    int
	Error            string
	SEO              SEOData // SEO metadata for meta tags and Open Graph
}

func (website *Website) ExecuteTemplate(w http.ResponseWriter, tpl configs.TemplateConfig, pageData PageData) {

	pageData.Template = tpl

	// get the error template if we have anything other than a 200 status // TODO?
	if pageData.StatusCode != 200 {
		website.RenderError(w, pageData)
		return
	}

	funcMap := sprig.TxtFuncMap()
	funcMap["sitename"] = func() string {
		return website.WebsiteConfig.SiteName
	}
	funcMap["mediaproxyurl"] = func() string {
		if website.WebsiteConfig.MediaProxyURL != "" {
			return website.WebsiteConfig.MediaProxyURL
		}
		return fmt.Sprintf("//%s/media-proxy", website.WebsiteConfig.SiteName)
	}
	funcMap["mediaproxy"] = func(width int, url string) string {
		if website.WebsiteConfig.MediaProxyURL != "" && url != "" && width > 0 {
			return fmt.Sprintf("%s/width/%d?url=%s", website.WebsiteConfig.MediaProxyURL, width, url)
		}
		return fmt.Sprintf("//%s/media-proxy/width/%d?url=%s", website.WebsiteConfig.SiteName, width, url)
	}
	funcMap["hash"] = func() string {
		return website.Hash
	}

	// Load the template file
	tplName := fmt.Sprintf("%s.tpl", tpl.Name)
	tplPath := fmt.Sprintf("%s/%s", tpl.Directory, tplName)
	tmpl, err := template.New(tplName).Funcs(funcMap).ParseFiles(tplPath)

	if err != nil {
		if tpl.Name != "error" {
			pageData.ErrorString = err.Error()
			pageData.StatusCode = 500
			website.RenderError(w, pageData)
			return
		}
		// TODO edge case or never? ^^
		log.Println(err)
		return
	}

	// Load any required template files
	if len(tpl.Requires) > 0 {
		var requiredFiles []string
		for _, required := range tpl.Requires {

			walkFn := func(path string, fileInfo os.FileInfo, inErr error) (err error) {
				if inErr == nil && !fileInfo.IsDir() && strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".tpl") {
					requiredFiles = append(requiredFiles, path)
				}
				return
			}

			reqTpl := website.GetTemplate(required)
			err = filepath.Walk(reqTpl.Directory, walkFn)
			if err != nil {
				pageData.ErrorString = err.Error()
				pageData.StatusCode = 500
				website.RenderError(w, pageData)
				return
			}
		}

		// Parse the required templates
		if len(requiredFiles) > 0 {
			tmpl, err = tmpl.ParseFiles(requiredFiles...)
			if err != nil {
				pageData.ErrorString = err.Error()
				pageData.StatusCode = 500
				website.RenderError(w, pageData)
				return
			}
		}
	}

	// Execute the template with the pageData into a buffer
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, pageData)
	if err != nil {
		if tpl.Name != "error" {
			// switch to the error template
			pageData.ErrorString = err.Error()
			pageData.StatusCode = 502
			website.RenderError(w, pageData)
			return
		}
	}

	if tpl.MimeType != "" {
		w.Header().Set("Content-Type", tpl.MimeType)
	} else {
		w.Header().Set("Content-Type", "text/html")
	}

	// If template execution is successful, write buffer contents to the http.ResponseWriter
	if tpl.NoCache {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate;")
		w.Header().Set("pragma", "no-cache")
	} else {
		// Set up caching based on the page type
		expiration := 2592000
		if tpl.CacheTime > 0 {
			expiration = tpl.CacheTime
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("public, s-maxage=%d, max-age=0", expiration))
	}

	w.WriteHeader(pageData.StatusCode)
	_, err = buffer.WriteTo(w)
	if err != nil {
		// TODO edge case or never? ^^
		log.Println(err)
	}
}

func (website *Website) RenderError(w http.ResponseWriter, pageData PageData) {

	funcMap := sprig.TxtFuncMap()
	funcMap["sitename"] = func() string {
		return website.WebsiteConfig.SiteName
	}
	funcMap["hash"] = func() string {
		return website.Hash
	}

	if pageData.ProdMode == false && pageData.HideErrors == false {
		tmpl := template.New("devError").Funcs(funcMap)
		tmpl, _ = tmpl.Parse(devErrTemplate)
		tmpl.Execute(w, pageData)
		return
	}

	// Load the template file
	tpl := website.GetTemplate("error")
	tplName := fmt.Sprintf("%s.tpl", tpl.Name)
	tplPath := fmt.Sprintf("%s/%s", tpl.Directory, tplName)
	tmpl, err := template.New(tplName).Funcs(funcMap).ParseFiles(tplPath)

	if err != nil {
		// TODO handle this type of error?
		log.Println(err.Error())
	}

	w.WriteHeader(pageData.StatusCode)
	tmpl.Execute(w, pageData)
}

var devErrTemplate = `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Error {{ .StatusCode }} - {{ .ErrorString }}</title>
		<style>
			body {
				font-family: 'Helvetica', sans-serif;
				color: #ffffff;
				background-color: #101010;
				margin: 0;
				display: flex;
				flex-direction: column;
				min-height: 100vh;
			}

			.content {
				flex: 1;
				padding: 20px;
			}

			.error-container {
				position: fixed;
				top: 0;
				left: 0;
				width: 100%;
				height: 100%;
				display: flex;
				flex-direction: column;
				justify-content: center;
				align-items: center;
				text-align: center;
				z-index: 9999;
				background-color: rgba(0, 0, 0, 0.8);
			}

			.error-code {
				font-size: 6rem;
				font-weight: bold;
				letter-spacing: -3px;
				margin-bottom: 15px;
				color: #ffffff;
				text-shadow: 0 3px 5px rgba(0, 0, 0, 0.2);
			}

			.error-message {
				font-size: 2rem;
				font-weight: bold;
				text-transform: uppercase;
				margin: 50px;
				color: #ffffff;
				text-shadow: 0 2px 3px rgba(0, 0, 0, 0.1);
			}

			.error-description {
				font-size: 1.2rem;
				margin-bottom: 20px;
				color: #ffffff;
				text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
			}

		</style>
	</head>
	<body>
		<div class="error-container">
			<div class="error-code">ERROR {{ .StatusCode }}</div>
			<div class="error-message">{{ .ErrorString }}</div>
			<div class="error-description">{{ .ErrorDescription }}</div>
		</div>
	</body>
	</html>`

func (website *Website) GetTemplate(name string) configs.TemplateConfig {
	for _, template := range *website.TemplateConfigs {
		if template.Name == name {
			return template
		}
	}
	return configs.TemplateConfig{}
}
