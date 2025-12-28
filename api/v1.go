package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/email"
	"github.com/murdinc/stencil2/session"
	"github.com/murdinc/stencil2/shippo"
	"github.com/murdinc/stencil2/structs"
	"github.com/murdinc/stencil2/twilio"
	"github.com/murdinc/stencil2/utils"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/webhook"
)

// Simple rate limiter for contact form submissions
type contactRateLimiter struct {
	mu          sync.Mutex
	submissions map[string][]time.Time // IP -> timestamps
}

var rateLimiter = &contactRateLimiter{
	submissions: make(map[string][]time.Time),
}

// checkRateLimit returns true if the IP is allowed to submit
func (rl *contactRateLimiter) checkRateLimit(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-1 * time.Hour) // 1 hour window

	// Clean up old submissions
	if submissions, exists := rl.submissions[ip]; exists {
		var recent []time.Time
		for _, t := range submissions {
			if t.After(windowStart) {
				recent = append(recent, t)
			}
		}
		rl.submissions[ip] = recent

		// Check if limit exceeded (max 3 per hour)
		if len(recent) >= 3 {
			return false
		}
	}

	// Record this submission
	rl.submissions[ip] = append(rl.submissions[ip], now)
	return true
}

// cleanup removes old entries from all IPs to prevent memory leak
func (rl *contactRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Hour)

	for ip, times := range rl.submissions {
		validTimes := []time.Time{}
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) == 0 {
			delete(rl.submissions, ip)
		} else {
			rl.submissions[ip] = validTimes
		}
	}
}

// startCleanup starts a background goroutine to periodically clean up old entries
func (rl *contactRateLimiter) startCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()
}

// API represents the V1 API instance.
type APIV1 struct {
	Routes        []Route
	dbConn        *database.DBConnection
	websiteConfig *configs.WebsiteConfig
	envConfig     *configs.EnvironmentConfig
	shippoClient  *shippo.Client
}

type ErrorResponse struct {
	StatusCode  int    `json:"status_code"`
	ErrorString string `json:"error_message"`
}

// NewAPIV1 creates and returns a new instance of the V1 API.
func NewAPIV1(dbConn *database.DBConnection, websiteConfig *configs.WebsiteConfig, envConfig *configs.EnvironmentConfig) *APIV1 {
	// Get Shippo API key from site config
	shippoKey := websiteConfig.Shippo.APIKey

	// Start rate limiter cleanup (only once for all sites)
	rateLimiter.startCleanup()

	api := &APIV1{
		Routes:        make([]Route, 0),
		dbConn:        dbConn,
		websiteConfig: websiteConfig,
		envConfig:     envConfig,
		shippoClient:  shippo.NewClient(shippoKey),
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
	api.addRoute("/api/v1/config", "GET", api.getConfig, "config")
	api.addRoute("/api/v1/validate-address", "POST", api.validateAddress, "address")
	api.addRoute("/api/v1/create-payment-intent", "POST", api.createPaymentIntent, "payment")
	api.addRoute("/api/v1/checkout", "POST", api.createOrder, "order")
	api.addRoute("/api/v1/order/{orderNumber}", "GET", api.getOrder, "order")
	api.addRoute("/api/v1/tracking/{carrier}/{trackingNumber}", "GET", api.getTracking, "tracking")
	api.addRoute("/api/v1/webhook/stripe", "GET", api.webhookInfo, "webhook")
	api.addRoute("/api/v1/webhook/stripe", "POST", api.handleStripeWebhook, "webhook")
	api.addRoute("/api/v1/webhook/shippo", "GET", api.webhookInfo, "webhook")
	api.addRoute("/api/v1/webhook/shippo", "POST", api.handleShippoWebhook, "webhook")

	// Marketing
	api.addRoute("/api/v1/sms-signup", "POST", api.createSMSSignup, "sms")
	api.addRoute("/api/v1/sms-verify", "POST", api.verifySMSCode, "sms")
	api.addRoute("/api/v1/sms-webhook", "POST", api.handleSMSWebhook, "sms")

	// Analytics
	api.addRoute("/api/v1/track", "POST", api.trackAnalytics, "analytics")

	// Contact
	api.addRoute("/api/v1/contact", "POST", api.submitContactForm, "contact")
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
		carrier := chi.URLParam(r, "carrier")
		trackingNumber := chi.URLParam(r, "trackingNumber")
		orderNumber := chi.URLParam(r, "orderNumber")

		vars := map[string]string{
			"taxonomy":       taxonomy,
			"slug":           slug,
			"count":          count,
			"offset":         offset,
			"page":           page,
			"carrier":        carrier,
			"trackingNumber": trackingNumber,
			"orderNumber":    orderNumber,
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

	// Validate quantity (1-100)
	if reqBody.Quantity < 1 {
		http.Error(w, "Quantity must be at least 1", http.StatusBadRequest)
		return
	}
	if reqBody.Quantity > 100 {
		http.Error(w, "Quantity cannot exceed 100", http.StatusBadRequest)
		return
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
	itemIDStr := chi.URLParam(r, "itemId")
	itemID, err := strconv.Atoi(itemIDStr)
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

	// Validate quantity (1-100)
	if reqBody.Quantity < 1 {
		http.Error(w, "Quantity must be at least 1", http.StatusBadRequest)
		return
	}
	if reqBody.Quantity > 100 {
		http.Error(w, "Quantity cannot exceed 100", http.StatusBadRequest)
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
	itemIDStr := chi.URLParam(r, "itemId")
	itemID, err := strconv.Atoi(itemIDStr)
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

	// Get tax rate and shipping cost from config (0 is valid)
	orderData["tax_rate"] = api.websiteConfig.Ecommerce.TaxRate
	orderData["shipping_cost"] = api.websiteConfig.Ecommerce.ShippingCost

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

// getConfig returns public configuration (like Stripe publishable key)
func (api *APIV1) getConfig(w http.ResponseWriter, r *http.Request) {
	// Get Stripe publishable key from site config
	publishableKey := api.websiteConfig.Stripe.PublishableKey

	// Get tax rate and shipping cost from config (0 is valid)
	taxRate := api.websiteConfig.Ecommerce.TaxRate
	shippingCost := api.websiteConfig.Ecommerce.ShippingCost

	response := map[string]interface{}{
		"stripePublishableKey": publishableKey,
		"taxRate":              taxRate,
		"shippingCost":         shippingCost,
	}

	jsonData, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// createPaymentIntent creates a Stripe payment intent for the cart
func (api *APIV1) createPaymentIntent(w http.ResponseWriter, r *http.Request) {
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

	// Parse request body to extract customer email and shipping address
	var requestBody map[string]interface{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err == nil {
		json.Unmarshal(bodyBytes, &requestBody)
		// Restore body for potential future reads
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Calculate total (subtotal + tax + shipping)
	subtotal := cart.Subtotal

	// Get tax rate and shipping cost from config (0 is valid)
	taxRate := api.websiteConfig.Ecommerce.TaxRate
	tax := subtotal * taxRate

	shippingCost := api.websiteConfig.Ecommerce.ShippingCost

	total := subtotal + tax + shippingCost

	// Get Stripe secret key from site config
	stripeKey := api.websiteConfig.Stripe.SecretKey
	if stripeKey == "" {
		http.Error(w, "Stripe not configured", http.StatusInternalServerError)
		return
	}

	// Set Stripe API key
	stripe.Key = stripeKey

	// Try to get/create customer and link to Stripe
	var stripeCustomerID string
	if requestBody != nil {
		if email, ok := requestBody["email"].(string); ok && email != "" {
			if shippingAddr, ok := requestBody["shipping_address"].(map[string]interface{}); ok {
				firstName, _ := shippingAddr["first_name"].(string)
				lastName, _ := shippingAddr["last_name"].(string)

				if firstName != "" && lastName != "" {
					// Get or create local customer record
					cust, err := api.dbConn.GetOrCreateCustomer(email, firstName, lastName)
					if err == nil {
						// Check if customer already has Stripe ID
						if cust.StripeCustomerID != "" {
							stripeCustomerID = cust.StripeCustomerID
						} else {
							// Create Stripe customer
							stripeParams := &stripe.CustomerParams{
								Email: stripe.String(email),
								Name:  stripe.String(firstName + " " + lastName),
							}
							stripeCust, err := customer.New(stripeParams)
							if err == nil && stripeCust != nil {
								stripeCustomerID = stripeCust.ID
								// Update local customer record with Stripe ID
								api.dbConn.UpdateCustomerStripeID(cust.ID, stripeCust.ID)
							}
						}
					}
				}
			}
		}
	}

	// Create payment intent
	params := &stripe.PaymentIntentParams{
		Amount:              stripe.Int64(int64(total * 100)), // Convert to cents
		Currency:            stripe.String(string(stripe.CurrencyUSD)),
		PaymentMethodTypes:  stripe.StringSlice([]string{"card", "link"}), // Card payments, Apple Pay, Google Pay, and Link
	}

	// Link to Stripe customer if we have one
	if stripeCustomerID != "" {
		params.Customer = stripe.String(stripeCustomerID)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create payment intent: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"clientSecret": pi.ClientSecret,
		"amount":       total,
		"subtotal":     subtotal,
		"tax":          tax,
		"shipping":     shippingCost,
	}

	jsonData, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// webhookInfo returns 200 OK for webhook endpoints when accessed via GET
func (api *APIV1) webhookInfo(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleStripeWebhook handles Stripe webhook events
func (api *APIV1) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Get Stripe secret key from site config
	stripeKey := api.websiteConfig.Stripe.SecretKey

	// Verify webhook signature
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), stripeKey)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Error parsing webhook", http.StatusBadRequest)
			return
		}

		// Find order by payment intent ID and update status
		err = api.handlePaymentSuccess(paymentIntent.ID)
		if err != nil {
			log.Printf("Error handling payment success: %v", err)
			// Don't return error to Stripe, we've received the webhook
		}

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Error parsing webhook", http.StatusBadRequest)
			return
		}

		// Update order status to failed
		err = api.dbConn.UpdateOrderPaymentStatusByIntentID(paymentIntent.ID, "failed")
		if err != nil {
			log.Printf("Error updating payment status: %v", err)
		}

	case "charge.refunded":
		var charge stripe.Charge
		err := json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Error parsing webhook", http.StatusBadRequest)
			return
		}

		// Update order status to refunded
		err = api.dbConn.UpdateOrderPaymentStatusByIntentID(charge.PaymentIntent.ID, "refunded")
		if err != nil {
			log.Printf("Error updating payment status: %v", err)
		}

	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// handlePaymentSuccess updates order status and sends confirmation email
func (api *APIV1) handlePaymentSuccess(paymentIntentID string) error {
	// Update payment status in database
	err := api.dbConn.UpdateOrderPaymentStatusByIntentID(paymentIntentID, "paid")
	if err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Get the order to send confirmation email
	order, err := api.dbConn.GetOrderByPaymentIntentID(paymentIntentID)
	if err != nil {
		return fmt.Errorf("failed to get order: %v", err)
	}

	// Send confirmation email
	emailService, err := email.NewEmailService()
	if err != nil {
		log.Printf("Failed to create email service: %v", err)
		return nil // Don't fail the webhook if email fails
	}

	// Convert order items to email format
	emailItems := make([]email.OrderItem, len(order.Items))
	for i, item := range order.Items {
		emailItems[i] = email.OrderItem{
			ProductName:  item.ProductName,
			VariantTitle: item.VariantTitle,
			Quantity:     item.Quantity,
			Price:        item.Price,
			Total:        item.Total,
		}
	}

	err = emailService.SendOrderConfirmation(
		api.websiteConfig,
		order.OrderNumber,
		order.CustomerEmail,
		order.CustomerName,
		emailItems,
		order.Subtotal,
		order.Tax,
		order.ShippingCost,
		order.Total,
	)
	if err != nil {
		log.Printf("Failed to send confirmation email: %v", err)
		// Don't fail the webhook if email fails
	}

	// Send admin notification email
	err = emailService.SendAdminOrderNotification(
		api.websiteConfig,
		order.OrderNumber,
		order.CustomerEmail,
		order.CustomerName,
		emailItems,
		order.Subtotal,
		order.Tax,
		order.ShippingCost,
		order.Total,
	)
	if err != nil {
		log.Printf("Failed to send admin notification email: %v", err)
		// Don't fail the webhook if admin email fails
	}

	return nil
}

// handleShippoWebhook handles incoming Shippo tracking webhooks
func (api *APIV1) handleShippoWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading Shippo webhook body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the webhook payload
	var webhookData map[string]interface{}
	if err := json.Unmarshal(body, &webhookData); err != nil {
		log.Printf("Error parsing Shippo webhook JSON: %v", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Log the webhook for debugging
	log.Printf("Shippo webhook received: %v", webhookData)

	// Extract tracking information
	trackingNumber, ok := webhookData["tracking_number"].(string)
	if !ok || trackingNumber == "" {
		log.Printf("No tracking number in Shippo webhook")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get tracking status data
	trackingStatusData, ok := webhookData["tracking_status"].(map[string]interface{})
	if !ok {
		log.Printf("No tracking_status in Shippo webhook")
		w.WriteHeader(http.StatusOK)
		return
	}

	status, ok := trackingStatusData["status"].(string)
	if !ok || status == "" {
		log.Printf("No status in tracking_status")
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Shippo tracking update - Tracking: %s, Status: %s", trackingNumber, status)

	// Map Shippo status to our fulfillment status
	var fulfillmentStatus string
	switch strings.ToUpper(status) {
	case "PRE_TRANSIT":
		fulfillmentStatus = "processing"
	case "TRANSIT":
		fulfillmentStatus = "shipped"
	case "DELIVERED":
		fulfillmentStatus = "delivered"
	case "RETURNED":
		fulfillmentStatus = "returned"
	case "FAILURE":
		fulfillmentStatus = "failed"
	default:
		// Unknown status, don't update
		log.Printf("Unknown Shippo status: %s", status)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get the order by tracking number
	order, err := api.dbConn.GetOrderByTrackingNumber(trackingNumber)
	if err != nil {
		log.Printf("Order not found for tracking number %s: %v", trackingNumber, err)
		w.WriteHeader(http.StatusOK) // Still return 200 to acknowledge webhook
		return
	}

	// Update the order fulfillment status
	err = api.dbConn.UpdateOrderTrackingStatus(trackingNumber, fulfillmentStatus)
	if err != nil {
		log.Printf("Failed to update tracking status: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Updated order %s to status: %s", order.OrderNumber, fulfillmentStatus)

	// Send delivery confirmation email if delivered
	if fulfillmentStatus == "delivered" {
		emailService, err := email.NewEmailService()
		if err == nil {
			err = emailService.SendDeliveryConfirmation(
				api.websiteConfig,
				order.OrderNumber,
				order.CustomerEmail,
				order.CustomerName,
			)
			if err != nil {
				log.Printf("Failed to send delivery confirmation email: %v", err)
			} else {
				log.Printf("Sent delivery confirmation email for order %s", order.OrderNumber)
			}
		} else {
			log.Printf("Failed to create email service: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// validateAddress validates a shipping address using Shippo
func (api *APIV1) validateAddress(w http.ResponseWriter, r *http.Request) {
	var addressData map[string]string
	err := json.NewDecoder(r.Body).Decode(&addressData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create Shippo address from request
	addr := shippo.Address{
		Name:    addressData["name"],
		Street1: addressData["street1"],
		Street2: addressData["street2"],
		City:    addressData["city"],
		State:   addressData["state"],
		Zip:     addressData["zip"],
		Country: addressData["country"],
		Email:   addressData["email"],
		Phone:   addressData["phone"],
	}

	// Validate with Shippo
	validatedAddr, err := api.shippoClient.ValidateAddress(addr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Address validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return validation results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validatedAddr)
}

// getTracking retrieves package tracking information using Shippo
func (api *APIV1) getTracking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars, ok := ctx.Value("vars").(map[string]string)
	if !ok {
		http.Error(w, http.StatusText(422), 422)
		return
	}

	carrier := vars["carrier"]
	trackingNumber := vars["trackingNumber"]

	if carrier == "" || trackingNumber == "" {
		http.Error(w, "Carrier and tracking number are required", http.StatusBadRequest)
		return
	}

	// Get tracking from Shippo
	tracking, err := api.shippoClient.GetTracking(carrier, trackingNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve tracking: %v", err), http.StatusInternalServerError)
		return
	}

	// Return tracking information
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracking)
}

func (api *APIV1) createSMSSignup(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		CountryCode string `json:"countryCode"`
		Phone       string `json:"phone"`
		Email       string `json:"email"`
		Source      string `json:"source"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Phone == "" {
		http.Error(w, "Phone number is required", http.StatusBadRequest)
		return
	}

	// Default to +1 if not provided
	if reqBody.CountryCode == "" {
		reqBody.CountryCode = "+1"
	}

	// Generate 6-digit verification code
	code, err := utils.GenerateVerificationCode()
	if err != nil {
		log.Printf("Failed to generate verification code: %v", err)
		http.Error(w, "Failed to generate verification code", http.StatusInternalServerError)
		return
	}

	// Set expiration to 10 minutes from now
	expiresAt := time.Now().Add(10 * time.Minute)

	// Store verification code in database
	err = api.dbConn.SetSMSVerificationCode(reqBody.CountryCode, reqBody.Phone, code, expiresAt)
	if err != nil {
		log.Printf("Failed to store verification code: %v", err)
		http.Error(w, "Failed to initiate verification", http.StatusInternalServerError)
		return
	}

	// Send verification code via Twilio
	twilioClient := twilio.NewClient(
		api.websiteConfig.Twilio.AccountSID,
		api.websiteConfig.Twilio.AuthToken,
		api.websiteConfig.Twilio.FromPhone,
	)

	// Format phone number for Twilio (E.164 format)
	toPhone := twilio.FormatPhoneNumber(reqBody.CountryCode, reqBody.Phone)

	err = twilioClient.SendVerificationCode(toPhone, code)
	if err != nil {
		log.Printf("Failed to send verification code via Twilio: %v", err)
		http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Verification code sent to your phone",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *APIV1) verifySMSCode(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		CountryCode string `json:"countryCode"`
		Phone       string `json:"phone"`
		Code        string `json:"code"`
		Email       string `json:"email"`
		Source      string `json:"source"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Phone == "" || reqBody.Code == "" {
		http.Error(w, "Phone number and verification code are required", http.StatusBadRequest)
		return
	}

	// Default to +1 if not provided
	if reqBody.CountryCode == "" {
		reqBody.CountryCode = "+1"
	}

	// Verify the code
	verified, err := api.dbConn.VerifySMSCode(reqBody.CountryCode, reqBody.Phone, reqBody.Code)
	if err != nil {
		log.Printf("Failed to verify code: %v", err)
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}

	if !verified {
		response := map[string]interface{}{
			"success": false,
			"message": "Invalid or expired verification code",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update email and source if provided
	if reqBody.Email != "" || reqBody.Source != "" {
		_, err = api.dbConn.CreateSMSSignup(reqBody.CountryCode, reqBody.Phone, reqBody.Email, reqBody.Source)
		if err != nil {
			log.Printf("Failed to update signup details: %v", err)
			// Continue anyway since verification succeeded
		}
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Phone number verified successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSMSWebhook handles incoming SMS from Twilio (for unsubscribe)
func (api *APIV1) handleSMSWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse Twilio webhook form data
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get incoming message details from Twilio
	from := r.FormValue("From")        // Phone number (E.164 format: +14155551234)
	body := r.FormValue("Body")        // Message text

	if from == "" || body == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Normalize message body for comparison
	bodyLower := strings.ToLower(strings.TrimSpace(body))

	// Check if this is an unsubscribe request
	// Common opt-out keywords per CTIA guidelines
	unsubscribeKeywords := []string{
		"stop", "stopall", "unsubscribe", "cancel", "end", "quit",
	}

	isUnsubscribe := false
	for _, keyword := range unsubscribeKeywords {
		if bodyLower == keyword {
			isUnsubscribe = true
			break
		}
	}

	if isUnsubscribe {
		// Parse phone number to extract country code and phone
		// E.164 format: +14155551234
		countryCode := ""
		phone := ""

		if len(from) > 2 && from[0] == '+' {
			// Extract country code (1-3 digits after +)
			if len(from) >= 12 { // +1 country code (11 digits total)
				countryCode = from[:2]  // +1
				phone = from[2:]        // 4155551234
			} else if len(from) >= 11 { // Other country codes
				countryCode = from[:3]  // +44, etc
				phone = from[3:]
			}
		}

		// Remove any non-digit characters from phone
		cleanPhone := ""
		for _, c := range phone {
			if c >= '0' && c <= '9' {
				cleanPhone += string(c)
			}
		}

		if countryCode != "" && cleanPhone != "" {
			// Mark as unsubscribed in database
			err := api.dbConn.UnsubscribeSMS(countryCode, cleanPhone)
			if err != nil {
				log.Printf("Error unsubscribing %s: %v", from, err)
			}
		}

		// Send TwiML response confirming unsubscribe
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>You have been unsubscribed from SMS notifications. You will not receive further messages.</Message>
</Response>`)
		return
	}

	// For other messages, just send empty response (no action)
	w.Header().Set("Content-Type", "text/xml")
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Response></Response>`)
}

// ===============================
// Analytics Handler
// ===============================

func (api *APIV1) trackAnalytics(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		VisitorID    string                 `json:"v"`
		SessionID    string                 `json:"s"`
		EventType    string                 `json:"t"`
		Path         string                 `json:"p"`
		Referrer     string                 `json:"r"`
		EventName    string                 `json:"e"`
		EventData    map[string]interface{} `json:"d"`
		ScreenWidth  int                    `json:"sw"`
		ScreenHeight int                    `json:"sh"`
		DeviceType   string                 `json:"dt"`
		PageviewID   int64                  `json:"pid"`
		TimeOnPage   int                    `json:"top"`
	}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		w.WriteHeader(http.StatusNoContent) // Return 204 even on error to keep beacon fast
		return
	}

	// Extract client info from request
	userAgent := r.UserAgent()
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.Header.Get("X-Real-IP")
	}
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	// Track based on type
	if reqBody.EventType == "u" {
		// Time update for existing pageview
		err = api.dbConn.UpdatePageViewTime(reqBody.PageviewID, reqBody.TimeOnPage)
		if err != nil {
			log.Printf("Failed to update time on page: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	} else if reqBody.EventType == "e" && reqBody.EventName != "" {
		// Custom event
		err = api.dbConn.TrackEvent(reqBody.VisitorID, reqBody.SessionID, reqBody.EventName, reqBody.Path, reqBody.EventData)
		if err != nil {
			log.Printf("Failed to track event: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	} else if reqBody.EventType == "h" {
		// Heartbeat - track as event to update session activity
		err = api.dbConn.TrackEvent(reqBody.VisitorID, reqBody.SessionID, "heartbeat", reqBody.Path, nil)
		if err != nil {
			log.Printf("Failed to track heartbeat: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		// Pageview (default) - return the pageview ID
		pageviewID, err := api.dbConn.TrackPageView(
			reqBody.VisitorID,
			reqBody.SessionID,
			reqBody.Path,
			reqBody.Referrer,
			userAgent,
			ipAddress,
			reqBody.ScreenWidth,
			reqBody.ScreenHeight,
		)
		if err != nil {
			log.Printf("Failed to track pageview: %v", err)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Return pageview ID for client to use in time updates
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pageview_id": pageviewID,
		})
	}
}

// submitContactForm handles contact form submissions
func (api *APIV1) submitContactForm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Message      string `json:"message"`
		Website      string `json:"website"`        // Honeypot field
		FormLoadedAt string `json:"form_loaded_at"` // Timestamp
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// SPAM PREVENTION 1: Honeypot - if filled, it's a bot
	if req.Website != "" {
		log.Printf("Spam detected: honeypot filled by %s", r.RemoteAddr)
		http.Error(w, "Invalid submission", http.StatusBadRequest)
		return
	}

	// SPAM PREVENTION 2: Time-based validation - form must be open for at least 3 seconds
	if req.FormLoadedAt != "" {
		loadedAt, err := strconv.ParseInt(req.FormLoadedAt, 10, 64)
		if err == nil {
			loadedTime := time.UnixMilli(loadedAt)
			timeTaken := time.Since(loadedTime)
			if timeTaken < 3*time.Second {
				log.Printf("Spam detected: form submitted too quickly (%v) from %s", timeTaken, r.RemoteAddr)
				http.Error(w, "Please wait a moment before submitting", http.StatusTooManyRequests)
				return
			}
		}
	}

	// SPAM PREVENTION 3: Rate limiting - max 3 submissions per hour per IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	}
	if !rateLimiter.checkRateLimit(clientIP) {
		log.Printf("Rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "Too many submissions. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Email == "" || req.Message == "" {
		http.Error(w, "Name, email, and message are required", http.StatusBadRequest)
		return
	}

	// Basic email validation
	if !strings.Contains(req.Email, "@") {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	// Save message to database (api.dbConn is already connected to website-specific database)
	err = api.dbConn.CreateMessage(req.Name, req.Email, req.Message)
	if err != nil {
		log.Printf("Error saving contact message: %v", err)
		http.Error(w, "Failed to submit message", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Thank you for your message! We'll get back to you soon.",
	})
}
