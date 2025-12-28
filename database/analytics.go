package database

import (
	"encoding/json"
	"fmt"
	"time"
)

// InitAnalyticsTables creates analytics tables if they don't exist
func (db *DBConnection) InitAnalyticsTables() error {
	if !db.Connected {
		return nil
	}

	schemas := []string{
		// Analytics - Page Views
		`CREATE TABLE IF NOT EXISTS analytics_pageviews (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			session_id VARCHAR(36) NOT NULL,
			path VARCHAR(500) NOT NULL,
			referrer VARCHAR(500) DEFAULT NULL,
			user_agent TEXT DEFAULT NULL,
			ip_address VARCHAR(45) DEFAULT NULL,
			screen_width INT DEFAULT NULL,
			screen_height INT DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_session_id (session_id),
			INDEX idx_path (path(255)),
			INDEX idx_created_at (created_at),
			INDEX idx_session_created (session_id, created_at)
		)`,

		// Analytics - Custom Events
		`CREATE TABLE IF NOT EXISTS analytics_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			session_id VARCHAR(36) NOT NULL,
			event_name VARCHAR(100) NOT NULL,
			event_data JSON DEFAULT NULL,
			path VARCHAR(500) DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_session_id (session_id),
			INDEX idx_event_name (event_name),
			INDEX idx_created_at (created_at),
			INDEX idx_event_created (event_name, created_at)
		)`,
	}

	for _, schema := range schemas {
		_, err := db.Database.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create analytics table: %v", err)
		}
	}

	return nil
}

// TrackPageView records a page view
func (db *DBConnection) TrackPageView(sessionID, path, referrer, userAgent, ipAddress string, screenWidth, screenHeight int) error {
	sqlQuery := `
		INSERT INTO analytics_pageviews
		(session_id, path, referrer, user_agent, ip_address, screen_width, screen_height)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.ExecuteQuery(sqlQuery, sessionID, path, referrer, userAgent, ipAddress, screenWidth, screenHeight)
	return err
}

// TrackEvent records a custom event
func (db *DBConnection) TrackEvent(sessionID, eventName, path string, eventData map[string]interface{}) error {
	var eventDataJSON []byte
	var err error

	if eventData != nil {
		eventDataJSON, err = json.Marshal(eventData)
		if err != nil {
			return err
		}
	}

	sqlQuery := `
		INSERT INTO analytics_events
		(session_id, event_name, event_data, path)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.ExecuteQuery(sqlQuery, sessionID, eventName, eventDataJSON, path)
	return err
}

// GetPageViewStats returns basic pageview statistics for a date range
func (db *DBConnection) GetPageViewStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total pageviews
	var totalViews int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
	`, startDate, endDate).Scan(&totalViews)
	if err != nil {
		return nil, err
	}
	stats["total_views"] = totalViews

	// Unique sessions
	var uniqueSessions int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT session_id) FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
	`, startDate, endDate).Scan(&uniqueSessions)
	if err != nil {
		return nil, err
	}
	stats["unique_sessions"] = uniqueSessions

	return stats, nil
}

// GetTopPages returns the most visited pages for a date range
func (db *DBConnection) GetTopPages(startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	sqlQuery := `
		SELECT path, COUNT(*) as views, COUNT(DISTINCT session_id) as unique_visitors
		FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
		GROUP BY path
		ORDER BY views DESC
		LIMIT ?
	`

	rows, err := db.Database.Query(sqlQuery, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []map[string]interface{}
	for rows.Next() {
		var path string
		var views, uniqueVisitors int
		err := rows.Scan(&path, &views, &uniqueVisitors)
		if err != nil {
			return nil, err
		}
		pages = append(pages, map[string]interface{}{
			"path":            path,
			"views":           views,
			"unique_visitors": uniqueVisitors,
		})
	}

	return pages, nil
}

// GetTopReferrers returns the top referrers for a date range
func (db *DBConnection) GetTopReferrers(startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	sqlQuery := `
		SELECT referrer, COUNT(*) as visits
		FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
		AND referrer IS NOT NULL
		AND referrer != ''
		GROUP BY referrer
		ORDER BY visits DESC
		LIMIT ?
	`

	rows, err := db.Database.Query(sqlQuery, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrers []map[string]interface{}
	for rows.Next() {
		var referrer string
		var visits int
		err := rows.Scan(&referrer, &visits)
		if err != nil {
			return nil, err
		}
		referrers = append(referrers, map[string]interface{}{
			"referrer": referrer,
			"visits":   visits,
		})
	}

	return referrers, nil
}

// GetEventStats returns statistics for custom events in a date range
func (db *DBConnection) GetEventStats(startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	sqlQuery := `
		SELECT event_name, COUNT(*) as count
		FROM analytics_events
		WHERE created_at BETWEEN ? AND ?
		GROUP BY event_name
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := db.Database.Query(sqlQuery, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var eventName string
		var count int
		err := rows.Scan(&eventName, &count)
		if err != nil {
			return nil, err
		}
		events = append(events, map[string]interface{}{
			"event_name": eventName,
			"count":      count,
		})
	}

	return events, nil
}
