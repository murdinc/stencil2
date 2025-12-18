package database

import (
	"fmt"
	"log"
)

// InitAdminTables creates admin-specific tables if they don't exist
func (db *DBConnection) InitAdminTables() error {
	if !db.Connected {
		return nil
	}

	schemas := []string{
		// Websites registry - tracks all configured websites
		`CREATE TABLE IF NOT EXISTS admin_websites (
			id INT PRIMARY KEY AUTO_INCREMENT,
			site_name VARCHAR(255) NOT NULL,
			directory VARCHAR(500) NOT NULL UNIQUE,
			database_name VARCHAR(255) NOT NULL,
			http_address VARCHAR(255),
			media_proxy_url VARCHAR(500),
			api_version INT DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_site_name (site_name),
			INDEX idx_directory (directory)
		)`,

		// Activity log for admin actions
		`CREATE TABLE IF NOT EXISTS admin_activity_log (
			id INT PRIMARY KEY AUTO_INCREMENT,
			action VARCHAR(255) NOT NULL,
			entity_type VARCHAR(100),
			entity_id INT,
			website_id INT,
			details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_action (action),
			INDEX idx_created_at (created_at),
			INDEX idx_website_id (website_id)
		)`,
	}

	for _, schema := range schemas {
		_, err := db.Database.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create admin table: %v", err)
		}
	}

	log.Println("Admin tables initialized successfully")
	return nil
}
