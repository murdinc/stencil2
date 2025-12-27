package email

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// MessageMatcher is an interface for matching incoming emails to existing messages
type MessageMatcher interface {
	FindMessageByEmail(email string) ([]int, error)
	CreateReply(messageID int, replyText, replyFrom string) error
}

// PollResult contains the results of a polling operation
type PollResult struct {
	EmailsChecked int
	RepliesAdded  int
	Errors        []error
}

// PollIncomingEmails checks IMAP for new emails and adds them as message replies
func PollIncomingEmails(config IMAPConfig, matcher MessageMatcher) (*PollResult, error) {
	result := &PollResult{
		Errors: make([]error, 0),
	}

	// Fetch new emails from IMAP
	emails, err := FetchNewEmails(config)
	if err != nil {
		return result, fmt.Errorf("failed to fetch emails: %v", err)
	}

	result.EmailsChecked = len(emails)
	log.Printf("Found %d unread emails to process", len(emails))

	var processedMessageIDs []string

	// Process each email
	for _, incomingEmail := range emails {
		// Find messages from this sender
		messageIDs, err := matcher.FindMessageByEmail(incomingEmail.From)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error finding messages for %s: %v", incomingEmail.From, err))
			continue
		}

		if len(messageIDs) == 0 {
			log.Printf("No matching message found for email from %s (subject: %s)", incomingEmail.From, incomingEmail.Subject)
			continue
		}

		// Add reply to the most recent message from this sender
		// (messageIDs are assumed to be sorted by most recent first)
		messageID := messageIDs[0]

		// Prepare reply text
		replyText := incomingEmail.Body
		if replyText == "" {
			replyText = incomingEmail.HTMLBody
		}

		// Add metadata about the email
		replyWithMetadata := fmt.Sprintf("[Email received: %s]\n\n%s",
			incomingEmail.Date.Format("2006-01-02 15:04:05"),
			replyText)

		// Save as reply
		err = matcher.CreateReply(messageID, replyWithMetadata, "customer")
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error creating reply for message %d: %v", messageID, err))
			continue
		}

		log.Printf("Added email reply from %s to message %d", incomingEmail.From, messageID)
		result.RepliesAdded++

		// Track this message ID for marking as read
		processedMessageIDs = append(processedMessageIDs, incomingEmail.MessageID)
	}

	// Mark processed emails as read
	if len(processedMessageIDs) > 0 {
		err = MarkAsRead(config, processedMessageIDs)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error marking emails as read: %v", err))
		} else {
			log.Printf("Marked %d emails as read", len(processedMessageIDs))
		}
	}

	return result, nil
}

// DBMessageMatcher implements MessageMatcher for a SQL database
type DBMessageMatcher struct {
	DB *sql.DB
}

// FindMessageByEmail finds all message IDs for a given email address, sorted by most recent first
func (m *DBMessageMatcher) FindMessageByEmail(email string) ([]int, error) {
	// Normalize email to lowercase for case-insensitive matching
	email = strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT id
		FROM messages
		WHERE LOWER(TRIM(email)) = ?
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messageIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, id)
	}

	return messageIDs, nil
}

// CreateReply adds a reply to a message
func (m *DBMessageMatcher) CreateReply(messageID int, replyText, replyFrom string) error {
	query := `
		INSERT INTO message_replies (message_id, reply_text, reply_from, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := m.DB.Exec(query, messageID, replyText, replyFrom, time.Now())
	return err
}
