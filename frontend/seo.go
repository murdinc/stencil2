package frontend

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/structs"
)

// GenerateArticleSchema generates schema.org Article structured data
func GenerateArticleSchema(post structs.Post, siteConfig configs.WebsiteConfig) string {
	if post.Slug == "" {
		return ""
	}

	baseURL := "https://" + siteConfig.SiteName

	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Article",
		"headline": post.Title,
	}

	if post.Description != "" {
		schema["description"] = post.Description
	}

	if post.Image.URL != "" {
		schema["image"] = post.Image.URL
	}

	// Set article URL
	articleURL := baseURL + "/" + post.Slug
	if post.URL != "" {
		articleURL = baseURL + post.URL
	}
	schema["url"] = articleURL

	// Add canonical URL if available
	if post.CanonicalURL != "" {
		schema["mainEntityOfPage"] = post.CanonicalURL
	} else {
		schema["mainEntityOfPage"] = articleURL
	}

	// Add dates
	if !post.PublishedDate.IsZero() {
		schema["datePublished"] = post.PublishedDate.Format("2006-01-02T15:04:05-07:00")
	}
	if !post.Modified.IsZero() {
		schema["dateModified"] = post.Modified.Format("2006-01-02T15:04:05-07:00")
	} else if !post.Updated.IsZero() {
		schema["dateModified"] = post.Updated.Format("2006-01-02T15:04:05-07:00")
	}

	// Add author (using site name as publisher)
	schema["author"] = map[string]interface{}{
		"@type": "Organization",
		"name":  siteConfig.SiteName,
	}

	schema["publisher"] = map[string]interface{}{
		"@type": "Organization",
		"name":  siteConfig.SiteName,
	}

	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// GenerateProductSchema generates schema.org Product structured data
func GenerateProductSchema(product structs.Product, siteConfig configs.WebsiteConfig) string {
	if product.Slug == "" {
		return ""
	}

	baseURL := "https://" + siteConfig.SiteName

	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Product",
		"name":     product.Name,
		"url":      baseURL + "/products/" + product.Slug,
	}

	if product.Description != "" {
		schema["description"] = product.Description
	}

	if product.SKU != "" {
		schema["sku"] = product.SKU
	}

	// Add image
	if len(product.Images) > 0 && product.Images[0].Image.URL != "" {
		schema["image"] = product.Images[0].Image.URL
	}

	// Add offers (price and availability)
	offer := map[string]interface{}{
		"@type": "Offer",
		"url":   baseURL + "/products/" + product.Slug,
	}

	if product.Price > 0 {
		offer["price"] = fmt.Sprintf("%.2f", product.Price)
		offer["priceCurrency"] = "USD"
	}

	// Set availability based on inventory
	availability := "https://schema.org/OutOfStock"
	if product.InventoryQuantity > 0 {
		availability = "https://schema.org/InStock"
	} else if product.InventoryPolicy == "continue" {
		availability = "https://schema.org/PreOrder"
	}
	offer["availability"] = availability

	schema["offers"] = offer

	// Add brand (using site name)
	schema["brand"] = map[string]interface{}{
		"@type": "Brand",
		"name":  siteConfig.SiteName,
	}

	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// GenerateCollectionSchema generates schema.org ItemList structured data for collections
func GenerateCollectionSchema(collection structs.Collection, products []structs.Product, siteConfig configs.WebsiteConfig) string {
	if collection.Slug == "" {
		return ""
	}

	baseURL := "https://" + siteConfig.SiteName

	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "ItemList",
		"name":     collection.Name,
		"url":      baseURL + "/collections/" + collection.Slug,
	}

	if collection.Description != "" {
		schema["description"] = collection.Description
	}

	// Add collection items (products)
	var itemListElements []map[string]interface{}
	for i, product := range products {
		if product.Slug == "" {
			continue
		}

		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"item": map[string]interface{}{
				"@type": "Product",
				"name":  product.Name,
				"url":   baseURL + "/products/" + product.Slug,
			},
		}

		if len(product.Images) > 0 && product.Images[0].Image.URL != "" {
			item["item"].(map[string]interface{})["image"] = product.Images[0].Image.URL
		}

		itemListElements = append(itemListElements, item)
	}

	if len(itemListElements) > 0 {
		schema["itemListElement"] = itemListElements
		schema["numberOfItems"] = len(itemListElements)
	}

	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// GenerateBreadcrumbSchema generates schema.org BreadcrumbList structured data
func GenerateBreadcrumbSchema(breadcrumbs []map[string]string, siteConfig configs.WebsiteConfig) string {
	if len(breadcrumbs) == 0 {
		return ""
	}

	baseURL := "https://" + siteConfig.SiteName

	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "BreadcrumbList",
	}

	var itemListElements []map[string]interface{}
	for i, crumb := range breadcrumbs {
		name, nameOk := crumb["name"]
		path, pathOk := crumb["path"]

		if !nameOk || !pathOk {
			continue
		}

		url := baseURL + path
		if strings.HasPrefix(path, "http") {
			url = path
		}

		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     name,
			"item":     url,
		}

		itemListElements = append(itemListElements, item)
	}

	if len(itemListElements) > 0 {
		schema["itemListElement"] = itemListElements
	}

	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}
