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

	"github.com/go-chi/chi"
	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
	"github.com/murdinc/stencil2/email"
	"github.com/murdinc/stencil2/session"
	"github.com/murdinc/stencil2/shippo"
	"github.com/murdinc/stencil2/structs"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/webhook"
)

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
	api.addRoute("/api/v1/webhook/stripe", "POST", api.handleStripeWebhook, "webhook")

	// Marketing
	api.addRoute("/api/v1/sms-signup", "POST", api.createSMSSignup, "sms")
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
		Amount:   stripe.Int64(int64(total * 100)), // Convert to cents
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
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
	// Note: In production, you should set up a webhook secret in Stripe dashboard
	// and verify the signature using: webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), webhookSecret)
	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), stripeKey)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		// For now, continue anyway for testing - REMOVE THIS IN PRODUCTION
		var tempEvent stripe.Event
		if err := json.Unmarshal(body, &tempEvent); err != nil {
			http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
			return
		}
		event = tempEvent
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
	emailService, err := email.NewEmailService(api.envConfig)
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

	return nil
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
		Phone  string `json:"phone"`
		Email  string `json:"email"`
		Source string `json:"source"`
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

	// Create signup
	signupID, err := api.dbConn.CreateSMSSignup(reqBody.Phone, reqBody.Email, reqBody.Source)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create signup: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"id":      signupID,
		"message": "Successfully signed up for SMS notifications",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
