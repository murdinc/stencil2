package frontend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/murdinc/stencil2/api"
	"github.com/murdinc/stencil2/media"
)

type NoListFile struct {
	http.File
}

type NoListFileSystem struct {
	base http.FileSystem
}

func (website *Website) NotFoundHandler(w http.ResponseWriter, r *http.Request) {

	pageData := PageData{
		ErrorString: "Page Not Found!",
		StatusCode:  404,
		ProdMode:    website.EnvironmentConfig.ProdMode,
		HideErrors:  website.EnvironmentConfig.HideErrors,
	}
	website.RenderError(w, pageData)
}

// Builds out a route for a given route name
func (website *Website) GetRoute(name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// get the template
		tpl := website.GetTemplate(name)

		// get the slug context
		ctx := r.Context()
		vars, ok := ctx.Value("vars").(map[string]string)
		if !ok {
			// TODO what to actually do here?
			http.Error(w, http.StatusText(422), 422)
			return
		}

		// override variables from template config?
		if tpl.APISlug != "" {
			vars["slug"] = tpl.APISlug
		}
		if tpl.APITaxonomy != "" {
			vars["taxonomy"] = tpl.APITaxonomy
		}
		if tpl.APICount != 0 {
			vars["count"] = strconv.Itoa(tpl.APICount)
		}
		if tpl.APIOffset != 0 {
			vars["offset"] = strconv.Itoa(tpl.APIOffset)
		}

		// Create and assign to pageDataVariable
		pageData := PageData{
			Slug:       vars["slug"],
			Page:       vars["page"],
			StatusCode: 200,
			ProdMode:   website.EnvironmentConfig.ProdMode,
			HideErrors: website.EnvironmentConfig.HideErrors,
		}

		// get internal API Handler if specified
		internalHandler, URLParams, err := website.APIHandler.API.GetInternalHandler(tpl.APIEndpoint)
		if err != nil {
			// TODO ??
			log.Println(err.Error())
		}

		// add "preview" param from main request
		URLParams["preview"] = r.URL.Query().Get("preview")

		// add "preview" as bool to pagedata
		previewBool, _ := strconv.ParseBool(URLParams["preview"])
		pageData.Preview = previewBool
		// override to nocache on preview pages
		if pageData.Preview {
			tpl.NoCache = true
		}

		// Default categories for nav, if the internalHandler isnt for categories specifically
		if website.WebsiteConfig.Database.Name != "" && internalHandler != "categories" {
			categories, err := website.DBConn.GetCategories(map[string]string{})
			if err != nil {
				// TODO?
				pageData.ErrorDescription = err.Error()
				pageData.StatusCode = 500
			}
			pageData.Categories = categories
		}

		if internalHandler != "" {
			switch internalHandler {
			case "post":
				post, err := website.DBConn.GetSingularPost(vars, URLParams)
				if err != nil {
					// TODO?
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if post.Slug == "" {
					pageData.ErrorString = "Page not found!"
					pageData.StatusCode = 404
				}
				pageData.Post = post

			case "posts":
				posts, err := website.DBConn.GetMultiplePosts(vars, URLParams)
				if err != nil {
					// TODO?
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if len(posts) == 0 {
					pageData.ErrorString = "Page not found!"
					pageData.StatusCode = 404
				}
				pageData.Posts = posts

				// If filtering by category taxonomy, also fetch the category details
				if vars["taxonomy"] == "category" && vars["slug"] != "" {
					category, err := website.DBConn.GetCategoryBySlug(vars["slug"])
					if err == nil {
						pageData.Category = category
					}
				}

			case "categories":
				categories, err := website.DBConn.GetCategories(URLParams)
				if err != nil {
					// TODO?
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if len(categories) == 0 {
					pageData.ErrorString = "Page not found!"
					pageData.StatusCode = 404
				}
				pageData.Categories = categories

			case "product":
				product, err := website.DBConn.GetProduct(vars["slug"])
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if product.Slug == "" {
					pageData.ErrorString = "Product not found!"
					pageData.StatusCode = 404
				}
				pageData.Product = product

			case "products":
				products, err := website.DBConn.GetProducts(vars, URLParams)
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if len(products) == 0 {
					pageData.ErrorString = "No products available!"
					pageData.StatusCode = 404
				}
				pageData.Products = products

			case "featured-products":
				products, err := website.DBConn.GetFeaturedProducts(vars, URLParams)
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if len(products) == 0 {
					pageData.ErrorString = "No featured products available!"
					pageData.StatusCode = 404
				}
				pageData.Products = products

			case "collection":
				collection, err := website.DBConn.GetCollection(vars["slug"])
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if collection.Slug == "" {
					pageData.ErrorString = "Collection not found!"
					pageData.StatusCode = 404
				}
				pageData.Collection = collection

				// Also get products in this collection
				products, err := website.DBConn.GetCollectionProducts(vars["slug"], vars, URLParams)
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				pageData.Products = products

			case "collections":
				collections, err := website.DBConn.GetCollections()
				if err != nil {
					pageData.ErrorDescription = err.Error()
					pageData.StatusCode = 500
				}
				if len(collections) == 0 {
					pageData.ErrorString = "No collections available!"
					pageData.StatusCode = 404
				}
				pageData.Collections = collections

			default:
				//
			}
		}

		website.ExecuteTemplate(w, tpl, pageData)
	}
}

func (website *Website) GetRouter() func() chi.Router {

	log.Printf("Building routes for: [%s]", website.WebsiteConfig.SiteName)
	router := func() chi.Router {
		r := chi.NewRouter()

		r.NotFound(website.NotFoundHandler)

		// Load Website templates
		for _, template := range *website.TemplateConfigs {
			if template.Path != "" {
				fmt.Printf("			> Setting up route: %s%s\n", website.WebsiteConfig.SiteName, template.Path)
				r.With(RouterCtx).Get(template.Path, website.GetRoute(template.Name))
				if template.PaginateType != 0 {
					paginatePath := path.Join(template.Path, "/{page:[0-9]+}")
					fmt.Printf("			> Setting up route: %s%s\n", website.WebsiteConfig.SiteName, paginatePath)
					switch template.PaginateType {
					case 1:
						// use same router
						r.With(RouterCtx).Get(paginatePath, website.GetRoute(template.Name))
					case 2:
						// 302 redirtect to slug
						r.With(RouterCtx).Get(paginatePath, paginate302Redirect)
					}

				}
			}
		}

		// Load API routes
		if website.WebsiteConfig.APIVersion == 1 {
			apiV1 := api.NewAPIV1(website.DBConn)
			r.Mount("/api", apiV1.APIRouter(website.WebsiteConfig.SiteName))
			website.APIHandler = &api.APIHandler{API: apiV1}
		}

		workDir, _ := os.Getwd()

		// add /public directory
		publicDir := http.Dir(filepath.Join(workDir, website.WebsiteConfig.Directory, "public"))
		fmt.Printf("			> Setting up public folder: %s\n", publicDir)
		FileServer(r, "/public/", publicDir)

		// add /sitemaps directory
		sitemapsDir := http.Dir(filepath.Join(workDir, website.WebsiteConfig.Directory, "sitemaps"))
		fmt.Printf("			> Setting up sitemaps folder: %s\n", sitemapsDir)
		FileServer(r, "/sitemaps/", sitemapsDir)

		// start media resizer
		r.Get("/media-proxy/width/{width}", func(w http.ResponseWriter, r *http.Request) {
			imageURL := r.URL.Query().Get("url")

			widthStr := chi.URLParam(r, "width")
			width, err := strconv.Atoi(widthStr)
			if err != nil {
				http.Error(w, "Invalid width parameter", http.StatusBadRequest)
				return
			}

			if imageURL == "" {
				http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
				return
			}

			acceptWebP := strings.Contains(r.Header.Get("Accept"), "image/webp")

			err = media.ProxyAndResizeImage(imageURL, width, w, acceptWebP)
			if err != nil {
				http.Error(w, "Error resizing and proxying image", http.StatusInternalServerError)
				return
			}
		})

		return r
	}
	return router
}

func paginate302Redirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Get the existing query parameters
	existingQuery := r.URL.RawQuery

	// Create the new redirect URL with preserved parameters
	redirectURL := fmt.Sprintf("/%s", slug)
	if existingQuery != "" {
		redirectURL = fmt.Sprintf("%s?%s", redirectURL, existingQuery)
	}

	// Redirect to the main route with preserved parameters
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func RouterCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		count := chi.URLParam(r, "count")
		offset := chi.URLParam(r, "offset")
		page := chi.URLParam(r, "page")
		if page == "" {
			page = "1"
		}

		taxonomy := ""

		// Check if the requesrt URL has a taxonomy in it
		urlPath := r.URL.Path
		if strings.HasPrefix(urlPath, "/category/") {
			taxonomy = "category"
		} else if strings.HasPrefix(urlPath, "/tag/") {
			taxonomy = "tag"
		} else if strings.HasPrefix(urlPath, "/author/") {
			taxonomy = "author"
		}

		vars := map[string]string{
			"taxonomy": taxonomy,
			"slug":     slug,
			"count":    count,
			"offset":   offset,
			"page":     page,
		}

		ctx := context.WithValue(r.Context(), "vars", vars)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.") // TODO strip?
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		nlfs := NoListFileSystem{root}
		fs := http.StripPrefix(pathPrefix, http.FileServer(nlfs))
		fs.ServeHTTP(w, r)
	})
}

func (f NoListFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (fs NoListFileSystem) Open(name string) (http.File, error) {
	f, err := fs.base.Open(name)
	if err != nil {
		return nil, err
	}
	return NoListFile{f}, nil
}
