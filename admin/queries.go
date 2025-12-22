package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Website represents a managed website
type Website struct {
	ID            string    `json:"id"` // Using database name as ID
	SiteName      string    `json:"siteName"`
	Directory     string    `json:"directory"`
	DatabaseName  string    `json:"databaseName"`
	HTTPAddress   string    `json:"httpAddress"`
	MediaProxyURL string    `json:"mediaProxyUrl"`
	APIVersion    int       `json:"apiVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Article represents an article/post
type Article struct {
	ID            int       `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Content       string    `json:"content"`
	Excerpt       string    `json:"excerpt"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	ThumbnailID   int       `json:"thumbnailId"`
	PublishedDate time.Time `json:"publishedDate"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Product represents an e-commerce product
type Product struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Description       string    `json:"description"`
	Price             float64   `json:"price"`
	CompareAtPrice    float64   `json:"compareAtPrice"`
	SKU               string    `json:"sku"`
	InventoryQuantity int       `json:"inventoryQuantity"`
	InventoryPolicy   string    `json:"inventoryPolicy"`
	Status            string    `json:"status"`
	Featured          bool      `json:"featured"`
	ReleasedDate      time.Time `json:"releasedDate"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Category represents an article category
type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Collection represents a product collection
type Collection struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ImageID     int       `json:"imageId"`
	SortOrder   int       `json:"sortOrder"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Image represents an uploaded image
type Image struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	AltText   string    `json:"altText"`
	Credit    string    `json:"credit"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetAllWebsites retrieves all websites from disk
func (s *AdminServer) GetAllWebsites() ([]Website, error) {
	websites := []Website{}

	// Determine config file name based on prod mode
	configName := "config-dev.json"
	if s.EnvConfig.ProdMode {
		configName = "config-prod.json"
	}

	// Walk the websites directory
	err := filepath.Walk("websites", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for config files
		if !info.IsDir() && info.Name() == configName {
			// Read the config file
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			var config struct {
				SiteName      string `json:"siteName"`
				APIVersion    int    `json:"apiVersion"`
				Database      struct {
					Name string `json:"name"`
				} `json:"database"`
				MediaProxyURL string `json:"mediaProxyUrl"`
				HTTP          struct {
					Address string `json:"address"`
				} `json:"http"`
			}

			if err := json.Unmarshal(data, &config); err != nil {
				return nil // Skip invalid JSON
			}

			// Extract directory (parent of config file)
			dir := filepath.Dir(path)
			relDir, _ := filepath.Rel("websites", dir)

			website := Website{
				ID:            config.Database.Name, // Use database name as ID
				SiteName:      config.SiteName,
				Directory:     relDir,
				DatabaseName:  config.Database.Name,
				HTTPAddress:   config.HTTP.Address,
				MediaProxyURL: config.MediaProxyURL,
				APIVersion:    config.APIVersion,
			}

			websites = append(websites, website)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return websites, nil
}

// GetWebsite retrieves a single website by ID (which is the database name)
func (s *AdminServer) GetWebsite(id string) (Website, error) {
	websites, err := s.GetAllWebsites()
	if err != nil {
		return Website{}, err
	}

	for _, website := range websites {
		if website.ID == id {
			return website, nil
		}
	}

	return Website{}, fmt.Errorf("website not found")
}

// GetWebsiteByDatabase retrieves a single website by database name (same as GetWebsite now)
func (s *AdminServer) GetWebsiteByDatabase(dbName string) (Website, error) {
	return s.GetWebsite(dbName)
}

// CreateWebsite creates a new website on disk (no database needed)
func (s *AdminServer) CreateWebsite(w Website) (int64, error) {
	// Website creation is handled in the handler by creating files directly
	// This function just returns success
	websites, _ := s.GetAllWebsites()
	return int64(len(websites) + 1), nil
}

// UpdateWebsite updates an existing website config file
func (s *AdminServer) UpdateWebsite(w Website) error {
	// Determine config file name
	configName := "config-dev.json"
	if s.EnvConfig.ProdMode {
		configName = "config-prod.json"
	}

	configPath := filepath.Join("websites", w.Directory, configName)

	// Read existing config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update fields
	config["siteName"] = w.SiteName
	config["apiVersion"] = w.APIVersion

	if config["database"] == nil {
		config["database"] = make(map[string]interface{})
	}
	config["database"].(map[string]interface{})["name"] = w.DatabaseName

	if config["http"] == nil {
		config["http"] = make(map[string]interface{})
	}
	config["http"].(map[string]interface{})["address"] = w.HTTPAddress

	if w.MediaProxyURL != "" {
		config["mediaProxyUrl"] = w.MediaProxyURL
	}

	// Write back to disk
	updatedData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, updatedData, 0644)
}

// DeleteWebsite deletes a website directory
func (s *AdminServer) DeleteWebsite(id string) error {
	website, err := s.GetWebsite(id)
	if err != nil {
		return err
	}

	websiteDir := filepath.Join("websites", website.Directory)
	return os.RemoveAll(websiteDir)
}

// GetWebsiteConnection gets a database connection for a specific website by ID (database name)
func (s *AdminServer) GetWebsiteConnection(websiteID string) (*sql.DB, error) {
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	connectionString := s.EnvConfig.Database.User + ":" + s.EnvConfig.Database.Password +
		"@tcp(" + s.EnvConfig.Database.Host + ":" + s.EnvConfig.Database.Port + ")/" +
		website.DatabaseName + "?parseTime=true"

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// GetWebsiteConnectionByDB gets a database connection for a specific website by database name
func (s *AdminServer) GetWebsiteConnectionByDB(dbName string) (*sql.DB, error) {
	connectionString := s.EnvConfig.Database.User + ":" + s.EnvConfig.Database.Password +
		"@tcp(" + s.EnvConfig.Database.Host + ":" + s.EnvConfig.Database.Port + ")/" +
		dbName + "?parseTime=true"

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// GetArticles retrieves articles for a specific website
func (s *AdminServer) GetArticles(websiteID string, limit, offset int) ([]Article, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, slug, title, description, content, excerpt, type, status, thumbnail_id, published_date, created_at, updated_at
		FROM articles_unified ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var a Article
		var publishedDate sql.NullTime
		var thumbnailID sql.NullInt64
		err := rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Description, &a.Content, &a.Excerpt, &a.Type, &a.Status, &thumbnailID, &publishedDate, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if publishedDate.Valid {
			a.PublishedDate = publishedDate.Time
		}
		if thumbnailID.Valid {
			a.ThumbnailID = int(thumbnailID.Int64)
		}
		articles = append(articles, a)
	}

	return articles, nil
}

// GetArticle retrieves a single article
func (s *AdminServer) GetArticle(websiteID string, articleID int) (Article, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Article{}, err
	}
	defer db.Close()

	query := `SELECT id, slug, title, description, content, excerpt, type, status, thumbnail_id, published_date, created_at, updated_at
		FROM articles_unified WHERE id = ?`

	var a Article
	var publishedDate sql.NullTime
	var thumbnailID sql.NullInt64
	err = db.QueryRow(query, articleID).Scan(&a.ID, &a.Slug, &a.Title, &a.Description, &a.Content, &a.Excerpt, &a.Type, &a.Status, &thumbnailID, &publishedDate, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Article{}, err
	}
	if publishedDate.Valid {
		a.PublishedDate = publishedDate.Time
	}
	if thumbnailID.Valid {
		a.ThumbnailID = int(thumbnailID.Int64)
	}

	return a, nil
}

// CreateArticle creates a new article
func (s *AdminServer) CreateArticle(websiteID string, a Article) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO articles_unified (slug, title, description, content, excerpt, type, status, thumbnail_id, published_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var publishedDate interface{}
	if a.PublishedDate.IsZero() {
		publishedDate = nil
	} else {
		publishedDate = a.PublishedDate
	}

	var thumbnailID interface{}
	if a.ThumbnailID == 0 {
		thumbnailID = nil
	} else {
		thumbnailID = a.ThumbnailID
	}

	result, err := db.Exec(query, a.Slug, a.Title, a.Description, a.Content, a.Excerpt, a.Type, a.Status, thumbnailID, publishedDate)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateArticle updates an existing article
func (s *AdminServer) UpdateArticle(websiteID string, a Article) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE articles_unified SET slug = ?, title = ?, description = ?, content = ?, excerpt = ?, type = ?, status = ?, thumbnail_id = ?, published_date = ?
		WHERE id = ?`

	var publishedDate interface{}
	if a.PublishedDate.IsZero() {
		publishedDate = nil
	} else {
		publishedDate = a.PublishedDate
	}

	var thumbnailID interface{}
	if a.ThumbnailID == 0 {
		thumbnailID = nil
	} else {
		thumbnailID = a.ThumbnailID
	}

	_, err = db.Exec(query, a.Slug, a.Title, a.Description, a.Content, a.Excerpt, a.Type, a.Status, thumbnailID, publishedDate, a.ID)
	return err
}

// DeleteArticle deletes an article
func (s *AdminServer) DeleteArticle(websiteID string, articleID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM articles_unified WHERE id = ?`
	_, err = db.Exec(query, articleID)
	return err
}

// GetProducts retrieves products for a specific website
func (s *AdminServer) GetProducts(websiteID string, limit, offset int) ([]Product, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, created_at, updated_at
		FROM products_unified ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// GetProduct retrieves a single product
func (s *AdminServer) GetProduct(websiteID string, productID int) (Product, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Product{}, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, released_date, created_at, updated_at
		FROM products_unified WHERE id = ?`

	var p Product
	var releasedDate sql.NullTime
	err = db.QueryRow(query, productID).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &releasedDate, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Product{}, err
	}
	if releasedDate.Valid {
		p.ReleasedDate = releasedDate.Time
	}

	return p, nil
}

// CreateProduct creates a new product
func (s *AdminServer) CreateProduct(websiteID string, p Product) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO products_unified (name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, released_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	result, err := db.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, releasedDate)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateProduct updates an existing product
func (s *AdminServer) UpdateProduct(websiteID string, p Product) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE products_unified SET name = ?, slug = ?, description = ?, price = ?, compare_at_price = ?, sku = ?, inventory_quantity = ?, inventory_policy = ?, status = ?, featured = ?, released_date = ?
		WHERE id = ?`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	_, err = db.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, releasedDate, p.ID)
	return err
}

// DeleteProduct deletes a product
func (s *AdminServer) DeleteProduct(websiteID string, productID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM products_unified WHERE id = ?`
	_, err = db.Exec(query, productID)
	return err
}

// GetCategories retrieves categories for a specific website
func (s *AdminServer) GetCategories(websiteID string) ([]Category, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, count, created_at, updated_at FROM categories_unified ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Count, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// CreateCategory creates a new category
func (s *AdminServer) CreateCategory(websiteID string, c Category) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO categories_unified (name, slug) VALUES (?, ?)`
	result, err := db.Exec(query, c.Name, c.Slug)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteCategory deletes a category
func (s *AdminServer) DeleteCategory(websiteID string, categoryID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM categories_unified WHERE id = ?`
	_, err = db.Exec(query, categoryID)
	return err
}

// GetCollections retrieves collections for a specific website
func (s *AdminServer) GetCollections(websiteID string) ([]Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, image_id, sort_order, status, created_at, updated_at
		FROM collections_unified ORDER BY sort_order, name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collections := []Collection{}
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.ImageID, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}

	return collections, nil
}

// CreateCollection creates a new collection at the top and pushes others down
func (s *AdminServer) CreateCollection(websiteID string, c Collection) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Increment all existing collections' sort_order by 1
	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = sort_order + 1`)
	if err != nil {
		return 0, err
	}

	// Insert new collection with sort_order = 0 (top position)
	query := `INSERT INTO collections_unified (name, slug, description, image_id, sort_order, status)
		VALUES (?, ?, ?, ?, 0, ?)`

	result, err := tx.Exec(query, c.Name, c.Slug, c.Description, c.ImageID, c.Status)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetCollection retrieves a single collection by ID
func (s *AdminServer) GetCollection(websiteID string, collectionID int) (Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Collection{}, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, image_id, sort_order, status, created_at, updated_at
		FROM collections_unified WHERE id = ?`

	var c Collection
	var imageID sql.NullInt64
	err = db.QueryRow(query, collectionID).Scan(
		&c.ID, &c.Name, &c.Slug, &c.Description, &imageID, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return Collection{}, err
	}

	if imageID.Valid {
		c.ImageID = int(imageID.Int64)
	}

	return c, nil
}

// UpdateCollection updates an existing collection
func (s *AdminServer) UpdateCollection(websiteID string, c Collection) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE collections_unified SET name = ?, slug = ?, description = ?, image_id = ?, sort_order = ?, status = ?
		WHERE id = ?`

	_, err = db.Exec(query, c.Name, c.Slug, c.Description, c.ImageID, c.SortOrder, c.Status, c.ID)
	return err
}

// DeleteCollection deletes a collection and renumbers the remaining ones
func (s *AdminServer) DeleteCollection(websiteID string, collectionID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete the collection
	_, err = tx.Exec(`DELETE FROM collections_unified WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	// Renumber all collections sequentially starting from 0
	// Get all collections ordered by current sort_order
	rows, err := tx.Query(`SELECT id FROM collections_unified ORDER BY sort_order, name`)
	if err != nil {
		return err
	}

	var collectionIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		collectionIDs = append(collectionIDs, id)
	}
	rows.Close()

	// Update each collection with sequential sort_order
	for i, id := range collectionIDs {
		_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, i, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ReorderCollection moves a collection up or down in sort order
func (s *AdminServer) ReorderCollection(websiteID string, collectionID int, direction string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get current collection's sort_order
	var currentSortOrder int
	err = db.QueryRow(`SELECT sort_order FROM collections_unified WHERE id = ?`, collectionID).Scan(&currentSortOrder)
	if err != nil {
		return err
	}

	var targetSortOrder int
	var targetID int
	if direction == "up" {
		// Find the collection with the next lower sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM collections_unified WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	} else if direction == "down" {
		// Find the collection with the next higher sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM collections_unified WHERE sort_order > ? ORDER BY sort_order ASC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	}

	if err == sql.ErrNoRows {
		// Already at top/bottom, nothing to do - just return success
		return nil
	}
	if err != nil {
		return err
	}

	// Swap sort_order values
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Swap the two collections' sort_order values
	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, targetSortOrder, collectionID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, currentSortOrder, targetID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetImages retrieves images for a specific website
func (s *AdminServer) GetImages(websiteID string, limit, offset int) ([]Image, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, url, alt_text, credit, filename, size, width, height, created_at
		FROM images_unified ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := []Image{}
	for rows.Next() {
		var img Image
		var filename, altText, credit sql.NullString
		var size sql.NullInt64
		var width, height sql.NullInt64
		err := rows.Scan(&img.ID, &img.URL, &altText, &credit, &filename, &size, &width, &height, &img.CreatedAt)
		if err != nil {
			return nil, err
		}
		if altText.Valid {
			img.AltText = altText.String
		}
		if credit.Valid {
			img.Credit = credit.String
		}
		if filename.Valid {
			img.Filename = filename.String
		}
		if size.Valid {
			img.Size = size.Int64
		}
		if width.Valid {
			img.Width = int(width.Int64)
		}
		if height.Valid {
			img.Height = int(height.Int64)
		}
		images = append(images, img)
	}

	return images, nil
}

// CreateImage creates a new image entry
func (s *AdminServer) CreateImage(websiteID string, img Image) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO images_unified (url, alt_text, credit, filename, size, width, height) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(query, img.URL, img.AltText, img.Credit, img.Filename, img.Size, img.Width, img.Height)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteImage deletes an image
func (s *AdminServer) DeleteImage(websiteID string, imageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM images_unified WHERE id = ?`
	_, err = db.Exec(query, imageID)
	return err
}

// LogActivity logs an admin action (currently just to stdout, no DB needed)
func (s *AdminServer) LogActivity(action, entityType string, entityID int, websiteID string, details interface{}) error {
	// Just log to stdout for now - could write to a file later if needed
	fmt.Printf("[ADMIN] %s %s (ID: %d, Website: %s)\n", action, entityType, entityID, websiteID)
	return nil
}

// ====================
// Product Images
// ====================

// ProductImage represents the product-image relationship
type ProductImage struct {
	ID       int    `json:"id"`
	ImageID  int    `json:"imageId"`
	Position int    `json:"position"`
	Image    Image  `json:"image"`
}

// GetProductImages retrieves all images for a product
func (s *AdminServer) GetProductImages(websiteID string, productID int) ([]ProductImage, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT pi.id, pi.image_id, pi.position,
		       i.id, i.url, i.alt_text, i.credit, i.filename, i.size
		FROM product_images pi
		JOIN images_unified i ON pi.image_id = i.id
		WHERE pi.product_id = ?
		ORDER BY pi.position ASC
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ProductImage
	for rows.Next() {
		var pi ProductImage
		var altText, credit, filename sql.NullString
		var size sql.NullInt64
		err := rows.Scan(&pi.ID, &pi.ImageID, &pi.Position,
			&pi.Image.ID, &pi.Image.URL, &altText, &credit, &filename, &size)
		if err != nil {
			return nil, err
		}
		if altText.Valid {
			pi.Image.AltText = altText.String
		}
		if credit.Valid {
			pi.Image.Credit = credit.String
		}
		if filename.Valid {
			pi.Image.Filename = filename.String
		}
		if size.Valid {
			pi.Image.Size = size.Int64
		}
		images = append(images, pi)
	}

	return images, nil
}

// AddProductImage adds an image to a product
func (s *AdminServer) AddProductImage(websiteID string, productID, imageID, position int) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO product_images (product_id, image_id, position) VALUES (?, ?, ?)`
	result, err := db.Exec(query, productID, imageID, position)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// RemoveProductImage removes an image from a product
func (s *AdminServer) RemoveProductImage(websiteID string, productImageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM product_images WHERE id = ?`
	_, err = db.Exec(query, productImageID)
	return err
}

// ClearProductImages removes all images from a product
func (s *AdminServer) ClearProductImages(websiteID string, productID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM product_images WHERE product_id = ?`
	_, err = db.Exec(query, productID)
	return err
}

// ====================
// Product-Collection Relationships
// ====================

// GetProductCollections retrieves all collections for a product
func (s *AdminServer) GetProductCollections(websiteID string, productID int) ([]Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT c.id, c.name, c.slug, c.status, c.sort_order
		FROM collections_unified c
		INNER JOIN product_collections pc ON c.id = pc.collection_id
		WHERE pc.product_id = ?
		ORDER BY pc.position
	`
	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Status, &c.SortOrder)
		if err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}

	return collections, nil
}

// SetProductCollections replaces all collections for a product
func (s *AdminServer) SetProductCollections(websiteID string, productID int, collectionIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing relationships
	_, err = tx.Exec("DELETE FROM product_collections WHERE product_id = ?", productID)
	if err != nil {
		return err
	}

	// Insert new relationships
	stmt, err := tx.Prepare("INSERT INTO product_collections (product_id, collection_id, position) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for position, collectionID := range collectionIDs {
		_, err = stmt.Exec(productID, collectionID, position)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ====================
// Article-Category Relationships
// ====================

// GetArticleCategories retrieves all categories for an article
func (s *AdminServer) GetArticleCategories(websiteID string, articleID int) ([]Category, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT c.id, c.name, c.slug, c.count
		FROM categories_unified c
		INNER JOIN article_categories ac ON c.id = ac.category_id
		WHERE ac.post_id = ?
		ORDER BY c.name
	`
	rows, err := db.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Count)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// SetArticleCategories replaces all categories for an article
func (s *AdminServer) SetArticleCategories(websiteID string, articleID int, categoryIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get old category IDs before we delete them
	oldCategoryIDsQuery := "SELECT category_id FROM article_categories WHERE post_id = ?"
	rows, err := db.Query(oldCategoryIDsQuery, articleID)
	if err != nil {
		return err
	}

	var oldCategoryIDs []int
	for rows.Next() {
		var catID int
		if err := rows.Scan(&catID); err != nil {
			rows.Close()
			return err
		}
		oldCategoryIDs = append(oldCategoryIDs, catID)
	}
	rows.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing relationships
	_, err = tx.Exec("DELETE FROM article_categories WHERE post_id = ?", articleID)
	if err != nil {
		return err
	}

	// Insert new relationships
	stmt, err := tx.Prepare("INSERT INTO article_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, categoryID := range categoryIDs {
		_, err = stmt.Exec(articleID, categoryID)
		if err != nil {
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Update counts for all affected categories (old and new)
	affectedCategories := make(map[int]bool)
	for _, id := range oldCategoryIDs {
		affectedCategories[id] = true
	}
	for _, id := range categoryIDs {
		affectedCategories[id] = true
	}

	// Update count for each affected category
	for catID := range affectedCategories {
		if err := s.UpdateCategoryCount(websiteID, catID); err != nil {
			// Log the error but don't fail the whole operation
			log.Printf("Error updating category count for category %d: %v", catID, err)
		}
	}

	return nil
}


// UpdateCategoryCount recalculates the post count for a category
func (s *AdminServer) UpdateCategoryCount(websiteID string, categoryID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
		UPDATE categories_unified
		SET count = (
			SELECT COUNT(*)
			FROM article_categories
			WHERE category_id = ?
		)
		WHERE id = ?
	`
	_, err = db.Exec(query, categoryID, categoryID)
	return err
}
