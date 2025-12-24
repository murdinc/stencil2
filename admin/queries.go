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

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/shippo"
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

	// Stripe
	StripePublishableKey string `json:"stripePublishableKey"`
	StripeSecretKey      string `json:"stripeSecretKey"`

	// Shippo
	ShippoAPIKey  string `json:"shippoApiKey"`
	LabelFormat   string `json:"labelFormat"` // PDF, PDF_4x6, ZPLII, PNG

	// Email
	EmailFromAddress string `json:"emailFromAddress"`
	EmailFromName    string `json:"emailFromName"`
	EmailReplyTo     string `json:"emailReplyTo"`

	// Ecommerce
	TaxRate      float64 `json:"taxRate"`
	ShippingCost float64 `json:"shippingCost"`

	// Early Access
	EarlyAccessEnabled  bool   `json:"earlyAccessEnabled"`
	EarlyAccessPassword string `json:"earlyAccessPassword"`

	// ShipFrom Address
	ShipFromName    string `json:"shipFromName"`
	ShipFromStreet1 string `json:"shipFromStreet1"`
	ShipFromStreet2 string `json:"shipFromStreet2"`
	ShipFromCity    string `json:"shipFromCity"`
	ShipFromState   string `json:"shipFromState"`
	ShipFromZip     string `json:"shipFromZip"`
	ShipFromCountry string `json:"shipFromCountry"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	SortOrder         int       `json:"sortOrder"`
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

// Order represents a customer order
type Order struct {
	ID                   int       `json:"id"`
	OrderNumber          string    `json:"orderNumber"`
	CustomerEmail        string    `json:"customerEmail"`
	CustomerName         string    `json:"customerName"`
	ShippingAddressLine1 string    `json:"shippingAddressLine1"`
	ShippingAddressLine2 string    `json:"shippingAddressLine2"`
	ShippingCity         string    `json:"shippingCity"`
	ShippingState        string    `json:"shippingState"`
	ShippingZip          string    `json:"shippingZip"`
	ShippingCountry      string    `json:"shippingCountry"`
	Subtotal             float64   `json:"subtotal"`
	Tax                  float64   `json:"tax"`
	ShippingCost         float64   `json:"shippingCost"`
	Total                float64   `json:"total"`
	PaymentStatus        string    `json:"paymentStatus"`
	FulfillmentStatus    string    `json:"fulfillmentStatus"`
	PaymentMethod        string    `json:"paymentMethod"`
	ShippingLabelCost    *float64  `json:"shippingLabelCost"`
	TrackingNumber       string    `json:"trackingNumber"`
	ShippingCarrier      string    `json:"shippingCarrier"`
	ShippingLabelURL     string    `json:"shippingLabelUrl"`
	Items                []OrderItem `json:"items"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// OrderItem represents a line item in an order
type OrderItem struct {
	ID           int     `json:"id"`
	ProductID    int     `json:"productId"`
	ProductName  string  `json:"productName"`
	VariantTitle string  `json:"variantTitle"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
	Total        float64 `json:"total"`
}

// ProductImageData represents a product-specific image (not shared with articles)
type ProductImageData struct {
	ID        int       `json:"id"`
	ProductID int       `json:"productId"`
	URL       string    `json:"url"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	AltText   string    `json:"altText"`
	Credit    string    `json:"credit"`
	Size      int64     `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
}

// OrderFilters represents filters for order queries
type OrderFilters struct {
	PaymentStatus     string
	FulfillmentStatus string
	Sort              string
}

// CustomerFilters represents filters for customer queries
type CustomerFilters struct {
	Sort string // total_desc, total_asc, orders_desc, date_desc, date_asc
}

// Customer represents a customer with aggregate statistics
type Customer struct {
	ID               int        `json:"id"`
	Email            string     `json:"email"`
	StripeCustomerID string     `json:"stripeCustomerId"`
	FirstName        string     `json:"firstName"`
	LastName         string     `json:"lastName"`
	Phone            string     `json:"phone"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	// Aggregate fields
	OrderCount int        `json:"orderCount"`
	TotalSpent float64    `json:"totalSpent"`
	FirstOrder *time.Time `json:"firstOrderDate"`
	LastOrder  *time.Time `json:"lastOrderDate"`
}

type SMSSignup struct {
	ID        int       `json:"id"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
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
				Stripe struct {
					PublishableKey string `json:"publishableKey"`
					SecretKey      string `json:"secretKey"`
				} `json:"stripe"`
				Shippo struct {
					APIKey      string `json:"apiKey"`
					LabelFormat string `json:"labelFormat"`
				} `json:"shippo"`
				Email struct {
					FromAddress string `json:"fromAddress"`
					FromName    string `json:"fromName"`
					ReplyTo     string `json:"replyTo"`
				} `json:"email"`
				Ecommerce struct {
					TaxRate      float64 `json:"taxRate"`
					ShippingCost float64 `json:"shippingCost"`
				} `json:"ecommerce"`
				EarlyAccess struct {
					Enabled  bool   `json:"enabled"`
					Password string `json:"password"`
				} `json:"earlyAccess"`
				ShipFrom struct {
					Name    string `json:"name"`
					Street1 string `json:"street1"`
					Street2 string `json:"street2"`
					City    string `json:"city"`
					State   string `json:"state"`
					Zip     string `json:"zip"`
					Country string `json:"country"`
				} `json:"shipFrom"`
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

				StripePublishableKey: config.Stripe.PublishableKey,
				StripeSecretKey:      config.Stripe.SecretKey,

				ShippoAPIKey: config.Shippo.APIKey,
				LabelFormat:  config.Shippo.LabelFormat,

				EmailFromAddress: config.Email.FromAddress,
				EmailFromName:    config.Email.FromName,
				EmailReplyTo:     config.Email.ReplyTo,

				TaxRate:      config.Ecommerce.TaxRate,
				ShippingCost: config.Ecommerce.ShippingCost,

				EarlyAccessEnabled:  config.EarlyAccess.Enabled,
				EarlyAccessPassword: config.EarlyAccess.Password,

				ShipFromName:    config.ShipFrom.Name,
				ShipFromStreet1: config.ShipFrom.Street1,
				ShipFromStreet2: config.ShipFrom.Street2,
				ShipFromCity:    config.ShipFrom.City,
				ShipFromState:   config.ShipFrom.State,
				ShipFromZip:     config.ShipFrom.Zip,
				ShipFromCountry: config.ShipFrom.Country,
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

	// Stripe
	if config["stripe"] == nil {
		config["stripe"] = make(map[string]interface{})
	}
	config["stripe"].(map[string]interface{})["publishableKey"] = w.StripePublishableKey
	config["stripe"].(map[string]interface{})["secretKey"] = w.StripeSecretKey

	// Shippo
	if config["shippo"] == nil {
		config["shippo"] = make(map[string]interface{})
	}
	config["shippo"].(map[string]interface{})["apiKey"] = w.ShippoAPIKey
	if w.LabelFormat != "" {
		config["shippo"].(map[string]interface{})["labelFormat"] = w.LabelFormat
	}

	// Email
	if config["email"] == nil {
		config["email"] = make(map[string]interface{})
	}
	config["email"].(map[string]interface{})["fromAddress"] = w.EmailFromAddress
	config["email"].(map[string]interface{})["fromName"] = w.EmailFromName
	config["email"].(map[string]interface{})["replyTo"] = w.EmailReplyTo

	// Ecommerce
	if config["ecommerce"] == nil {
		config["ecommerce"] = make(map[string]interface{})
	}
	config["ecommerce"].(map[string]interface{})["taxRate"] = w.TaxRate
	config["ecommerce"].(map[string]interface{})["shippingCost"] = w.ShippingCost

	// Early Access
	if config["earlyAccess"] == nil {
		config["earlyAccess"] = make(map[string]interface{})
	}
	config["earlyAccess"].(map[string]interface{})["enabled"] = w.EarlyAccessEnabled
	config["earlyAccess"].(map[string]interface{})["password"] = w.EarlyAccessPassword

	// ShipFrom
	if config["shipFrom"] == nil {
		config["shipFrom"] = make(map[string]interface{})
	}
	config["shipFrom"].(map[string]interface{})["name"] = w.ShipFromName
	config["shipFrom"].(map[string]interface{})["street1"] = w.ShipFromStreet1
	config["shipFrom"].(map[string]interface{})["street2"] = w.ShipFromStreet2
	config["shipFrom"].(map[string]interface{})["city"] = w.ShipFromCity
	config["shipFrom"].(map[string]interface{})["state"] = w.ShipFromState
	config["shipFrom"].(map[string]interface{})["zip"] = w.ShipFromZip
	config["shipFrom"].(map[string]interface{})["country"] = w.ShipFromCountry

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

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, created_at, updated_at
		FROM products_unified ORDER BY sort_order ASC, created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
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

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, released_date, created_at, updated_at
		FROM products_unified WHERE id = ?`

	var p Product
	var releasedDate sql.NullTime
	err = db.QueryRow(query, productID).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &p.SortOrder, &releasedDate, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Product{}, err
	}
	if releasedDate.Valid {
		p.ReleasedDate = releasedDate.Time
	}

	return p, nil
}

// CreateProduct creates a new product at the top and pushes others down
func (s *AdminServer) CreateProduct(websiteID string, p Product) (int64, error) {
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

	// Increment all existing products' sort_order by 1
	_, err = tx.Exec(`UPDATE products_unified SET sort_order = sort_order + 1`)
	if err != nil {
		return 0, err
	}

	// Insert new product with sort_order = 0 (top position)
	query := `INSERT INTO products_unified (name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, released_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	result, err := tx.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, releasedDate)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
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

	query := `UPDATE products_unified SET name = ?, slug = ?, description = ?, price = ?, compare_at_price = ?, sku = ?, inventory_quantity = ?, inventory_policy = ?, status = ?, featured = ?, sort_order = ?, released_date = ?
		WHERE id = ?`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	_, err = db.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, p.SortOrder, releasedDate, p.ID)
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

// ReorderProduct swaps the sort_order of a product with an adjacent product
func (s *AdminServer) ReorderProduct(websiteID string, productID int, direction string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get current product's sort_order
	var currentSortOrder int
	err = db.QueryRow(`SELECT sort_order FROM products_unified WHERE id = ?`, productID).Scan(&currentSortOrder)
	if err != nil {
		return err
	}

	var targetSortOrder int
	var targetID int
	if direction == "up" {
		// Find the product with the next lower sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM products_unified WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	} else if direction == "down" {
		// Find the product with the next higher sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM products_unified WHERE sort_order > ? ORDER BY sort_order ASC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
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

	// Swap the two products' sort_order values
	_, err = tx.Exec(`UPDATE products_unified SET sort_order = ? WHERE id = ?`, targetSortOrder, productID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE products_unified SET sort_order = ? WHERE id = ?`, currentSortOrder, targetID)
	if err != nil {
		return err
	}

	return tx.Commit()
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

// GetOrders retrieves all orders for a website
func (s *AdminServer) GetOrders(websiteID string) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetOrdersFiltered retrieves orders with filters and sorting
func (s *AdminServer) GetOrdersFiltered(websiteID string, filters OrderFilters) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE 1=1
	`

	var args []interface{}

	// Add payment status filter
	if filters.PaymentStatus != "" {
		query += " AND payment_status = ?"
		args = append(args, filters.PaymentStatus)
	}

	// Add fulfillment status filter
	if filters.FulfillmentStatus != "" {
		query += " AND fulfillment_status = ?"
		args = append(args, filters.FulfillmentStatus)
	}

	// Add sorting
	switch filters.Sort {
	case "date_asc":
		query += " ORDER BY created_at ASC"
	case "total_desc":
		query += " ORDER BY total DESC"
	case "total_asc":
		query += " ORDER BY total ASC"
	default: // date_desc
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetOrder retrieves a single order with its items
func (s *AdminServer) GetOrder(websiteID string, orderID int) (Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Order{}, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE id = ?
	`

	var o Order
	var shippingLine2, paymentMethod sql.NullString

	err = db.QueryRow(query, orderID).Scan(
		&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
		&o.ShippingAddressLine1, &shippingLine2,
		&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
		&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
		&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return Order{}, err
	}

	o.ShippingAddressLine2 = shippingLine2.String
	o.PaymentMethod = paymentMethod.String

	// Get order items
	itemsQuery := `
		SELECT id, product_id, product_name, variant_title, quantity, price, total
		FROM order_items
		WHERE order_id = ?
	`

	rows, err := db.Query(itemsQuery, orderID)
	if err != nil {
		return o, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem
		var variantTitle sql.NullString

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName, &variantTitle,
			&item.Quantity, &item.Price, &item.Total,
		)
		if err != nil {
			return o, err
		}

		item.VariantTitle = variantTitle.String
		o.Items = append(o.Items, item)
	}

	return o, nil
}

// UpdateOrderFulfillmentStatus updates the fulfillment status of an order
func (s *AdminServer) UpdateOrderFulfillmentStatus(websiteID string, orderID int, status string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE orders SET fulfillment_status = ?, updated_at = NOW() WHERE id = ?`
	_, err = db.Exec(query, status, orderID)
	return err
}

// LabelInfo contains information about a purchased shipping label
type LabelInfo struct {
	TrackingNumber string  `json:"trackingNumber"`
	LabelURL       string  `json:"labelUrl"`
	Carrier        string  `json:"carrier"`
	Cost           float64 `json:"cost"`
}

// GetShippingRates gets shipping rates from Shippo for an order
func (s *AdminServer) GetShippingRates(websiteID string, order Order, length, width, height, weight float64) ([]shippo.Rate, error) {
	// Get website config
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	// Load website config to get ship-from address
	configPath := filepath.Join("websites", website.Directory, "config-dev.json")
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read website config: %w", err)
	}

	var siteConfig configs.WebsiteConfig
	if err := json.Unmarshal(configData, &siteConfig); err != nil {
		return nil, fmt.Errorf("failed to parse website config: %w", err)
	}

	// Get Shippo API key from site config
	shippoKey := siteConfig.Shippo.APIKey

	// Create Shippo client
	client := shippo.NewClient(shippoKey)

	// Build addresses
	fromAddress := shippo.Address{
		Name:    siteConfig.ShipFrom.Name,
		Street1: siteConfig.ShipFrom.Street1,
		Street2: siteConfig.ShipFrom.Street2,
		City:    siteConfig.ShipFrom.City,
		State:   siteConfig.ShipFrom.State,
		Zip:     siteConfig.ShipFrom.Zip,
		Country: siteConfig.ShipFrom.Country,
	}

	toAddress := shippo.Address{
		Name:    order.CustomerName,
		Street1: order.ShippingAddressLine1,
		Street2: order.ShippingAddressLine2,
		City:    order.ShippingCity,
		State:   order.ShippingState,
		Zip:     order.ShippingZip,
		Country: order.ShippingCountry,
		Email:   order.CustomerEmail,
	}

	parcel := shippo.Parcel{
		Length:       fmt.Sprintf("%.2f", length),
		Width:        fmt.Sprintf("%.2f", width),
		Height:       fmt.Sprintf("%.2f", height),
		DistanceUnit: "in",
		Weight:       fmt.Sprintf("%.2f", weight),
		MassUnit:     "lb",
	}

	// Get rates from Shippo
	shipmentResp, err := client.GetRates(fromAddress, toAddress, parcel)
	if err != nil {
		return nil, err
	}

	return shipmentResp.Rates, nil
}

// PurchaseShippingLabel purchases a shipping label from Shippo and updates the order
func (s *AdminServer) PurchaseShippingLabel(websiteID string, orderID int, rateID string) (*LabelInfo, error) {
	// Get website config for site-specific Shippo key
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	// Get Shippo API key from site config
	shippoKey := website.ShippoAPIKey

	// Get label format - use site-specific if available, otherwise default to PDF
	labelFormat := "PDF"
	if website.LabelFormat != "" {
		labelFormat = website.LabelFormat
	}

	// Create Shippo client
	client := shippo.NewClient(shippoKey)

	// Purchase label
	transaction, err := client.PurchaseLabel(rateID, labelFormat)
	if err != nil {
		return nil, err
	}

	// Extract cost from rate amount
	cost := 0.0
	fmt.Sscanf(transaction.Rate, "%f", &cost)

	// Update order in database with label information
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Determine carrier from transaction
	carrier := "USPS" // Default, would need to parse from rate details

	query := `
		UPDATE orders
		SET tracking_number = ?,
		    shipping_carrier = ?,
		    shipping_label_url = ?,
		    shipping_label_cost = ?,
		    updated_at = NOW()
		WHERE id = ?
	`

	_, err = db.Exec(query, transaction.TrackingNumber, carrier, transaction.LabelURL, cost, orderID)
	if err != nil {
		return nil, err
	}

	return &LabelInfo{
		TrackingNumber: transaction.TrackingNumber,
		LabelURL:       transaction.LabelURL,
		Carrier:        carrier,
		Cost:           cost,
	}, nil
}

// GetProductImagesData retrieves all images for a product from product_images_data
func (s *AdminServer) GetProductImagesData(websiteID string, productID int) ([]ProductImageData, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, product_id, url, filename, filepath, alt_text, credit, size, width, height, position, created_at
		FROM product_images_data
		WHERE product_id = ?
		ORDER BY position ASC
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ProductImageData
	for rows.Next() {
		var img ProductImageData
		var altText, credit sql.NullString
		var size sql.NullInt64
		var width, height sql.NullInt64

		err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Filename, &img.Filepath,
			&altText, &credit, &size, &width, &height, &img.Position, &img.CreatedAt)
		if err != nil {
			return nil, err
		}

		if altText.Valid {
			img.AltText = altText.String
		}
		if credit.Valid {
			img.Credit = credit.String
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

// AddProductImageData adds a new image directly to a product
func (s *AdminServer) AddProductImageData(websiteID string, img ProductImageData) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO product_images_data (product_id, url, filename, filepath, alt_text, credit, size, width, height, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, img.ProductID, img.URL, img.Filename, img.Filepath,
		img.AltText, img.Credit, img.Size, img.Width, img.Height, img.Position)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteProductImageData deletes an image and its file from disk
func (s *AdminServer) DeleteProductImageData(websiteID string, imageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// First get the filepath
	var filepath string
	err = db.QueryRow("SELECT filepath FROM product_images_data WHERE id = ?", imageID).Scan(&filepath)
	if err != nil {
		return err
	}

	// Delete from database
	_, err = db.Exec("DELETE FROM product_images_data WHERE id = ?", imageID)
	if err != nil {
		return err
	}

	// Delete file from disk
	if err := os.Remove(filepath); err != nil {
		log.Printf("Warning: Failed to delete image file %s: %v", filepath, err)
		// Don't fail the operation if file doesn't exist
	}

	return nil
}

// UpdateProductImagePositions updates the position values for reordering
func (s *AdminServer) UpdateProductImagePositions(websiteID string, imageIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE product_images_data SET position = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for position, imageID := range imageIDs {
		_, err = stmt.Exec(position, imageID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ====================
// Customer Management
// ====================

// GetCustomers retrieves all customers with aggregate statistics
func (s *AdminServer) GetCustomers(websiteID string, filters CustomerFilters) ([]Customer, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone,
			c.created_at, c.updated_at,
			COUNT(CASE WHEN o.payment_status = 'paid' THEN 1 END) as order_count,
			COALESCE(SUM(CASE WHEN o.payment_status = 'paid' THEN o.total ELSE 0 END), 0) as total_spent,
			MIN(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as first_order,
			MAX(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as last_order
		FROM customers c
		LEFT JOIN orders o ON c.id = o.customer_id
		GROUP BY c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone, c.created_at, c.updated_at
	`

	// Add sorting
	switch filters.Sort {
	case "total_desc":
		query += " ORDER BY total_spent DESC"
	case "total_asc":
		query += " ORDER BY total_spent ASC"
	case "orders_desc":
		query += " ORDER BY order_count DESC"
	case "date_asc":
		query += " ORDER BY c.created_at ASC"
	case "date_desc":
		query += " ORDER BY c.created_at DESC"
	default: // Default to total spent descending
		query += " ORDER BY total_spent DESC"
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		var stripeCustomerID, phone sql.NullString
		var firstOrder, lastOrder sql.NullTime

		err := rows.Scan(
			&c.ID, &c.Email, &stripeCustomerID, &c.FirstName, &c.LastName, &phone,
			&c.CreatedAt, &c.UpdatedAt,
			&c.OrderCount, &c.TotalSpent, &firstOrder, &lastOrder,
		)
		if err != nil {
			return nil, err
		}

		if stripeCustomerID.Valid {
			c.StripeCustomerID = stripeCustomerID.String
		}
		if phone.Valid {
			c.Phone = phone.String
		}
		if firstOrder.Valid {
			c.FirstOrder = &firstOrder.Time
		}
		if lastOrder.Valid {
			c.LastOrder = &lastOrder.Time
		}

		customers = append(customers, c)
	}

	return customers, nil
}

// GetCustomer retrieves a single customer with statistics
func (s *AdminServer) GetCustomer(websiteID string, customerID int) (Customer, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Customer{}, err
	}
	defer db.Close()

	query := `
		SELECT
			c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone,
			c.created_at, c.updated_at,
			COUNT(CASE WHEN o.payment_status = 'paid' THEN 1 END) as order_count,
			COALESCE(SUM(CASE WHEN o.payment_status = 'paid' THEN o.total ELSE 0 END), 0) as total_spent,
			MIN(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as first_order,
			MAX(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as last_order
		FROM customers c
		LEFT JOIN orders o ON c.id = o.customer_id
		WHERE c.id = ?
		GROUP BY c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone, c.created_at, c.updated_at
	`

	var c Customer
	var stripeCustomerID, phone sql.NullString
	var firstOrder, lastOrder sql.NullTime

	err = db.QueryRow(query, customerID).Scan(
		&c.ID, &c.Email, &stripeCustomerID, &c.FirstName, &c.LastName, &phone,
		&c.CreatedAt, &c.UpdatedAt,
		&c.OrderCount, &c.TotalSpent, &firstOrder, &lastOrder,
	)
	if err != nil {
		return Customer{}, err
	}

	if stripeCustomerID.Valid {
		c.StripeCustomerID = stripeCustomerID.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if firstOrder.Valid {
		c.FirstOrder = &firstOrder.Time
	}
	if lastOrder.Valid {
		c.LastOrder = &lastOrder.Time
	}

	return c, nil
}

// GetCustomerOrders retrieves all orders for a specific customer
func (s *AdminServer) GetCustomerOrders(websiteID string, customerID int) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE customer_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetSMSSignups retrieves all SMS signups for a website
func (s *AdminServer) GetSMSSignups(websiteID string) ([]SMSSignup, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, phone, COALESCE(email, ''), COALESCE(source, ''), created_at
		FROM sms_signups
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signups []SMSSignup
	for rows.Next() {
		var s SMSSignup
		err := rows.Scan(&s.ID, &s.Phone, &s.Email, &s.Source, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		signups = append(signups, s)
	}

	return signups, nil
}

// DeleteSMSSignup deletes an SMS signup by ID
func (s *AdminServer) DeleteSMSSignup(websiteID string, signupID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM sms_signups WHERE id = ?`
	_, err = db.Exec(query, signupID)
	return err
}
