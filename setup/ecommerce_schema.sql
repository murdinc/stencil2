-- E-commerce Schema for Stencil2
-- This file contains the database schema needed for e-commerce functionality

-- Products table
CREATE TABLE IF NOT EXISTS products_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    compare_at_price DECIMAL(10, 2) DEFAULT NULL,
    sku VARCHAR(100),
    inventory_quantity INT DEFAULT 0,
    inventory_policy VARCHAR(50) DEFAULT 'deny',  -- 'deny' or 'continue'
    status VARCHAR(50) DEFAULT 'draft',  -- published, draft, archived
    featured TINYINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    published_date DATETIME,
    INDEX idx_slug (slug),
    INDEX idx_status (status),
    INDEX idx_featured (featured)
);

-- Collections (like categories for products)
CREATE TABLE IF NOT EXISTS collections_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    image_id INT,
    sort_order INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'published',  -- published, draft
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_slug (slug),
    INDEX idx_status (status),
    INDEX idx_sort_order (sort_order)
);

-- Product-Collection relationships (many-to-many)
CREATE TABLE IF NOT EXISTS product_collections (
    product_id INT NOT NULL,
    collection_id INT NOT NULL,
    position INT DEFAULT 0,
    PRIMARY KEY (product_id, collection_id),
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
    FOREIGN KEY (collection_id) REFERENCES collections_unified(id) ON DELETE CASCADE,
    INDEX idx_collection_id (collection_id),
    INDEX idx_position (position)
);

-- Product Images
CREATE TABLE IF NOT EXISTS product_images (
    id INT PRIMARY KEY AUTO_INCREMENT,
    product_id INT NOT NULL,
    image_id INT NOT NULL,
    position INT DEFAULT 0,
    alt_text VARCHAR(255),
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images_unified(id) ON DELETE CASCADE,
    INDEX idx_product_id (product_id),
    INDEX idx_position (position)
);

-- Product Variants (for size, color, etc.)
CREATE TABLE IF NOT EXISTS product_variants (
    id INT PRIMARY KEY AUTO_INCREMENT,
    product_id INT NOT NULL,
    title VARCHAR(255),  -- e.g., "Small / Red"
    option1 VARCHAR(100),  -- e.g., "Small"
    option2 VARCHAR(100),  -- e.g., "Red"
    option3 VARCHAR(100),
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    compare_at_price DECIMAL(10, 2) DEFAULT NULL,
    sku VARCHAR(100),
    inventory_quantity INT DEFAULT 0,
    position INT DEFAULT 0,
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
    INDEX idx_product_id (product_id),
    INDEX idx_sku (sku)
);

-- Shopping Carts
CREATE TABLE IF NOT EXISTS carts (
    id VARCHAR(255) PRIMARY KEY,  -- session ID
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    INDEX idx_expires_at (expires_at)
);

-- Cart Items
CREATE TABLE IF NOT EXISTS cart_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    cart_id VARCHAR(255) NOT NULL,
    product_id INT NOT NULL,
    variant_id INT DEFAULT 0,
    quantity INT NOT NULL DEFAULT 1,
    price DECIMAL(10, 2) NOT NULL,  -- snapshot price at time of add
    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
    INDEX idx_cart_id (cart_id)
);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
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
    payment_status VARCHAR(50) DEFAULT 'pending',  -- pending, paid, failed, refunded
    fulfillment_status VARCHAR(50) DEFAULT 'unfulfilled',  -- unfulfilled, fulfilled, shipped
    payment_method VARCHAR(50),
    stripe_payment_intent_id VARCHAR(255),
    shipping_label_cost DECIMAL(10, 2) DEFAULT NULL,  -- actual cost paid for shipping label
    tracking_number VARCHAR(100) DEFAULT NULL,
    shipping_carrier VARCHAR(50) DEFAULT NULL,  -- USPS, UPS, FedEx, etc.
    shipping_label_url VARCHAR(500) DEFAULT NULL,  -- URL to download label PDF
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_order_number (order_number),
    INDEX idx_customer_email (customer_email),
    INDEX idx_customer_id (customer_id),
    INDEX idx_payment_status (payment_status),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL
);

-- Order Items
CREATE TABLE IF NOT EXISTS order_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    variant_id INT DEFAULT 0,
    product_name VARCHAR(255) NOT NULL,  -- snapshot at time of order
    variant_title VARCHAR(255),
    quantity INT NOT NULL DEFAULT 1,
    price DECIMAL(10, 2) NOT NULL,  -- snapshot price
    total DECIMAL(10, 2) NOT NULL,  -- price * quantity
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    INDEX idx_order_id (order_id),
    INDEX idx_product_id (product_id)
);

-- Customers table for customer tracking
CREATE TABLE IF NOT EXISTS customers (
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
);

-- Clean up expired carts (run this periodically via cron)
-- DELETE FROM carts WHERE expires_at < NOW();
