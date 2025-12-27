package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig holds SMTP connection configuration
type SMTPConfig struct {
	Server   string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

// OutgoingEmail represents an email to be sent
type OutgoingEmail struct {
	From        string
	FromName    string
	To          string
	ReplyTo     string
	Subject     string
	Body        string
	HTMLBody    string
	InReplyTo   string
	References  string
}

// SendEmail sends an email via SMTP
func SendEmail(config SMTPConfig, email OutgoingEmail) error {
	// Build email headers
	headers := make(map[string]string)

	// From header with name
	if email.FromName != "" {
		headers["From"] = fmt.Sprintf("%s <%s>", email.FromName, email.From)
	} else {
		headers["From"] = email.From
	}

	headers["To"] = email.To
	headers["Subject"] = email.Subject
	headers["MIME-Version"] = "1.0"

	// Add threading headers for email replies
	if email.InReplyTo != "" {
		headers["In-Reply-To"] = email.InReplyTo
	}
	if email.References != "" {
		headers["References"] = email.References
	}
	if email.ReplyTo != "" {
		headers["Reply-To"] = email.ReplyTo
	}

	// Determine content type based on what we have
	var body string
	if email.HTMLBody != "" && email.Body != "" {
		// Send multipart message with both plain text and HTML
		boundary := "boundary-string-12345"
		headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=\"%s\"", boundary)

		body = fmt.Sprintf("--%s\r\n", boundary)
		body += "Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n"
		body += email.Body + "\r\n\r\n"
		body += fmt.Sprintf("--%s\r\n", boundary)
		body += "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
		body += email.HTMLBody + "\r\n\r\n"
		body += fmt.Sprintf("--%s--", boundary)
	} else if email.HTMLBody != "" {
		// HTML only
		headers["Content-Type"] = "text/html; charset=\"UTF-8\""
		body = email.HTMLBody
	} else {
		// Plain text only
		headers["Content-Type"] = "text/plain; charset=\"UTF-8\""
		body = email.Body
	}

	// Build message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)

	// Authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Server)

	// Send email
	err := smtp.SendMail(addr, auth, email.From, []string{email.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// SendReplyEmail sends a reply email that will thread properly
func SendReplyEmail(config SMTPConfig, originalEmail IncomingEmail, replyBody string, fromAddress, fromName string) error {
	// Build References header for proper threading
	references := originalEmail.References
	if references != "" && originalEmail.MessageID != "" {
		references = references + " " + originalEmail.MessageID
	} else if originalEmail.MessageID != "" {
		references = originalEmail.MessageID
	}

	// Build subject with Re: prefix if not already present
	subject := originalEmail.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	email := OutgoingEmail{
		From:       fromAddress,
		FromName:   fromName,
		To:         originalEmail.From,
		ReplyTo:    fromAddress,
		Subject:    subject,
		Body:       replyBody,
		InReplyTo:  originalEmail.MessageID,
		References: references,
	}

	return SendEmail(config, email)
}
