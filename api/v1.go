package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/murdinc/stencil2/database"
)

// API represents the V1 API instance.
type APIV1 struct {
	Routes []Route
	dbConn *database.DBConnection
}

type ErrorResponse struct {
	StatusCode  int    `json:"status_code"`
	ErrorString string `json:"error_message"`
}

// NewAPIV1 creates and returns a new instance of the V1 API.
func NewAPIV1(dbConn *database.DBConnection) *APIV1 {
	api := &APIV1{
		Routes: make([]Route, 0),
		dbConn: dbConn,
	}

	api.initRoutesV1()

	return api
}

// initRoutesV1 initializes the routes for V1 API.
func (api *APIV1) initRoutesV1() {
	// Define V1 routes and associate them with their corresponding handlers.

	//categories list
	api.addRoute("/api/v1/categories", "GET", api.getCategories, "categories")

	// posts lists
	api.addRoute("/api/v1/posts", "GET", api.getPosts, "posts")
	api.addRoute("/api/v1/posts/{count}", "GET", api.getPosts, "posts")
	api.addRoute("/api/v1/posts/{count}/{offset}", "GET", api.getPosts, "posts")

	// category/tag/author posts
	api.addRoute("/api/v1/{taxonomy}/{slug}/posts", "GET", api.getPosts, "posts")
	api.addRoute("/api/v1/{taxonomy}/{slug}/posts/{count}", "GET", api.getPosts, "posts")
	api.addRoute("/api/v1/{taxonomy}/{slug}/posts/{count}/{offset}", "GET", api.getPosts, "posts")

	// single post
	api.addRoute("/api/v1/post/{slug}", "GET", api.getPost, "post")

}

func (api *APIV1) APIRouter(siteName string) chi.Router {
	r := chi.NewRouter()
	r.NotFound(api.NotFoundHandler)

	for _, route := range api.Routes {
		fmt.Printf("			> Setting up API route: %s %s%s\n", route.Method, siteName, route.Path)
		if route.Method == "GET" {
			r.With(APIRouterCtx).Get(route.Path, route.HTTPHandler)
		}
	}
	return r
}

func APIRouterCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taxonomy := chi.URLParam(r, "taxonomy")
		slug := chi.URLParam(r, "slug")
		count := chi.URLParam(r, "count")
		offset := chi.URLParam(r, "offset")
		page := chi.URLParam(r, "page")

		vars := map[string]string{
			"taxonomy": taxonomy,
			"slug":     slug,
			"count":    count,
			"offset":   offset,
			"page":     page,
		}

		ctx := context.WithValue(r.Context(), "vars", vars)

		// set expiration
		w.Header().Set("Cache-Control", "public, s-maxage=300, max-age=0")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *APIV1) GetInternalHandler(path string) (string, map[string]string, error) {
	params := make(map[string]string)
	if path == "" {
		return "", params, nil
	}

	// Parse the path and separate URL parameters
	u, err := url.Parse(path)
	if err != nil {
		return "", params, err
	}
	path = u.Path

	// Extract URL parameters into a map
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	for _, route := range api.Routes {
		if route.Path == path {
			return route.InternalHandler, params, nil
		}
	}
	return "", params, errors.New("internal handler not found!")
}

// addRoute adds a new route to the API.
func (api *APIV1) addRoute(path string, method string, httpHandler http.HandlerFunc, internalHandler string) {
	api.Routes = append(api.Routes, Route{
		Path:            path,
		Method:          method,
		HTTPHandler:     httpHandler,
		InternalHandler: internalHandler,
	})
}

func (api *APIV1) getCategories(w http.ResponseWriter, r *http.Request) {

	// Parse the path and separate URL parameters
	u, err := url.Parse(r.URL.String())
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Extract URL parameters into a map
	params := make(map[string]string)
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	categories, err := api.dbConn.GetCategories(params)
	if err != nil {
		// 500? 404?
		fmt.Println("Error:", err)
	}

	jsonData, err := json.MarshalIndent(categories, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON data to the response writer
	w.Write(jsonData)
}

func (api *APIV1) getPost(w http.ResponseWriter, r *http.Request) {

	// get the slug context
	ctx := r.Context()

	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		// TODO what to actually do here?
		http.Error(w, http.StatusText(422), 422)
		return
	}

	// Parse the path and separate URL parameters
	u, err := url.Parse(r.URL.String())
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Extract URL parameters into a map
	params := make(map[string]string)
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	post, err := api.dbConn.GetSingularPost(vars, params)
	if err != nil {
		// 500? 404?
		fmt.Println("Error:", err)
	}

	jsonData, err := json.MarshalIndent(post, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON data to the response writer
	w.Write(jsonData)
}

func (api *APIV1) getPosts(w http.ResponseWriter, r *http.Request) {

	// get the slug context
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		// TODO what to actually do here?
		http.Error(w, http.StatusText(422), 422)
		return
	}

	// Parse the path and separate URL parameters
	u, err := url.Parse(r.URL.String())
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Extract URL parameters into a map
	params := make(map[string]string)
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	posts, err := api.dbConn.GetMultiplePosts(vars, params)
	if err != nil {
		// 500? 404?
		fmt.Println("Error:", err)
	}

	jsonData, err := json.MarshalIndent(posts, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON data to the response writer
	w.Write(jsonData)
}

func (api *APIV1) NotFoundHandler(w http.ResponseWriter, r *http.Request) {

	// Create the error response struct
	errResponse := ErrorResponse{
		StatusCode:  404,
		ErrorString: "endpoint not found",
	}

	// Convert the struct to JSON
	jsonData, err := json.Marshal(errResponse)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	// Set the status code for the response
	w.WriteHeader(errResponse.StatusCode)
	// Write the JSON data to the response writer
	w.Write(jsonData)
}
