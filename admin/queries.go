package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/shippo"
	"github.com/murdinc/stencil2/structs"
)

// Website represents a managed website
type Website struct {
	ID            string    `json:"id"` // Using database name as ID
	SiteName      string    `json:"siteName"`
	Directory     string    `json:"directory"`
	DatabaseName  string    `json:"databaseName"`
	HTTPAddress   string    `json:"httpAddress"`
	MediaProxyURL string    `json:"mediaProxyUrl"`
	APIVersion    int       `json:"apiVersion"`
	Timezone      string    `json:"timezone"` // IANA timezone (e.g., "America/Los_Angeles")

	// Stripe
	StripePublishableKey string `json:"stripePublishableKey"`
	StripeSecretKey      string `json:"stripeSecretKey"`

	// Shippo
	ShippoAPIKey  string `json:"shippoApiKey"`
	LabelFormat   string `json:"labelFormat"` // PDF, PDF_4x6, ZPLII, PNG

	// Twilio
	TwilioAccountSID string `json:"twilioAccountSid"`
	TwilioAuthToken  string `json:"twilioAuthToken"`
	TwilioFromPhone  string `json:"twilioFromPhone"`

	// Email
	EmailFromAddress string `json:"emailFromAddress"`
	EmailFromName    string `json:"emailFromName"`
	EmailReplyTo     string `json:"emailReplyTo"`

	// IMAP (for receiving emails)
	IMAPServer   string `json:"imapServer"`
	IMAPPort     int    `json:"imapPort"`
	IMAPUsername string `json:"imapUsername"`
	IMAPPassword string `json:"imapPassword"`
	IMAPUseTLS   bool   `json:"imapUseTLS"`

	// SMTP (for sending emails)
	SMTPServer   string `json:"smtpServer"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUsername string `json:"smtpUsername"`
	SMTPPassword string `json:"smtpPassword"`
	SMTPUseTLS   bool   `json:"smtpUseTLS"`

	// Ecommerce
	TaxRate      float64 `json:"taxRate"`
	ShippingCost float64 `json:"shippingCost"`

	// Early Access
	EarlyAccessEnabled  bool   `json:"earlyAccessEnabled"`
	EarlyAccessPassword string `json:"earlyAccessPassword"`

	// ShipFrom Address
	ShipFromName    string `json:"shipFromName"`
	ShipFromStreet1 string `json:"shipFromStreet1"`
	ShipFromStreet2 string `json:"shipFromStreet2"`
	ShipFromCity    string `json:"shipFromCity"`
	ShipFromState   string `json:"shipFromState"`
	ShipFromZip     string `json:"shipFromZip"`
	ShipFromCountry string `json:"shipFromCountry"`

	// SEO
	RobotsTxt string `json:"robotsTxt"`

	// Branding
	Logo string `json:"logo"` // Path or URL to site logo for packing slips

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Article represents an article/post
type Article struct {
	ID            int       `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Content       string    `json:"content"`
	Excerpt       string    `json:"excerpt"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	ThumbnailID   int       `json:"thumbnailId"`
	PublishedDate time.Time `json:"publishedDate"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Product represents an e-commerce product
type Product struct {
	ID                int                      `json:"id"`
	Name              string                   `json:"name"`
	Slug              string                   `json:"slug"`
	Description       string                   `json:"description"`
	Price             float64                  `json:"price"`
	CompareAtPrice    float64                  `json:"compareAtPrice"`
	SKU               string                   `json:"sku"`
	InventoryQuantity int                      `json:"inventoryQuantity"`
	InventoryPolicy   string                   `json:"inventoryPolicy"`
	Status            string                   `json:"status"`
	Featured          bool                     `json:"featured"`
	SortOrder         int                      `json:"sortOrder"`
	ReleasedDate      time.Time                `json:"releasedDate"`
	CreatedAt         time.Time                `json:"createdAt"`
	UpdatedAt         time.Time                `json:"updatedAt"`
	Variants          []structs.ProductVariant `json:"variants"`
}

// Category represents an article category
type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Collection represents a product collection
type Collection struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ImageID     int       `json:"imageId"`
	SortOrder   int       `json:"sortOrder"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Image represents an uploaded image
type Image struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	AltText   string    `json:"altText"`
	Credit    string    `json:"credit"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

// Order represents a customer order
type Order struct {
	ID                   int       `json:"id"`
	OrderNumber          string    `json:"orderNumber"`
	CustomerEmail        string    `json:"customerEmail"`
	CustomerName         string    `json:"customerName"`
	ShippingAddressLine1 string    `json:"shippingAddressLine1"`
	ShippingAddressLine2 string    `json:"shippingAddressLine2"`
	ShippingCity         string    `json:"shippingCity"`
	ShippingState        string    `json:"shippingState"`
	ShippingZip          string    `json:"shippingZip"`
	ShippingCountry      string    `json:"shippingCountry"`
	Subtotal             float64   `json:"subtotal"`
	Tax                  float64   `json:"tax"`
	ShippingCost         float64   `json:"shippingCost"`
	Total                float64   `json:"total"`
	PaymentStatus        string    `json:"paymentStatus"`
	FulfillmentStatus    string    `json:"fulfillmentStatus"`
	PaymentMethod        string    `json:"paymentMethod"`
	ShippingLabelCost    *float64  `json:"shippingLabelCost"`
	TrackingNumber       string    `json:"trackingNumber"`
	ShippingCarrier      string    `json:"shippingCarrier"`
	ShippingLabelURL     string    `json:"shippingLabelUrl"`
	Items                []OrderItem `json:"items"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// OrderItem represents a line item in an order
type OrderItem struct {
	ID           int     `json:"id"`
	ProductID    int     `json:"productId"`
	ProductName  string  `json:"productName"`
	VariantTitle string  `json:"variantTitle"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
	Total        float64 `json:"total"`
}

// ProductImageData represents a product-specific image (not shared with articles)
type ProductImageData struct {
	ID        int       `json:"id"`
	ProductID int       `json:"productId"`
	URL       string    `json:"url"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	AltText   string    `json:"altText"`
	Credit    string    `json:"credit"`
	Size      int64     `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
}

// OrderFilters represents filters for order queries
type OrderFilters struct {
	PaymentStatus     string
	FulfillmentStatus string
	Sort              string
}

// CustomerFilters represents filters for customer queries
type CustomerFilters struct {
	Sort string // total_desc, total_asc, orders_desc, date_desc, date_asc
}

// Customer represents a customer with aggregate statistics
type Customer struct {
	ID               int        `json:"id"`
	Email            string     `json:"email"`
	StripeCustomerID string     `json:"stripeCustomerId"`
	FirstName        string     `json:"firstName"`
	LastName         string     `json:"lastName"`
	Phone            string     `json:"phone"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	// Aggregate fields
	OrderCount int        `json:"orderCount"`
	TotalSpent float64    `json:"totalSpent"`
	FirstOrder *time.Time `json:"firstOrderDate"`
	LastOrder  *time.Time `json:"lastOrderDate"`
}

type SMSSignup struct {
	ID          int       `json:"id"`
	CountryCode string    `json:"countryCode"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SMSSignupFilters struct {
	CountryCode string
	Source      string
	DateFrom    string
	DateTo      string
	Sort        string
}

// GetAllWebsites retrieves all websites from disk
func (s *AdminServer) GetAllWebsites() ([]Website, error) {
	websites := []Website{}

	// Determine config file name based on prod mode
	configName := "config-dev.json"
	if s.EnvConfig.ProdMode {
		configName = "config-prod.json"
	}

	// Walk the websites directory
	err := filepath.Walk("websites", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for config files
		if !info.IsDir() && info.Name() == configName {
			// Read the config file
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			var config struct {
				SiteName      string `json:"siteName"`
				APIVersion    int    `json:"apiVersion"`
				Timezone      string `json:"timezone"`
				Database struct {
					Name string `json:"name"`
				} `json:"database"`
				MediaProxyURL string `json:"mediaProxyUrl"`
				HTTP          struct {
					Address string `json:"address"`
				} `json:"http"`
				Stripe struct {
					PublishableKey string `json:"publishableKey"`
					SecretKey      string `json:"secretKey"`
				} `json:"stripe"`
				Shippo struct {
					APIKey      string `json:"apiKey"`
					LabelFormat string `json:"labelFormat"`
				} `json:"shippo"`
				Twilio struct {
					AccountSID string `json:"accountSid"`
					AuthToken  string `json:"authToken"`
					FromPhone  string `json:"fromPhone"`
				} `json:"twilio"`
				Email struct {
					FromAddress string `json:"fromAddress"`
					FromName    string `json:"fromName"`
					ReplyTo     string `json:"replyTo"`
					IMAP        struct {
						Server   string `json:"server"`
						Port     int    `json:"port"`
						Username string `json:"username"`
						Password string `json:"password"`
						UseTLS   bool   `json:"useTLS"`
					} `json:"imap"`
					SMTP struct {
						Server   string `json:"server"`
						Port     int    `json:"port"`
						Username string `json:"username"`
						Password string `json:"password"`
						UseTLS   bool   `json:"useTLS"`
					} `json:"smtp"`
				} `json:"email"`
				Ecommerce struct {
					TaxRate      float64 `json:"taxRate"`
					ShippingCost float64 `json:"shippingCost"`
				} `json:"ecommerce"`
				EarlyAccess struct {
					Enabled  bool   `json:"enabled"`
					Password string `json:"password"`
				} `json:"earlyAccess"`
				ShipFrom struct {
					Name    string `json:"name"`
					Street1 string `json:"street1"`
					Street2 string `json:"street2"`
					City    string `json:"city"`
					State   string `json:"state"`
					Zip     string `json:"zip"`
					Country string `json:"country"`
				} `json:"shipFrom"`
				RobotsTxt string `json:"robotsTxt"`
				Logo      string `json:"logo"`
			}

			if err := json.Unmarshal(data, &config); err != nil {
				return nil // Skip invalid JSON
			}

			// Default timezone to PST if not set
			if config.Timezone == "" {
				config.Timezone = "America/Los_Angeles"
			}

			// Extract directory (parent of config file)
			dir := filepath.Dir(path)
			relDir, _ := filepath.Rel("websites", dir)

			website := Website{
				ID:            config.Database.Name, // Use database name as ID
				SiteName:      config.SiteName,
				Directory:     relDir,
				DatabaseName:  config.Database.Name,
				HTTPAddress:   config.HTTP.Address,
				MediaProxyURL: config.MediaProxyURL,
				APIVersion:    config.APIVersion,
				Timezone:      config.Timezone,

				StripePublishableKey: config.Stripe.PublishableKey,
				StripeSecretKey:      config.Stripe.SecretKey,

				ShippoAPIKey: config.Shippo.APIKey,
				LabelFormat:  config.Shippo.LabelFormat,

				TwilioAccountSID: config.Twilio.AccountSID,
				TwilioAuthToken:  config.Twilio.AuthToken,
				TwilioFromPhone:  config.Twilio.FromPhone,

				EmailFromAddress: config.Email.FromAddress,
				EmailFromName:    config.Email.FromName,
				EmailReplyTo:     config.Email.ReplyTo,

				IMAPServer:   config.Email.IMAP.Server,
				IMAPPort:     config.Email.IMAP.Port,
				IMAPUsername: config.Email.IMAP.Username,
				IMAPPassword: config.Email.IMAP.Password,
				IMAPUseTLS:   config.Email.IMAP.UseTLS,

				SMTPServer:   config.Email.SMTP.Server,
				SMTPPort:     config.Email.SMTP.Port,
				SMTPUsername: config.Email.SMTP.Username,
				SMTPPassword: config.Email.SMTP.Password,
				SMTPUseTLS:   config.Email.SMTP.UseTLS,

				TaxRate:      config.Ecommerce.TaxRate,
				ShippingCost: config.Ecommerce.ShippingCost,

				EarlyAccessEnabled:  config.EarlyAccess.Enabled,
				EarlyAccessPassword: config.EarlyAccess.Password,

				ShipFromName:    config.ShipFrom.Name,
				ShipFromStreet1: config.ShipFrom.Street1,
				ShipFromStreet2: config.ShipFrom.Street2,
				ShipFromCity:    config.ShipFrom.City,
				ShipFromState:   config.ShipFrom.State,
				ShipFromZip:     config.ShipFrom.Zip,
				ShipFromCountry: config.ShipFrom.Country,

				RobotsTxt: config.RobotsTxt,
				Logo:      config.Logo,
			}

			websites = append(websites, website)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return websites, nil
}

// GetWebsite retrieves a single website by ID (which is the database name)
func (s *AdminServer) GetWebsite(id string) (Website, error) {
	websites, err := s.GetAllWebsites()
	if err != nil {
		return Website{}, err
	}

	for _, website := range websites {
		if website.ID == id {
			return website, nil
		}
	}

	return Website{}, fmt.Errorf("website not found")
}

// GetWebsiteByDatabase retrieves a single website by database name (same as GetWebsite now)
func (s *AdminServer) GetWebsiteByDatabase(dbName string) (Website, error) {
	return s.GetWebsite(dbName)
}

// CreateWebsite creates a new website on disk (no database needed)
func (s *AdminServer) CreateWebsite(w Website) (int64, error) {
	// Website creation is handled in the handler by creating files directly
	// This function just returns success
	websites, _ := s.GetAllWebsites()
	return int64(len(websites) + 1), nil
}

// UpdateWebsite updates an existing website config file
func (s *AdminServer) UpdateWebsite(w Website) error {
	// Determine config file name
	configName := "config-dev.json"
	if s.EnvConfig.ProdMode {
		configName = "config-prod.json"
	}

	configPath := filepath.Join("websites", w.Directory, configName)

	// Read existing config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update fields
	config["siteName"] = w.SiteName
	config["apiVersion"] = w.APIVersion
	config["timezone"] = w.Timezone

	if config["database"] == nil {
		config["database"] = make(map[string]interface{})
	}
	config["database"].(map[string]interface{})["name"] = w.DatabaseName

	if config["http"] == nil {
		config["http"] = make(map[string]interface{})
	}
	config["http"].(map[string]interface{})["address"] = w.HTTPAddress

	if w.MediaProxyURL != "" {
		config["mediaProxyUrl"] = w.MediaProxyURL
	}

	// Stripe
	if config["stripe"] == nil {
		config["stripe"] = make(map[string]interface{})
	}
	config["stripe"].(map[string]interface{})["publishableKey"] = w.StripePublishableKey
	config["stripe"].(map[string]interface{})["secretKey"] = w.StripeSecretKey

	// Shippo
	if config["shippo"] == nil {
		config["shippo"] = make(map[string]interface{})
	}
	config["shippo"].(map[string]interface{})["apiKey"] = w.ShippoAPIKey
	if w.LabelFormat != "" {
		config["shippo"].(map[string]interface{})["labelFormat"] = w.LabelFormat
	}

	// Twilio
	if config["twilio"] == nil {
		config["twilio"] = make(map[string]interface{})
	}
	config["twilio"].(map[string]interface{})["accountSid"] = w.TwilioAccountSID
	config["twilio"].(map[string]interface{})["authToken"] = w.TwilioAuthToken
	config["twilio"].(map[string]interface{})["fromPhone"] = w.TwilioFromPhone

	// Email
	if config["email"] == nil {
		config["email"] = make(map[string]interface{})
	}
	config["email"].(map[string]interface{})["fromAddress"] = w.EmailFromAddress
	config["email"].(map[string]interface{})["fromName"] = w.EmailFromName
	config["email"].(map[string]interface{})["replyTo"] = w.EmailReplyTo

	// IMAP
	emailMap := config["email"].(map[string]interface{})
	if emailMap["imap"] == nil {
		emailMap["imap"] = make(map[string]interface{})
	}
	emailMap["imap"].(map[string]interface{})["server"] = w.IMAPServer
	emailMap["imap"].(map[string]interface{})["port"] = w.IMAPPort
	emailMap["imap"].(map[string]interface{})["username"] = w.IMAPUsername
	emailMap["imap"].(map[string]interface{})["password"] = w.IMAPPassword
	emailMap["imap"].(map[string]interface{})["useTLS"] = w.IMAPUseTLS

	// SMTP
	if emailMap["smtp"] == nil {
		emailMap["smtp"] = make(map[string]interface{})
	}
	emailMap["smtp"].(map[string]interface{})["server"] = w.SMTPServer
	emailMap["smtp"].(map[string]interface{})["port"] = w.SMTPPort
	emailMap["smtp"].(map[string]interface{})["username"] = w.SMTPUsername
	emailMap["smtp"].(map[string]interface{})["password"] = w.SMTPPassword
	emailMap["smtp"].(map[string]interface{})["useTLS"] = w.SMTPUseTLS

	// Ecommerce
	if config["ecommerce"] == nil {
		config["ecommerce"] = make(map[string]interface{})
	}
	config["ecommerce"].(map[string]interface{})["taxRate"] = w.TaxRate
	config["ecommerce"].(map[string]interface{})["shippingCost"] = w.ShippingCost

	// Early Access
	if config["earlyAccess"] == nil {
		config["earlyAccess"] = make(map[string]interface{})
	}
	config["earlyAccess"].(map[string]interface{})["enabled"] = w.EarlyAccessEnabled
	config["earlyAccess"].(map[string]interface{})["password"] = w.EarlyAccessPassword

	// ShipFrom
	if config["shipFrom"] == nil {
		config["shipFrom"] = make(map[string]interface{})
	}
	config["shipFrom"].(map[string]interface{})["name"] = w.ShipFromName
	config["shipFrom"].(map[string]interface{})["street1"] = w.ShipFromStreet1
	config["shipFrom"].(map[string]interface{})["street2"] = w.ShipFromStreet2
	config["shipFrom"].(map[string]interface{})["city"] = w.ShipFromCity
	config["shipFrom"].(map[string]interface{})["state"] = w.ShipFromState
	config["shipFrom"].(map[string]interface{})["zip"] = w.ShipFromZip
	config["shipFrom"].(map[string]interface{})["country"] = w.ShipFromCountry

	// robots.txt
	if w.RobotsTxt != "" {
		config["robotsTxt"] = w.RobotsTxt
	}

	// Logo
	if w.Logo != "" {
		config["logo"] = w.Logo
	}

	// Write back to disk
	updatedData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, updatedData, 0644)
}

// DeleteWebsite deletes a website directory
func (s *AdminServer) DeleteWebsite(id string) error {
	website, err := s.GetWebsite(id)
	if err != nil {
		return err
	}

	websiteDir := filepath.Join("websites", website.Directory)
	return os.RemoveAll(websiteDir)
}

// GetWebsiteConnection gets a database connection for a specific website by ID (database name)
func (s *AdminServer) GetWebsiteConnection(websiteID string) (*sql.DB, error) {
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	connectionString := s.EnvConfig.Database.User + ":" + s.EnvConfig.Database.Password +
		"@tcp(" + s.EnvConfig.Database.Host + ":" + s.EnvConfig.Database.Port + ")/" +
		website.DatabaseName + "?parseTime=true"

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// GetWebsiteConnectionByDB gets a database connection for a specific website by database name
func (s *AdminServer) GetWebsiteConnectionByDB(dbName string) (*sql.DB, error) {
	connectionString := s.EnvConfig.Database.User + ":" + s.EnvConfig.Database.Password +
		"@tcp(" + s.EnvConfig.Database.Host + ":" + s.EnvConfig.Database.Port + ")/" +
		dbName + "?parseTime=true"

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// timezoneToOffset converts IANA timezone names to UTC offsets for MySQL
func timezoneToOffset(tz string) string {
	offsets := map[string]string{
		"America/Los_Angeles": "-08:00", // PST (standard time)
		"America/Denver":      "-07:00", // MST
		"America/Chicago":     "-06:00", // CST
		"America/New_York":    "-05:00", // EST
		"America/Anchorage":   "-09:00", // AKST
		"Pacific/Honolulu":    "-10:00", // HST
		"UTC":                 "+00:00",
	}
	if offset, ok := offsets[tz]; ok {
		return offset
	}
	return "-08:00" // Default to PST
}

// GetAnalyticsTimeSeries gets daily pageviews, unique visitors, and revenue for charting
func (s *AdminServer) GetAnalyticsTimeSeries(websiteID string, startDate, endDate time.Time, timezone string) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Default to PST if no timezone specified
	if timezone == "" {
		timezone = "America/Los_Angeles"
	}

	// Convert timezone name to UTC offset for MySQL compatibility
	offset := timezoneToOffset(timezone)

	// Build query with dynamic timezone conversion
	// Convert UTC timestamps to user's timezone before extracting dates
	query := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(dates.date, '%%Y-%%m-%%d') as date,
			COALESCE(a.pageviews, 0) as pageviews,
			COALESCE(a.visitors, 0) as visitors,
			COALESCE(a.sessions, 0) as sessions,
			COALESCE(o.revenue, 0) as revenue
		FROM (
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM analytics_pageviews
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			UNION
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM orders
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
		) dates
		LEFT JOIN (
			SELECT
				DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
				COUNT(*) as pageviews,
				COUNT(DISTINCT visitor_id) as visitors,
				COUNT(DISTINCT session_id) as sessions
			FROM analytics_pageviews
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			GROUP BY DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
		) a ON dates.date = a.date
		LEFT JOIN (
			SELECT
				DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
				SUM(total) as revenue
			FROM orders
			WHERE payment_status = 'paid'
				AND DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			GROUP BY DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
		) o ON dates.date = o.date
		ORDER BY dates.date ASC
	`, offset, offset, offset, offset, offset, offset, offset, offset, offset, offset)

	rows, err := db.Query(query, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a map of actual data indexed by date
	dataMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var date string
		var pageviews, visitors, sessions int
		var revenue float64
		err := rows.Scan(&date, &pageviews, &visitors, &sessions, &revenue)
		if err != nil {
			continue
		}
		dataMap[date] = map[string]interface{}{
			"date":      date,
			"pageviews": pageviews,
			"visitors":  visitors,
			"sessions":  sessions,
			"revenue":   revenue,
		}
	}

	// Generate complete date range with zeros for missing days
	// Normalize to midnight for clean date iteration
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var results []map[string]interface{}
	currentDate := start
	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")

		if data, exists := dataMap[dateStr]; exists {
			// Use actual data
			results = append(results, data)
		} else {
			// Fill with zeros
			results = append(results, map[string]interface{}{
				"date":      dateStr,
				"pageviews": 0,
				"visitors":  0,
				"sessions":  0,
				"revenue":   0.0,
			})
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return results, nil
}

// GetEngagementTimeSeries gets daily order count, avg pages per visit, and avg time on site
func (s *AdminServer) GetEngagementTimeSeries(websiteID string, startDate, endDate time.Time, timezone string) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Default to PST if no timezone specified
	if timezone == "" {
		timezone = "America/Los_Angeles"
	}

	// Convert timezone name to UTC offset for MySQL compatibility
	offset := timezoneToOffset(timezone)

	// Query for engagement metrics
	query := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(dates.date, '%%Y-%%m-%%d') as date,
			COALESCE(o.paid_orders, 0) as paid_orders,
			COALESCE(o.pending_orders, 0) as pending_orders,
			COALESCE(e.avg_pages_per_visit, 0) as avg_pages_per_visit,
			COALESCE(e.avg_time_on_site, 0) as avg_time_on_site
		FROM (
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM analytics_pageviews
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			UNION
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM orders
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
		) dates
		LEFT JOIN (
			SELECT
				DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
				SUM(CASE WHEN payment_status = 'paid' THEN 1 ELSE 0 END) as paid_orders,
				SUM(CASE WHEN payment_status = 'pending' THEN 1 ELSE 0 END) as pending_orders
			FROM orders
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			GROUP BY DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
		) o ON dates.date = o.date
		LEFT JOIN (
			SELECT
				date,
				ROUND(COUNT(*) / NULLIF(COUNT(DISTINCT session_id), 0), 2) as avg_pages_per_visit,
				ROUND(AVG(total_time), 0) as avg_time_on_site
			FROM (
				SELECT
					session_id,
					DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
					SUM(time_on_page) as total_time
				FROM analytics_pageviews
				WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
				GROUP BY session_id, DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
			) sessions
			GROUP BY date
		) e ON dates.date = e.date
		ORDER BY dates.date ASC
	`, offset, offset, offset, offset, offset, offset, offset, offset, offset, offset)

	rows, err := db.Query(query, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a map of actual data indexed by date
	dataMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var date string
		var paidOrders, pendingOrders int
		var avgPagesPerVisit, avgTimeOnSite float64
		err := rows.Scan(&date, &paidOrders, &pendingOrders, &avgPagesPerVisit, &avgTimeOnSite)
		if err != nil {
			continue
		}
		dataMap[date] = map[string]interface{}{
			"date":                date,
			"paid_orders":         paidOrders,
			"pending_orders":      pendingOrders,
			"avg_pages_per_visit": avgPagesPerVisit,
			"avg_time_on_site":    avgTimeOnSite,
		}
	}

	// Generate complete date range with zeros for missing days
	// Normalize to midnight for clean date iteration
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var results []map[string]interface{}
	currentDate := start
	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")

		if data, exists := dataMap[dateStr]; exists {
			// Use actual data
			results = append(results, data)
		} else {
			// Fill with zeros
			results = append(results, map[string]interface{}{
				"date":                dateStr,
				"paid_orders":         0,
				"pending_orders":      0,
				"avg_pages_per_visit": 0.0,
				"avg_time_on_site":    0.0,
			})
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return results, nil
}

// GetGrowthTimeSeries gets daily new customers and SMS signups for charting
func (s *AdminServer) GetGrowthTimeSeries(websiteID string, startDate, endDate time.Time, timezone string) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Default to PST if no timezone specified
	if timezone == "" {
		timezone = "America/Los_Angeles"
	}

	// Convert timezone name to UTC offset for MySQL compatibility
	offset := timezoneToOffset(timezone)

	// Query for growth metrics
	query := fmt.Sprintf(`
		SELECT
			DATE_FORMAT(dates.date, '%%Y-%%m-%%d') as date,
			COALESCE(c.new_customers, 0) as new_customers,
			COALESCE(s.new_sms_signups, 0) as new_sms_signups
		FROM (
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM customers
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			UNION
			SELECT DISTINCT DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date
			FROM sms_signups
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
		) dates
		LEFT JOIN (
			SELECT
				DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
				COUNT(*) as new_customers
			FROM customers
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			GROUP BY DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
		) c ON dates.date = c.date
		LEFT JOIN (
			SELECT
				DATE(CONVERT_TZ(created_at, '+00:00', '%s')) as date,
				COUNT(*) as new_sms_signups
			FROM sms_signups
			WHERE DATE(CONVERT_TZ(created_at, '+00:00', '%s')) BETWEEN DATE(?) AND DATE(?)
			GROUP BY DATE(CONVERT_TZ(created_at, '+00:00', '%s'))
		) s ON dates.date = s.date
		ORDER BY dates.date ASC
	`, offset, offset, offset, offset, offset, offset, offset, offset, offset, offset)

	rows, err := db.Query(query, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a map of actual data indexed by date
	dataMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var date string
		var newCustomers, newSMSSignups int
		err := rows.Scan(&date, &newCustomers, &newSMSSignups)
		if err != nil {
			continue
		}
		dataMap[date] = map[string]interface{}{
			"date":            date,
			"new_customers":   newCustomers,
			"new_sms_signups": newSMSSignups,
		}
	}

	// Generate complete date range with zeros for missing days
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var results []map[string]interface{}
	currentDate := start
	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")

		if data, exists := dataMap[dateStr]; exists {
			results = append(results, data)
		} else {
			results = append(results, map[string]interface{}{
				"date":            dateStr,
				"new_customers":   0,
				"new_sms_signups": 0,
			})
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return results, nil
}

// GetArticles retrieves articles for a specific website
func (s *AdminServer) GetArticles(websiteID string, limit, offset int) ([]Article, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, slug, title, description, content, excerpt, type, status, thumbnail_id, published_date, created_at, updated_at
		FROM articles_unified ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var a Article
		var publishedDate sql.NullTime
		var thumbnailID sql.NullInt64
		err := rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Description, &a.Content, &a.Excerpt, &a.Type, &a.Status, &thumbnailID, &publishedDate, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if publishedDate.Valid {
			a.PublishedDate = publishedDate.Time
		}
		if thumbnailID.Valid {
			a.ThumbnailID = int(thumbnailID.Int64)
		}
		articles = append(articles, a)
	}

	return articles, nil
}

// GetArticle retrieves a single article
func (s *AdminServer) GetArticle(websiteID string, articleID int) (Article, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Article{}, err
	}
	defer db.Close()

	query := `SELECT id, slug, title, description, content, excerpt, type, status, thumbnail_id, published_date, created_at, updated_at
		FROM articles_unified WHERE id = ?`

	var a Article
	var publishedDate sql.NullTime
	var thumbnailID sql.NullInt64
	err = db.QueryRow(query, articleID).Scan(&a.ID, &a.Slug, &a.Title, &a.Description, &a.Content, &a.Excerpt, &a.Type, &a.Status, &thumbnailID, &publishedDate, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Article{}, err
	}
	if publishedDate.Valid {
		a.PublishedDate = publishedDate.Time
	}
	if thumbnailID.Valid {
		a.ThumbnailID = int(thumbnailID.Int64)
	}

	return a, nil
}

// CreateArticle creates a new article
func (s *AdminServer) CreateArticle(websiteID string, a Article) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO articles_unified (slug, title, description, content, excerpt, type, status, thumbnail_id, published_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var publishedDate interface{}
	if a.PublishedDate.IsZero() {
		publishedDate = nil
	} else {
		publishedDate = a.PublishedDate
	}

	var thumbnailID interface{}
	if a.ThumbnailID == 0 {
		thumbnailID = nil
	} else {
		thumbnailID = a.ThumbnailID
	}

	result, err := db.Exec(query, a.Slug, a.Title, a.Description, a.Content, a.Excerpt, a.Type, a.Status, thumbnailID, publishedDate)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateArticle updates an existing article
func (s *AdminServer) UpdateArticle(websiteID string, a Article) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE articles_unified SET slug = ?, title = ?, description = ?, content = ?, excerpt = ?, type = ?, status = ?, thumbnail_id = ?, published_date = ?
		WHERE id = ?`

	var publishedDate interface{}
	if a.PublishedDate.IsZero() {
		publishedDate = nil
	} else {
		publishedDate = a.PublishedDate
	}

	var thumbnailID interface{}
	if a.ThumbnailID == 0 {
		thumbnailID = nil
	} else {
		thumbnailID = a.ThumbnailID
	}

	_, err = db.Exec(query, a.Slug, a.Title, a.Description, a.Content, a.Excerpt, a.Type, a.Status, thumbnailID, publishedDate, a.ID)
	return err
}

// DeleteArticle deletes an article
func (s *AdminServer) DeleteArticle(websiteID string, articleID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM articles_unified WHERE id = ?`
	_, err = db.Exec(query, articleID)
	return err
}

// GetProducts retrieves products for a specific website
func (s *AdminServer) GetProducts(websiteID string, limit, offset int) ([]Product, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, created_at, updated_at
		FROM products_unified ORDER BY sort_order ASC, created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

// GetProduct retrieves a single product
func (s *AdminServer) GetProduct(websiteID string, productID int) (Product, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Product{}, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, released_date, created_at, updated_at
		FROM products_unified WHERE id = ?`

	var p Product
	var releasedDate sql.NullTime
	err = db.QueryRow(query, productID).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.CompareAtPrice, &p.SKU, &p.InventoryQuantity, &p.InventoryPolicy, &p.Status, &p.Featured, &p.SortOrder, &releasedDate, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Product{}, err
	}
	if releasedDate.Valid {
		p.ReleasedDate = releasedDate.Time
	}

	// Load variants for this product
	p.Variants, err = s.getProductVariants(websiteID, productID)
	if err != nil {
		// Don't fail if variants can't be loaded, just log it
		log.Printf("Warning: Failed to load variants for product %d: %v", productID, err)
		p.Variants = []structs.ProductVariant{}
	}

	return p, nil
}

// CreateProduct creates a new product at the top and pushes others down
func (s *AdminServer) CreateProduct(websiteID string, p Product) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Increment all existing products' sort_order by 1
	_, err = tx.Exec(`UPDATE products_unified SET sort_order = sort_order + 1`)
	if err != nil {
		return 0, err
	}

	// Insert new product with sort_order = 0 (top position)
	query := `INSERT INTO products_unified (name, slug, description, price, compare_at_price, sku, inventory_quantity, inventory_policy, status, featured, sort_order, released_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	result, err := tx.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, releasedDate)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateProduct updates an existing product
func (s *AdminServer) UpdateProduct(websiteID string, p Product) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE products_unified SET name = ?, slug = ?, description = ?, price = ?, compare_at_price = ?, sku = ?, inventory_quantity = ?, inventory_policy = ?, status = ?, featured = ?, sort_order = ?, released_date = ?
		WHERE id = ?`

	var releasedDate interface{}
	if !p.ReleasedDate.IsZero() {
		releasedDate = p.ReleasedDate
	}

	_, err = db.Exec(query, p.Name, p.Slug, p.Description, p.Price, p.CompareAtPrice, p.SKU, p.InventoryQuantity, p.InventoryPolicy, p.Status, p.Featured, p.SortOrder, releasedDate, p.ID)
	return err
}

// DeleteProduct deletes a product
func (s *AdminServer) DeleteProduct(websiteID string, productID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM products_unified WHERE id = ?`
	_, err = db.Exec(query, productID)
	return err
}

// ReorderProduct swaps the sort_order of a product with an adjacent product
func (s *AdminServer) ReorderProduct(websiteID string, productID int, direction string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get current product's sort_order
	var currentSortOrder int
	err = db.QueryRow(`SELECT sort_order FROM products_unified WHERE id = ?`, productID).Scan(&currentSortOrder)
	if err != nil {
		return err
	}

	var targetSortOrder int
	var targetID int
	if direction == "up" {
		// Find the product with the next lower sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM products_unified WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	} else if direction == "down" {
		// Find the product with the next higher sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM products_unified WHERE sort_order > ? ORDER BY sort_order ASC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	}

	if err == sql.ErrNoRows {
		// Already at top/bottom, nothing to do - just return success
		return nil
	}
	if err != nil {
		return err
	}

	// Swap sort_order values
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Swap the two products' sort_order values
	_, err = tx.Exec(`UPDATE products_unified SET sort_order = ? WHERE id = ?`, targetSortOrder, productID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE products_unified SET sort_order = ? WHERE id = ?`, currentSortOrder, targetID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Helper function to load variants for a product
func (s *AdminServer) getProductVariants(websiteID string, productID int) ([]structs.ProductVariant, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, product_id, title, price_modifier, sku, inventory_quantity, position
		FROM product_variants
		WHERE product_id = ?
		ORDER BY position ASC
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []structs.ProductVariant
	for rows.Next() {
		var variant structs.ProductVariant
		var sku sql.NullString

		err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.Title,
			&variant.PriceModifier,
			&sku,
			&variant.InventoryQuantity,
			&variant.Position,
		)
		if err != nil {
			continue
		}

		variant.SKU = sku.String

		variants = append(variants, variant)
	}

	return variants, nil
}

// Variant management functions
func (s *AdminServer) CreateVariant(websiteID string, productID int, data map[string]interface{}) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get the max position for this product
	var maxPosition int
	err = db.QueryRow(`SELECT COALESCE(MAX(position), 0) FROM product_variants WHERE product_id = ?`, productID).Scan(&maxPosition)
	if err != nil {
		maxPosition = 0
	}

	query := `
		INSERT INTO product_variants (
			product_id, title, price_modifier, sku, inventory_quantity, position
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(query,
		productID,
		data["title"],
		data["priceModifier"],
		data["sku"],
		data["inventoryQuantity"],
		maxPosition+1,
	)

	return err
}

func (s *AdminServer) GetVariant(websiteID string, variantID int) (structs.ProductVariant, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return structs.ProductVariant{}, err
	}
	defer db.Close()

	query := `
		SELECT id, product_id, title, price_modifier, sku, inventory_quantity, position
		FROM product_variants
		WHERE id = ?
	`

	var variant structs.ProductVariant
	var sku sql.NullString

	err = db.QueryRow(query, variantID).Scan(
		&variant.ID,
		&variant.ProductID,
		&variant.Title,
		&variant.PriceModifier,
		&sku,
		&variant.InventoryQuantity,
		&variant.Position,
	)

	if err != nil {
		return structs.ProductVariant{}, err
	}

	variant.SKU = sku.String

	return variant, nil
}

func (s *AdminServer) UpdateVariant(websiteID string, variantID int, data map[string]interface{}) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
		UPDATE product_variants
		SET title = ?, price_modifier = ?, sku = ?, inventory_quantity = ?
		WHERE id = ?
	`

	_, err = db.Exec(query,
		data["title"],
		data["priceModifier"],
		data["sku"],
		data["inventoryQuantity"],
		variantID,
	)

	return err
}

func (s *AdminServer) DeleteVariant(websiteID string, variantID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM product_variants WHERE id = ?`
	_, err = db.Exec(query, variantID)
	return err
}

// ReorderVariant moves a variant up or down in the list
func (s *AdminServer) ReorderVariant(websiteID string, variantID int, direction string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get current variant's position and product_id
	var currentPosition, productID int
	err = db.QueryRow(`SELECT position, product_id FROM product_variants WHERE id = ?`, variantID).Scan(&currentPosition, &productID)
	if err != nil {
		return err
	}

	var targetPosition int
	var targetID int
	if direction == "up" {
		// Find the variant with the next lower position (same product)
		err = db.QueryRow(`SELECT id, position FROM product_variants WHERE product_id = ? AND position < ? ORDER BY position DESC LIMIT 1`, productID, currentPosition).Scan(&targetID, &targetPosition)
	} else if direction == "down" {
		// Find the variant with the next higher position (same product)
		err = db.QueryRow(`SELECT id, position FROM product_variants WHERE product_id = ? AND position > ? ORDER BY position ASC LIMIT 1`, productID, currentPosition).Scan(&targetID, &targetPosition)
	}

	if err == sql.ErrNoRows {
		// Already at top/bottom, nothing to do
		return nil
	}
	if err != nil {
		return err
	}

	// Swap position values
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE product_variants SET position = ? WHERE id = ?`, targetPosition, variantID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE product_variants SET position = ? WHERE id = ?`, currentPosition, targetID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetCategories retrieves categories for a specific website
func (s *AdminServer) GetCategories(websiteID string) ([]Category, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, count, created_at, updated_at FROM categories_unified ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Count, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// CreateCategory creates a new category
func (s *AdminServer) CreateCategory(websiteID string, c Category) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO categories_unified (name, slug) VALUES (?, ?)`
	result, err := db.Exec(query, c.Name, c.Slug)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteCategory deletes a category
func (s *AdminServer) DeleteCategory(websiteID string, categoryID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM categories_unified WHERE id = ?`
	_, err = db.Exec(query, categoryID)
	return err
}

// GetCollections retrieves collections for a specific website
func (s *AdminServer) GetCollections(websiteID string) ([]Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, image_id, sort_order, status, created_at, updated_at
		FROM collections_unified ORDER BY sort_order, name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collections := []Collection{}
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.ImageID, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}

	return collections, nil
}

// CreateCollection creates a new collection at the top and pushes others down
func (s *AdminServer) CreateCollection(websiteID string, c Collection) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Increment all existing collections' sort_order by 1
	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = sort_order + 1`)
	if err != nil {
		return 0, err
	}

	// Insert new collection with sort_order = 0 (top position)
	query := `INSERT INTO collections_unified (name, slug, description, image_id, sort_order, status)
		VALUES (?, ?, ?, ?, 0, ?)`

	result, err := tx.Exec(query, c.Name, c.Slug, c.Description, c.ImageID, c.Status)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetCollection retrieves a single collection by ID
func (s *AdminServer) GetCollection(websiteID string, collectionID int) (Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Collection{}, err
	}
	defer db.Close()

	query := `SELECT id, name, slug, description, image_id, sort_order, status, created_at, updated_at
		FROM collections_unified WHERE id = ?`

	var c Collection
	var imageID sql.NullInt64
	err = db.QueryRow(query, collectionID).Scan(
		&c.ID, &c.Name, &c.Slug, &c.Description, &imageID, &c.SortOrder, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return Collection{}, err
	}

	if imageID.Valid {
		c.ImageID = int(imageID.Int64)
	}

	return c, nil
}

// UpdateCollection updates an existing collection
func (s *AdminServer) UpdateCollection(websiteID string, c Collection) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE collections_unified SET name = ?, slug = ?, description = ?, image_id = ?, sort_order = ?, status = ?
		WHERE id = ?`

	_, err = db.Exec(query, c.Name, c.Slug, c.Description, c.ImageID, c.SortOrder, c.Status, c.ID)
	return err
}

// DeleteCollection deletes a collection and renumbers the remaining ones
func (s *AdminServer) DeleteCollection(websiteID string, collectionID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete the collection
	_, err = tx.Exec(`DELETE FROM collections_unified WHERE id = ?`, collectionID)
	if err != nil {
		return err
	}

	// Renumber all collections sequentially starting from 0
	// Get all collections ordered by current sort_order
	rows, err := tx.Query(`SELECT id FROM collections_unified ORDER BY sort_order, name`)
	if err != nil {
		return err
	}

	var collectionIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		collectionIDs = append(collectionIDs, id)
	}
	rows.Close()

	// Update each collection with sequential sort_order
	for i, id := range collectionIDs {
		_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, i, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ReorderCollection moves a collection up or down in sort order
func (s *AdminServer) ReorderCollection(websiteID string, collectionID int, direction string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get current collection's sort_order
	var currentSortOrder int
	err = db.QueryRow(`SELECT sort_order FROM collections_unified WHERE id = ?`, collectionID).Scan(&currentSortOrder)
	if err != nil {
		return err
	}

	var targetSortOrder int
	var targetID int
	if direction == "up" {
		// Find the collection with the next lower sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM collections_unified WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	} else if direction == "down" {
		// Find the collection with the next higher sort_order
		err = db.QueryRow(`SELECT id, sort_order FROM collections_unified WHERE sort_order > ? ORDER BY sort_order ASC LIMIT 1`, currentSortOrder).Scan(&targetID, &targetSortOrder)
	}

	if err == sql.ErrNoRows {
		// Already at top/bottom, nothing to do - just return success
		return nil
	}
	if err != nil {
		return err
	}

	// Swap sort_order values
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Swap the two collections' sort_order values
	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, targetSortOrder, collectionID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE collections_unified SET sort_order = ? WHERE id = ?`, currentSortOrder, targetID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetImages retrieves images for a specific website
func (s *AdminServer) GetImages(websiteID string, limit, offset int) ([]Image, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, url, alt_text, credit, filename, size, width, height, created_at
		FROM images_unified ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := []Image{}
	for rows.Next() {
		var img Image
		var filename, altText, credit sql.NullString
		var size sql.NullInt64
		var width, height sql.NullInt64
		err := rows.Scan(&img.ID, &img.URL, &altText, &credit, &filename, &size, &width, &height, &img.CreatedAt)
		if err != nil {
			return nil, err
		}
		if altText.Valid {
			img.AltText = altText.String
		}
		if credit.Valid {
			img.Credit = credit.String
		}
		if filename.Valid {
			img.Filename = filename.String
		}
		if size.Valid {
			img.Size = size.Int64
		}
		if width.Valid {
			img.Width = int(width.Int64)
		}
		if height.Valid {
			img.Height = int(height.Int64)
		}
		images = append(images, img)
	}

	return images, nil
}

// CreateImage creates a new image entry
func (s *AdminServer) CreateImage(websiteID string, img Image) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO images_unified (url, alt_text, credit, filename, size, width, height) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := db.Exec(query, img.URL, img.AltText, img.Credit, img.Filename, img.Size, img.Width, img.Height)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteImage deletes an image
func (s *AdminServer) DeleteImage(websiteID string, imageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM images_unified WHERE id = ?`
	_, err = db.Exec(query, imageID)
	return err
}

// LogActivity logs an admin action (currently just to stdout, no DB needed)
func (s *AdminServer) LogActivity(action, entityType string, entityID int, websiteID string, details interface{}) error {
	// Just log to stdout for now - could write to a file later if needed
	fmt.Printf("[ADMIN] %s %s (ID: %d, Website: %s)\n", action, entityType, entityID, websiteID)
	return nil
}

// ====================
// Product Images
// ====================

// ProductImage represents the product-image relationship
type ProductImage struct {
	ID       int    `json:"id"`
	ImageID  int    `json:"imageId"`
	Position int    `json:"position"`
	Image    Image  `json:"image"`
}

// ====================
// Product-Collection Relationships
// ====================

// GetProductCollections retrieves all collections for a product
func (s *AdminServer) GetProductCollections(websiteID string, productID int) ([]Collection, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT c.id, c.name, c.slug, c.status, c.sort_order
		FROM collections_unified c
		INNER JOIN product_collections pc ON c.id = pc.collection_id
		WHERE pc.product_id = ?
		ORDER BY pc.position
	`
	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Status, &c.SortOrder)
		if err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}

	return collections, nil
}

// SetProductCollections replaces all collections for a product
func (s *AdminServer) SetProductCollections(websiteID string, productID int, collectionIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing relationships
	_, err = tx.Exec("DELETE FROM product_collections WHERE product_id = ?", productID)
	if err != nil {
		return err
	}

	// Insert new relationships
	stmt, err := tx.Prepare("INSERT INTO product_collections (product_id, collection_id, position) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for position, collectionID := range collectionIDs {
		_, err = stmt.Exec(productID, collectionID, position)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ====================
// Article-Category Relationships
// ====================

// GetArticleCategories retrieves all categories for an article
func (s *AdminServer) GetArticleCategories(websiteID string, articleID int) ([]Category, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT c.id, c.name, c.slug, c.count
		FROM categories_unified c
		INNER JOIN article_categories ac ON c.id = ac.category_id
		WHERE ac.post_id = ?
		ORDER BY c.name
	`
	rows, err := db.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Count)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// SetArticleCategories replaces all categories for an article
func (s *AdminServer) SetArticleCategories(websiteID string, articleID int, categoryIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get old category IDs before we delete them
	oldCategoryIDsQuery := "SELECT category_id FROM article_categories WHERE post_id = ?"
	rows, err := db.Query(oldCategoryIDsQuery, articleID)
	if err != nil {
		return err
	}

	var oldCategoryIDs []int
	for rows.Next() {
		var catID int
		if err := rows.Scan(&catID); err != nil {
			rows.Close()
			return err
		}
		oldCategoryIDs = append(oldCategoryIDs, catID)
	}
	rows.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing relationships
	_, err = tx.Exec("DELETE FROM article_categories WHERE post_id = ?", articleID)
	if err != nil {
		return err
	}

	// Insert new relationships
	stmt, err := tx.Prepare("INSERT INTO article_categories (post_id, category_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, categoryID := range categoryIDs {
		_, err = stmt.Exec(articleID, categoryID)
		if err != nil {
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Update counts for all affected categories (old and new)
	affectedCategories := make(map[int]bool)
	for _, id := range oldCategoryIDs {
		affectedCategories[id] = true
	}
	for _, id := range categoryIDs {
		affectedCategories[id] = true
	}

	// Update count for each affected category
	for catID := range affectedCategories {
		if err := s.UpdateCategoryCount(websiteID, catID); err != nil {
			// Log the error but don't fail the whole operation
			log.Printf("Error updating category count for category %d: %v", catID, err)
		}
	}

	return nil
}


// UpdateCategoryCount recalculates the post count for a category
func (s *AdminServer) UpdateCategoryCount(websiteID string, categoryID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
		UPDATE categories_unified
		SET count = (
			SELECT COUNT(*)
			FROM article_categories
			WHERE category_id = ?
		)
		WHERE id = ?
	`
	_, err = db.Exec(query, categoryID, categoryID)
	return err
}

// GetOrders retrieves all orders for a website
func (s *AdminServer) GetOrders(websiteID string) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetOrdersFiltered retrieves orders with filters and sorting
func (s *AdminServer) GetOrdersFiltered(websiteID string, filters OrderFilters) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE 1=1
	`

	var args []interface{}

	// Add payment status filter
	if filters.PaymentStatus != "" {
		query += " AND payment_status = ?"
		args = append(args, filters.PaymentStatus)
	}

	// Add fulfillment status filter
	if filters.FulfillmentStatus != "" {
		query += " AND fulfillment_status = ?"
		args = append(args, filters.FulfillmentStatus)
	}

	// Add sorting
	switch filters.Sort {
	case "date_asc":
		query += " ORDER BY created_at ASC"
	case "total_desc":
		query += " ORDER BY total DESC"
	case "total_asc":
		query += " ORDER BY total ASC"
	default: // date_desc
		query += " ORDER BY created_at DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetOrder retrieves a single order with its items
func (s *AdminServer) GetOrder(websiteID string, orderID int) (Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Order{}, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE id = ?
	`

	var o Order
	var shippingLine2, paymentMethod sql.NullString

	err = db.QueryRow(query, orderID).Scan(
		&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
		&o.ShippingAddressLine1, &shippingLine2,
		&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
		&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
		&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return Order{}, err
	}

	o.ShippingAddressLine2 = shippingLine2.String
	o.PaymentMethod = paymentMethod.String

	// Get order items
	itemsQuery := `
		SELECT id, product_id, product_name, variant_title, quantity, price, total
		FROM order_items
		WHERE order_id = ?
	`

	rows, err := db.Query(itemsQuery, orderID)
	if err != nil {
		return o, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem
		var variantTitle sql.NullString

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName, &variantTitle,
			&item.Quantity, &item.Price, &item.Total,
		)
		if err != nil {
			return o, err
		}

		item.VariantTitle = variantTitle.String
		o.Items = append(o.Items, item)
	}

	return o, nil
}

// UpdateOrderFulfillmentStatus updates the fulfillment status of an order
func (s *AdminServer) UpdateOrderFulfillmentStatus(websiteID string, orderID int, status string) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `UPDATE orders SET fulfillment_status = ?, updated_at = NOW() WHERE id = ?`
	_, err = db.Exec(query, status, orderID)
	return err
}

// LabelInfo contains information about a purchased shipping label
type LabelInfo struct {
	TrackingNumber string  `json:"trackingNumber"`
	LabelURL       string  `json:"labelUrl"`
	Carrier        string  `json:"carrier"`
	Cost           float64 `json:"cost"`
}

// GetShippingRates gets shipping rates from Shippo for an order
func (s *AdminServer) GetShippingRates(websiteID string, order Order, length, width, height, weight float64) ([]shippo.Rate, error) {
	// Get website config
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	// Load website config to get ship-from address
	configPath := filepath.Join("websites", website.Directory, "config-dev.json")
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read website config: %w", err)
	}

	var siteConfig configs.WebsiteConfig
	if err := json.Unmarshal(configData, &siteConfig); err != nil {
		return nil, fmt.Errorf("failed to parse website config: %w", err)
	}

	// Get Shippo API key from site config
	shippoKey := siteConfig.Shippo.APIKey

	// Create Shippo client
	client := shippo.NewClient(shippoKey)

	// Build addresses
	fromAddress := shippo.Address{
		Name:    siteConfig.ShipFrom.Name,
		Street1: siteConfig.ShipFrom.Street1,
		Street2: siteConfig.ShipFrom.Street2,
		City:    siteConfig.ShipFrom.City,
		State:   siteConfig.ShipFrom.State,
		Zip:     siteConfig.ShipFrom.Zip,
		Country: siteConfig.ShipFrom.Country,
	}

	toAddress := shippo.Address{
		Name:    order.CustomerName,
		Street1: order.ShippingAddressLine1,
		Street2: order.ShippingAddressLine2,
		City:    order.ShippingCity,
		State:   order.ShippingState,
		Zip:     order.ShippingZip,
		Country: order.ShippingCountry,
		Email:   order.CustomerEmail,
	}

	parcel := shippo.Parcel{
		Length:       fmt.Sprintf("%.2f", length),
		Width:        fmt.Sprintf("%.2f", width),
		Height:       fmt.Sprintf("%.2f", height),
		DistanceUnit: "in",
		Weight:       fmt.Sprintf("%.2f", weight),
		MassUnit:     "lb",
	}

	// Get rates from Shippo
	shipmentResp, err := client.GetRates(fromAddress, toAddress, parcel)
	if err != nil {
		return nil, err
	}

	return shipmentResp.Rates, nil
}

// PurchaseShippingLabel purchases a shipping label from Shippo and updates the order
func (s *AdminServer) PurchaseShippingLabel(websiteID string, orderID int, rateID string) (*LabelInfo, error) {
	// Get website config for site-specific Shippo key
	website, err := s.GetWebsite(websiteID)
	if err != nil {
		return nil, err
	}

	// Get Shippo API key from site config
	shippoKey := website.ShippoAPIKey

	// Get label format - use site-specific if available, otherwise default to PDF
	labelFormat := "PDF"
	if website.LabelFormat != "" {
		labelFormat = website.LabelFormat
	}

	// Create Shippo client
	client := shippo.NewClient(shippoKey)

	// Purchase label
	transaction, err := client.PurchaseLabel(rateID, labelFormat)
	if err != nil {
		return nil, err
	}

	// Extract cost from rate amount
	cost := 0.0
	fmt.Sscanf(transaction.Rate, "%f", &cost)

	// Update order in database with label information
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Determine carrier from transaction
	carrier := "USPS" // Default, would need to parse from rate details

	query := `
		UPDATE orders
		SET tracking_number = ?,
		    shipping_carrier = ?,
		    shipping_label_url = ?,
		    shipping_label_cost = ?,
		    updated_at = NOW()
		WHERE id = ?
	`

	_, err = db.Exec(query, transaction.TrackingNumber, carrier, transaction.LabelURL, cost, orderID)
	if err != nil {
		return nil, err
	}

	return &LabelInfo{
		TrackingNumber: transaction.TrackingNumber,
		LabelURL:       transaction.LabelURL,
		Carrier:        carrier,
		Cost:           cost,
	}, nil
}

// GetProductImagesData retrieves all images for a product from product_images_data
func (s *AdminServer) GetProductImagesData(websiteID string, productID int) ([]ProductImageData, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, product_id, url, filename, filepath, alt_text, credit, size, width, height, position, created_at
		FROM product_images_data
		WHERE product_id = ?
		ORDER BY position ASC
	`

	rows, err := db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ProductImageData
	for rows.Next() {
		var img ProductImageData
		var altText, credit sql.NullString
		var size sql.NullInt64
		var width, height sql.NullInt64

		err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Filename, &img.Filepath,
			&altText, &credit, &size, &width, &height, &img.Position, &img.CreatedAt)
		if err != nil {
			return nil, err
		}

		if altText.Valid {
			img.AltText = altText.String
		}
		if credit.Valid {
			img.Credit = credit.String
		}
		if size.Valid {
			img.Size = size.Int64
		}
		if width.Valid {
			img.Width = int(width.Int64)
		}
		if height.Valid {
			img.Height = int(height.Int64)
		}

		images = append(images, img)
	}

	return images, nil
}

// AddProductImageData adds a new image directly to a product
func (s *AdminServer) AddProductImageData(websiteID string, img ProductImageData) (int64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `INSERT INTO product_images_data (product_id, url, filename, filepath, alt_text, credit, size, width, height, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, img.ProductID, img.URL, img.Filename, img.Filepath,
		img.AltText, img.Credit, img.Size, img.Width, img.Height, img.Position)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// DeleteProductImageData deletes an image and its file from disk
func (s *AdminServer) DeleteProductImageData(websiteID string, imageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// First get the filepath
	var filepath string
	err = db.QueryRow("SELECT filepath FROM product_images_data WHERE id = ?", imageID).Scan(&filepath)
	if err != nil {
		return err
	}

	// Delete from database
	_, err = db.Exec("DELETE FROM product_images_data WHERE id = ?", imageID)
	if err != nil {
		return err
	}

	// Delete file from disk
	if err := os.Remove(filepath); err != nil {
		log.Printf("Warning: Failed to delete image file %s: %v", filepath, err)
		// Don't fail the operation if file doesn't exist
	}

	return nil
}

// UpdateProductImagePositions updates the position values for reordering
func (s *AdminServer) UpdateProductImagePositions(websiteID string, imageIDs []int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE product_images_data SET position = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for position, imageID := range imageIDs {
		_, err = stmt.Exec(position, imageID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ====================
// Customer Management
// ====================

// GetCustomers retrieves all customers with aggregate statistics
func (s *AdminServer) GetCustomers(websiteID string, filters CustomerFilters) ([]Customer, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone,
			c.created_at, c.updated_at,
			COUNT(CASE WHEN o.payment_status = 'paid' THEN 1 END) as order_count,
			COALESCE(SUM(CASE WHEN o.payment_status = 'paid' THEN o.total ELSE 0 END), 0) as total_spent,
			MIN(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as first_order,
			MAX(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as last_order
		FROM customers c
		LEFT JOIN orders o ON c.id = o.customer_id
		GROUP BY c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone, c.created_at, c.updated_at
	`

	// Add sorting
	switch filters.Sort {
	case "total_desc":
		query += " ORDER BY total_spent DESC"
	case "total_asc":
		query += " ORDER BY total_spent ASC"
	case "orders_desc":
		query += " ORDER BY order_count DESC"
	case "date_asc":
		query += " ORDER BY c.created_at ASC"
	case "date_desc":
		query += " ORDER BY c.created_at DESC"
	default: // Default to total spent descending
		query += " ORDER BY total_spent DESC"
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		var stripeCustomerID, phone sql.NullString
		var firstOrder, lastOrder sql.NullTime

		err := rows.Scan(
			&c.ID, &c.Email, &stripeCustomerID, &c.FirstName, &c.LastName, &phone,
			&c.CreatedAt, &c.UpdatedAt,
			&c.OrderCount, &c.TotalSpent, &firstOrder, &lastOrder,
		)
		if err != nil {
			return nil, err
		}

		if stripeCustomerID.Valid {
			c.StripeCustomerID = stripeCustomerID.String
		}
		if phone.Valid {
			c.Phone = phone.String
		}
		if firstOrder.Valid {
			c.FirstOrder = &firstOrder.Time
		}
		if lastOrder.Valid {
			c.LastOrder = &lastOrder.Time
		}

		customers = append(customers, c)
	}

	return customers, nil
}

// GetCustomer retrieves a single customer with statistics
func (s *AdminServer) GetCustomer(websiteID string, customerID int) (Customer, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return Customer{}, err
	}
	defer db.Close()

	query := `
		SELECT
			c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone,
			c.created_at, c.updated_at,
			COUNT(CASE WHEN o.payment_status = 'paid' THEN 1 END) as order_count,
			COALESCE(SUM(CASE WHEN o.payment_status = 'paid' THEN o.total ELSE 0 END), 0) as total_spent,
			MIN(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as first_order,
			MAX(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as last_order
		FROM customers c
		LEFT JOIN orders o ON c.id = o.customer_id
		WHERE c.id = ?
		GROUP BY c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone, c.created_at, c.updated_at
	`

	var c Customer
	var stripeCustomerID, phone sql.NullString
	var firstOrder, lastOrder sql.NullTime

	err = db.QueryRow(query, customerID).Scan(
		&c.ID, &c.Email, &stripeCustomerID, &c.FirstName, &c.LastName, &phone,
		&c.CreatedAt, &c.UpdatedAt,
		&c.OrderCount, &c.TotalSpent, &firstOrder, &lastOrder,
	)
	if err != nil {
		return Customer{}, err
	}

	if stripeCustomerID.Valid {
		c.StripeCustomerID = stripeCustomerID.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if firstOrder.Valid {
		c.FirstOrder = &firstOrder.Time
	}
	if lastOrder.Valid {
		c.LastOrder = &lastOrder.Time
	}

	return c, nil
}

// GetCustomerByEmail retrieves a customer by email address with statistics
func (s *AdminServer) GetCustomerByEmail(websiteID string, email string) (*Customer, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone,
			c.created_at, c.updated_at,
			COUNT(CASE WHEN o.payment_status = 'paid' THEN 1 END) as order_count,
			COALESCE(SUM(CASE WHEN o.payment_status = 'paid' THEN o.total ELSE 0 END), 0) as total_spent,
			MIN(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as first_order,
			MAX(CASE WHEN o.payment_status = 'paid' THEN o.created_at END) as last_order
		FROM customers c
		LEFT JOIN orders o ON c.id = o.customer_id
		WHERE c.email = ?
		GROUP BY c.id, c.email, c.stripe_customer_id, c.first_name, c.last_name, c.phone, c.created_at, c.updated_at
	`

	var c Customer
	var stripeCustomerID, phone sql.NullString
	var firstOrder, lastOrder sql.NullTime

	err = db.QueryRow(query, email).Scan(
		&c.ID, &c.Email, &stripeCustomerID, &c.FirstName, &c.LastName, &phone,
		&c.CreatedAt, &c.UpdatedAt,
		&c.OrderCount, &c.TotalSpent, &firstOrder, &lastOrder,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No customer found, not an error
		}
		return nil, err
	}

	if stripeCustomerID.Valid {
		c.StripeCustomerID = stripeCustomerID.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if firstOrder.Valid {
		c.FirstOrder = &firstOrder.Time
	}
	if lastOrder.Valid {
		c.LastOrder = &lastOrder.Time
	}

	return &c, nil
}

// GetCustomerOrders retrieves all orders for a specific customer
func (s *AdminServer) GetCustomerOrders(websiteID string, customerID int) ([]Order, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id, order_number, customer_email, customer_name,
			shipping_address_line1, shipping_address_line2,
			shipping_city, shipping_state, shipping_zip, shipping_country,
			subtotal, tax, shipping_cost, total,
			payment_status, fulfillment_status, payment_method,
			created_at, updated_at
		FROM orders
		WHERE customer_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var shippingLine2, paymentMethod sql.NullString

		err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerEmail, &o.CustomerName,
			&o.ShippingAddressLine1, &shippingLine2,
			&o.ShippingCity, &o.ShippingState, &o.ShippingZip, &o.ShippingCountry,
			&o.Subtotal, &o.Tax, &o.ShippingCost, &o.Total,
			&o.PaymentStatus, &o.FulfillmentStatus, &paymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		o.ShippingAddressLine2 = shippingLine2.String
		o.PaymentMethod = paymentMethod.String

		orders = append(orders, o)
	}

	return orders, nil
}

// GetSMSSignups retrieves SMS signups for a website with filters
func (s *AdminServer) GetSMSSignups(websiteID string, filters SMSSignupFilters) ([]SMSSignup, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, COALESCE(country_code, '+1'), phone, COALESCE(email, ''), COALESCE(source, ''), created_at
		FROM sms_signups
		WHERE 1=1
	`

	var args []interface{}

	// Filter by country code
	if filters.CountryCode != "" {
		query += ` AND country_code = ?`
		args = append(args, filters.CountryCode)
	}

	// Filter by source
	if filters.Source != "" {
		query += ` AND source = ?`
		args = append(args, filters.Source)
	}

	// Filter by date range
	if filters.DateFrom != "" {
		query += ` AND created_at >= ?`
		args = append(args, filters.DateFrom+" 00:00:00")
	}
	if filters.DateTo != "" {
		query += ` AND created_at <= ?`
		args = append(args, filters.DateTo+" 23:59:59")
	}

	// Sorting
	switch filters.Sort {
	case "date_asc":
		query += ` ORDER BY created_at ASC`
	case "phone_asc":
		query += ` ORDER BY phone ASC`
	case "phone_desc":
		query += ` ORDER BY phone DESC`
	default:
		query += ` ORDER BY created_at DESC`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signups []SMSSignup
	for rows.Next() {
		var s SMSSignup
		err := rows.Scan(&s.ID, &s.CountryCode, &s.Phone, &s.Email, &s.Source, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		signups = append(signups, s)
	}

	return signups, nil
}

// GetUniqueCountryCodes retrieves unique country codes from SMS signups
func (s *AdminServer) GetUniqueCountryCodes(websiteID string) ([]string, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT DISTINCT country_code FROM sms_signups WHERE country_code IS NOT NULL ORDER BY country_code`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}

	return codes, nil
}

// GetUniqueSources retrieves unique sources from SMS signups
func (s *AdminServer) GetUniqueSources(websiteID string) ([]string, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT DISTINCT source FROM sms_signups WHERE source IS NOT NULL AND source != '' ORDER BY source`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// DeleteSMSSignup deletes an SMS signup by ID
func (s *AdminServer) DeleteSMSSignup(websiteID string, signupID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `DELETE FROM sms_signups WHERE id = ?`
	_, err = db.Exec(query, signupID)
	return err
}

// GetVerifiedSMSSignups retrieves only verified SMS signups with filters
func (s *AdminServer) GetVerifiedSMSSignups(websiteID string, filters SMSSignupFilters) ([]SMSSignup, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, COALESCE(country_code, '+1'), phone, COALESCE(email, ''), COALESCE(source, ''), created_at
		FROM sms_signups
		WHERE verified = 1 AND unsubscribed = 0
	`

	var args []interface{}

	// Filter by country code
	if filters.CountryCode != "" {
		query += ` AND country_code = ?`
		args = append(args, filters.CountryCode)
	}

	// Filter by source
	if filters.Source != "" {
		query += ` AND source = ?`
		args = append(args, filters.Source)
	}

	// Filter by date range
	if filters.DateFrom != "" {
		query += ` AND created_at >= ?`
		args = append(args, filters.DateFrom+" 00:00:00")
	}
	if filters.DateTo != "" {
		query += ` AND created_at <= ?`
		args = append(args, filters.DateTo+" 23:59:59")
	}

	// Sorting
	switch filters.Sort {
	case "date_asc":
		query += ` ORDER BY created_at ASC`
	case "phone_asc":
		query += ` ORDER BY phone ASC`
	case "phone_desc":
		query += ` ORDER BY phone DESC`
	default:
		query += ` ORDER BY created_at DESC`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signups []SMSSignup
	for rows.Next() {
		var s SMSSignup
		err := rows.Scan(&s.ID, &s.CountryCode, &s.Phone, &s.Email, &s.Source, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		signups = append(signups, s)
	}

	return signups, nil
}

// ===============================
// Analytics Queries
// ===============================

// GetPageViewStats returns basic pageview statistics for a date range
func (s *AdminServer) GetPageViewStats(websiteID string, startDate, endDate time.Time) (map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stats := make(map[string]interface{})

	// Total pageviews
	var totalViews int
	err = db.QueryRow(`
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
func (s *AdminServer) GetTopPages(websiteID string, startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT path, COUNT(*) as views, COUNT(DISTINCT session_id) as unique_visitors
		FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
		GROUP BY path
		ORDER BY views DESC
		LIMIT ?
	`

	rows, err := db.Query(query, startDate, endDate, limit)
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
func (s *AdminServer) GetTopReferrers(websiteID string, startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT referrer, COUNT(*) as visits
		FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
		AND referrer IS NOT NULL
		AND referrer != ''
		GROUP BY referrer
		ORDER BY visits DESC
		LIMIT ?
	`

	rows, err := db.Query(query, startDate, endDate, limit)
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
func (s *AdminServer) GetEventStats(websiteID string, startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT event_name, COUNT(*) as count
		FROM analytics_events
		WHERE created_at BETWEEN ? AND ?
		GROUP BY event_name
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := db.Query(query, startDate, endDate, limit)
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

// ===============================
// Real-Time Analytics
// ===============================

// GetActiveUsers returns count of users active in the last N minutes
func (s *AdminServer) GetActiveUsers(websiteID string, minutesAgo int) (int, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	cutoffTime := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)

	query := `
		SELECT COUNT(DISTINCT session_id) as active_users
		FROM (
			SELECT session_id, MAX(created_at) as last_activity
			FROM analytics_pageviews
			WHERE created_at >= ?
			GROUP BY session_id
			UNION
			SELECT session_id, MAX(created_at) as last_activity
			FROM analytics_events
			WHERE created_at >= ?
			GROUP BY session_id
		) as combined
	`

	var activeUsers int
	err = db.QueryRow(query, cutoffTime, cutoffTime).Scan(&activeUsers)
	if err != nil {
		return 0, err
	}

	return activeUsers, nil
}

// GetCurrentPages returns pages currently being viewed by active users
func (s *AdminServer) GetCurrentPages(websiteID string, minutesAgo int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	cutoffTime := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)

	query := `
		SELECT path, COUNT(DISTINCT session_id) as active_viewers
		FROM (
			SELECT session_id, path, created_at,
				ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at DESC) as rn
			FROM analytics_pageviews
			WHERE created_at >= ?
		) as recent_views
		WHERE rn = 1
		GROUP BY path
		ORDER BY active_viewers DESC
		LIMIT 20
	`

	rows, err := db.Query(query, cutoffTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []map[string]interface{}
	for rows.Next() {
		var path string
		var viewers int
		err := rows.Scan(&path, &viewers)
		if err != nil {
			return nil, err
		}
		pages = append(pages, map[string]interface{}{
			"path":    path,
			"viewers": viewers,
		})
	}

	return pages, nil
}

// ===============================
// Engagement Metrics
// ===============================

// GetBounceRate returns the bounce rate (single-page sessions) for a date range
func (s *AdminServer) GetBounceRate(websiteID string, startDate, endDate time.Time) (float64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `
		SELECT
			COUNT(DISTINCT CASE WHEN pageview_count = 1 THEN session_id END) as bounced_sessions,
			COUNT(DISTINCT session_id) as total_sessions
		FROM (
			SELECT session_id, COUNT(*) as pageview_count
			FROM analytics_pageviews
			WHERE created_at BETWEEN ? AND ?
			GROUP BY session_id
		) as session_stats
	`

	var bouncedSessions, totalSessions int
	err = db.QueryRow(query, startDate, endDate).Scan(&bouncedSessions, &totalSessions)
	if err != nil {
		return 0, err
	}

	if totalSessions == 0 {
		return 0, nil
	}

	bounceRate := (float64(bouncedSessions) / float64(totalSessions)) * 100
	return bounceRate, nil
}

// GetAverageSessionDuration returns average session duration in seconds
func (s *AdminServer) GetAverageSessionDuration(websiteID string, startDate, endDate time.Time) (float64, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `
		SELECT AVG(duration) as avg_duration
		FROM (
			SELECT
				session_id,
				SUM(time_on_page) as duration
			FROM analytics_pageviews
			WHERE created_at BETWEEN ? AND ?
			GROUP BY session_id
			HAVING SUM(time_on_page) > 0
		) as session_durations
	`

	var avgDuration sql.NullFloat64
	err = db.QueryRow(query, startDate, endDate).Scan(&avgDuration)
	if err != nil {
		return 0, err
	}

	if !avgDuration.Valid {
		return 0, nil
	}

	return avgDuration.Float64, nil
}

// GetDeviceBreakdown returns breakdown of traffic by device type
func (s *AdminServer) GetDeviceBreakdown(websiteID string, startDate, endDate time.Time) (map[string]int, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Categorize by screen width since we track that
	query := `
		SELECT
			CASE
				WHEN screen_width < 768 THEN 'mobile'
				WHEN screen_width < 1024 THEN 'tablet'
				ELSE 'desktop'
			END as device_type,
			COUNT(DISTINCT session_id) as sessions
		FROM analytics_pageviews
		WHERE created_at BETWEEN ? AND ?
		GROUP BY device_type
	`

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	for rows.Next() {
		var deviceType string
		var sessions int
		err := rows.Scan(&deviceType, &sessions)
		if err != nil {
			return nil, err
		}
		breakdown[deviceType] = sessions
	}

	return breakdown, nil
}

// GetEntryPages returns the top pages where users enter the site
func (s *AdminServer) GetEntryPages(websiteID string, startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT path, COUNT(*) as entries
		FROM (
			SELECT session_id, path,
				ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at ASC) as rn
			FROM analytics_pageviews
			WHERE created_at BETWEEN ? AND ?
		) as first_pages
		WHERE rn = 1
		GROUP BY path
		ORDER BY entries DESC
		LIMIT ?
	`

	rows, err := db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []map[string]interface{}
	for rows.Next() {
		var path string
		var entries int
		err := rows.Scan(&path, &entries)
		if err != nil {
			return nil, err
		}
		pages = append(pages, map[string]interface{}{
			"path":    path,
			"entries": entries,
		})
	}

	return pages, nil
}

// GetExitPages returns the top pages where users leave the site
func (s *AdminServer) GetExitPages(websiteID string, startDate, endDate time.Time, limit int) ([]map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT path, COUNT(*) as exits
		FROM (
			SELECT session_id, path,
				ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at DESC) as rn
			FROM analytics_pageviews
			WHERE created_at BETWEEN ? AND ?
		) as last_pages
		WHERE rn = 1
		GROUP BY path
		ORDER BY exits DESC
		LIMIT ?
	`

	rows, err := db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []map[string]interface{}
	for rows.Next() {
		var path string
		var exits int
		err := rows.Scan(&path, &exits)
		if err != nil {
			return nil, err
		}
		pages = append(pages, map[string]interface{}{
			"path":  path,
			"exits": exits,
		})
	}

	return pages, nil
}

// ===============================
// E-Commerce Analytics
// ===============================

// GetConversionRate returns the conversion rate (% of sessions that result in purchase)
func (s *AdminServer) GetConversionRate(websiteID string, startDate, endDate time.Time) (float64, int, int, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer db.Close()

	query := `
		SELECT
			COUNT(DISTINCT pv.session_id) as total_sessions,
			COUNT(DISTINCT CASE WHEN e.event_name = 'purchase' THEN pv.session_id END) as converted_sessions
		FROM analytics_pageviews pv
		LEFT JOIN analytics_events e ON pv.session_id = e.session_id
			AND e.event_name = 'purchase'
			AND e.created_at BETWEEN ? AND ?
		WHERE pv.created_at BETWEEN ? AND ?
	`

	var totalSessions, convertedSessions int
	err = db.QueryRow(query, startDate, endDate, startDate, endDate).Scan(&totalSessions, &convertedSessions)
	if err != nil {
		return 0, 0, 0, err
	}

	var conversionRate float64
	if totalSessions > 0 {
		conversionRate = (float64(convertedSessions) / float64(totalSessions)) * 100
	}

	return conversionRate, convertedSessions, totalSessions, nil
}

// GetCartAbandonmentRate returns cart abandonment metrics
func (s *AdminServer) GetCartAbandonmentRate(websiteID string, startDate, endDate time.Time) (float64, int, int, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer db.Close()

	query := `
		SELECT
			COUNT(DISTINCT CASE WHEN added.session_id IS NOT NULL THEN added.session_id END) as sessions_with_cart,
			COUNT(DISTINCT CASE WHEN purchased.session_id IS NOT NULL THEN purchased.session_id END) as sessions_with_purchase
		FROM (
			SELECT DISTINCT session_id
			FROM analytics_events
			WHERE event_name = 'add_to_cart'
			AND created_at BETWEEN ? AND ?
		) as added
		LEFT JOIN (
			SELECT DISTINCT session_id
			FROM analytics_events
			WHERE event_name = 'purchase'
			AND created_at BETWEEN ? AND ?
		) as purchased ON added.session_id = purchased.session_id
	`

	var sessionsWithCart, sessionsWithPurchase int
	err = db.QueryRow(query, startDate, endDate, startDate, endDate).Scan(&sessionsWithCart, &sessionsWithPurchase)
	if err != nil {
		return 0, 0, 0, err
	}

	var abandonmentRate float64
	if sessionsWithCart > 0 {
		abandoned := sessionsWithCart - sessionsWithPurchase
		abandonmentRate = (float64(abandoned) / float64(sessionsWithCart)) * 100
	}

	return abandonmentRate, sessionsWithCart - sessionsWithPurchase, sessionsWithCart, nil
}

// GetRevenueMetrics returns revenue statistics for a date range
func (s *AdminServer) GetRevenueMetrics(websiteID string, startDate, endDate time.Time) (map[string]interface{}, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			COUNT(*) as total_orders,
			SUM(total) as total_revenue,
			AVG(total) as avg_order_value,
			MAX(total) as highest_order
		FROM orders
		WHERE payment_status = 'paid'
		AND created_at BETWEEN ? AND ?
	`

	var totalOrders int
	var totalRevenue, avgOrderValue, highestOrder sql.NullFloat64

	err = db.QueryRow(query, startDate, endDate).Scan(&totalOrders, &totalRevenue, &avgOrderValue, &highestOrder)
	if err != nil {
		return nil, err
	}

	metrics := map[string]interface{}{
		"total_orders": totalOrders,
		"total_revenue": func() float64 {
			if totalRevenue.Valid {
				return totalRevenue.Float64
			}
			return 0
		}(),
		"avg_order_value": func() float64 {
			if avgOrderValue.Valid {
				return avgOrderValue.Float64
			}
			return 0
		}(),
		"highest_order": func() float64 {
			if highestOrder.Valid {
				return highestOrder.Float64
			}
			return 0
		}(),
	}

	return metrics, nil
}

// Overview Dashboard Queries

type OverviewStats struct {
	// Content Stats
	TotalArticles   int
	PublishedArticles int
	DraftArticles   int

	// Analytics Stats
	TotalPageviews    int
	PageviewsToday    int
	PageviewsThisWeek int
	PageviewsThisMonth int
	UniqueVisitorsToday int

	// E-commerce Stats
	TotalProducts    int
	RepeatCustomers  int
	TotalOrders      int
	TotalRevenue     float64
	TotalCustomers   int

	// Recent Activity
	OrdersToday     int
	RevenueToday    float64
	OrdersThisWeek  int
	RevenueThisWeek float64
	OrdersThisMonth int
	RevenueThisMonth float64

	// Marketing Stats
	TotalSMSSignups int

	// Messages Stats
	UnreadMessages int
	TotalMessages  int
}

func (s *AdminServer) GetOverviewStats(websiteID string) (*OverviewStats, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stats := &OverviewStats{}

	// Content Stats
	err = db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END) as published,
			SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END) as draft
		FROM articles_unified
	`).Scan(&stats.TotalArticles, &stats.PublishedArticles, &stats.DraftArticles)
	if err != nil && err != sql.ErrNoRows {
		// If table doesn't exist, continue with zeros
		stats.TotalArticles = 0
		stats.PublishedArticles = 0
		stats.DraftArticles = 0
	}

	// E-commerce Stats - Products
	err = db.QueryRow(`SELECT COUNT(*) FROM products_unified`).Scan(&stats.TotalProducts)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalProducts = 0
	}

	// E-commerce Stats - Repeat Customers (customers with 2+ paid orders)
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM customers c
		WHERE (
			SELECT COUNT(*)
			FROM orders o
			WHERE o.customer_id = c.id
			AND o.payment_status = 'paid'
		) >= 2
	`).Scan(&stats.RepeatCustomers)
	if err != nil && err != sql.ErrNoRows {
		stats.RepeatCustomers = 0
	}

	// E-commerce Stats - Orders (only paid orders)
	var totalRevenue sql.NullFloat64
	err = db.QueryRow(`
		SELECT
			COUNT(*) as total_orders,
			COALESCE(SUM(total), 0) as total_revenue
		FROM orders
		WHERE payment_status = 'paid'
	`).Scan(&stats.TotalOrders, &totalRevenue)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalOrders = 0
		stats.TotalRevenue = 0
	} else if totalRevenue.Valid {
		stats.TotalRevenue = totalRevenue.Float64
	}

	// E-commerce Stats - Customers
	err = db.QueryRow(`SELECT COUNT(*) FROM customers`).Scan(&stats.TotalCustomers)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalCustomers = 0
	}

	// Today's Activity
	todayStart := time.Now().Truncate(24 * time.Hour)
	var revenueToday sql.NullFloat64
	err = db.QueryRow(`
		SELECT
			COUNT(*) as orders_today,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN total ELSE 0 END), 0) as revenue_today
		FROM orders
		WHERE created_at >= ?
	`, todayStart).Scan(&stats.OrdersToday, &revenueToday)
	if err != nil && err != sql.ErrNoRows {
		stats.OrdersToday = 0
		stats.RevenueToday = 0
	} else if revenueToday.Valid {
		stats.RevenueToday = revenueToday.Float64
	}

	// This Week's Activity
	weekStart := time.Now().AddDate(0, 0, -7)
	var revenueWeek sql.NullFloat64
	err = db.QueryRow(`
		SELECT
			COUNT(*) as orders_week,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN total ELSE 0 END), 0) as revenue_week
		FROM orders
		WHERE created_at >= ?
	`, weekStart).Scan(&stats.OrdersThisWeek, &revenueWeek)
	if err != nil && err != sql.ErrNoRows {
		stats.OrdersThisWeek = 0
		stats.RevenueThisWeek = 0
	} else if revenueWeek.Valid {
		stats.RevenueThisWeek = revenueWeek.Float64
	}

	// This Month's Activity
	monthStart := time.Now().AddDate(0, 0, -30)
	var revenueMonth sql.NullFloat64
	err = db.QueryRow(`
		SELECT
			COUNT(*) as orders_month,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN total ELSE 0 END), 0) as revenue_month
		FROM orders
		WHERE created_at >= ?
	`, monthStart).Scan(&stats.OrdersThisMonth, &revenueMonth)
	if err != nil && err != sql.ErrNoRows {
		stats.OrdersThisMonth = 0
		stats.RevenueThisMonth = 0
	} else if revenueMonth.Valid {
		stats.RevenueThisMonth = revenueMonth.Float64
	}

	// Marketing Stats
	err = db.QueryRow(`SELECT COUNT(*) FROM sms_signups`).Scan(&stats.TotalSMSSignups)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalSMSSignups = 0
	}

	// Analytics Stats
	err = db.QueryRow(`SELECT COUNT(*) FROM analytics_pageviews`).Scan(&stats.TotalPageviews)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalPageviews = 0
	}

	// Pageviews Today
	err = db.QueryRow(`
		SELECT COUNT(*) FROM analytics_pageviews
		WHERE created_at >= ?
	`, todayStart).Scan(&stats.PageviewsToday)
	if err != nil && err != sql.ErrNoRows {
		stats.PageviewsToday = 0
	}

	// Pageviews This Week
	err = db.QueryRow(`
		SELECT COUNT(*) FROM analytics_pageviews
		WHERE created_at >= ?
	`, weekStart).Scan(&stats.PageviewsThisWeek)
	if err != nil && err != sql.ErrNoRows {
		stats.PageviewsThisWeek = 0
	}

	// Pageviews This Month
	err = db.QueryRow(`
		SELECT COUNT(*) FROM analytics_pageviews
		WHERE created_at >= ?
	`, monthStart).Scan(&stats.PageviewsThisMonth)
	if err != nil && err != sql.ErrNoRows {
		stats.PageviewsThisMonth = 0
	}

	// Unique Visitors Today
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT session_id) FROM analytics_pageviews
		WHERE created_at >= ?
	`, todayStart).Scan(&stats.UniqueVisitorsToday)
	if err != nil && err != sql.ErrNoRows {
		stats.UniqueVisitorsToday = 0
	}

	// Messages Stats
	err = db.QueryRow(`SELECT COUNT(*) FROM messages WHERE status = 'unread'`).Scan(&stats.UnreadMessages)
	if err != nil && err != sql.ErrNoRows {
		stats.UnreadMessages = 0
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&stats.TotalMessages)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalMessages = 0
	}

	return stats, nil
}

type RecentOrder struct {
	ID            int
	OrderNumber   string
	CustomerName  string
	CustomerEmail string
	Total         float64
	PaymentStatus string
	CreatedAt     time.Time
}

func (s *AdminServer) GetRecentOrders(websiteID string, limit int) ([]RecentOrder, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			id,
			order_number,
			customer_name,
			customer_email,
			total,
			payment_status,
			created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []RecentOrder
	for rows.Next() {
		var order RecentOrder
		err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.CustomerName,
			&order.CustomerEmail,
			&order.Total,
			&order.PaymentStatus,
			&order.CreatedAt,
		)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// Message Management Queries

type Message struct {
	ID           int
	Name         string
	Email        string
	Message      string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ReplyCount   int
	LastReplyAt  *time.Time
}

type MessageWithReplies struct {
	ID        int
	Name      string
	Email     string
	Message   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Replies   []MessageReply
}

type MessageReply struct {
	ID        int
	MessageID int
	ReplyText string
	SentAt    time.Time
	SentBy    string
}

func (s *AdminServer) GetMessages(websiteID string) ([]Message, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			m.id,
			m.name,
			m.email,
			m.message,
			m.status,
			m.created_at,
			m.updated_at,
			COALESCE(COUNT(r.id), 0) as reply_count,
			MAX(r.sent_at) as last_reply_at
		FROM messages m
		LEFT JOIN message_replies r ON m.id = r.message_id
		GROUP BY m.id, m.name, m.email, m.message, m.status, m.created_at, m.updated_at
		ORDER BY m.created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.ID,
			&msg.Name,
			&msg.Email,
			&msg.Message,
			&msg.Status,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.ReplyCount,
			&msg.LastReplyAt,
		)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *AdminServer) GetMessage(websiteID string, messageID int) (*MessageWithReplies, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Get the message
	messageQuery := `
		SELECT id, name, email, message, status, created_at, updated_at
		FROM messages
		WHERE id = ?
	`

	var msg MessageWithReplies
	err = db.QueryRow(messageQuery, messageID).Scan(
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

	// Get replies
	repliesQuery := `
		SELECT id, message_id, reply_text, sent_at, sent_by
		FROM message_replies
		WHERE message_id = ?
		ORDER BY sent_at ASC
	`

	rows, err := db.Query(repliesQuery, messageID)
	if err != nil {
		return &msg, nil // Return message even if replies fail
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

	msg.Replies = replies
	return &msg, nil
}

func (s *AdminServer) GetUnreadMessageCount(websiteID string) (int, error) {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `SELECT COUNT(*) FROM messages WHERE status = 'unread'`
	var count int
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *AdminServer) DeleteMessage(websiteID string, messageID int) error {
	db, err := s.GetWebsiteConnection(websiteID)
	if err != nil {
		return err
	}
	defer db.Close()

	// Delete the message (replies will be cascade deleted due to FOREIGN KEY constraint)
	query := `DELETE FROM messages WHERE id = ?`
	_, err = db.Exec(query, messageID)
	return err
}
