package database

import (
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
			ifnull(A.description, '') as description
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
			&category.ID, &category.Name, &category.Slug, &category.Description,
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
			A.name AS slug,
			A.title AS title,
			A.type AS type,
			A.published_date AS published_date,
			A.modified AS modified,
			A.updated AS updated,
			A.content AS content,
			A.deck AS deck,
			A.coverline AS coverline,
			A.status AS status,
			A.thumbnail_id AS thumbnail_id,
			A.url AS url,
			A.canonical_url AS canonical_url,
			A.keywords AS keywords,
			B.authors AS authors,
			B.categories AS categories,
			B.tags AS tags,
			B.image AS image,
			ifnull(C.article_slide_id, 0) as duplication_id
		FROM articles_unified A
			JOIN article_information B ON B.post_id = A.id
			LEFT JOIN article_duplicates_slides C ON C.duplication_id = A.id
		WHERE A.status = 'published' AND A.url = concat('/', ?)
		ORDER BY A.published_date DESC
		LIMIT 1;
	`

	if params["preview"] == "true" {
		sqlQuery = `
			SELECT
				A.id AS id,
				A.name AS slug,
				A.title AS title,
				A.type AS type,
				A.published_date AS published_date,
				A.modified AS modified,
				A.updated AS updated,
				A.content AS content,
				A.deck AS deck,
				A.coverline AS coverline,
				A.status AS status,
				A.thumbnail_id AS thumbnail_id,
				A.url AS url,
				A.canonical_url AS canonical_url,
				A.keywords AS keywords,
				B.authors AS authors,
				B.categories AS categories,
				B.tags AS tags,
				B.image AS image,
				ifnull(C.article_slide_id, 0) as duplication_id
			FROM history_articles_unified A
				JOIN preview_article_information B ON B.post_id = A.id
				LEFT JOIN article_duplicates_slides C ON C.duplication_id = A.id
			WHERE A.url = concat('/', ?) AND A.status = 'preview_draft'
			ORDER BY A.date_changed DESC
			LIMIT 1;`
	}

	var post structs.Post
	var authorsJSON, categoriesJSON, tagsJSON, imageJSON string

	err := db.QueryRow(sqlQuery, vars["slug"]).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Type, &post.PublishedDate,
		&post.Modified, &post.Updated, &post.Content, &post.Deck, &post.Coverline, &post.Status,
		&post.ThumbnailID, &post.URL, &post.CanonicalURL, &post.Keywords, &authorsJSON,
		&categoriesJSON, &tagsJSON, &imageJSON, &post.DuplicationID,
	)

	if err != nil {
		return structs.Post{}, err
	}

	json.Unmarshal([]byte(authorsJSON), &post.Authors)
	json.Unmarshal([]byte(categoriesJSON), &post.Categories)
	json.Unmarshal([]byte(tagsJSON), &post.Tags)
	json.Unmarshal([]byte(imageJSON), &post.Image)

	if post.Type == "gallery" || (post.Type == "article" && post.Content == "") {
		slidesQuery := `
			SELECT
				A.slide_position,
				A.title,
				ifnull(A.pre_image_desc, '') as pre_image_desc,
				ifnull(A.description, '') as description,
				ifnull(B.id, 0) as id,
				ifnull(B.url, '') as url,
				ifnull(B.alt_text, '') as alt_text,
				ifnull(B.credit, '') as credit,
				ifnull(C.duplication_id, 0) as duplication_found
			FROM article_slides A
				LEFT JOIN images_unified B ON A.image_id = B.id
				LEFT JOIN article_duplicates_slides C on A.post_id = C.article_slide_id
			WHERE A.post_id in (?, ?)
				AND B.url IS NOT NULL AND B.url <> ''
			GROUP BY A.slide_position
			ORDER BY A.slide_position ASC;
		`

		if params["preview"] == "true" {
			slidesQuery = `
				SELECT
					A.slide_position,
					A.title,
					ifnull(A.pre_image_desc, '') as pre_image_desc,
					ifnull(A.description, '') as description,
					ifnull(B.id, 0) as id,
					ifnull(B.url, '') as url,
					ifnull(B.alt_text, '') as alt_text,
					ifnull(B.credit, '') as credit,
					ifnull(C.duplication_id, 0) as duplication_found
				FROM preview_article_slides A
					LEFT JOIN images_unified B ON A.image_id = B.id
					LEFT JOIN article_duplicates_slides C ON A.post_id = C.duplication_id
				WHERE A.post_id in (?, ?)
				ORDER BY A.slide_position ASC;
			`
		}

		rows, err := db.QueryRows(slidesQuery, post.ID, post.DuplicationID)
		if err != nil {
			return structs.Post{}, err
		}
		defer rows.Close()

		var slides []structs.Slide
		var filteredSlides []structs.Slide
		var firstGallery *structs.Slide
		var openingContent string

		for rows.Next() {
			var slide structs.Slide
			var image structs.Image

			err := rows.Scan(
				&slide.SlidePosition, &slide.Title, &slide.PreImageDesc,
				&slide.Description, &image.ID, &image.URL, &image.AltText, &image.Credit, &slide.DuplicationFound,
			)
			if err != nil {
				return post, err
			}

			slide.Image = image
			slide.Title = slide.Title

			// Gather the pre_image_desc for the first slide (in case order is changed)
			if slide.SlidePosition == 1 && slide.DuplicationFound != 0 {
				openingContent = slide.PreImageDesc
				slide.PreImageDesc = ""
			}

			// If this image is the one that SHOULD be first, save it
			if image.ID == post.Image.ID && post.Image.ID != 0 && slide.DuplicationFound != 0 && firstGallery == nil {
				firstGallery = &slide
			} else {
				slides = append(slides, slide)
			}
		}

		if firstGallery != nil {
			firstGallery.PreImageDesc = openingContent

			// Add firstGallery to filteredSlides and then add the rest of the slides
			filteredSlides = append(filteredSlides, *firstGallery)
			filteredSlides = append(filteredSlides, slides...)

			slides = filteredSlides
		}

		post.Slides = slides
	}

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

	featured := `AND A.featured = 1`
	if value, exists := params["featured"]; exists && value == "false" {
		featured = ``
	}

	orderby := `A.published_date DESC`
	if value, exists := params["sort"]; exists && value == "modified" {
		orderby = `A.modified DESC`
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
			A.name AS slug,
			A.title AS title,
			A.type AS type,
			A.published_date AS published_date,
			A.modified AS modified,
			A.updated AS updated,
			%s
			A.deck AS deck,
			A.coverline AS coverline,
			A.thumbnail_id AS thumbnail_id,
			A.url AS url,
			A.canonical_url AS canonical_url,
			A.keywords AS keywords,
			B.authors AS authors,
			B.categories AS categories,
			B.tags AS tags,
			B.image AS image,
			ifnull(C.article_slide_id, 0) as duplication_id
		FROM articles_unified A
		JOIN article_information B ON B.post_id = A.id
		LEFT JOIN article_duplicates_slides C ON C.duplication_id = A.id
		%s
		WHERE A.status = 'published' AND A.type NOT IN ('page')	%s
		%s
		ORDER BY %s
		LIMIT %d, %d;
	`, fullFeed, queryJoin, featured, queryWhereAnd, orderby, offset, count)

	rows, err := db.QueryRows(sqlQuery, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []structs.Post

	for rows.Next() {
		var post structs.Post
		var authorsJSON, categoriesJSON, tagsJSON, imageJSON string
		if err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Type, &post.PublishedDate,
			&post.Modified, &post.Updated, &post.Content, &post.Deck, &post.Coverline, &post.ThumbnailID,
			&post.URL, &post.CanonicalURL, &post.Keywords, &authorsJSON, &categoriesJSON,
			&tagsJSON, &imageJSON, &post.DuplicationID,
		); err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(authorsJSON), &post.Authors)
		json.Unmarshal([]byte(categoriesJSON), &post.Categories)
		json.Unmarshal([]byte(tagsJSON), &post.Tags)
		json.Unmarshal([]byte(imageJSON), &post.Image)

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

	// This is so that if a gallery doens't have any slides, it won't be returned in the feed
	var filteredPosts []structs.Post

	if value, exists := params["full"]; exists && value == "true" {
		for i, post := range posts {
			if post.Type == "gallery" {
				slidesQuery := `
					SELECT
						F.slide_position,
						F.title,
						ifnull(F.pre_image_desc, '') as pre_image_desc,
						ifnull(F.description, '') as description,
						ifnull(G.id, 0) as id,
						ifnull(G.url, '') as url,
						ifnull(G.alt_text, '') as alt_text,
						ifnull(G.credit, '') as credit,
						ifnull(H.duplication_id, 0) as duplication_found
					FROM article_slides F
						LEFT JOIN images_unified G ON F.image_id = G.id
						LEFT JOIN article_duplicates_slides H on F.post_id = H.article_slide_id
					WHERE F.post_id in (?, ?)
						AND G.url IS NOT NULL AND G.url <> ''
					GROUP BY F.slide_position
					ORDER BY F.slide_position ASC;
				`

				rows, _ := db.QueryRows(slidesQuery, post.ID, post.DuplicationID)
				defer rows.Close()
				if err != nil {
					return posts, err
				}

				var slides []structs.Slide
				var filteredSlides []structs.Slide
				var firstGallery *structs.Slide
				var openingContent string
				
				for rows.Next() {
					var slide structs.Slide
					var image structs.Image

					err := rows.Scan(
						&slide.SlidePosition, &slide.Title, &slide.PreImageDesc,
						&slide.Description, &image.ID, &image.URL, &image.AltText, &image.Credit, &slide.DuplicationFound,
					)
					
					if err != nil {
						return posts, err
					}

					slide.Image = image

					// Gather the pre_image_desc for the first slide (in case order is changed for duplicates)
					if slide.SlidePosition == 1 && slide.DuplicationFound != 0 {
						openingContent = slide.PreImageDesc
						slide.PreImageDesc = ""
					}

					// If this image is the one that SHOULD be first, save it
					if image.ID == post.Image.ID && post.Image.ID != 0 && slide.DuplicationFound != 0 && firstGallery == nil {
						firstGallery = &slide
					} else {
						slides = append(slides, slide)
					}
				}

				if firstGallery != nil {
					firstGallery.PreImageDesc = openingContent

					// Add firstGallery to filteredSlides and then add the rest of the slides
					filteredSlides = append(filteredSlides, *firstGallery)
					filteredSlides = append(filteredSlides, slides...)

					slides = filteredSlides
				}

				// Only add slides if there are any, otherwise remove this post from the posts array
				if len(slides) > 0 {
					posts[i].Slides = slides

					posts[i].ParseContent(&structs.ParserOptions{})
					filteredPosts = append(filteredPosts, posts[i])
				}

				// Re-set variables from this article to prepare for the next one
				firstGallery = nil
				openingContent = ""
				slides = []structs.Slide{}
				filteredSlides = []structs.Slide{}
			} else {
				posts[i].ParseContent(&structs.ParserOptions{})
				filteredPosts = append(filteredPosts, posts[i])
			}
		}
		
		posts = filteredPosts
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
			name,
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
