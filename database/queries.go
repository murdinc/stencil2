package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/murdinc/stencil2/structs"
)

// GetCategories
func (db *DBConnection) GetCategories(params map[string]string) ([]structs.Category, error) {

	// check if this is a full category request (includes image)
	fullCategory := `
			,
			'' as image_url,
			'' as alt_text
		`
	fullCategoryJoin := ``
	if value, exists := params["full"]; exists && value == "true" {
		fullCategory = `
			,
			ifnull(B.url, '') as image_url,
			ifnull(B.alt_text, '') as alt_text
		`
		fullCategoryJoin = `left join images_unified B on A.image_id = B.id`
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			A.id,
			A.name,
			A.slug,
			ifnull(A.description, '') as description,
			A.count
			%s
		FROM categories_unified A
		%s
		WHERE A.count > 0 ORDER BY
		A.name ASC
	`, fullCategory, fullCategoryJoin)

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []structs.Category
	for rows.Next() {
		var category structs.Category
		if err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &category.Description, &category.Count,
			&category.ImageUrl, &category.AltText,
		); err != nil {
			return nil, err
		}

		categories = append(categories, category)
	}

	return categories, nil
}

// GetSingularPost retrieves a singular post from the database
func (db *DBConnection) GetSingularPost(vars map[string]string, params map[string]string) (structs.Post, error) {

	sqlQuery := `
		SELECT
			A.id AS id,
			A.slug AS slug,
			A.title AS title,
			A.type AS type,
			A.published_date AS published_date,
			A.updated_at AS modified,
			A.updated_at AS updated,
			A.content AS content,
			A.description AS description,
			A.deck AS deck,
			A.coverline AS coverline,
			A.status AS status,
			A.thumbnail_id AS thumbnail_id,
			A.slug AS url,
			A.canonical_url AS canonical_url,
			A.keywords AS keywords,
			B.authors AS authors,
			B.categories AS categories,
			B.tags AS tags,
			I.id AS image_id,
			I.url AS image_url,
			I.alt_text AS image_alt,
			I.credit AS image_credit,
			0 as duplication_id
		FROM articles_unified A
			LEFT JOIN article_information B ON B.post_id = A.id
			LEFT JOIN images_unified I ON I.id = A.thumbnail_id
		WHERE A.status = 'published' AND A.slug = ?
		ORDER BY A.published_date DESC
		LIMIT 1;
	`

	if params["preview"] == "true" {
		sqlQuery = `
			SELECT
				A.id AS id,
				A.slug AS slug,
				A.title AS title,
				A.type AS type,
				A.published_date AS published_date,
				A.updated_at AS modified,
				A.updated_at AS updated,
				A.content AS content,
				A.description AS description,
				A.deck AS deck,
				A.coverline AS coverline,
				A.status AS status,
				A.thumbnail_id AS thumbnail_id,
				A.slug AS url,
				A.canonical_url AS canonical_url,
				A.keywords AS keywords,
				B.authors AS authors,
				B.categories AS categories,
				B.tags AS tags,
				I.id AS image_id,
				I.url AS image_url,
				I.alt_text AS image_alt,
				I.credit AS image_credit,
				0 as duplication_id
			FROM history_articles_unified A
				JOIN preview_article_information B ON B.post_id = A.id
				LEFT JOIN images_unified I ON I.id = A.thumbnail_id
			WHERE A.slug = ? AND A.status = 'preview_draft'
			ORDER BY A.date_changed DESC
			LIMIT 1;`
	}

	var post structs.Post
	var authorsJSON, categoriesJSON, tagsJSON sql.NullString
	var publishedDate sql.NullTime
	var description, deck, coverline, canonicalURL, keywords sql.NullString
	var thumbnailID sql.NullInt64
	var imageID sql.NullInt64
	var imageURL, imageAlt, imageCredit sql.NullString

	err := db.QueryRow(sqlQuery, vars["slug"]).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Type, &publishedDate,
		&post.Modified, &post.Updated, &post.Content, &description, &deck, &coverline, &post.Status,
		&thumbnailID, &post.URL, &canonicalURL, &keywords, &authorsJSON,
		&categoriesJSON, &tagsJSON, &imageID, &imageURL, &imageAlt, &imageCredit, &post.DuplicationID,
	)

	if err != nil {
		return structs.Post{}, err
	}

	if publishedDate.Valid {
		post.PublishedDate = publishedDate.Time
	}
	if description.Valid {
		post.Description = description.String
	}
	if deck.Valid {
		post.Deck = deck.String
	}
	if coverline.Valid {
		post.Coverline = coverline.String
	}
	if canonicalURL.Valid {
		post.CanonicalURL = canonicalURL.String
	}
	if keywords.Valid {
		post.Keywords = keywords.String
	}
	if thumbnailID.Valid {
		post.ThumbnailID = int(thumbnailID.Int64)
	}

	// Build image from joined data
	if imageID.Valid {
		post.Image.ID = int(imageID.Int64)
	}
	if imageURL.Valid {
		post.Image.URL = imageURL.String
	}
	if imageAlt.Valid {
		post.Image.AltText = imageAlt.String
	}
	if imageCredit.Valid {
		post.Image.Credit = imageCredit.String
	}

	if authorsJSON.Valid {
		json.Unmarshal([]byte(authorsJSON.String), &post.Authors)
	}
	if categoriesJSON.Valid {
		json.Unmarshal([]byte(categoriesJSON.String), &post.Categories)
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &post.Tags)
	}

	// Slides removed - not needed
	post.Slides = []structs.Slide{}

	post.ParseContent(&structs.ParserOptions{})

	return post, nil
}

// GetMultiplePosts retrieves multiple posts from the database
func (db *DBConnection) GetMultiplePosts(vars map[string]string, params map[string]string) ([]structs.Post, error) {

	// check if this is a full feed request
	fullFeed := `'' AS content,`
	if value, exists := params["full"]; exists && value == "true" {
		fullFeed = `A.content AS content,`
	}

	orderby := `A.published_date DESC`
	if value, exists := params["sort"]; exists && value == "modified" {
		orderby = `A.updated_at DESC`
	}

	offset, count := defaultOffsetCount(vars)

	queryArgs := []interface{}{}

	queryJoin := ``
	queryWhereAnd := ``

	if vars["slug"] != "" {
		switch vars["taxonomy"] {
		case "tag":
			queryArgs = append(queryArgs, vars["slug"])
			queryJoin = `
				JOIN article_tags C ON C.post_id = A.id
				JOIN tags_unified D ON D.id = C.tag_id
				`
			queryWhereAnd = `AND D.slug = ?`

		case "category":
			queryArgs = append(queryArgs, vars["slug"])
			queryJoin = `
				JOIN article_categories C ON C.post_id = A.id
				JOIN categories_unified D ON D.id = C.category_id
				`
			queryWhereAnd = `AND D.slug = ?`

		case "author":
			queryArgs = append(queryArgs, vars["slug"])
			queryJoin = `
				JOIN article_authors C ON C.post_id = A.id
				JOIN authors_unified D ON D.id = C.author_id
				`
			queryWhereAnd = `AND D.slug = ?`

		case "type":
			queryArgs = append(queryArgs, vars["slug"])

			queryWhereAnd = `AND A.type = ?`
		}
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			A.id AS id,
			A.slug AS slug,
			A.title AS title,
			A.type AS type,
			A.published_date AS published_date,
			A.updated_at AS modified,
			A.updated_at AS updated,
			%s
			A.deck AS deck,
			A.coverline AS coverline,
			A.thumbnail_id AS thumbnail_id,
			A.slug AS url,
			A.canonical_url AS canonical_url,
			A.keywords AS keywords,
			B.authors AS authors,
			B.categories AS categories,
			B.tags AS tags,
			I.id AS image_id,
			I.url AS image_url,
			I.alt_text AS image_alt,
			I.credit AS image_credit,
			0 as duplication_id
		FROM articles_unified A
		LEFT JOIN article_information B ON B.post_id = A.id
		LEFT JOIN images_unified I ON I.id = A.thumbnail_id
		%s
		WHERE A.status = 'published' AND A.type NOT IN ('page')	%s
		ORDER BY %s
		LIMIT %d, %d;
	`, fullFeed, queryJoin, queryWhereAnd, orderby, offset, count)

	rows, err := db.QueryRows(sqlQuery, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []structs.Post

	for rows.Next() {
		var post structs.Post
		var authorsJSON, categoriesJSON, tagsJSON sql.NullString
		var publishedDate sql.NullTime
		var deck, coverline, canonicalURL, keywords sql.NullString
		var thumbnailID sql.NullInt64
		var imageID sql.NullInt64
		var imageURL, imageAlt, imageCredit sql.NullString
		if err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Type, &publishedDate,
			&post.Modified, &post.Updated, &post.Content, &deck, &coverline, &thumbnailID,
			&post.URL, &canonicalURL, &keywords, &authorsJSON, &categoriesJSON,
			&tagsJSON, &imageID, &imageURL, &imageAlt, &imageCredit, &post.DuplicationID,
		); err != nil {
			return nil, err
		}

		if publishedDate.Valid {
			post.PublishedDate = publishedDate.Time
		}
		if deck.Valid {
			post.Deck = deck.String
		}
		if coverline.Valid {
			post.Coverline = coverline.String
		}
		if canonicalURL.Valid {
			post.CanonicalURL = canonicalURL.String
		}
		if keywords.Valid {
			post.Keywords = keywords.String
		}
		if thumbnailID.Valid {
			post.ThumbnailID = int(thumbnailID.Int64)
		}

		// Build image from joined data
		if imageID.Valid {
			post.Image.ID = int(imageID.Int64)
		}
		if imageURL.Valid {
			post.Image.URL = imageURL.String
		}
		if imageAlt.Valid {
			post.Image.AltText = imageAlt.String
		}
		if imageCredit.Valid {
			post.Image.Credit = imageCredit.String
		}

		if authorsJSON.Valid {
			json.Unmarshal([]byte(authorsJSON.String), &post.Authors)
		}
		if categoriesJSON.Valid {
			json.Unmarshal([]byte(categoriesJSON.String), &post.Categories)
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &post.Tags)
		}

		// rearrange if requesting by taxonomy
		if vars["slug"] != "" {
			switch vars["taxonomy"] {
			case "tag":
				sort.Slice(post.Tags, func(i, j int) bool {
					return post.Tags[i].Slug == vars["slug"]
				})
			case "category":
				sort.Slice(post.Categories, func(i, j int) bool {
					return post.Categories[i].Slug == vars["slug"]
				})
			case "author":
				sort.Slice(post.Authors, func(i, j int) bool {
					return post.Authors[i].Slug == vars["slug"]
				})
			}

		}

		post.ParseContent(&structs.ParserOptions{})
		posts = append(posts, post)
	}

	return posts, nil
}

func (db *DBConnection) InitSitemaps() error {
	query := `
		INSERT INTO article_sitemaps (sitemap_date, complete, completed_time)
		SELECT DISTINCT DATE_SUB(published_date, INTERVAL DAY(published_date) - 1 DAY), 0, NULL
		FROM articles_unified
		ON DUPLICATE KEY UPDATE complete = 0, completed_time = NULL;
	`

	_, err := db.ExecuteQuery(query)
	return err
}

func (db *DBConnection) GetIncompleteSitemaps() ([]time.Time, error) {
	sqlQuery := `SELECT sitemap_date FROM article_sitemaps WHERE complete = 0;`

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incompleteDates []time.Time
	for rows.Next() {
		var incompleteDate time.Time
		if err := rows.Scan(&incompleteDate); err != nil {
			return nil, err
		}
		incompleteDates = append(incompleteDates, incompleteDate)
	}

	return incompleteDates, nil
}

func (db *DBConnection) MarkSitemapsAsComplete() error {
	query := `UPDATE article_sitemaps SET complete = 1 WHERE complete = 0;`
	_, err := db.ExecuteQuery(query)
	return err
}

func (db *DBConnection) GetPublishedPostsByMonth(month time.Time) ([]structs.Post, error) {
	sqlQuery := `
		SELECT
			title,
			slug,
			published_date,
			modified,
			updated,
			type,
			url
		FROM
			articles_unified
		WHERE
			published_date BETWEEN ? AND ? + INTERVAL 1 MONTH
			AND status = 'published'
		ORDER BY
			published_date ASC;
	`

	rows, err := db.QueryRows(sqlQuery, month, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []structs.Post
	for rows.Next() {
		var post structs.Post
		if err := rows.Scan(
			&post.Title,
			&post.Slug,
			&post.PublishedDate,
			&post.Modified,
			&post.Updated,
			&post.Type,
			&post.URL,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (db *DBConnection) GetAllPublishedProducts() ([]structs.Product, error) {
	sqlQuery := `
		SELECT
			slug,
			updated_at
		FROM
			products_unified
		WHERE
			status = 'published'
		ORDER BY
			updated_at DESC;
	`

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []structs.Product
	for rows.Next() {
		var product structs.Product
		if err := rows.Scan(
			&product.Slug,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (db *DBConnection) GetAllPublishedCollections() ([]structs.Collection, error) {
	sqlQuery := `
		SELECT
			slug,
			updated_at
		FROM
			collections_unified
		WHERE
			status = 'published'
		ORDER BY
			updated_at DESC;
	`

	rows, err := db.QueryRows(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []structs.Collection
	for rows.Next() {
		var collection structs.Collection
		if err := rows.Scan(
			&collection.Slug,
			&collection.UpdatedAt,
		); err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

func defaultOffsetCount(vars map[string]string) (int, int) {

	// Convert offset to an integer, default to 0 if not a number
	offset, err := strconv.Atoi(vars["offset"])
	if err != nil || offset < 0 {
		offset = 0
	}

	// Convert count to an integer, default to 30 if not a number
	count, err := strconv.Atoi(vars["count"])
	if err != nil || count <= 0 {
		count = 30
	}
	// Enforce maximum to prevent memory exhaustion
	if count > 100 {
		count = 100
	}

	if offset < 0 {
		offset = 0
	}

	// Convert page to an integer, default to 1 if not a number
	page, err := strconv.Atoi(vars["page"])
	if err != nil || page < 1 {
		page = 1
	}

	// use page if offset wasn't specified
	if offset == 0 && page > 1 {
		offset = page * count
	}

	return offset, count
}

// GetCategoryBySlug returns a single category by its slug
func (db *DBConnection) GetCategoryBySlug(slug string) (structs.Category, error) {
	sqlQuery := `
		SELECT
			A.id,
			A.name,
			A.slug,
			ifnull(A.description, '') as description,
			A.count
		FROM categories_unified A
		WHERE A.slug = ?
		LIMIT 1
	`

	var category structs.Category
	err := db.QueryRow(sqlQuery, slug).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Description, &category.Count,
	)
	if err != nil {
		return structs.Category{}, err
	}

	return category, nil
}
