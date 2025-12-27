# Stencil2

A high-performance, multi-site template engine and content management platform written in Go. Stencil2 enables you to serve multiple independent websites from a single server instance, each with its own configuration, templates, and database.

## Features

### Core Platform
- **Multi-Site Hosting**: Host multiple independent websites with separate configurations, templates, and databases
- **Built-in Admin CMS**: Web-based admin interface for managing websites, articles, products, orders, and customers
- **Powerful Template Engine**: Go templates with custom functions and Sprig library integration
- **Asset Pipeline**: Automatic CSS/JS minification and combination with cache busting
- **REST API**: Comprehensive JSON API (v1) for programmatic content access
- **Dynamic Routing**: Template-based route generation with pagination support
- **Media Proxy**: On-the-fly image resizing with width parameter
- **Sitemap Generation**: Automatic XML sitemap generation from database content
- **Development Tools**: File watcher for hot-reload and error debugging
- **Production Ready**: Includes systemd service and Nginx configuration examples

### E-commerce Features
- **Product Management**: Products with variants, SKUs, inventory tracking, and pricing
- **Collections**: Organize products into collections with sort ordering
- **Shopping Cart**: Session-based cart with 7-day expiry
- **Customer Tracking**: Automatic customer creation with order history and spending analytics
- **Stripe Integration**: Payment processing with Stripe payment intents and customer objects
- **Shippo Shipping**: Real-time shipping rate calculation, label generation, and tracking
- **Order Management**: Complete order workflow with fulfillment status and tracking
- **Email Notifications**: AWS SES integration for order confirmation emails
- **Tax Calculation**: Configurable tax rates per website
- **Address Validation**: Shippo-powered address validation during checkout

### Content Features
- **Articles & Posts**: Full-featured content management with multiple article types
- **Categories & Tags**: Organize content with categories, tags, and authors
- **Gallery Support**: Multi-slide galleries with images and captions
- **Featured Content**: Flag articles and products as featured
- **Preview Mode**: Preview draft content before publishing
- **SEO Support**: Canonical URLs, keywords, and meta descriptions

### Marketing Features
- **SMS Signups**: Collect phone numbers for marketing with country code support
- **Early Access Control**: Password-protect sites during development with public page exceptions
- **Email Marketing**: Customer and SMS signup lists for marketing campaigns

### Analytics & Insights
- **Custom Analytics System**: Privacy-focused, lightweight analytics built into the platform (no external dependencies)
- **Real-Time Monitoring**: Live active user count and current page views (last 5 minutes)
- **Traffic Analytics**: Pageviews, unique visitors, sessions, bounce rate, and session duration
- **E-Commerce Analytics**: Conversion rate, cart abandonment rate, revenue metrics, and average order value
- **User Behavior**: Entry pages, exit pages, top pages, and visitor referral sources
- **Device Analytics**: Mobile, tablet, and desktop traffic breakdown
- **Custom Event Tracking**: JavaScript API for tracking custom events (add to cart, checkout, purchases, etc.)
- **Heartbeat Tracking**: 30-second heartbeat signals for accurate session duration and active user detection
- **Session Management**: Automatic session detection with 30-minute timeout and localStorage persistence
- **Admin Dashboard**: Beautiful analytics dashboard with time period selectors (7/30/90/365 days)

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Environment Configuration](#environment-configuration)
  - [Website Configuration](#website-configuration)
  - [Template Configuration](#template-configuration)
- [CLI Commands](#cli-commands)
- [Directory Structure](#directory-structure)
- [Template System](#template-system)
- [Analytics System](#analytics-system)
  - [How It Works](#how-it-works)
  - [JavaScript API](#javascript-api)
  - [Admin Dashboard](#admin-dashboard)
  - [Privacy & Performance](#privacy--performance)
- [API Endpoints](#api-endpoints)
- [Database Schema](#database-schema)
- [Deployment](#deployment)
- [Development](#development)

## Installation

### Prerequisites

- Go 1.20 or higher
- MySQL 5.5+ or MariaDB 10.1+

### Build from Source

```bash
# Clone the repository
git clone git@github.com:murdinc/stencil2.git
cd stencil2

# Install dependencies
go mod download

# Build for your platform
go build -o stencil2 main.go

# Or cross-compile for different platforms
env GOOS=linux GOARCH=amd64 go build -o ./builds/linux/stencil2 main.go
env GOOS=darwin GOARCH=arm64 go build -o ./builds/osx_m1/stencil2 main.go
env GOOS=darwin GOARCH=amd64 go build -o ./builds/osx_intel/stencil2 main.go
```

## Quick Start

### 1. Set Up Configuration

Create environment configuration file:

```bash
# For development
cat > websites/env-dev.json << EOF
{
  "database": {
    "host": "localhost",
    "user": "root",
    "port": "3306",
    "password": "",
    "name": "stencil2"
  },
  "http": {
    "port": "8080"
  }
}
EOF
```

Create a website configuration:

```bash
mkdir -p websites/example.com
cat > websites/example.com/config-dev.json << EOF
{
  "siteName": "example.com",
  "apiVersion": 1,
  "database": {
    "name": "example_db"
  },
  "http": {
    "address": "example.com"
  }
}
EOF
```

### 2. Create Templates

```bash
mkdir -p websites/example.com/templates/homepage
mkdir -p websites/example.com/public
mkdir -p websites/example.com/sitemaps
```

Create a template configuration (`templates/homepage/homepage.json`):

```json
{
  "name": "homepage",
  "path": "/",
  "apiEndpoint": "/api/v1/posts",
  "cacheTime": 300
}
```

Create a template file (`templates/homepage/homepage.tpl`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ sitename }}</title>
</head>
<body>
    <h1>Welcome to {{ sitename }}</h1>
    {{ range .Posts }}
        <article>
            <h2>{{ .Title }}</h2>
            <p>{{ .Deck }}</p>
        </article>
    {{ end }}
</body>
</html>
```

### 3. Start the Server

```bash
# Development mode
./stencil2 serve

# Or in production mode
./stencil2 --prod-mode serve
```

**Note**: On first startup, Stencil2 automatically creates all necessary database tables (article tables and e-commerce tables) if they don't exist. No manual SQL imports required!

### 4. Access Your Site

Add to `/etc/hosts`:
```
127.0.0.1 example.com
```

Visit `http://example.com:8080`

## Admin Backend (CMS)

Stencil2 includes a built-in web-based admin interface for managing websites, articles, and products.

### Enabling the Admin

The admin backend is configured in `websites/env-dev.json`:

```json
{
  "admin": {
    "enabled": true,
    "port": "8081",
    "password": "your-secure-password",
    "database": {
      "name": "stencil_admin"
    }
  }
}
```

**Configuration options**:
- `enabled`: Set to `true` to start the admin server
- `port`: Port for admin interface (default: 8081)
- `password`: Admin login password (change this!)
- `database.name`: Database name for admin data (default: stencil_admin)

### Accessing the Admin

1. Start the server: `./stencil2 serve`
2. Visit: `http://localhost:8081/login`
3. Enter your password
4. You'll see the dashboard with all your websites

### Admin Features

**Website Management**:
- Create new websites (automatically creates folder structure and config files)
- Edit website settings (Stripe keys, Shippo credentials, email config, tax rates, shipping)
- Delete websites
- Each website gets its own database automatically created
- Configure early access password protection

**Article/Content Management**:
- Create, edit, and delete articles
- Set article type (article, page, gallery)
- Set status (draft, published, archived)
- Manage article content, excerpts, and metadata
- Set published dates and featured flag
- Assign categories, authors, and tags
- Manage multi-slide galleries with images

**Product Management**:
- Create, edit, and delete products
- Set pricing and compare-at pricing
- Manage inventory and SKUs
- Set product status and featured flag
- Configure inventory policies
- Add product variants (size, color, etc.)
- Upload multiple product images with ordering
- Assign products to collections
- Reorder products with up/down controls
- Set release dates

**Order Management**:
- View all orders with filtering and sorting
- View order details (items, customer info, shipping, payment)
- Update order status (pending, processing, fulfilled, cancelled)
- View payment and fulfillment status
- Add tracking numbers
- Resend order confirmation emails
- View order timeline and notes

**Customer Management**:
- View all customers with stats (order count, total spent)
- Filter and sort customers by total spent, order count, date joined
- View customer details and order history
- View Stripe customer ID integration
- Track first and last order dates
- Calculate average order value

**SMS Signups Management**:
- View all SMS signups
- Filter by country code, source, and date range
- Sort by date or phone number
- Export filtered data to CSV
- Delete signups
- Track signup source (which page/form)

**Category & Collection Management**:
- Create and delete article categories
- Create and delete product collections
- Automatically generates slugs
- Assign multiple collections to products

**Image Management**:
- Upload and manage images
- Track image URLs and metadata
- Use images in articles, products, and galleries
- Set alt text and credits

**Site Settings**:
- Configure Stripe integration
- Configure Shippo shipping
- Set email sender details (AWS SES)
- Configure tax rates
- Set flat shipping costs
- Manage early access settings

### Admin Database

The admin uses its own database (`stencil_admin` by default) which stores:
- `admin_websites` - Registry of all websites
- `admin_activity_log` - Audit log of all admin actions

Each website's content (articles, products) is stored in that website's own database, keeping data isolated.

### Security Notes

- **Change the default password** in `env-dev.json`
- Admin uses session-based authentication (24-hour sessions)
- Sessions are stored in memory (will be lost on server restart)
- For production, use HTTPS and a strong password
- Consider adding IP restrictions via firewall

## Site Types

Stencil2 supports two types of websites, and **a single site can be both**:

### Article/Content Sites

For blogs, news sites, magazines, and content-driven websites.

**Auto-created tables**:
- `articles_unified` - Articles, blog posts, pages, galleries
- `categories_unified` - Article categories
- `authors_unified` - Author profiles
- `tags_unified` - Article tags
- `images_unified` - Image library
- `article_information` - Denormalized JSON data for fast queries
- Relationship tables: `article_authors`, `article_categories`, `article_tags`
- Gallery support: `article_slides`
- Preview mode: `preview_article_information`, `preview_article_slides`

**API Endpoints** (see [API Endpoints](#api-endpoints) for full list):
- `GET /api/v1/posts` - List articles
- `GET /api/v1/post/{slug}` - Single article
- `GET /api/v1/category/{slug}/posts` - Articles by category
- `GET /api/v1/author/{slug}/posts` - Articles by author
- `GET /api/v1/tag/{slug}/posts` - Articles by tag

**Example template config**:
```json
{
  "name": "homepage",
  "path": "/",
  "apiEndpoint": "/api/v1/posts",
  "apiCount": 10,
  "cacheTime": 300
}
```

### E-commerce Sites

For online stores, product catalogs, and shopping experiences.

**Auto-created tables**:
- `products_unified` - Product catalog with pricing, inventory, SKUs
- `collections_unified` - Product collections (like categories)
- `product_variants` - Size, color, and other variations
- `product_images` - Product image galleries
- `carts` - Shopping cart sessions (7-day expiry)
- `cart_items` - Items in shopping carts
- `orders` - Customer orders with shipping/billing
- `order_items` - Order line items

**API Endpoints** (see [ECOMMERCE.md](ECOMMERCE.md) for full documentation):
- `GET /api/v1/products` - List products
- `GET /api/v1/product/{slug}` - Single product
- `GET /api/v1/collections` - List collections
- `GET /api/v1/collection/{slug}/products` - Products in collection
- `POST /api/v1/cart/add` - Add to cart
- `POST /api/v1/checkout` - Process checkout
- `GET /api/v1/order/{orderNumber}` - View order

**Example template config**:
```json
{
  "name": "store",
  "path": "/store",
  "apiEndpoint": "/api/v1/products",
  "apiCount": 12,
  "cacheTime": 300
}
```

### Hybrid Sites (Both Article + E-commerce)

A single website can use both article and e-commerce features simultaneously. For example:
- A blog with a merch store
- A news site with subscription products
- A magazine with an e-commerce section

Simply use both types of API endpoints in different templates:

```json
// Homepage with latest articles
{
  "name": "homepage",
  "path": "/",
  "apiEndpoint": "/api/v1/posts"
}
```

```json
// Store page with products
{
  "name": "store",
  "path": "/store",
  "apiEndpoint": "/api/v1/products"
}
```

**All tables are created automatically** when the server starts, so you can use whichever features you need without any manual database setup.

## Configuration

### Environment Configuration

Located at `websites/env-dev.json` or `websites/env-prod.json`:

```json
{
  "database": {
    "host": "localhost",      // Database host
    "user": "root",            // Database user
    "port": "3306",            // Database port
    "password": "",            // Database password
    "name": "stencil2"         // Root database name
  },
  "http": {
    "port": "80"               // HTTP server port
  }
}
```

### Website Configuration

Located at `websites/{site}/config-dev.json` or `websites/{site}/config-prod.json`:

```json
{
  "siteName": "example.com",
  "apiVersion": 1,
  "database": {
    "name": "example_db"
  },
  "mediaProxyUrl": "https://media.example.com",
  "http": {
    "address": "example.com"
  },
  "stripe": {
    "publishableKey": "pk_test_...",
    "secretKey": "sk_test_..."
  },
  "shippo": {
    "apiKey": "shippo_test_...",
    "labelFormat": "PDF"
  },
  "email": {
    "provider": "ses",
    "fromAddress": "orders@example.com",
    "fromName": "Example Store",
    "replyTo": "support@example.com"
  },
  "ecommerce": {
    "taxRate": 0.08,
    "flatShippingCost": 5.00
  },
  "earlyAccess": {
    "enabled": false,
    "password": "your-password-here"
  },
  "shipFrom": {
    "name": "Example Warehouse",
    "street1": "123 Main St",
    "city": "San Francisco",
    "state": "CA",
    "zip": "94102",
    "country": "US",
    "phone": "415-555-0100"
  }
}
```

**Configuration Fields:**

| Field | Description |
|-------|-------------|
| `siteName` | Domain name for the website |
| `apiVersion` | API version (currently only v1 supported) |
| `database.name` | Site-specific database name |
| `mediaProxyUrl` | Optional media proxy URL for image resizing |
| `http.address` | Host header for routing requests |
| `stripe.publishableKey` | Stripe publishable key for frontend |
| `stripe.secretKey` | Stripe secret key for backend |
| `shippo.apiKey` | Shippo API key for shipping |
| `shippo.labelFormat` | Label format (PDF, PNG, ZPLII) |
| `email.provider` | Email provider (currently only "ses" supported) |
| `email.fromAddress` | Sender email address |
| `email.fromName` | Sender name |
| `email.replyTo` | Reply-to email address |
| `ecommerce.taxRate` | Tax rate as decimal (0.08 = 8%) |
| `ecommerce.flatShippingCost` | Flat shipping cost (if not using Shippo) |
| `earlyAccess.enabled` | Enable early access password protection |
| `earlyAccess.password` | Password for early access |
| `shipFrom.*` | Default shipping origin address for Shippo |

### Template Configuration

Located at `websites/{site}/templates/{template-name}/{template-name}.json`:

#### All Available Options

```json
{
  "name": "homepage",
  "path": "/",
  "paginateType": 0,
  "requires": ["common"],
  "jsFile": "main.js",
  "cssFile": "main.css",
  "queryRow": "custom_query",
  "apiEndpoint": "/api/v1/posts",
  "apiTaxonomy": "category",
  "apiSlug": "technology",
  "apiCount": 10,
  "apiOffset": 0,
  "mimeType": "text/html",
  "noCache": false,
  "cacheTime": 300,
  "publicAccess": false
}
```

#### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | **Required.** Template identifier, must match directory name |
| `path` | string | URL path for this template (e.g., `/`, `/about`, `/store/product/{slug}`) |
| `paginateType` | int | Pagination mode: `0` = none, `1` = paginated URLs, `2` = 302 redirect to paginated URL |
| `requires` | string[] | List of template directories to include (e.g., `["common", "sidebar"]`) |
| `jsFile` | string | JavaScript file to load from template directory |
| `cssFile` | string | CSS file to load from template directory |
| `queryRow` | string | Custom database query identifier (advanced) |
| `apiEndpoint` | string | API endpoint to fetch data from (e.g., `/api/v1/posts`, `/api/v1/products`) |
| `apiTaxonomy` | string | Filter by taxonomy: `category`, `tag`, `author`, or `type` |
| `apiSlug` | string | Slug value for taxonomy filter (e.g., `technology` for category) |
| `apiCount` | int | Number of items to fetch from API (default varies by endpoint) |
| `apiOffset` | int | Offset for pagination (skip first N items) |
| `mimeType` | string | Response content type (default: `text/html`) |
| `noCache` | bool | If `true`, disables all caching for this template |
| `cacheTime` | int | Cache TTL in seconds (default: 0 = no cache) |
| `publicAccess` | bool | If `true`, page is accessible even when early access protection is enabled |

#### Common Examples

**Simple Homepage**:
```json
{
  "name": "homepage",
  "path": "/",
  "apiEndpoint": "/api/v1/posts",
  "apiCount": 10,
  "cacheTime": 300
}
```

**Product Page with Dynamic Slug**:
```json
{
  "name": "product",
  "path": "/store/product/{slug}",
  "apiEndpoint": "/api/v1/product/{slug}",
  "requires": ["common"],
  "noCache": true
}
```

**Category Archive with Pagination**:
```json
{
  "name": "category",
  "path": "/category/{slug}",
  "paginateType": 1,
  "apiEndpoint": "/api/v1/category/{slug}/posts",
  "apiCount": 20,
  "cacheTime": 600
}
```

**Public Page (Accessible During Early Access Lockdown)**:
```json
{
  "name": "sms-signup",
  "path": "/sms-signup",
  "noCache": true,
  "publicAccess": true
}
```

**Custom MIME Type (JSON API)**:
```json
{
  "name": "api-posts",
  "path": "/posts.json",
  "apiEndpoint": "/api/v1/posts",
  "mimeType": "application/json",
  "noCache": true
}
```

## CLI Commands

### serve

Start the HTTP server to serve all configured websites.

```bash
./stencil2 serve                    # Development mode
./stencil2 --prod-mode serve        # Production mode
./stencil2 serve --hide-errors      # Hide friendly error pages (dev only)
```

### sitemaps

Generate XML sitemaps for all configured websites.

```bash
./stencil2 sitemaps              # Build sitemaps
./stencil2 sitemaps --init       # Initialize sitemap tables
```

Sitemaps are generated at:
- `websites/{site}/sitemaps/sitemap-YYYY-MM.xml` (monthly sitemaps)
- `websites/{site}/sitemaps/sitemaps-index.xml` (sitemap index)

## Directory Structure

```
stencil2/
├── api/                          # API route handlers
│   ├── v1.go                     # V1 API implementation
│   └── routes.go                 # Route definitions
├── cmd/                          # CLI commands
│   ├── root.go                   # Root command with flags
│   ├── serve.go                  # Web server command
│   └── sitemaps.go               # Sitemap generation command
├── configs/                      # Configuration loaders
│   ├── env.go                    # Environment config loader
│   ├── website.go                # Website config loader
│   └── template.go               # Template config loader
├── database/                     # Database layer
│   ├── client.go                 # Connection management
│   └── queries.go                # Query methods
├── frontend/                     # Website rendering
│   ├── router.go                 # Route registration
│   ├── websites.go               # Website instance management
│   ├── templates.go              # Template rendering
│   ├── helpers.go                # File watchers and utilities
│   ├── sitemaps.go               # Sitemap generation
│   ├── css.go                    # CSS asset pipeline
│   └── js.go                     # JS asset pipeline
├── media/                        # Image processing
│   └── proxy.go                  # Image resizing and proxy
├── structs/                      # Data models
│   ├── post.go                   # Post/Article structure
│   ├── category.go               # Category structure
│   ├── author.go                 # Author structure
│   └── image.go                  # Image structure
├── setup/                        # Deployment configs
│   ├── stencil2.service          # Systemd service file
│   └── stencil2.conf             # Nginx configuration
├── websites/                     # Website configurations (gitignored)
│   ├── env-dev.json              # Dev environment config
│   ├── env-prod.json             # Prod environment config
│   └── {site-name}/
│       ├── config-dev.json       # Dev website config
│       ├── config-prod.json      # Prod website config
│       ├── templates/            # Template files and configs
│       │   └── {template-name}/
│       │       ├── {template-name}.json  # Template config
│       │       ├── {template-name}.tpl   # Template file
│       │       ├── *.css                 # CSS files
│       │       └── *.js                  # JavaScript files
│       ├── public/               # Static assets (served at /public/)
│       └── sitemaps/             # Generated sitemaps (served at /sitemaps/)
├── main.go                       # Application entry point
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
└── README.md                     # This file
```

## Template System

### Available Template Functions

Stencil2 includes all [Sprig template functions](http://masterminds.github.io/sprig/) plus custom functions:

- `{{ sitename }}` - Returns the configured site name
- `{{ hash }}` - Returns asset hash for cache busting (e.g., `/public/style.css?v={{ hash }}`)
- `{{ mediaproxyurl }}` - Returns the media proxy base URL
- `{{ mediaproxy 800 "https://example.com/image.jpg" }}` - Generates a resized image URL at 800px width

### Template Data

Templates receive a `PageData` object with the following fields:

```go
.ProdMode         // bool - Production mode flag
.HideErrors       // bool - Hide error details flag
.Slug             // string - Current URL slug
.Page             // string - Current page number
.Categories       // []Category - List of categories
.Post             // Post - Single post (for post templates)
.Posts            // []Post - List of posts (for list templates)
.Template         // TemplateConfig - Current template config
.Preview          // bool - Preview mode flag
```

### Template Inheritance

Templates can require other templates using the `requires` field:

```json
{
  "name": "article",
  "requires": ["common", "sidebar"]
}
```

All `.tpl` files from required template directories will be available for use with `{{ template "name" . }}`.

**Common Pattern - Shared Components**:

A typical pattern is to create a `common` template that defines reusable components like headers, footers, and base styles:

```html
<!-- templates/common/common.tpl -->
{{define "header"}}
<header>
    <nav>
        <a href="/">Home</a>
        <a href="/shop">Shop</a>
    </nav>
</header>
{{end}}

{{define "footer"}}
<footer>
    <p>&copy; 2025 My Site</p>
</footer>
{{end}}

{{define "styles"}}
<style>
    body { font-family: sans-serif; }
    header { background: #333; color: white; }
</style>
{{end}}
```

Then other templates can require and use these components:

```html
<!-- templates/homepage/homepage.tpl -->
<!DOCTYPE html>
<html>
<head>
    <title>{{ sitename }}</title>
    {{template "styles" .}}
    <style>
        /* Page-specific styles */
    </style>
</head>
<body>
    {{template "header" .}}

    <main>
        <!-- Page content -->
    </main>

    {{template "footer" .}}
</body>
</html>
```

```json
// templates/homepage/homepage.json
{
  "name": "homepage",
  "path": "/",
  "requires": ["common"]
}
```

This eliminates code duplication and makes it easy to maintain consistent branding across all pages.

### Example Templates

**Article Template**:
```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Post.Title }} - {{ sitename }}</title>
    <link rel="stylesheet" href="/public/style.css?v={{ hash }}">
</head>
<body>
    <article>
        <h1>{{ .Post.Title }}</h1>
        <div class="meta">
            Published: {{ .Post.PublishedDate.Format "January 2, 2006" }}
        </div>

        {{ if .Post.Image.URL }}
        <img src="{{ mediaproxy 1200 .Post.Image.URL }}" alt="{{ .Post.Image.AltText }}">
        {{ end }}

        <div class="content">
            {{ .Post.Content }}
        </div>

        {{ range .Post.Categories }}
            <a href="/category/{{ .Slug }}">{{ .Name }}</a>
        {{ end }}
    </article>
</body>
</html>
```

**Gallery Template**:
```html
{{ range .Post.Slides }}
<div class="slide">
    <h3>{{ .Title }}</h3>
    {{ if .PreImageDesc }}
        <div class="pre-desc">{{ .PreImageDesc }}</div>
    {{ end }}
    <img src="{{ mediaproxy 1200 .Image.URL }}" alt="{{ .Image.AltText }}">
    {{ if .Image.Credit }}
        <div class="credit">{{ .Image.Credit }}</div>
    {{ end }}
    {{ if .Description }}
        <div class="description">{{ .Description }}</div>
    {{ end }}
</div>
{{ end }}
```

## Analytics System

Stencil2 includes a built-in, privacy-focused analytics system that tracks visitor behavior, e-commerce conversions, and site performance without relying on external services like Google Analytics.

### How It Works

The analytics system uses a lightweight JavaScript tracker (~2KB) that automatically:
- Tracks pageviews on initial page load
- Generates unique session IDs stored in localStorage (30-minute timeout)
- Sends heartbeat signals every 30 seconds to track active sessions
- Detects device type (mobile/tablet/desktop) from screen dimensions
- Pauses tracking when the browser tab is hidden

All analytics data is stored in MySQL tables within each website's database:
- `analytics_pageviews` - Page visits with session, path, referrer, user agent, IP, and screen dimensions
- `analytics_events` - Custom events with event name, data payload, and session context

### JavaScript API

The analytics tracker is automatically loaded on all pages via `/public/analytics.js` and exposes a global `window.analytics` object:

#### Automatic Tracking

```javascript
// Pageviews are tracked automatically on page load
// No code needed - just include the script tag
```

#### Custom Event Tracking

```javascript
// Track a custom event
analytics.track('event_name', { key: 'value' });

// E-commerce helpers
analytics.trackAddToCart(productId, 'Product Name', 29.99, 1);
analytics.trackRemoveFromCart(productId);
analytics.trackCheckoutStarted(149.99, 3); // cart value, item count
analytics.trackPurchase('ORD-12345', 149.99, 3); // order ID, total, item count

// Content engagement helpers
analytics.trackScrollDepth(75); // percentage
analytics.trackClick('button', 'Subscribe CTA');
```

#### Session Management

Sessions are automatically managed:
- New session created on first visit
- Session ID persists in localStorage for 30 minutes of inactivity
- Session extends with each pageview or heartbeat
- Sessions expire after 30 minutes of no activity

### Admin Dashboard

Access analytics for each website via the admin panel at `/site/{id}/analytics`.

**Available Metrics:**

**Real-Time**
- Active users (last 5 minutes)
- Current pages being viewed
- Live activity feed

**Traffic Overview**
- Total pageviews
- Unique visitors (sessions)
- Average pages per visit
- Bounce rate (single-page sessions)
- Average session duration

**E-Commerce** (requires purchase tracking)
- Total revenue
- Number of orders
- Average order value
- Conversion rate (% of sessions with purchases)
- Cart abandonment rate (% who add to cart but don't buy)

**User Behavior**
- Top pages (most viewed)
- Entry pages (where users land)
- Exit pages (where users leave)
- Top referrers (traffic sources)
- Device breakdown (mobile/tablet/desktop)

**Custom Events**
- All tracked custom events with counts
- Filtered view (heartbeats hidden)

**Time Periods**
- Last 7 days
- Last 30 days (default)
- Last 90 days
- Last year

### Privacy & Performance

**Privacy Features:**
- No cookies required (uses localStorage for session management)
- No third-party requests (all data stays on your server)
- IP addresses stored but not used for tracking individuals
- No cross-site tracking or advertising IDs
- Full data ownership and control

**Performance:**
- Minimal JavaScript footprint (~2KB gzipped)
- Async beacon API (doesn't block page load)
- Automatic heartbeat pauses when tab is hidden
- Database indexes on frequently queried columns
- Efficient aggregation queries for dashboard

**Database Tables:**

```sql
CREATE TABLE analytics_pageviews (
    id INT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(100) NOT NULL,
    path VARCHAR(500) NOT NULL,
    referrer VARCHAR(500),
    user_agent VARCHAR(500),
    ip_address VARCHAR(100),
    screen_width INT,
    screen_height INT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session (session_id),
    INDEX idx_created (created_at),
    INDEX idx_path (path(255))
);

CREATE TABLE analytics_events (
    id INT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(100) NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    event_data JSON,
    path VARCHAR(500),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session (session_id),
    INDEX idx_event (event_name),
    INDEX idx_created (created_at)
);
```

**Auto-Deployment:**

The analytics JavaScript file is automatically copied from `frontend/static/analytics.js` to each website's `public/` directory on server startup, ensuring all sites stay in sync with the latest tracker version.

## API Endpoints

Stencil2 provides a comprehensive RESTful JSON API (v1) for all configured websites.

### Content Endpoints

#### Categories

**GET** `/api/v1/categories` - Get all categories

Query Parameters:
- `full=true` - Include category images

#### Posts

**GET** `/api/v1/posts` - Get all posts
**GET** `/api/v1/posts/{count}` - Get N posts
**GET** `/api/v1/posts/{count}/{offset}` - Get N posts with offset

Query Parameters:
- `full=true` - Include post content and slides
- `featured=false` - Exclude featured posts
- `sort=modified` - Sort by modified date instead of published date

**GET** `/api/v1/post/{slug}` - Get single post by slug

Query Parameters:
- `preview=true` - Get draft/preview version of post

#### Taxonomy Posts

**GET** `/api/v1/{taxonomy}/{slug}/posts` - Get posts by taxonomy
**GET** `/api/v1/{taxonomy}/{slug}/posts/{count}/{offset}` - With pagination

Taxonomy types: `category`, `tag`, `author`, `type`

---

### E-commerce Endpoints

#### Products

**GET** `/api/v1/products` - Get all products
**GET** `/api/v1/products/{count}` - Get N products
**GET** `/api/v1/products/{count}/{offset}` - Get N products with offset

**GET** `/api/v1/product/{slug}` - Get single product by slug

#### Collections

**GET** `/api/v1/collections` - Get all collections
**GET** `/api/v1/collection/{slug}/products` - Get products in collection
**GET** `/api/v1/collection/{slug}/products/{count}/{offset}` - With pagination

#### Shopping Cart

**POST** `/api/v1/cart/add` - Add item to cart

Request body:
```json
{
  "product_id": 123,
  "variant_id": 456,
  "quantity": 2
}
```

**POST** `/api/v1/cart/update` - Update cart item quantity

Request body:
```json
{
  "item_id": 789,
  "quantity": 3
}
```

**POST** `/api/v1/cart/remove` - Remove item from cart

Request body:
```json
{
  "item_id": 789
}
```

**GET** `/api/v1/cart` - Get current cart contents

#### Checkout & Orders

**POST** `/api/v1/payment-intent` - Create Stripe payment intent

Request body:
```json
{
  "email": "customer@example.com",
  "shipping": {
    "name": "John Doe",
    "address": {
      "line1": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "postal_code": "94102",
      "country": "US"
    },
    "phone": "415-555-0100"
  }
}
```

**POST** `/api/v1/checkout` - Create order from cart

Request body:
```json
{
  "payment_intent_id": "pi_...",
  "customer_email": "customer@example.com",
  "customer_name": "John Doe",
  "shipping_address": {...},
  "billing_address": {...}
}
```

**GET** `/api/v1/order/{orderNumber}` - Get order details

**POST** `/api/v1/webhook/stripe` - Stripe webhook handler (for payment events)

#### Shipping

**POST** `/api/v1/shipping/rates` - Get shipping rates

Request body:
```json
{
  "address": {
    "name": "John Doe",
    "street1": "123 Main St",
    "city": "San Francisco",
    "state": "CA",
    "zip": "94102",
    "country": "US"
  },
  "parcel": {
    "length": "10",
    "width": "8",
    "height": "4",
    "weight": "1.5"
  }
}
```

**POST** `/api/v1/shipping/validate-address` - Validate shipping address

Request body:
```json
{
  "name": "John Doe",
  "street1": "123 Main St",
  "city": "San Francisco",
  "state": "CA",
  "zip": "94102",
  "country": "US"
}
```

**POST** `/api/v1/shipping/purchase-label` - Purchase shipping label

Request body:
```json
{
  "rate_id": "rate_...",
  "label_format": "PDF"
}
```

**GET** `/api/v1/shipping/track/{carrier}/{trackingNumber}` - Track shipment

---

### Marketing Endpoints

**POST** `/api/v1/sms-signup` - Submit SMS signup

Request body:
```json
{
  "countryCode": "+1",
  "phone": "4155550100",
  "email": "customer@example.com",
  "source": "homepage-banner"
}
```

---

### Analytics Endpoints

**POST** `/api/v1/track` - Track analytics events

The analytics tracking endpoint accepts three types of events: pageviews, custom events, and heartbeats. All requests return `204 No Content` for minimal overhead.

**Pageview Tracking:**

Request body:
```json
{
  "s": "session-uuid",
  "t": "p",
  "p": "/products/example",
  "r": "https://google.com",
  "sw": 1920,
  "sh": 1080,
  "dt": "desktop"
}
```

**Custom Event Tracking:**

Request body:
```json
{
  "s": "session-uuid",
  "t": "e",
  "p": "/products/example",
  "e": "add_to_cart",
  "d": {
    "product_id": "123",
    "product_name": "Example Product",
    "price": 29.99,
    "quantity": 1
  },
  "dt": "mobile"
}
```

**Heartbeat (Session Extension):**

Request body:
```json
{
  "s": "session-uuid",
  "t": "h",
  "p": "/products/example"
}
```

**Request Parameters:**
- `s` - Session ID (UUID stored in localStorage)
- `t` - Event type: `p` (pageview), `e` (event), `h` (heartbeat)
- `p` - Current page path
- `r` - Referrer URL (pageviews only)
- `e` - Event name (custom events only)
- `d` - Event data object (custom events only)
- `sw` - Screen width in pixels
- `sh` - Screen height in pixels
- `dt` - Device type: `mobile`, `tablet`, `desktop`

**Response:** `204 No Content` (always, even on errors)

**E-Commerce Events:**

Built-in event tracking for e-commerce conversions:
- `add_to_cart` - Product added to cart
- `checkout_started` - Customer initiated checkout
- `purchase` - Order completed and paid

---

### Configuration

**GET** `/api/v1/config` - Get website configuration (Stripe publishable key, etc.)

## Database Schema

Stencil2 automatically creates all necessary tables on first startup. Here's the complete schema:

### Content Tables

```sql
-- Articles/Posts
CREATE TABLE articles_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) UNIQUE,    -- slug
    title VARCHAR(255),
    type VARCHAR(50),            -- article, gallery, page
    published_date DATETIME,
    modified DATETIME,
    updated DATETIME,
    content TEXT,
    deck TEXT,                   -- summary/excerpt
    coverline VARCHAR(255),
    status VARCHAR(50),          -- published, draft, archived
    thumbnail_id INT,
    url VARCHAR(255),
    canonical_url VARCHAR(255),
    keywords TEXT,
    featured TINYINT DEFAULT 0,
    INDEX idx_status (status),
    INDEX idx_published_date (published_date)
);

-- Categories
CREATE TABLE categories_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    image_id INT,
    count INT DEFAULT 0
);

-- Authors
CREATE TABLE authors_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    bio TEXT,
    image_id INT
);

-- Tags
CREATE TABLE tags_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE
);

-- Images
CREATE TABLE images_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    url VARCHAR(500),
    alt_text VARCHAR(255),
    credit VARCHAR(255)
);

-- Gallery Slides
CREATE TABLE article_slides (
    id INT PRIMARY KEY AUTO_INCREMENT,
    post_id INT,
    slide_position INT,
    title VARCHAR(255),
    pre_image_desc TEXT,
    description TEXT,
    image_id INT,
    INDEX idx_post_id (post_id)
);

-- Relationship Tables
CREATE TABLE article_authors (
    post_id INT,
    author_id INT,
    PRIMARY KEY (post_id, author_id)
);

CREATE TABLE article_categories (
    post_id INT,
    category_id INT,
    PRIMARY KEY (post_id, category_id)
);

CREATE TABLE article_tags (
    post_id INT,
    tag_id INT,
    PRIMARY KEY (post_id, tag_id)
);

-- Sitemap Management
CREATE TABLE article_sitemaps (
    sitemap_date DATE PRIMARY KEY,
    complete TINYINT DEFAULT 0,
    completed_time DATETIME
);
```

### E-commerce Tables

```sql
-- Customers
CREATE TABLE customers (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    stripe_customer_id VARCHAR(255) UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    phone VARCHAR(50),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_stripe_customer_id (stripe_customer_id)
);

-- Products
CREATE TABLE products_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    price DECIMAL(10, 2),
    compare_at_price DECIMAL(10, 2),
    sku VARCHAR(255),
    inventory_quantity INT DEFAULT 0,
    inventory_policy VARCHAR(50),  -- deny, continue
    status VARCHAR(50),             -- active, draft, archived
    featured TINYINT DEFAULT 0,
    released_date DATETIME,
    sort_order INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_slug (slug),
    INDEX idx_status (status),
    INDEX idx_sort_order (sort_order)
);

-- Product Variants
CREATE TABLE product_variants (
    id INT PRIMARY KEY AUTO_INCREMENT,
    product_id INT,
    title VARCHAR(255),
    price DECIMAL(10, 2),
    sku VARCHAR(255),
    inventory_quantity INT DEFAULT 0,
    option1 VARCHAR(255),
    option2 VARCHAR(255),
    option3 VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_product_id (product_id),
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE
);

-- Product Images
CREATE TABLE product_images_data (
    id INT PRIMARY KEY AUTO_INCREMENT,
    product_id INT,
    url VARCHAR(500),
    alt_text VARCHAR(255),
    position INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_product_id (product_id),
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE
);

-- Collections
CREATE TABLE collections_unified (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    image_url VARCHAR(500),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Product-Collection Relationship
CREATE TABLE product_collections (
    product_id INT,
    collection_id INT,
    PRIMARY KEY (product_id, collection_id),
    FOREIGN KEY (product_id) REFERENCES products_unified(id) ON DELETE CASCADE,
    FOREIGN KEY (collection_id) REFERENCES collections_unified(id) ON DELETE CASCADE
);

-- Shopping Carts
CREATE TABLE carts (
    id VARCHAR(255) PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    INDEX idx_expires_at (expires_at)
);

-- Cart Items
CREATE TABLE cart_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    cart_id VARCHAR(255),
    product_id INT,
    variant_id INT,
    quantity INT DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_cart_id (cart_id),
    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE
);

-- Orders
CREATE TABLE orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_number VARCHAR(50) UNIQUE,
    customer_id INT,
    customer_email VARCHAR(255),
    customer_name VARCHAR(255),
    stripe_payment_intent_id VARCHAR(255),
    subtotal DECIMAL(10, 2),
    tax DECIMAL(10, 2),
    shipping DECIMAL(10, 2),
    total DECIMAL(10, 2),
    status VARCHAR(50),              -- pending, processing, fulfilled, cancelled
    payment_status VARCHAR(50),      -- pending, paid, failed
    fulfillment_status VARCHAR(50),  -- unfulfilled, fulfilled, partial
    shipping_name VARCHAR(255),
    shipping_address_line1 VARCHAR(255),
    shipping_address_line2 VARCHAR(255),
    shipping_city VARCHAR(255),
    shipping_state VARCHAR(50),
    shipping_postal_code VARCHAR(50),
    shipping_country VARCHAR(50),
    shipping_phone VARCHAR(50),
    billing_name VARCHAR(255),
    billing_address_line1 VARCHAR(255),
    billing_address_line2 VARCHAR(255),
    billing_city VARCHAR(255),
    billing_state VARCHAR(50),
    billing_postal_code VARCHAR(50),
    billing_country VARCHAR(50),
    tracking_number VARCHAR(255),
    tracking_carrier VARCHAR(255),
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_order_number (order_number),
    INDEX idx_customer_id (customer_id),
    INDEX idx_customer_email (customer_email),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL
);

-- Order Items
CREATE TABLE order_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id INT,
    product_id INT,
    variant_id INT,
    product_name VARCHAR(255),
    variant_title VARCHAR(255),
    sku VARCHAR(255),
    quantity INT,
    price DECIMAL(10, 2),
    INDEX idx_order_id (order_id),
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);
```

### Marketing Tables

```sql
-- SMS Signups
CREATE TABLE sms_signups (
    id INT PRIMARY KEY AUTO_INCREMENT,
    country_code VARCHAR(10) DEFAULT '+1',
    phone VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) DEFAULT NULL,
    source VARCHAR(100) DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_created_at (created_at),
    INDEX idx_country_code (country_code)
);
```

### Analytics Tables

```sql
-- Page Views
CREATE TABLE analytics_pageviews (
    id INT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(100),
    path VARCHAR(500),
    referrer VARCHAR(500),
    user_agent TEXT,
    ip_address VARCHAR(45),
    screen_width INT,
    screen_height INT,
    device_type VARCHAR(20),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session (session_id),
    INDEX idx_path (path),
    INDEX idx_created (created_at)
);

-- Custom Events
CREATE TABLE analytics_events (
    id INT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(100),
    event_name VARCHAR(100),
    event_data JSON,
    path VARCHAR(500),
    device_type VARCHAR(20),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session (session_id),
    INDEX idx_event (event_name),
    INDEX idx_created (created_at)
);
```

### Admin Tables

The admin uses its own database (`stencil_admin` by default):

```sql
-- Admin Website Registry
CREATE TABLE admin_websites (
    id INT PRIMARY KEY AUTO_INCREMENT,
    site_name VARCHAR(255) UNIQUE,
    database_name VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Admin Activity Log
CREATE TABLE admin_activity_log (
    id INT PRIMARY KEY AUTO_INCREMENT,
    website_id INT,
    action VARCHAR(255),
    entity_type VARCHAR(100),
    entity_id INT,
    details TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_website_id (website_id),
    INDEX idx_created_at (created_at)
);
```

## Deployment

### Production Build

```bash
# Build for Linux
env GOOS=linux GOARCH=amd64 go build -o stencil2 main.go

# Copy to server
scp stencil2 user@server:/www/stencil2/
scp -r websites user@server:/www/stencil2/
```

### Systemd Service

Copy `setup/stencil2.service` to `/etc/systemd/system/`:

```bash
sudo cp setup/stencil2.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable stencil2
sudo systemctl start stencil2
```

Check status:
```bash
sudo systemctl status stencil2
sudo journalctl -u stencil2 -f
```

### Nginx Configuration

Use `setup/stencil2.conf` as a reference for your Nginx configuration:

```nginx
upstream stencil2 {
    server 127.0.0.1:80;
    keepalive 64;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://stencil2;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Development

### File Watching

In development mode, Stencil2 automatically watches for changes:

- `.css` files - Automatically recompiled and minified
- `.js` files - Automatically recompiled and minified
- `.json` template configs - Automatically reloaded

No server restart required!

### Error Debugging

Development mode shows detailed error pages by default:

```bash
./stencil2 serve                    # Shows detailed errors
./stencil2 serve --hide-errors      # Uses custom error template
./stencil2 --prod-mode serve        # Production (always uses custom error template)
```

### Preview Mode

Access draft content with the `preview=true` query parameter:

```
http://example.com/article-slug?preview=true
```

This queries the `history_articles_unified` and `preview_article_information` tables.

### Asset Cache Busting

The `{{ hash }}` function generates an MD5 hash of your `/public/` directory:

```html
<link rel="stylesheet" href="/public/style.css?v={{ hash }}">
```

When files change, the hash updates automatically, busting browser caches.

## Recent Updates & Bug Fixes

### Admin CMS Improvements (December 2024)

#### Product Date Field Persistence
Fixed an issue where the `released_date` field on products was not being saved or loaded correctly:
- Added `released_date` column to SELECT, INSERT, and UPDATE queries (`admin/queries.go`)
- Implemented proper NULL handling using `sql.NullTime` for nullable datetime fields
- Products can now have optional release dates that persist correctly

#### Article Date Field Persistence
Fixed an issue where the `published_date` field was not being parsed from the admin form:
- Added form parsing for `published_date` in both create and update handlers (`admin/handlers.go`)
- Form field uses `datetime-local` input type with format `2006-01-02T15:04`
- Published dates now persist correctly when manually set in the admin

#### Product Collections Association
Fixed checkbox logic in the product form that prevented collections from being properly displayed:
- Corrected template comparison logic in `admin/templates/product_form_content.html`
- Changed from incorrect `{{if eq .ID $.ID}}` to correct `{{if eq .ID $collection.ID}}`
- Collections now correctly show as checked when editing products

These fixes ensure that all metadata fields in the admin CMS persist correctly across saves and page reloads.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For issues, questions, or contributions, please visit:
https://github.com/murdinc/stencil2
