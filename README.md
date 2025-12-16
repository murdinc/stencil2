# Stencil2

A high-performance, multi-site template engine and content management platform written in Go. Stencil2 enables you to serve multiple independent websites from a single server instance, each with its own configuration, templates, and database.

## Features

- **Multi-Site Hosting**: Host multiple independent websites with separate configurations, templates, and databases
- **Powerful Template Engine**: Go templates with custom functions and Sprig library integration
- **Asset Pipeline**: Automatic CSS/JS minification and combination with cache busting
- **REST API**: JSON API (v1) for programmatic content access
- **Dynamic Routing**: Template-based route generation with pagination support
- **Media Proxy**: On-the-fly image resizing with WebP support
- **Sitemap Generation**: Automatic XML sitemap generation from database content
- **Development Tools**: File watcher for hot-reload, in-memory database, and error debugging
- **Production Ready**: Includes systemd service and Nginx configuration examples

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
- [API Endpoints](#api-endpoints)
- [Database Schema](#database-schema)
- [Deployment](#deployment)
- [Development](#development)

## Installation

### Prerequisites

- Go 1.20 or higher
- MySQL 5.5+ or MariaDB 10.1+ (or use the built-in local database for development)

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
# Development mode with local database
./stencil2 localdb

# In another terminal, start the web server
./stencil2 serve

# Or in production mode
./stencil2 --prod-mode serve
```

### 4. Access Your Site

Add to `/etc/hosts`:
```
127.0.0.1 example.com
```

Visit `http://example.com:8080`

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
  "siteName": "example.com",           // Domain name
  "apiVersion": 1,                     // API version (currently only v1)
  "database": {
    "name": "example_db"               // Site-specific database
  },
  "mediaProxyUrl": "https://media.example.com",  // Optional media proxy URL
  "http": {
    "address": "example.com"           // Host header for routing
  }
}
```

### Template Configuration

Located at `websites/{site}/templates/{template-name}/{template-name}.json`:

```json
{
  "name": "homepage",                  // Template identifier
  "path": "/",                         // URL path
  "paginateType": 0,                   // 0=none, 1=paginate, 2=302-redirect
  "requires": ["common"],              // Required template dependencies
  "jsFile": "main.js",                 // JavaScript file to load
  "cssFile": "main.css",               // CSS file to load
  "apiEndpoint": "/api/v1/posts",      // API endpoint for data
  "apiTaxonomy": "category",           // Taxonomy type (category/tag/author)
  "apiSlug": "technology",             // Taxonomy slug filter
  "apiCount": 10,                      // Number of items to fetch
  "apiOffset": 0,                      // Offset for pagination
  "noCache": false,                    // Disable caching
  "cacheTime": 300,                    // Cache TTL in seconds
  "mimeType": "text/html"              // Response content type
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

### localdb

Start an in-memory MySQL database for local development.

```bash
./stencil2 localdb
```

The local database will:
- Create an in-memory MySQL engine
- Load all `.sql` files from `websites/{site}/data/` directories
- Stay running until stopped (Ctrl+C)

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
│   ├── localdb.go                # Local database command
│   └── sitemaps.go               # Sitemap generation command
├── configs/                      # Configuration loaders
│   ├── env.go                    # Environment config loader
│   ├── website.go                # Website config loader
│   └── template.go               # Template config loader
├── database/                     # Database layer
│   ├── connection.go             # Connection management
│   ├── queries.go                # Query methods
│   └── localdb.go                # In-memory database setup
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
│       ├── sitemaps/             # Generated sitemaps (served at /sitemaps/)
│       └── data/                 # SQL dump files for localdb
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

## API Endpoints

Stencil2 provides a RESTful JSON API (v1) for all configured websites.

### Categories

**GET** `/api/v1/categories`

Query Parameters:
- `full=true` - Include category images

Response:
```json
[
  {
    "id": 1,
    "name": "Technology",
    "slug": "technology",
    "description": "Latest tech news",
    "image_url": "https://example.com/tech.jpg",
    "alt_text": "Technology"
  }
]
```

### Posts List

**GET** `/api/v1/posts`
**GET** `/api/v1/posts/{count}`
**GET** `/api/v1/posts/{count}/{offset}`

Query Parameters:
- `full=true` - Include post content and slides
- `featured=false` - Exclude featured posts
- `sort=modified` - Sort by modified date instead of published date

Response:
```json
[
  {
    "id": 123,
    "slug": "example-article",
    "title": "Example Article",
    "type": "article",
    "published_date": "2025-01-15T10:00:00Z",
    "deck": "Article summary",
    "url": "/example-article",
    "image": {
      "id": 456,
      "url": "https://example.com/image.jpg",
      "alt_text": "Example"
    },
    "authors": [...],
    "categories": [...],
    "tags": [...]
  }
]
```

### Single Post

**GET** `/api/v1/post/{slug}`

Query Parameters:
- `preview=true` - Get draft/preview version of post

Response: Single post object with full content and slides

### Taxonomy Posts

**GET** `/api/v1/{taxonomy}/{slug}/posts`
**GET** `/api/v1/{taxonomy}/{slug}/posts/{count}/{offset}`

Taxonomy types: `category`, `tag`, `author`, `type`

Example:
- `/api/v1/category/technology/posts/10`
- `/api/v1/author/john-doe/posts`
- `/api/v1/tag/ai/posts/20/40`

## Database Schema

Stencil2 expects the following core tables (minimal schema):

```sql
-- Articles/Posts
CREATE TABLE articles_unified (
    id INT PRIMARY KEY,
    name VARCHAR(255),           -- slug
    title VARCHAR(255),
    type VARCHAR(50),            -- article, gallery, page
    published_date DATETIME,
    modified DATETIME,
    updated DATETIME,
    content TEXT,
    deck TEXT,                   -- summary/excerpt
    coverline VARCHAR(255),
    status VARCHAR(50),          -- published, draft
    thumbnail_id INT,
    url VARCHAR(255),
    canonical_url VARCHAR(255),
    keywords TEXT,
    featured TINYINT DEFAULT 0
);

-- Categories
CREATE TABLE categories_unified (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    slug VARCHAR(255),
    description TEXT,
    image_id INT,
    count INT DEFAULT 0
);

-- Authors
CREATE TABLE authors_unified (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    slug VARCHAR(255),
    bio TEXT,
    image_id INT
);

-- Tags
CREATE TABLE tags_unified (
    id INT PRIMARY KEY,
    name VARCHAR(255),
    slug VARCHAR(255)
);

-- Images
CREATE TABLE images_unified (
    id INT PRIMARY KEY,
    url VARCHAR(500),
    alt_text VARCHAR(255),
    credit VARCHAR(255)
);

-- Gallery Slides
CREATE TABLE article_slides (
    id INT PRIMARY KEY,
    post_id INT,
    slide_position INT,
    title VARCHAR(255),
    pre_image_desc TEXT,
    description TEXT,
    image_id INT
);

-- Relationship Tables
CREATE TABLE article_authors (
    post_id INT,
    author_id INT
);

CREATE TABLE article_categories (
    post_id INT,
    category_id INT
);

CREATE TABLE article_tags (
    post_id INT,
    tag_id INT
);

-- Sitemap Management
CREATE TABLE article_sitemaps (
    sitemap_date DATE PRIMARY KEY,
    complete TINYINT DEFAULT 0,
    completed_time DATETIME
);
```

For a complete schema, place `.sql` dump files in `websites/{site}/data/` for use with `localdb`.

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

### Local Database

Use the in-memory database for development:

```bash
# Terminal 1: Start local database
./stencil2 localdb

# Terminal 2: Start server
./stencil2 serve

# Terminal 3: Load sample data
mysql -h 127.0.0.1 -P 3306 -u root your_database < sample_data.sql
```

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
