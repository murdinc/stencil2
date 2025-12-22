package database

import (
	"fmt"
	"log"
)

// InitArticleTables creates article/content tables if they don't exist
func (db *DBConnection) InitArticleTables() error {
	if !db.Connected {
		return nil
	}

	schemas := []string{
		// Categories table
		`CREATE TABLE IF NOT EXISTS categories_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			count INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug)
		)`,

		// Authors table
		`CREATE TABLE IF NOT EXISTS authors_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			bio TEXT,
			image_url VARCHAR(500),
			count INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug)
		)`,

		// Tags table
		`CREATE TABLE IF NOT EXISTS tags_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			count INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug)
		)`,

		// Images table
		`CREATE TABLE IF NOT EXISTS images_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			url VARCHAR(500) NOT NULL,
			alt_text VARCHAR(255),
			filename VARCHAR(255),
			size BIGINT,
			credit VARCHAR(255),
			width INT,
			height INT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_url (url)
		)`,

		// Articles table
		`CREATE TABLE IF NOT EXISTS articles_unified (
			id INT PRIMARY KEY AUTO_INCREMENT,
			slug VARCHAR(500) UNIQUE NOT NULL,
			title VARCHAR(500) NOT NULL,
			description TEXT,
			content LONGTEXT,
			excerpt TEXT,
			deck TEXT,
			coverline VARCHAR(255),
			type VARCHAR(50) DEFAULT 'article',
			status VARCHAR(50) DEFAULT 'draft',
			published_date DATETIME,
			thumbnail_id INT,
			canonical_url VARCHAR(500),
			keywords TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug),
			INDEX idx_status (status),
			INDEX idx_type (type),
			INDEX idx_published_date (published_date)
		)`,

		// Article information (denormalized JSON storage)
		`CREATE TABLE IF NOT EXISTS article_information (
			post_id INT PRIMARY KEY,
			authors JSON,
			categories JSON,
			tags JSON,
			image JSON,
			FOREIGN KEY (post_id) REFERENCES articles_unified(id) ON DELETE CASCADE
		)`,

		// Article-Author relationships
		`CREATE TABLE IF NOT EXISTS article_authors (
			post_id INT NOT NULL,
			author_id INT NOT NULL,
			position INT DEFAULT 0,
			PRIMARY KEY (post_id, author_id),
			INDEX idx_author_id (author_id),
			INDEX idx_position (position)
		)`,

		// Article-Category relationships
		`CREATE TABLE IF NOT EXISTS article_categories (
			post_id INT NOT NULL,
			category_id INT NOT NULL,
			PRIMARY KEY (post_id, category_id),
			INDEX idx_category_id (category_id)
		)`,

		// Article-Tag relationships
		`CREATE TABLE IF NOT EXISTS article_tags (
			post_id INT NOT NULL,
			tag_id INT NOT NULL,
			PRIMARY KEY (post_id, tag_id),
			INDEX idx_tag_id (tag_id)
		)`,

		// Preview tables for draft mode
		`CREATE TABLE IF NOT EXISTS preview_article_information (
			post_id INT PRIMARY KEY,
			authors JSON,
			categories JSON,
			tags JSON,
			image JSON
		)`,

		// Article sitemaps tracking
		"CREATE TABLE IF NOT EXISTS article_sitemaps (" +
			"id INT PRIMARY KEY AUTO_INCREMENT, " +
			"`year_month` VARCHAR(7) NOT NULL, " +
			"generated_at DATETIME DEFAULT CURRENT_TIMESTAMP, " +
			"article_count INT DEFAULT 0, " +
			"UNIQUE INDEX idx_year_month (`year_month`)" +
		")",

		// Article duplicates for slides
		`CREATE TABLE IF NOT EXISTS article_duplicates_slides (
			id INT PRIMARY KEY AUTO_INCREMENT,
			original_post_id INT NOT NULL,
			duplicate_post_id INT NOT NULL,
			slide_id INT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_original (original_post_id),
			INDEX idx_duplicate (duplicate_post_id)
		)`,
	}

	for _, schema := range schemas {
		_, err := db.Database.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create article table: %v", err)
		}
	}

	log.Println("Article/content tables initialized successfully")
	return nil
}
