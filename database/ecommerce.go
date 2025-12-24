package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/murdinc/stencil2/structs"
)

// InitEcommerceTables creates e-commerce tables if they don't exist
func (db *DBConnection) InitEcommerceTables() error {
	if !db.Connected {
		return nil
	}

	schemas := []string{
		// Customers table (must be first for foreign key references)
		`CREATE TABLE IF NOT EXISTS customers (
			id INT PRIMARY KEY AUTO_INCREMENT,
			email VARCHAR(255) NOT NULL UNIQUE,
			stripe_customer_id VARCHAR(255) UNIQUE DEFAULT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			phone VARCHAR(50) DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_email (email),
			INDEX idx_stripe_customer_id (stripe_customer_id)
		)`,

		// Products table
		`CREATE TABLE IF NOT EXISTS products_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			compare_at_price DECIMAL(10, 2) DEFAULT NULL,
			sku VARCHAR(100),
			inventory_quantity INT DEFAULT 0,
			inventory_policy VARCHAR(50) DEFAULT 'deny',
			status VARCHAR(50) DEFAULT 'draft',
			featured BOOLEAN DEFAULT FALSE,
			sort_order INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			released_date DATETIME,
			INDEX idx_slug (slug),
			INDEX idx_status (status),
			INDEX idx_featured (featured),
			INDEX idx_sort_order (sort_order)
		)`,

		// Collections table
		`CREATE TABLE IF NOT EXISTS collections_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			image_id INT,
			sort_order INT DEFAULT 0,
			status VARCHAR(50) DEFAULT 'published',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug),
			INDEX idx_status (status),
			INDEX idx_sort_order (sort_order)
		)`,

		// Product-Collection relationships
		`CREATE TABLE IF NOT EXISTS product_collections (
			product_id INT NOT NULL,
			collection_id INT NOT NULL,
			position INT DEFAULT 0,
			PRIMARY KEY (product_id, collection_id),
			INDEX idx_collection_id (collection_id),
			INDEX idx_position (position)
		)`,

		// Product Images Data (dedicated table for product images)
		`CREATE TABLE IF NOT EXISTS product_images_data (
			id INT PRIMARY KEY AUTO_INCREMENT,
			product_id INT NOT NULL,
			url VARCHAR(500) NOT NULL,
			filename VARCHAR(255) NOT NULL,
			filepath VARCHAR(500) NOT NULL,
			alt_text VARCHAR(255),
			credit VARCHAR(255),
			size BIGINT,
			width INT,
			height INT,
			position INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
			INDEX idx_product_id (product_id),
			INDEX idx_position (position)
		)`,

		// Product Variants
		`CREATE TABLE IF NOT EXISTS product_variants (
			id INT PRIMARY KEY AUTO_INCREMENT,
			product_id INT NOT NULL,
			title VARCHAR(255),
			option1 VARCHAR(100),
			option2 VARCHAR(100),
			option3 VARCHAR(100),
			price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			compare_at_price DECIMAL(10, 2) DEFAULT NULL,
			sku VARCHAR(100),
			inventory_quantity INT DEFAULT 0,
			position INT DEFAULT 0,
			INDEX idx_product_id (product_id),
			INDEX idx_sku (sku)
		)`,

		// Shopping Carts
		`CREATE TABLE IF NOT EXISTS carts (
			id VARCHAR(255) PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			INDEX idx_expires_at (expires_at)
		)`,

		// Cart Items
		`CREATE TABLE IF NOT EXISTS cart_items (
			id INT PRIMARY KEY AUTO_INCREMENT,
			cart_id VARCHAR(255) NOT NULL,
			product_id INT NOT NULL,
			variant_id INT DEFAULT 0,
			quantity INT NOT NULL DEFAULT 1,
			price DECIMAL(10, 2) NOT NULL,
			INDEX idx_cart_id (cart_id)
		)`,

		// Orders
		`CREATE TABLE IF NOT EXISTS orders (
			id INT PRIMARY KEY AUTO_INCREMENT,
			order_number VARCHAR(50) UNIQUE NOT NULL,
			customer_email VARCHAR(255) NOT NULL,
			customer_name VARCHAR(255) NOT NULL,
			customer_id INT DEFAULT NULL,
			shipping_address_line1 VARCHAR(255),
			shipping_address_line2 VARCHAR(255),
			shipping_city VARCHAR(100),
			shipping_state VARCHAR(100),
			shipping_zip VARCHAR(20),
			shipping_country VARCHAR(100),
			billing_address_line1 VARCHAR(255),
			billing_city VARCHAR(100),
			billing_state VARCHAR(100),
			billing_zip VARCHAR(20),
			billing_country VARCHAR(100),
			subtotal DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			tax DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			shipping_cost DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			payment_status VARCHAR(50) DEFAULT 'pending',
			fulfillment_status VARCHAR(50) DEFAULT 'unfulfilled',
			payment_method VARCHAR(50),
			stripe_payment_intent_id VARCHAR(255),
			shipping_label_cost DECIMAL(10, 2) DEFAULT NULL,
			tracking_number VARCHAR(100) DEFAULT NULL,
			shipping_carrier VARCHAR(50) DEFAULT NULL,
			shipping_label_url VARCHAR(500) DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_order_number (order_number),
			INDEX idx_customer_email (customer_email),
			INDEX idx_customer_id (customer_id),
			INDEX idx_payment_status (payment_status),
			INDEX idx_created_at (created_at),
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL
		)`,

		// Order Items
		`CREATE TABLE IF NOT EXISTS order_items (
			id INT PRIMARY KEY AUTO_INCREMENT,
			order_id INT NOT NULL,
			product_id INT NOT NULL,
			variant_id INT DEFAULT 0,
			product_name VARCHAR(255) NOT NULL,
			variant_title VARCHAR(255),
			quantity INT NOT NULL DEFAULT 1,
			price DECIMAL(10, 2) NOT NULL,
			total DECIMAL(10, 2) NOT NULL,
			INDEX idx_order_id (order_id),
			INDEX idx_product_id (product_id)
		)`,

		// SMS Signups (marketing list)
		`CREATE TABLE IF NOT EXISTS sms_signups (
			id INT PRIMARY KEY AUTO_INCREMENT,
			country_code VARCHAR(10) DEFAULT '+1',
			phone VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(255) DEFAULT NULL,
			source VARCHAR(100) DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_created_at (created_at)
		)`,
	}

	for _, schema := range schemas {
		_, err := db.Database.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create e-commerce table: %v", err)
		}
	}

	log.Println("E-commerce tables initialized successfully")
	return nil
}

// GetProduct retrieves a single product by slug
func (db *DBConnection) GetProduct(slug string) (structs.Product, error) {
	sqlQuery := `
		SELECT
			id, name, slug, description, price, compare_at_price,
			sku, inventory_quantity, inventory_policy, status, featured,
			created_at, updated_at, released_date
		FROM products_unified
		WHERE slug = ? AND status = 'published'
		LIMIT 1
	`

	var product structs.Product
	var releasedDate sql.NullTime
	err := db.QueryRow(sqlQuery, slug).Scan(
		&product.ID, &product.Name, &product.Slug, &product.Description,
		&product.Price, &product.CompareAtPrice, &product.SKU,
		&product.InventoryQuantity, &product.InventoryPolicy, &product.Status, &product.Featured,
		&product.CreatedAt, &product.UpdatedAt, &releasedDate,
	)

	if err != nil {
		return structs.Product{}, err
	}

	if releasedDate.Valid {
		product.ReleasedDate = releasedDate.Time
	}

	if err != nil {
		return structs.Product{}, err
	}

	// Get product images
	product.Images, err = db.getProductImages(product.ID)
	if err != nil {
		return product, err
	}

	// Get product variants
	product.Variants, err = db.getProductVariants(product.ID)
	if err != nil {
		return product, err
	}

	// Get product collections
	product.Collections, err = db.getProductCollections(product.ID)
	if err != nil {
		return product, err
	}

	return product, nil
}

// GetProducts retrieves multiple products with pagination
func (db *DBConnection) GetProducts(vars map[string]string, params map[string]string) ([]structs.Product, error) {
	offset, count := defaultOffsetCount(vars)

	orderby := `sort_order ASC, released_date DESC`
	if value, exists := params["sort"]; exists {
		switch value {
		case "price_asc":
			orderby = `price ASC`
		case "price_desc":
			orderby = `price DESC`
		case "name":
			orderby = `name ASC`
		}
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			id, name, slug, description, price, compare_at_price,
			sku, inventory_quantity, inventory_policy, status, featured, sort_order,
			created_at, updated_at, released_date
		FROM products_unified
		WHERE status = 'published'
		ORDER BY %s
		LIMIT %d, %d
	`, orderby, offset, count)

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []structs.Product
	for rows.Next() {
		var product structs.Product
		var releasedDate sql.NullTime
		err := rows.Scan(
			&product.ID, &product.Name, &product.Slug, &product.Description,
			&product.Price, &product.CompareAtPrice, &product.SKU,
			&product.InventoryQuantity, &product.InventoryPolicy, &product.Status, &product.Featured, &product.SortOrder,
			&product.CreatedAt, &product.UpdatedAt, &releasedDate,
		)
		if err != nil {
			return nil, err
		}

		if releasedDate.Valid {
			product.ReleasedDate = releasedDate.Time
		}

		// Get product images
		product.Images, _ = db.getProductImages(product.ID)

		// Get product variants
		product.Variants, _ = db.getProductVariants(product.ID)

		products = append(products, product)
	}

	return products, nil
}

// GetFeaturedProducts retrieves featured products with pagination
func (db *DBConnection) GetFeaturedProducts(vars map[string]string, params map[string]string) ([]structs.Product, error) {
	offset, count := defaultOffsetCount(vars)

	orderby := `sort_order ASC, released_date DESC`
	if value, exists := params["sort"]; exists {
		switch value {
		case "price_asc":
			orderby = `price ASC`
		case "price_desc":
			orderby = `price DESC`
		case "name":
			orderby = `name ASC`
		}
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			id, name, slug, description, price, compare_at_price,
			sku, inventory_quantity, inventory_policy, status, featured, sort_order,
			created_at, updated_at, released_date
		FROM products_unified
		WHERE status = 'published' AND featured = 1
		ORDER BY %s
		LIMIT %d, %d
	`, orderby, offset, count)

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []structs.Product
	for rows.Next() {
		var product structs.Product
		var releasedDate sql.NullTime
		err := rows.Scan(
			&product.ID, &product.Name, &product.Slug, &product.Description,
			&product.Price, &product.CompareAtPrice, &product.SKU,
			&product.InventoryQuantity, &product.InventoryPolicy, &product.Status, &product.Featured, &product.SortOrder,
			&product.CreatedAt, &product.UpdatedAt, &releasedDate,
		)
		if err != nil {
			return nil, err
		}

		if releasedDate.Valid {
			product.ReleasedDate = releasedDate.Time
		}

		// Get product images
		product.Images, _ = db.getProductImages(product.ID)

		// Get product variants
		product.Variants, _ = db.getProductVariants(product.ID)

		products = append(products, product)
	}

	return products, nil
}

// GetCollection retrieves a single collection by slug
func (db *DBConnection) GetCollection(slug string) (structs.Collection, error) {
	sqlQuery := `
		SELECT
			c.id, c.name, c.slug, c.description, c.sort_order, c.status,
			c.created_at, c.updated_at,
			ifnull(i.id, 0), ifnull(i.url, ''), ifnull(i.alt_text, ''), ifnull(i.credit, ''),
			(SELECT COUNT(*) FROM product_collections pc
			 JOIN products_unified p ON pc.product_id = p.id
			 WHERE pc.collection_id = c.id AND p.status = 'published') as product_count
		FROM collections_unified c
		LEFT JOIN images_unified i ON c.image_id = i.id
		WHERE c.slug = ? AND c.status = 'published'
		LIMIT 1
	`

	var collection structs.Collection
	err := db.QueryRow(sqlQuery, slug).Scan(
		&collection.ID, &collection.Name, &collection.Slug, &collection.Description,
		&collection.SortOrder, &collection.Status, &collection.CreatedAt, &collection.UpdatedAt,
		&collection.Image.ID, &collection.Image.URL, &collection.Image.AltText, &collection.Image.Credit,
		&collection.ProductCount,
	)

	if err != nil {
		return structs.Collection{}, err
	}

	return collection, nil
}

// GetCollections retrieves all collections
func (db *DBConnection) GetCollections() ([]structs.Collection, error) {
	sqlQuery := `
		SELECT
			c.id, c.name, c.slug, c.description, c.sort_order, c.status,
			c.created_at, c.updated_at,
			ifnull(i.id, 0), ifnull(i.url, ''), ifnull(i.alt_text, ''), ifnull(i.credit, ''),
			(SELECT COUNT(*) FROM product_collections pc
			 JOIN products_unified p ON pc.product_id = p.id
			 WHERE pc.collection_id = c.id AND p.status = 'published') as product_count
		FROM collections_unified c
		LEFT JOIN images_unified i ON c.image_id = i.id
		WHERE c.status = 'published'
		ORDER BY c.sort_order ASC, c.name ASC
	`

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []structs.Collection
	for rows.Next() {
		var collection structs.Collection
		err := rows.Scan(
			&collection.ID, &collection.Name, &collection.Slug, &collection.Description,
			&collection.SortOrder, &collection.Status, &collection.CreatedAt, &collection.UpdatedAt,
			&collection.Image.ID, &collection.Image.URL, &collection.Image.AltText, &collection.Image.Credit,
			&collection.ProductCount,
		)
		if err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

// GetCollectionProducts retrieves products in a collection
func (db *DBConnection) GetCollectionProducts(collectionSlug string, vars map[string]string, params map[string]string) ([]structs.Product, error) {
	offset, count := defaultOffsetCount(vars)

	orderby := `p.released_date DESC`
	if value, exists := params["sort"]; exists {
		switch value {
		case "price_asc":
			orderby = `p.price ASC`
		case "price_desc":
			orderby = `p.price DESC`
		case "name":
			orderby = `p.name ASC`
		case "position":
			orderby = `pc.position ASC`
		}
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			p.id, p.name, p.slug, p.description, p.price, p.compare_at_price,
			p.sku, p.inventory_quantity, p.inventory_policy, p.status, p.featured,
			p.created_at, p.updated_at, p.released_date
		FROM products_unified p
		JOIN product_collections pc ON p.id = pc.product_id
		JOIN collections_unified c ON pc.collection_id = c.id
		WHERE c.slug = ? AND p.status = 'published' AND c.status = 'published'
		ORDER BY %s
		LIMIT %d, %d
	`, orderby, offset, count)

	rows, err := db.QueryRows(sqlQuery, collectionSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []structs.Product
	for rows.Next() {
		var product structs.Product
		var releasedDate sql.NullTime
		err := rows.Scan(
			&product.ID, &product.Name, &product.Slug, &product.Description,
			&product.Price, &product.CompareAtPrice, &product.SKU,
			&product.InventoryQuantity, &product.InventoryPolicy, &product.Status, &product.Featured,
			&product.CreatedAt, &product.UpdatedAt, &releasedDate,
		)
		if err != nil {
			return nil, err
		}

		if releasedDate.Valid {
			product.ReleasedDate = releasedDate.Time
		}

		// Get product images
		product.Images, _ = db.getProductImages(product.ID)

		// Get product variants
		product.Variants, _ = db.getProductVariants(product.ID)

		products = append(products, product)
	}

	return products, nil
}

// Helper functions

func (db *DBConnection) getProductImages(productID int) ([]structs.ProductImage, error) {
	sqlQuery := `
		SELECT
			id, url, alt_text, credit, position
		FROM product_images_data
		WHERE product_id = ?
		ORDER BY position ASC
	`

	rows, err := db.QueryRows(sqlQuery, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []structs.ProductImage
	for rows.Next() {
		var img structs.ProductImage
		var altText, credit sql.NullString
		err := rows.Scan(
			&img.ID, &img.Image.URL, &altText, &credit, &img.Position,
		)
		if err != nil {
			return nil, err
		}
		if altText.Valid {
			img.Image.AltText = altText.String
		}
		if credit.Valid {
			img.Image.Credit = credit.String
		}
		images = append(images, img)
	}

	return images, nil
}

func (db *DBConnection) getProductVariants(productID int) ([]structs.ProductVariant, error) {
	sqlQuery := `
		SELECT
			id, product_id, title, option1, option2, option3,
			price, compare_at_price, sku, inventory_quantity, position
		FROM product_variants
		WHERE product_id = ?
		ORDER BY position ASC
	`

	rows, err := db.QueryRows(sqlQuery, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []structs.ProductVariant
	for rows.Next() {
		var variant structs.ProductVariant
		var option1, option2, option3 sql.NullString

		err := rows.Scan(
			&variant.ID, &variant.ProductID, &variant.Title,
			&option1, &option2, &option3,
			&variant.Price, &variant.CompareAtPrice, &variant.SKU,
			&variant.InventoryQuantity, &variant.Position,
		)
		if err != nil {
			return nil, err
		}

		variant.Option1 = option1.String
		variant.Option2 = option2.String
		variant.Option3 = option3.String

		variants = append(variants, variant)
	}

	return variants, nil
}

func (db *DBConnection) getProductCollections(productID int) ([]structs.Collection, error) {
	sqlQuery := `
		SELECT
			c.id, c.name, c.slug, c.description, c.sort_order, c.status,
			c.created_at, c.updated_at
		FROM collections_unified c
		JOIN product_collections pc ON c.id = pc.collection_id
		WHERE pc.product_id = ? AND c.status = 'published'
		ORDER BY c.sort_order ASC
	`

	rows, err := db.QueryRows(sqlQuery, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []structs.Collection
	for rows.Next() {
		var collection structs.Collection
		err := rows.Scan(
			&collection.ID, &collection.Name, &collection.Slug, &collection.Description,
			&collection.SortOrder, &collection.Status, &collection.CreatedAt, &collection.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

// Cart operations

// GetCart retrieves or creates a cart by session ID
func (db *DBConnection) GetCart(sessionID string) (structs.Cart, error) {
	sqlQuery := `
		SELECT id, created_at, updated_at, expires_at
		FROM carts
		WHERE id = ? AND expires_at > NOW()
		LIMIT 1
	`

	var cart structs.Cart
	err := db.QueryRow(sqlQuery, sessionID).Scan(
		&cart.ID, &cart.CreatedAt, &cart.UpdatedAt, &cart.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		// Create new cart
		return db.createCart(sessionID)
	} else if err != nil {
		return structs.Cart{}, err
	}

	// Get cart items
	cart.Items, err = db.getCartItems(sessionID)
	if err != nil {
		return cart, err
	}

	// Calculate subtotal
	cart.Subtotal = 0
	for _, item := range cart.Items {
		cart.Subtotal += item.Total
	}

	return cart, nil
}

func (db *DBConnection) createCart(sessionID string) (structs.Cart, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour * 7) // 7 days

	sqlQuery := `
		INSERT INTO carts (id, created_at, updated_at, expires_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := db.ExecuteQuery(sqlQuery, sessionID, now, now, expiresAt)
	if err != nil {
		return structs.Cart{}, err
	}

	return structs.Cart{
		ID:        sessionID,
		Items:     []structs.CartItem{},
		Subtotal:  0,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

func (db *DBConnection) getCartItems(cartID string) ([]structs.CartItem, error) {
	sqlQuery := `
		SELECT
			ci.id, ci.product_id, ci.variant_id, ci.quantity, ci.price,
			p.name, p.slug, p.description, p.price,
			ifnull(pv.title, ''), ifnull(pv.option1, ''), ifnull(pv.option2, ''), ifnull(pv.option3, '')
		FROM cart_items ci
		JOIN products_unified p ON ci.product_id = p.id
		LEFT JOIN product_variants pv ON ci.variant_id = pv.id
		WHERE ci.cart_id = ?
	`

	rows, err := db.QueryRows(sqlQuery, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []structs.CartItem
	for rows.Next() {
		var item structs.CartItem
		err := rows.Scan(
			&item.ID, &item.ProductID, &item.VariantID, &item.Quantity, &item.Price,
			&item.Product.Name, &item.Product.Slug, &item.Product.Description, &item.Product.Price,
			&item.Variant.Title, &item.Variant.Option1, &item.Variant.Option2, &item.Variant.Option3,
		)
		if err != nil {
			return nil, err
		}

		// Load product images
		item.Product.Images, _ = db.getProductImages(item.ProductID)

		item.Total = item.Price * float64(item.Quantity)
		items = append(items, item)
	}

	return items, nil
}

// AddToCart adds an item to the cart
func (db *DBConnection) AddToCart(sessionID string, productID int, variantID int, quantity int) error {
	// Get the price
	var price float64
	if variantID > 0 {
		err := db.QueryRow("SELECT price FROM product_variants WHERE id = ?", variantID).Scan(&price)
		if err != nil {
			return err
		}
	} else {
		err := db.QueryRow("SELECT price FROM products_unified WHERE id = ?", productID).Scan(&price)
		if err != nil {
			return err
		}
	}

	// Check if item already exists in cart
	var existingID int
	var existingQuantity int
	err := db.QueryRow(`
		SELECT id, quantity FROM cart_items
		WHERE cart_id = ? AND product_id = ? AND variant_id = ?
	`, sessionID, productID, variantID).Scan(&existingID, &existingQuantity)

	if err == sql.ErrNoRows {
		// Insert new item
		sqlQuery := `
			INSERT INTO cart_items (cart_id, product_id, variant_id, quantity, price)
			VALUES (?, ?, ?, ?, ?)
		`
		_, err = db.ExecuteQuery(sqlQuery, sessionID, productID, variantID, quantity, price)
		return err
	} else if err != nil {
		return err
	}

	// Update existing item quantity
	newQuantity := existingQuantity + quantity
	return db.UpdateCartItem(existingID, newQuantity)
}

// UpdateCartItem updates the quantity of a cart item
func (db *DBConnection) UpdateCartItem(cartItemID int, quantity int) error {
	if quantity <= 0 {
		return db.RemoveFromCart(cartItemID)
	}

	sqlQuery := `UPDATE cart_items SET quantity = ? WHERE id = ?`
	_, err := db.ExecuteQuery(sqlQuery, quantity, cartItemID)
	return err
}

// RemoveFromCart removes an item from the cart
func (db *DBConnection) RemoveFromCart(cartItemID int) error {
	sqlQuery := `DELETE FROM cart_items WHERE id = ?`
	_, err := db.ExecuteQuery(sqlQuery, cartItemID)
	return err
}

// CreateOrder creates an order from cart data
func (db *DBConnection) CreateOrder(orderData map[string]interface{}) (structs.Order, error) {
	// Generate order number
	orderNumber := fmt.Sprintf("ORD-%d", time.Now().Unix())

	// Extract data from map
	cartItems := orderData["cart_items"].([]structs.CartItem)
	customerEmail := orderData["email"].(string)

	// Extract shipping address (nested object)
	shippingAddr := orderData["shipping_address"].(map[string]interface{})
	firstName := shippingAddr["first_name"].(string)
	lastName := shippingAddr["last_name"].(string)
	customerName := firstName + " " + lastName

	// Get or create customer record
	customer, err := db.GetOrCreateCustomer(customerEmail, firstName, lastName)
	if err != nil {
		// Log error but don't fail order creation - backwards compatibility
		// Customer tracking is supplementary feature
		log.Printf("Warning: failed to create/get customer: %v\n", err)
	}

	// Extract payment information (if provided)
	paymentIntentID := ""
	if val, ok := orderData["payment_intent_id"].(string); ok {
		paymentIntentID = val
	}
	paymentStatus := "pending"
	if val, ok := orderData["payment_status"].(string); ok {
		paymentStatus = val
	}

	// Calculate totals
	subtotal := 0.0
	for _, item := range cartItems {
		subtotal += item.Total
	}

	// Get tax rate and shipping cost from orderData (0 is valid)
	taxRate := 0.0
	if tr, ok := orderData["tax_rate"].(float64); ok {
		taxRate = tr
	}
	tax := subtotal * taxRate

	shippingCost := 0.0
	if sc, ok := orderData["shipping_cost"].(float64); ok {
		shippingCost = sc
	}

	total := subtotal + tax + shippingCost

	// Build full address from nested fields
	address1 := shippingAddr["address"].(string)
	address2 := ""
	if addr2, ok := shippingAddr["address2"].(string); ok {
		address2 = addr2
	}
	city := shippingAddr["city"].(string)
	state := shippingAddr["state"].(string)
	zip := shippingAddr["zip"].(string)
	country := shippingAddr["country"].(string)

	// Prepare customer ID for insertion (NULL if customer creation failed)
	var customerID interface{} = nil
	if customer.ID > 0 {
		customerID = customer.ID
	}

	// Insert order
	sqlQuery := `
		INSERT INTO orders (
			order_number, customer_email, customer_name, customer_id,
			shipping_address_line1, shipping_address_line2, shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, stripe_payment_intent_id, payment_method, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'unfulfilled', ?, 'card', NOW(), NOW())
	`

	result, err := db.ExecuteQuery(sqlQuery,
		orderNumber, customerEmail, customerName, customerID,
		address1, address2, city, state, zip, country,
		subtotal, tax, shippingCost, total,
		paymentStatus, paymentIntentID,
	)
	if err != nil {
		return structs.Order{}, err
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		return structs.Order{}, err
	}

	// Insert order items and deduct inventory
	for _, item := range cartItems {
		itemQuery := `
			INSERT INTO order_items (
				order_id, product_id, variant_id, product_name, variant_title,
				quantity, price, total
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = db.ExecuteQuery(itemQuery,
			orderID, item.ProductID, item.VariantID, item.Product.Name, item.Variant.Title,
			item.Quantity, item.Price, item.Total,
		)
		if err != nil {
			return structs.Order{}, err
		}

		// Deduct inventory
		if item.VariantID > 0 {
			// Deduct from variant inventory
			inventoryQuery := `
				UPDATE product_variants
				SET inventory_quantity = inventory_quantity - ?
				WHERE id = ? AND inventory_quantity >= ?
			`
			result, err := db.ExecuteQuery(inventoryQuery, item.Quantity, item.VariantID, item.Quantity)
			if err != nil {
				return structs.Order{}, fmt.Errorf("failed to deduct variant inventory: %v", err)
			}
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected == 0 {
				return structs.Order{}, fmt.Errorf("insufficient inventory for variant ID %d", item.VariantID)
			}
		} else {
			// Deduct from product inventory
			inventoryQuery := `
				UPDATE products_unified
				SET inventory_quantity = inventory_quantity - ?
				WHERE id = ? AND inventory_quantity >= ?
			`
			result, err := db.ExecuteQuery(inventoryQuery, item.Quantity, item.ProductID, item.Quantity)
			if err != nil {
				return structs.Order{}, fmt.Errorf("failed to deduct product inventory: %v", err)
			}
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected == 0 {
				return structs.Order{}, fmt.Errorf("insufficient inventory for product ID %d", item.ProductID)
			}
		}
	}

	// Return the created order
	return db.GetOrder(orderNumber)
}

// GetOrder retrieves an order by order number
func (db *DBConnection) GetOrder(orderNumber string) (structs.Order, error) {
	sqlQuery := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			stripe_payment_intent_id, created_at, updated_at
		FROM orders
		WHERE order_number = ?
		LIMIT 1
	`

	var order structs.Order
	var shippingLine2, paymentMethod, stripeIntent sql.NullString

	err := db.QueryRow(sqlQuery, orderNumber).Scan(
		&order.ID, &order.OrderNumber, &order.CustomerEmail, &order.CustomerName,
		&order.ShippingAddressLine1, &shippingLine2,
		&order.ShippingCity, &order.ShippingState, &order.ShippingZip, &order.ShippingCountry,
		&order.Subtotal, &order.Tax, &order.ShippingCost, &order.Total,
		&order.PaymentStatus, &order.FulfillmentStatus, &paymentMethod,
		&stripeIntent, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return structs.Order{}, err
	}

	order.ShippingAddressLine2 = shippingLine2.String
	order.PaymentMethod = paymentMethod.String
	order.StripePaymentIntent = stripeIntent.String

	// Get order items
	order.Items, err = db.getOrderItems(order.ID)
	if err != nil {
		return order, err
	}

	return order, nil
}

func (db *DBConnection) getOrderItems(orderID int) ([]structs.OrderItem, error) {
	sqlQuery := `
		SELECT
			id, product_id, variant_id, product_name, variant_title,
			quantity, price, total
		FROM order_items
		WHERE order_id = ?
	`

	rows, err := db.QueryRows(sqlQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []structs.OrderItem
	for rows.Next() {
		var item structs.OrderItem
		err := rows.Scan(
			&item.ID, &item.ProductID, &item.VariantID, &item.ProductName, &item.VariantTitle,
			&item.Quantity, &item.Price, &item.Total,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateOrderPaymentStatus updates the payment status of an order
func (db *DBConnection) UpdateOrderPaymentStatus(orderNumber string, status string, paymentIntentID string, paymentMethod string) error {
	sqlQuery := `
		UPDATE orders
		SET payment_status = ?, stripe_payment_intent_id = ?, payment_method = ?, updated_at = NOW()
		WHERE order_number = ?
	`
	_, err := db.ExecuteQuery(sqlQuery, status, paymentIntentID, paymentMethod, orderNumber)
	return err
}

// UpdateOrderPaymentStatusByIntentID updates payment status by payment intent ID
func (db *DBConnection) UpdateOrderPaymentStatusByIntentID(paymentIntentID string, status string) error {
	sqlQuery := `
		UPDATE orders
		SET payment_status = ?, updated_at = NOW()
		WHERE stripe_payment_intent_id = ?
	`
	_, err := db.ExecuteQuery(sqlQuery, status, paymentIntentID)
	return err
}

// GetOrderByPaymentIntentID retrieves an order by payment intent ID
func (db *DBConnection) GetOrderByPaymentIntentID(paymentIntentID string) (structs.Order, error) {
	sqlQuery := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			stripe_payment_intent_id, created_at, updated_at
		FROM orders
		WHERE stripe_payment_intent_id = ?
		LIMIT 1
	`

	var order structs.Order
	var shippingLine2, paymentMethod, stripeIntent sql.NullString

	err := db.QueryRow(sqlQuery, paymentIntentID).Scan(
		&order.ID, &order.OrderNumber, &order.CustomerEmail, &order.CustomerName,
		&order.ShippingAddressLine1, &shippingLine2,
		&order.ShippingCity, &order.ShippingState, &order.ShippingZip, &order.ShippingCountry,
		&order.Subtotal, &order.Tax, &order.ShippingCost, &order.Total,
		&order.PaymentStatus, &order.FulfillmentStatus, &paymentMethod,
		&stripeIntent, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return structs.Order{}, err
	}

	order.ShippingAddressLine2 = shippingLine2.String
	order.PaymentMethod = paymentMethod.String
	order.StripePaymentIntent = stripeIntent.String

	// Get order items
	order.Items, err = db.getOrderItems(order.ID)
	if err != nil {
		return order, err
	}

	return order, nil
}

// GetOrCreateCustomer finds an existing customer by email or creates a new one
// Email comparison is case-insensitive for deduplication
func (db *DBConnection) GetOrCreateCustomer(email, firstName, lastName string) (structs.Customer, error) {
	// Normalize email to lowercase for consistent lookups
	email = strings.ToLower(strings.TrimSpace(email))

	// First, try to find existing customer
	customer, err := db.GetCustomerByEmail(email)
	if err == nil {
		// Customer exists, return it
		return customer, nil
	}

	// Customer doesn't exist, create new one
	sqlQuery := `
		INSERT INTO customers (email, first_name, last_name, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
	`

	result, err := db.ExecuteQuery(sqlQuery, email, firstName, lastName)
	if err != nil {
		// Check if this is a duplicate key error (race condition)
		if strings.Contains(err.Error(), "Duplicate entry") {
			// Another request created the customer, fetch it
			return db.GetCustomerByEmail(email)
		}
		return structs.Customer{}, err
	}

	customerID, err := result.LastInsertId()
	if err != nil {
		return structs.Customer{}, err
	}

	// Return the newly created customer
	return db.GetCustomerByID(int(customerID))
}

// GetCustomerByEmail retrieves a customer by email address
func (db *DBConnection) GetCustomerByEmail(email string) (structs.Customer, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	sqlQuery := `
		SELECT id, email, COALESCE(stripe_customer_id, ''), first_name, last_name,
		       COALESCE(phone, ''), created_at, updated_at
		FROM customers
		WHERE LOWER(email) = ?
		LIMIT 1
	`

	var c structs.Customer
	err := db.QueryRow(sqlQuery, email).Scan(
		&c.ID, &c.Email, &c.StripeCustomerID, &c.FirstName, &c.LastName,
		&c.Phone, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		return structs.Customer{}, err
	}

	return c, nil
}

// GetCustomerByID retrieves a customer by ID
func (db *DBConnection) GetCustomerByID(customerID int) (structs.Customer, error) {
	sqlQuery := `
		SELECT id, email, COALESCE(stripe_customer_id, ''), first_name, last_name,
		       COALESCE(phone, ''), created_at, updated_at
		FROM customers
		WHERE id = ?
		LIMIT 1
	`

	var c structs.Customer
	err := db.QueryRow(sqlQuery, customerID).Scan(
		&c.ID, &c.Email, &c.StripeCustomerID, &c.FirstName, &c.LastName,
		&c.Phone, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		return structs.Customer{}, err
	}

	return c, nil
}

// UpdateCustomerStripeID updates the Stripe customer ID for a customer
func (db *DBConnection) UpdateCustomerStripeID(customerID int, stripeCustomerID string) error {
	sqlQuery := `
		UPDATE customers
		SET stripe_customer_id = ?, updated_at = NOW()
		WHERE id = ?
	`

	_, err := db.ExecuteQuery(sqlQuery, stripeCustomerID, customerID)
	return err
}

// CreateSMSSignup creates or updates an SMS signup entry
func (db *DBConnection) CreateSMSSignup(countryCode, phone, email, source string) (int64, error) {
	sqlQuery := `
		INSERT INTO sms_signups (country_code, phone, email, source)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			email = CASE
				WHEN ? != '' AND ? != email THEN ?
				ELSE email
			END,
			source = VALUES(source)
	`

	result, err := db.ExecuteQuery(sqlQuery, countryCode, phone, email, source, email, email, email)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetSMSSignups retrieves all SMS signups
func (db *DBConnection) GetSMSSignups() ([]structs.SMSSignup, error) {
	sqlQuery := `
		SELECT id, COALESCE(country_code, '+1'), phone, COALESCE(email, ''), COALESCE(source, ''), created_at
		FROM sms_signups
		ORDER BY created_at DESC
	`

	rows, err := db.Database.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signups []structs.SMSSignup
	for rows.Next() {
		var s structs.SMSSignup
		err := rows.Scan(&s.ID, &s.CountryCode, &s.Phone, &s.Email, &s.Source, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		signups = append(signups, s)
	}

	return signups, nil
}

// DeleteSMSSignup deletes an SMS signup by ID
func (db *DBConnection) DeleteSMSSignup(signupID int) error {
	sqlQuery := `DELETE FROM sms_signups WHERE id = ?`
	_, err := db.ExecuteQuery(sqlQuery, signupID)
	return err
}
