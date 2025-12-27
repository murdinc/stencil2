package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/murdinc/stencil2/configs"
)

type EmailService struct {
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

// NewEmailService creates a new email service
func NewEmailService(envConfig *configs.EnvironmentConfig) (*EmailService, error) {
	return &EmailService{
		envConfig: envConfig,
	}, nil
}

// SendEmail sends an email via SMTP
func (e *EmailService) SendEmail(msg EmailMessage) error {
	// We'll use the admin SMTP settings from env config as fallback
	// (Website-specific SMTP is passed in via the message from/reply-to fields)

	// For now, we expect SMTP settings to be configured in the website config
	// and passed through via the WebsiteConfig in the calling functions
	// This is a simplified SMTP implementation

	return fmt.Errorf("SendEmail called without SMTP config - use SendEmailWithSMTP instead")
}

// SendEmailWithSMTP sends an email using provided SMTP configuration
func (e *EmailService) SendEmailWithSMTP(msg EmailMessage, smtpHost string, smtpPort int, username, password string, useTLS bool) error {
	// Build from header
	from := msg.FromAddress
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.FromAddress)
	}

	// Build message headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(msg.To, ", ")
	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"

	if msg.ReplyTo != "" {
		headers["Reply-To"] = msg.ReplyTo
	}

	// Build email body with multipart support
	var message string
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	// If we have both HTML and text, use multipart/alternative
	if msg.HTMLBody != "" && msg.TextBody != "" {
		boundary := "----=_Part_0_1234567890.1234567890"
		headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=\"%s\"", boundary)

		message += fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

		// Text part
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/plain; charset=UTF-8\r\n\r\n"
		message += msg.TextBody + "\r\n\r\n"

		// HTML part
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/html; charset=UTF-8\r\n\r\n"
		message += msg.HTMLBody + "\r\n\r\n"

		message += fmt.Sprintf("--%s--", boundary)
	} else if msg.HTMLBody != "" {
		message += "Content-Type: text/html; charset=UTF-8\r\n\r\n"
		message += msg.HTMLBody
	} else {
		message += "Content-Type: text/plain; charset=UTF-8\r\n\r\n"
		message += msg.TextBody
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	if useTLS {
		// TLS connection (typically port 587 with STARTTLS)
		return e.sendViaTLS(addr, username, password, msg.FromAddress, msg.To, []byte(message))
	} else {
		// Plain SMTP (not recommended, but supported)
		return e.sendViaPlain(addr, username, password, msg.FromAddress, msg.To, []byte(message))
	}
}

func (e *EmailService) sendViaTLS(addr, username, password, from string, to []string, msg []byte) error {
	// Connect to server
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %v", err)
	}

	// TLS config
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	// Connect
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Close()

	// STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %v", err)
		}
	}

	// Auth
	if username != "" && password != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %v", err)
		}
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", recipient, err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %v", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %v", err)
	}

	return client.Quit()
}

func (e *EmailService) sendViaPlain(addr, username, password, from string, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %v", err)
	}

	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	return smtp.SendMail(addr, auth, from, to, msg)
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

	return e.SendEmailWithSMTP(
		EmailMessage{
			To:          []string{customerEmail},
			FromAddress: fromAddress,
			FromName:    fromName,
			ReplyTo:     replyTo,
			Subject:     fmt.Sprintf("Order Confirmation #%s", orderNumber),
			HTMLBody:    htmlBody,
			TextBody:    textBody,
		},
		siteConfig.Email.SMTP.Server,
		siteConfig.Email.SMTP.Port,
		siteConfig.Email.SMTP.Username,
		siteConfig.Email.SMTP.Password,
		siteConfig.Email.SMTP.UseTLS,
	)
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

// SendAdminOrderNotification sends a new order notification to the admin
func (e *EmailService) SendAdminOrderNotification(siteConfig *configs.WebsiteConfig, orderNumber, customerEmail, customerName string, items []OrderItem, subtotal, tax, shipping, total float64) error {
	adminEmail := siteConfig.Email.FromAddress
	if adminEmail == "" {
		adminEmail = e.envConfig.Email.Admin.FromAddress
	}

	if adminEmail == "" {
		return fmt.Errorf("no admin email configured")
	}

	htmlBody := e.buildAdminOrderNotificationHTML(siteConfig.SiteName, orderNumber, customerName, customerEmail, items, subtotal, tax, shipping, total)
	textBody := e.buildAdminOrderNotificationText(siteConfig.SiteName, orderNumber, customerName, customerEmail, items, subtotal, tax, shipping, total)

	fromAddress := e.envConfig.Email.Admin.FromAddress
	fromName := "Store Notifications"

	if fromAddress == "" {
		fromAddress = siteConfig.Email.FromAddress
	}

	return e.SendEmailWithSMTP(
		EmailMessage{
			To:          []string{adminEmail},
			FromAddress: fromAddress,
			FromName:    fromName,
			Subject:     fmt.Sprintf("New Order #%s", orderNumber),
			HTMLBody:    htmlBody,
			TextBody:    textBody,
		},
		siteConfig.Email.SMTP.Server,
		siteConfig.Email.SMTP.Port,
		siteConfig.Email.SMTP.Username,
		siteConfig.Email.SMTP.Password,
		siteConfig.Email.SMTP.UseTLS,
	)
}

func (e *EmailService) buildAdminOrderNotificationHTML(siteName, orderNumber, customerName, customerEmail string, items []OrderItem, subtotal, tax, shipping, total float64) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 2px solid #000; padding-bottom: 20px; margin-bottom: 30px; background: #f8f9fa; padding: 20px; border-radius: 8px; }
        .order-number { font-size: 24px; font-weight: 600; margin: 10px 0; color: #48bb78; }
        .alert { background: #fff4e6; border-left: 4px solid #f59e0b; padding: 16px; margin-bottom: 20px; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th { text-align: left; padding: 10px; border-bottom: 1px solid #ddd; font-weight: 600; background: #f8f9fa; }
        td { padding: 10px; border-bottom: 1px solid #eee; }
        .totals { margin-top: 20px; }
        .totals div { display: flex; justify-content: space-between; margin: 8px 0; }
        .total-row { font-size: 18px; font-weight: 600; padding-top: 10px; border-top: 2px solid #000; }
        .customer-info { background: #f8f9fa; padding: 16px; border-radius: 8px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="alert">
            <strong>⚡ Action Required:</strong> New order received and payment confirmed!
        </div>

        <div class="header">
            <h1>%s</h1>
            <div class="order-number">Order #%s</div>
        </div>

        <div class="customer-info">
            <h3 style="margin-top: 0;">Customer Information</h3>
            <p><strong>Name:</strong> %s</p>
            <p><strong>Email:</strong> %s</p>
        </div>

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
`, siteName, orderNumber, customerName, customerEmail)

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

        <div style="margin-top: 30px; padding: 20px; background: #e6ffed; border-radius: 8px; border: 1px solid #48bb78;">
            <p style="margin: 0; font-weight: 600;">✓ Payment confirmed via Stripe</p>
            <p style="margin: 8px 0 0 0; font-size: 14px; color: #666;">Log in to your admin panel to process this order.</p>
        </div>
    </div>
</body>
</html>
`, subtotal, tax, shipping, total)

	return html
}

func (e *EmailService) buildAdminOrderNotificationText(siteName, orderNumber, customerName, customerEmail string, items []OrderItem, subtotal, tax, shipping, total float64) string {
	text := fmt.Sprintf(`NEW ORDER RECEIVED
%s

Order #%s

CUSTOMER:
Name: %s
Email: %s

ORDER ITEMS:
`, siteName, orderNumber, customerName, customerEmail)

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

Payment confirmed via Stripe.
Log in to your admin panel to process this order.
`, subtotal, tax, shipping, total)

	return text
}

// SendShippingConfirmation sends a shipping confirmation email to the customer
func (e *EmailService) SendShippingConfirmation(siteConfig *configs.WebsiteConfig, orderNumber, customerEmail, customerName, trackingNumber, carrier string) error {
	htmlBody := e.buildShippingConfirmationHTML(siteConfig.SiteName, orderNumber, customerName, trackingNumber, carrier)
	textBody := e.buildShippingConfirmationText(siteConfig.SiteName, orderNumber, customerName, trackingNumber, carrier)

	fromAddress := siteConfig.Email.FromAddress
	fromName := siteConfig.Email.FromName
	replyTo := siteConfig.Email.ReplyTo

	if fromAddress == "" {
		fromAddress = e.envConfig.Email.Admin.FromAddress
		fromName = e.envConfig.Email.Admin.FromName
	}

	return e.SendEmailWithSMTP(
		EmailMessage{
			To:          []string{customerEmail},
			FromAddress: fromAddress,
			FromName:    fromName,
			ReplyTo:     replyTo,
			Subject:     fmt.Sprintf("Your Order #%s Has Shipped!", orderNumber),
			HTMLBody:    htmlBody,
			TextBody:    textBody,
		},
		siteConfig.Email.SMTP.Server,
		siteConfig.Email.SMTP.Port,
		siteConfig.Email.SMTP.Username,
		siteConfig.Email.SMTP.Password,
		siteConfig.Email.SMTP.UseTLS,
	)
}

func (e *EmailService) buildShippingConfirmationHTML(siteName, orderNumber, customerName, trackingNumber, carrier string) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 2px solid #000; padding-bottom: 20px; margin-bottom: 30px; }
        .order-number { font-size: 20px; font-weight: 600; margin: 10px 0; }
        .shipping-box { background: #e6ffed; border: 2px solid #48bb78; border-radius: 8px; padding: 24px; margin: 24px 0; text-align: center; }
        .tracking { font-size: 24px; font-weight: 700; color: #48bb78; font-family: monospace; margin: 16px 0; }
        .info-box { background: #f8f9fa; padding: 16px; border-radius: 8px; margin: 16px 0; }
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
        <p>Great news! Your order has been shipped and is on its way to you.</p>

        <div class="shipping-box">
            <div style="font-size: 48px; margin-bottom: 16px;">📦</div>
            <h2 style="margin: 0 0 8px 0; color: #48bb78;">Your Order Has Shipped!</h2>
            <p style="margin: 8px 0; color: #666;">Carrier: <strong>%s</strong></p>
            <p style="margin: 8px 0; color: #666;">Tracking Number:</p>
            <div class="tracking">%s</div>
        </div>

        <div class="footer">
            <p>You can use the tracking number above to monitor your delivery status with %s.</p>
            <p>If you have any questions, please reply to this email.</p>
            <p>Order Number: %s</p>
        </div>
    </div>
</body>
</html>
`, siteName, orderNumber, customerName, carrier, trackingNumber, carrier, orderNumber)

	return html
}

func (e *EmailService) buildShippingConfirmationText(siteName, orderNumber, customerName, trackingNumber, carrier string) string {
	return fmt.Sprintf(`%s

Order #%s

Hi %s,

Great news! Your order has shipped and is on its way to you.

SHIPPING DETAILS:
Carrier: %s
Tracking Number: %s

You can use the tracking number above to monitor your delivery status with %s.

If you have any questions, please reply to this email.

Order Number: %s
`, siteName, orderNumber, customerName, carrier, trackingNumber, carrier, orderNumber)
}

// SendDeliveryConfirmation sends a delivery confirmation email to the customer
func (e *EmailService) SendDeliveryConfirmation(siteConfig *configs.WebsiteConfig, orderNumber, customerEmail, customerName string) error {
	htmlBody := e.buildDeliveryConfirmationHTML(siteConfig.SiteName, orderNumber, customerName)
	textBody := e.buildDeliveryConfirmationText(siteConfig.SiteName, orderNumber, customerName)

	fromAddress := siteConfig.Email.FromAddress
	fromName := siteConfig.Email.FromName
	replyTo := siteConfig.Email.ReplyTo

	if fromAddress == "" {
		fromAddress = e.envConfig.Email.Admin.FromAddress
		fromName = e.envConfig.Email.Admin.FromName
	}

	return e.SendEmailWithSMTP(
		EmailMessage{
			To:          []string{customerEmail},
			FromAddress: fromAddress,
			FromName:    fromName,
			ReplyTo:     replyTo,
			Subject:     fmt.Sprintf("Your Order #%s Has Been Delivered!", orderNumber),
			HTMLBody:    htmlBody,
			TextBody:    textBody,
		},
		siteConfig.Email.SMTP.Server,
		siteConfig.Email.SMTP.Port,
		siteConfig.Email.SMTP.Username,
		siteConfig.Email.SMTP.Password,
		siteConfig.Email.SMTP.UseTLS,
	)
}

func (e *EmailService) buildDeliveryConfirmationHTML(siteName, orderNumber, customerName string) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 2px solid #000; padding-bottom: 20px; margin-bottom: 30px; }
        .order-number { font-size: 20px; font-weight: 600; margin: 10px 0; }
        .delivery-box { background: linear-gradient(135deg, #48bb78 0%%, #38a169 100%%); color: white; border-radius: 8px; padding: 32px; margin: 24px 0; text-align: center; }
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

        <div class="delivery-box">
            <div style="font-size: 64px; margin-bottom: 16px;">✓</div>
            <h2 style="margin: 0 0 8px 0; color: white;">Delivered!</h2>
            <p style="margin: 8px 0; color: rgba(255,255,255,0.9); font-size: 18px;">Your order has been delivered</p>
        </div>

        <p>We hope you love your purchase! If you have any questions or concerns, please don't hesitate to reach out.</p>

        <div class="footer">
            <p>Thank you for shopping with us!</p>
            <p>Order Number: %s</p>
        </div>
    </div>
</body>
</html>
`, siteName, orderNumber, customerName, orderNumber)

	return html
}

func (e *EmailService) buildDeliveryConfirmationText(siteName, orderNumber, customerName string) string {
	return fmt.Sprintf(`%s

Order #%s

Hi %s,

Your order has been delivered!

We hope you love your purchase! If you have any questions or concerns, please don't hesitate to reach out.

Thank you for shopping with us!

Order Number: %s
`, siteName, orderNumber, customerName, orderNumber)
}
