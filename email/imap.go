package email

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
)

// IMAPConfig holds IMAP connection configuration
type IMAPConfig struct {
	Server   string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

// IncomingEmail represents a parsed incoming email
type IncomingEmail struct {
	From        string
	To          string
	Subject     string
	Body        string
	HTMLBody    string
	MessageID   string
	InReplyTo   string
	References  string
	Date        time.Time
}

// FetchNewEmails connects to IMAP server and fetches unread emails
func FetchNewEmails(config IMAPConfig) ([]IncomingEmail, error) {
	// Connect to server
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)

	var c *client.Client
	var err error

	if config.UseTLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %v", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login(config.Username, config.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %v", err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %v", err)
	}

	// If there are no messages, return empty
	if mbox.Messages == 0 {
		return []IncomingEmail{}, nil
	}

	// Search for unseen messages
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	ids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}

	if len(ids) == 0 {
		return []IncomingEmail{}, nil
	}

	log.Printf("Found %d unread emails", len(ids))

	// Fetch messages
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, len(ids))
	done := make(chan error, 1)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	var emails []IncomingEmail
	for msg := range messages {
		email, err := parseMessage(msg, section)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %v", err)
	}

	return emails, nil
}

// parseMessage parses an IMAP message into an IncomingEmail
func parseMessage(msg *imap.Message, section *imap.BodySectionName) (IncomingEmail, error) {
	email := IncomingEmail{}

	if msg.Envelope != nil {
		if len(msg.Envelope.From) > 0 {
			email.From = msg.Envelope.From[0].Address()
		}
		if len(msg.Envelope.To) > 0 {
			email.To = msg.Envelope.To[0].Address()
		}
		email.Subject = msg.Envelope.Subject
		email.MessageID = msg.Envelope.MessageId
		email.InReplyTo = msg.Envelope.InReplyTo
		email.Date = msg.Envelope.Date
	}

	// Get message body
	r := msg.GetBody(section)
	if r == nil {
		return email, fmt.Errorf("message body is nil")
	}

	// Parse email message
	mr, err := mail.ReadMessage(r)
	if err != nil {
		return email, fmt.Errorf("failed to read message: %v", err)
	}

	// Extract References header
	email.References = mr.Header.Get("References")

	// Parse message body
	msgReader, err := message.Read(mr.Body)
	if err != nil {
		return email, fmt.Errorf("failed to parse message: %v", err)
	}

	// Extract text and HTML parts
	var plainText, htmlText string

	// Walk through all parts of the message
	var walkParts func(*message.Entity) error
	walkParts = func(entity *message.Entity) error {
		contentType, _, _ := entity.Header.ContentType()

		if strings.HasPrefix(contentType, "text/plain") {
			body, err := io.ReadAll(entity.Body)
			if err == nil {
				plainText = string(body)
			}
		} else if strings.HasPrefix(contentType, "text/html") {
			body, err := io.ReadAll(entity.Body)
			if err == nil {
				htmlText = string(body)
			}
		}

		// Check if multipart and walk children
		multipartReader := entity.MultipartReader()
		if multipartReader != nil {
			for {
				part, err := multipartReader.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				if err := walkParts(part); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if err := walkParts(msgReader); err != nil {
		log.Printf("Error walking message parts: %v", err)
	}

	// Prefer plain text, fall back to HTML
	if plainText != "" {
		email.Body = plainText
	} else if htmlText != "" {
		email.Body = stripHTML(htmlText)
		email.HTMLBody = htmlText
	}

	return email, nil
}

// stripHTML removes HTML tags from a string (basic implementation)
func stripHTML(html string) string {
	// Very basic HTML stripping - just removes tags
	result := html
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}

// MarkAsRead marks emails as read on the IMAP server
func MarkAsRead(config IMAPConfig, messageIDs []string) error {
	// Connect to server
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)

	var c *client.Client
	var err error

	if config.UseTLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %v", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login(config.Username, config.Password); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}

	// Select INBOX
	if _, err := c.Select("INBOX", false); err != nil {
		return fmt.Errorf("failed to select INBOX: %v", err)
	}

	// Search for messages by Message-ID
	for _, msgID := range messageIDs {
		criteria := imap.NewSearchCriteria()
		criteria.Header.Set("Message-ID", msgID)

		ids, err := c.Search(criteria)
		if err != nil || len(ids) == 0 {
			continue
		}

		seqset := new(imap.SeqSet)
		seqset.AddNum(ids...)

		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.SeenFlag}

		if err := c.Store(seqset, item, flags, nil); err != nil {
			log.Printf("Failed to mark message as read: %v", err)
		}
	}

	return nil
}
