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
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Description string `json:"description"`
	ImageUrl string `json:"image_url"`
	AltText string `json:"alt_text"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
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
