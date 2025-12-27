package database

import (
	"fmt"
	"log"
	"time"
)

// InitMessagesTables creates the tables for contact messages and replies
func (db *DBConnection) InitMessagesTables() error {
	if !db.Connected {
		return nil
	}

	schemas := []string{
		// Contact Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			status VARCHAR(20) DEFAULT 'unread',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_status (status),
			INDEX idx_created_at (created_at),
			INDEX idx_email (email)
		)`,

		// Message Replies table
		`CREATE TABLE IF NOT EXISTS message_replies (
			id INT PRIMARY KEY AUTO_INCREMENT,
			message_id INT NOT NULL,
			reply_text TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			sent_by VARCHAR(100) DEFAULT 'admin',
			INDEX idx_message_id (message_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,
	}

	for _, schema := range schemas {
		_, err := db.Database.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create messages table: %v", err)
		}
	}

	log.Println("Messages tables initialized successfully")
	return nil
}

// CreateMessage stores a new contact form submission
func (db *DBConnection) CreateMessage(name, email, message string) error {
	query := `
		INSERT INTO messages (name, email, message, status)
		VALUES (?, ?, ?, 'unread')
	`

	_, err := db.Database.Exec(query, name, email, message)
	if err != nil {
		return fmt.Errorf("failed to create message: %v", err)
	}

	return nil
}

// GetMessage retrieves a single message by ID
func (db *DBConnection) GetMessage(messageID int) (*Message, error) {
	query := `
		SELECT id, name, email, message, status, created_at, updated_at
		FROM messages
		WHERE id = ?
	`

	var msg Message
	err := db.Database.QueryRow(query, messageID).Scan(
		&msg.ID,
		&msg.Name,
		&msg.Email,
		&msg.Message,
		&msg.Status,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// MarkMessageAsRead updates message status to 'read'
func (db *DBConnection) MarkMessageAsRead(messageID int) error {
	query := `UPDATE messages SET status = 'read' WHERE id = ?`
	_, err := db.Database.Exec(query, messageID)
	return err
}

// MarkMessageAsUnread updates message status to 'unread'
func (db *DBConnection) MarkMessageAsUnread(messageID int) error {
	query := `UPDATE messages SET status = 'unread' WHERE id = ?`
	_, err := db.Database.Exec(query, messageID)
	return err
}

// CreateReply stores a reply to a message
func (db *DBConnection) CreateReply(messageID int, replyText, sentBy string) error {
	query := `
		INSERT INTO message_replies (message_id, reply_text, sent_by)
		VALUES (?, ?, ?)
	`

	_, err := db.Database.Exec(query, messageID, replyText, sentBy)
	if err != nil {
		return fmt.Errorf("failed to create reply: %v", err)
	}

	// Mark message as read when replying
	_ = db.MarkMessageAsRead(messageID)

	return nil
}

// GetMessageReplies retrieves all replies for a message
func (db *DBConnection) GetMessageReplies(messageID int) ([]MessageReply, error) {
	query := `
		SELECT id, message_id, reply_text, sent_at, sent_by
		FROM message_replies
		WHERE message_id = ?
		ORDER BY sent_at ASC
	`

	rows, err := db.Database.Query(query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []MessageReply
	for rows.Next() {
		var reply MessageReply
		err := rows.Scan(
			&reply.ID,
			&reply.MessageID,
			&reply.ReplyText,
			&reply.SentAt,
			&reply.SentBy,
		)
		if err != nil {
			continue
		}
		replies = append(replies, reply)
	}

	return replies, nil
}

// Message represents a contact form submission
type Message struct {
	ID        int
	Name      string
	Email     string
	Message   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MessageReply represents a reply to a contact message
type MessageReply struct {
	ID        int
	MessageID int
	ReplyText string
	SentAt    time.Time
	SentBy    string
}
