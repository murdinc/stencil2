package email

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/murdinc/stencil2/configs"
)

type EmailService struct {
	sesClient *ses.SES
	envConfig *configs.EnvironmentConfig
}

type EmailMessage struct {
	To          []string
	FromAddress string
	FromName    string
	ReplyTo     string
	Subject     string
	HTMLBody    string
	TextBody    string
}

// NewEmailService creates a new email service with SES
func NewEmailService(envConfig *configs.EnvironmentConfig) (*EmailService, error) {
	if envConfig.Email.Provider != "ses" {
		return nil, fmt.Errorf("only SES provider is supported")
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(envConfig.Email.SES.Region),
		Credentials: credentials.NewStaticCredentials(
			envConfig.Email.SES.AccessKeyID,
			envConfig.Email.SES.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create SES client
	sesClient := ses.New(sess)

	return &EmailService{
		sesClient: sesClient,
		envConfig: envConfig,
	}, nil
}

// SendEmail sends an email via SES
func (e *EmailService) SendEmail(msg EmailMessage) error {
	// Build from address with name
	from := msg.FromAddress
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.FromAddress)
	}

	// Prepare SES input
	input := &ses.SendEmailInput{
		Source: aws.String(from),
		Destination: &ses.Destination{
			ToAddresses: aws.StringSlice(msg.To),
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data: aws.String(msg.Subject),
			},
			Body: &ses.Body{},
		},
	}

	// Add HTML body if provided
	if msg.HTMLBody != "" {
		input.Message.Body.Html = &ses.Content{
			Data: aws.String(msg.HTMLBody),
		}
	}

	// Add text body if provided
	if msg.TextBody != "" {
		input.Message.Body.Text = &ses.Content{
			Data: aws.String(msg.TextBody),
		}
	}

	// Add reply-to if provided
	if msg.ReplyTo != "" {
		input.ReplyToAddresses = aws.StringSlice([]string{msg.ReplyTo})
	}

	// Send the email
	_, err := e.sesClient.SendEmail(input)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// SendOrderConfirmation sends an order confirmation email
func (e *EmailService) SendOrderConfirmation(siteConfig *configs.WebsiteConfig, orderNumber, customerEmail, customerName string, items []OrderItem, subtotal, tax, shipping, total float64) error {
	htmlBody := e.buildOrderConfirmationHTML(siteConfig.SiteName, orderNumber, customerName, items, subtotal, tax, shipping, total)
	textBody := e.buildOrderConfirmationText(siteConfig.SiteName, orderNumber, customerName, items, subtotal, tax, shipping, total)

	fromAddress := siteConfig.Email.FromAddress
	fromName := siteConfig.Email.FromName
	replyTo := siteConfig.Email.ReplyTo

	// Fallback to env config if site config doesn't have email settings
	if fromAddress == "" {
		fromAddress = e.envConfig.Email.Admin.FromAddress
		fromName = e.envConfig.Email.Admin.FromName
	}

	return e.SendEmail(EmailMessage{
		To:          []string{customerEmail},
		FromAddress: fromAddress,
		FromName:    fromName,
		ReplyTo:     replyTo,
		Subject:     fmt.Sprintf("Order Confirmation #%s", orderNumber),
		HTMLBody:    htmlBody,
		TextBody:    textBody,
	})
}

type OrderItem struct {
	ProductName   string
	VariantTitle  string
	Quantity      int
	Price         float64
	Total         float64
}

func (e *EmailService) buildOrderConfirmationHTML(siteName, orderNumber, customerName string, items []OrderItem, subtotal, tax, shipping, total float64) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 2px solid #000; padding-bottom: 20px; margin-bottom: 30px; }
        .order-number { font-size: 24px; font-weight: 600; margin: 10px 0; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th { text-align: left; padding: 10px; border-bottom: 1px solid #ddd; font-weight: 600; }
        td { padding: 10px; border-bottom: 1px solid #eee; }
        .totals { margin-top: 20px; }
        .totals div { display: flex; justify-content: space-between; margin: 8px 0; }
        .total-row { font-size: 18px; font-weight: 600; padding-top: 10px; border-top: 2px solid #000; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
            <div class="order-number">Order #%s</div>
        </div>

        <p>Hi %s,</p>
        <p>Thank you for your order! We've received your payment and will process your order shortly.</p>

        <table>
            <thead>
                <tr>
                    <th>Product</th>
                    <th>Qty</th>
                    <th style="text-align: right;">Price</th>
                    <th style="text-align: right;">Total</th>
                </tr>
            </thead>
            <tbody>
`, siteName, orderNumber, customerName)

	for _, item := range items {
		productName := item.ProductName
		if item.VariantTitle != "" {
			productName += fmt.Sprintf(" - %s", item.VariantTitle)
		}
		html += fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%d</td>
                    <td style="text-align: right;">$%.2f</td>
                    <td style="text-align: right;">$%.2f</td>
                </tr>
`, productName, item.Quantity, item.Price, item.Total)
	}

	html += fmt.Sprintf(`
            </tbody>
        </table>

        <div class="totals">
            <div><span>Subtotal:</span><span>$%.2f</span></div>
            <div><span>Tax:</span><span>$%.2f</span></div>
            <div><span>Shipping:</span><span>$%.2f</span></div>
            <div class="total-row"><span>Total:</span><span>$%.2f</span></div>
        </div>

        <div class="footer">
            <p>If you have any questions, please reply to this email.</p>
            <p>Order Number: %s</p>
        </div>
    </div>
</body>
</html>
`, subtotal, tax, shipping, total, orderNumber)

	return html
}

func (e *EmailService) buildOrderConfirmationText(siteName, orderNumber, customerName string, items []OrderItem, subtotal, tax, shipping, total float64) string {
	text := fmt.Sprintf(`%s

Order #%s

Hi %s,

Thank you for your order! We've received your payment and will process your order shortly.

ORDER ITEMS:
`, siteName, orderNumber, customerName)

	for _, item := range items {
		productName := item.ProductName
		if item.VariantTitle != "" {
			productName += fmt.Sprintf(" - %s", item.VariantTitle)
		}
		text += fmt.Sprintf("%s x%d - $%.2f\n", productName, item.Quantity, item.Total)
	}

	text += fmt.Sprintf(`
Subtotal: $%.2f
Tax: $%.2f
Shipping: $%.2f
Total: $%.2f

Order Number: %s

If you have any questions, please reply to this email.
`, subtotal, tax, shipping, total, orderNumber)

	return text
}
