package structs

import (
	"bytes"
	"html/template"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

type Post struct {
	ID            int           `json:"id"`
	Slug          string        `json:"slug"`
	Title         string        `json:"title"`
	Type          string        `json:"type"`
	PublishedDate time.Time     `json:"published_date"`
	Modified      time.Time     `json:"modified"`
	Updated       time.Time     `json:"updated"`
	Content       string        `json:"content"`
	ParsedContent template.HTML `json:"-"`
	Description   string        `json:"description"`
	Deck          string        `json:"deck"`
	Coverline     string        `json:"coverline"`
	Status        string        `json:"status"`
	ThumbnailID   int           `json:"thumbnail_id"`
	DuplicationID int           `json:"duplication_id"`
	URL           string        `json:"url"`
	CanonicalURL  string        `json:"canonical_url"`
	Keywords      string        `json:"keywords"`
	Authors       []Author      `json:"authors"`
	Categories    []Category    `json:"categories"`
	Tags          []Tag         `json:"tags"`
	Image         Image         `json:"image"`
	Slides        []Slide       `json:"slides"`
}

type Slide struct {
	SlidePosition      int           `json:"slide_position"`
	Title              string        `json:"title"`
	PreImageDesc       string        `json:"pre_image_desc"`
	ParsedPreImageDesc template.HTML `json:"-"`
	Description        string        `json:"description"`
	ParsedDescription  template.HTML `json:"-"`
	Image              Image         `json:"image"`
	DuplicationFound   int           `json:"duplication_found"`
}

type Image struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	AltText string `json:"alt_text"`
	Credit  string `json:"credit"`
}

type Author struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ShortBio string `json:"short_bio"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	ImageUrl    string `json:"image_url"`
	AltText     string `json:"alt_text"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// E-commerce Structs

type Product struct {
	ID                int              `json:"id"`
	Name              string           `json:"name"`
	Slug              string           `json:"slug"`
	Description       string           `json:"description"`
	Price             float64          `json:"price"`
	CompareAtPrice    float64          `json:"compare_at_price"`
	SKU               string           `json:"sku"`
	InventoryQuantity int              `json:"inventory_quantity"`
	InventoryPolicy   string           `json:"inventory_policy"`
	Status            string           `json:"status"`
	Featured          bool             `json:"featured"`
	SortOrder         int              `json:"sort_order"`
	Images            []ProductImage   `json:"images"`
	Variants          []ProductVariant `json:"variants"`
	Collections       []Collection     `json:"collections"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	ReleasedDate      time.Time        `json:"released_date"`
}

type Collection struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	Image        Image     `json:"image"`
	SortOrder    int       `json:"sort_order"`
	Status       string    `json:"status"`
	ProductCount int       `json:"product_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProductVariant struct {
	ID                int     `json:"id"`
	ProductID         int     `json:"product_id"`
	Title             string  `json:"title"`
	Option1           string  `json:"option1"`
	Option2           string  `json:"option2"`
	Option3           string  `json:"option3"`
	Price             float64 `json:"price"`
	CompareAtPrice    float64 `json:"compare_at_price"`
	SKU               string  `json:"sku"`
	InventoryQuantity int     `json:"inventory_quantity"`
	Position          int     `json:"position"`
}

type ProductImage struct {
	ID       int   `json:"id"`
	ImageID  int   `json:"image_id"`
	Position int   `json:"position"`
	Image    Image `json:"image"`
}

type Cart struct {
	ID        string     `json:"id"`
	Items     []CartItem `json:"items"`
	Subtotal  float64    `json:"subtotal"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

type CartItem struct {
	ID        int            `json:"id"`
	ProductID int            `json:"product_id"`
	VariantID int            `json:"variant_id"`
	Product   Product        `json:"product"`
	Variant   ProductVariant `json:"variant"`
	Quantity  int            `json:"quantity"`
	Price     float64        `json:"price"`
	Total     float64        `json:"total"`
}

type Order struct {
	ID                   int         `json:"id"`
	OrderNumber          string      `json:"order_number"`
	CustomerEmail        string      `json:"customer_email"`
	CustomerName         string      `json:"customer_name"`
	ShippingAddressLine1 string      `json:"shipping_address_line1"`
	ShippingAddressLine2 string      `json:"shipping_address_line2"`
	ShippingCity         string      `json:"shipping_city"`
	ShippingState        string      `json:"shipping_state"`
	ShippingZip          string      `json:"shipping_zip"`
	ShippingCountry      string      `json:"shipping_country"`
	BillingAddressLine1  string      `json:"billing_address_line1"`
	BillingCity          string      `json:"billing_city"`
	BillingState         string      `json:"billing_state"`
	BillingZip           string      `json:"billing_zip"`
	BillingCountry       string      `json:"billing_country"`
	Subtotal             float64     `json:"subtotal"`
	Tax                  float64     `json:"tax"`
	ShippingCost         float64     `json:"shipping_cost"`
	Total                float64     `json:"total"`
	PaymentStatus        string      `json:"payment_status"`
	FulfillmentStatus    string      `json:"fulfillment_status"`
	PaymentMethod        string      `json:"payment_method"`
	StripePaymentIntent  string      `json:"stripe_payment_intent_id"`
	Items                []OrderItem `json:"items"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID           int     `json:"id"`
	ProductID    int     `json:"product_id"`
	VariantID    int     `json:"variant_id"`
	ProductName  string  `json:"product_name"`
	VariantTitle string  `json:"variant_title"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
	Total        float64 `json:"total"`
}

type ParserOptions struct {
	StripTags bool
}

func (post *Post) ParseContent(options *ParserOptions) error {

	// parsed content
	content, err := parseHtml(post.Content, &ParserOptions{})
	if err != nil {
		return err
	}
	post.ParsedContent = content

	// parsed slide pre img desc and desc
	for i, slide := range post.Slides {

		preImgDesc, err := parseHtml(slide.PreImageDesc, &ParserOptions{})
		if err != nil {
			return err
		}
		post.Slides[i].ParsedPreImageDesc = preImgDesc

		desc, err := parseHtml(slide.Description, &ParserOptions{})
		if err != nil {
			return err
		}
		post.Slides[i].ParsedDescription = desc
	}

	return nil
}

// Parses a string with HTML into rendered HTML
func parseHtml(input string, options *ParserOptions) (template.HTML, error) {

	htmlInput, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	body := cascadia.MustCompile("body").MatchFirst(htmlInput)
	bodyTemplate := template.HTML(nodeString(body))

	/*var paragraphTemplates []template.HTML
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		childString := nodeString(child)
		if childString != "" {
			paragraphTemplates = append(paragraphTemplates, template.HTML(nodeString(child)))
		}
	}*/

	return bodyTemplate, nil
}

func nodeString(n *html.Node) string {
	buf := bytes.NewBufferString("")
	html.Render(buf, n)
	str := buf.String()
	if len(str) < 8 {
		return ""
	}
	return buf.String()
}
