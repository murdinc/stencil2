package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/session"
	"github.com/murdinc/stencil2/structs"
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

	// E-commerce routes

	// Collections
	api.addRoute("/api/v1/collections", "GET", api.getCollections, "collections")
	api.addRoute("/api/v1/collection/{slug}", "GET", api.getCollection, "collection")

	// Products
	api.addRoute("/api/v1/products", "GET", api.getProducts, "products")
	api.addRoute("/api/v1/products/{count}", "GET", api.getProducts, "products")
	api.addRoute("/api/v1/products/{count}/{offset}", "GET", api.getProducts, "products")
	api.addRoute("/api/v1/product/{slug}", "GET", api.getProduct, "product")

	// Collection Products
	api.addRoute("/api/v1/collection/{slug}/products", "GET", api.getCollectionProducts, "products")
	api.addRoute("/api/v1/collection/{slug}/products/{count}", "GET", api.getCollectionProducts, "products")
	api.addRoute("/api/v1/collection/{slug}/products/{count}/{offset}", "GET", api.getCollectionProducts, "products")

	// Cart
	api.addRoute("/api/v1/cart", "GET", api.getCart, "cart")
	api.addRoute("/api/v1/cart/add", "POST", api.addToCart, "cart")
	api.addRoute("/api/v1/cart/update/{itemId}", "POST", api.updateCartItem, "cart")
	api.addRoute("/api/v1/cart/remove/{itemId}", "POST", api.removeFromCart, "cart")

	// Checkout & Orders
	api.addRoute("/api/v1/checkout", "POST", api.createOrder, "order")
	api.addRoute("/api/v1/order/{orderNumber}", "GET", api.getOrder, "order")
}

func (api *APIV1) APIRouter(siteName string) chi.Router {
	r := chi.NewRouter()
	r.NotFound(api.NotFoundHandler)

	for _, route := range api.Routes {
		fmt.Printf("			> Setting up API route: %s %s%s\n", route.Method, siteName, route.Path)
		switch route.Method {
		case "GET":
			r.With(APIRouterCtx).Get(route.Path, route.HTTPHandler)
		case "POST":
			r.With(APIRouterCtx).Post(route.Path, route.HTTPHandler)
		case "PUT":
			r.With(APIRouterCtx).Put(route.Path, route.HTTPHandler)
		case "DELETE":
			r.With(APIRouterCtx).Delete(route.Path, route.HTTPHandler)
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

// E-commerce API Handlers

func (api *APIV1) getCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := api.dbConn.GetCollections()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(collections, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	collection, err := api.dbConn.GetCollection(vars["slug"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonData, err := json.MarshalIndent(collection, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := make(map[string]string)
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	products, err := api.dbConn.GetProducts(vars, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(products, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	product, err := api.dbConn.GetProduct(vars["slug"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonData, err := json.MarshalIndent(product, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getCollectionProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := make(map[string]string)
	queryParams := u.Query()
	for key := range queryParams {
		params[key] = queryParams.Get(key)
	}

	products, err := api.dbConn.GetCollectionProducts(vars["slug"], vars, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(products, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getCart(w http.ResponseWriter, r *http.Request) {
	sessionID := session.GetCartSession(r)
	if sessionID == "" {
		emptyCart := structs.Cart{
			Items:    []structs.CartItem{},
			Subtotal: 0,
		}
		jsonData, _ := json.MarshalIndent(emptyCart, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		return
	}

	cart, err := api.dbConn.GetCart(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(cart, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) addToCart(w http.ResponseWriter, r *http.Request) {
	sessionID := session.GetOrCreateCartSession(r, w)

	var reqBody struct {
		ProductID int `json:"product_id"`
		VariantID int `json:"variant_id"`
		Quantity  int `json:"quantity"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Quantity <= 0 {
		reqBody.Quantity = 1
	}

	err = api.dbConn.AddToCart(sessionID, reqBody.ProductID, reqBody.VariantID, reqBody.Quantity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cart, err := api.dbConn.GetCart(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(cart, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) updateCartItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	itemID, err := strconv.Atoi(vars["itemId"])
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	var reqBody struct {
		Quantity int `json:"quantity"`
	}

	err = json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = api.dbConn.UpdateCartItem(itemID, reqBody.Quantity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := session.GetCartSession(r)
	cart, err := api.dbConn.GetCart(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(cart, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) removeFromCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	itemID, err := strconv.Atoi(vars["itemId"])
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	err = api.dbConn.RemoveFromCart(itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := session.GetCartSession(r)
	cart, err := api.dbConn.GetCart(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(cart, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) createOrder(w http.ResponseWriter, r *http.Request) {
	sessionID := session.GetCartSession(r)
	if sessionID == "" {
		http.Error(w, "No cart session found", http.StatusBadRequest)
		return
	}

	cart, err := api.dbConn.GetCart(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(cart.Items) == 0 {
		http.Error(w, "Cart is empty", http.StatusBadRequest)
		return
	}

	var orderData map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&orderData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	orderData["cart_items"] = cart.Items

	order, err := api.dbConn.CreateOrder(orderData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.ClearCartSession(w)

	jsonData, err := json.MarshalIndent(order, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (api *APIV1) getOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	order, err := api.dbConn.GetOrder(vars["orderNumber"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonData, err := json.MarshalIndent(order, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
