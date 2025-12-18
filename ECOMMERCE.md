# E-commerce Implementation Guide for Stencil2

This document explains how to use the e-commerce features in Stencil2 to build online stores.

## What Was Implemented

Stencil2 now includes a complete e-commerce backend with:

1. **Database Schema** - Products, Collections, Variants, Cart, Orders
2. **Go Structs** - Data structures for all e-commerce entities
3. **Database Query Methods** - Functions to retrieve/manage e-commerce data
4. **REST API Endpoints** - JSON APIs for all e-commerce operations
5. **Session Management** - Cookie-based cart sessions

## Database Setup

The e-commerce tables are **automatically created** when you start the server. No manual setup required!

When Stencil2 connects to each website's database, it checks for and creates the following tables if they don't exist:

### Tables Created

- `products_unified` - Product catalog
- `collections_unified` - Product collections (like categories)
- `product_collections` - Many-to-many relationship
- `product_images` - Product images with position ordering
- `product_variants` - Size, color, etc. variations
- `carts` - Shopping cart sessions
- `cart_items` - Items in carts
- `orders` - Customer orders
- `order_items` - Line items in orders

## API Endpoints

All endpoints return JSON. Use these in your JavaScript to build the frontend.

### Collections

**GET** `/api/v1/collections`
- Returns all published collections
- Response: Array of Collection objects with product counts

**GET** `/api/v1/collection/{slug}`
- Returns single collection by slug
- Response: Collection object with image and product count

### Products

**GET** `/api/v1/products`
**GET** `/api/v1/products/{count}`
**GET** `/api/v1/products/{count}/{offset}`
- Returns products with pagination
- Query params:
  - `featured=true|false` - Filter by featured
  - `sort=price_asc|price_desc|name` - Sort order
- Response: Array of Product objects with images and variants

**GET** `/api/v1/product/{slug}`
- Returns single product by slug
- Response: Product object with full details (images, variants, collections)

**GET** `/api/v1/collection/{slug}/products`
**GET** `/api/v1/collection/{slug}/products/{count}`
**GET** `/api/v1/collection/{slug}/products/{count}/{offset}`
- Returns products in a specific collection
- Query params: `sort=price_asc|price_desc|name|position`
- Response: Array of Product objects

### Cart

**GET** `/api/v1/cart`
- Returns current cart for the session
- Response: Cart object with items and subtotal

**POST** `/api/v1/cart/add`
- Adds item to cart
- Request body:
  ```json
  {
    "product_id": 123,
    "variant_id": 456,  // optional, 0 for no variant
    "quantity": 1
  }
  ```
- Response: Updated Cart object

**POST** `/api/v1/cart/update/{itemId}`
- Updates quantity of cart item
- Request body:
  ```json
  {
    "quantity": 3
  }
  ```
- Response: Updated Cart object

**POST** `/api/v1/cart/remove/{itemId}`
- Removes item from cart
- Response: Updated Cart object

### Checkout & Orders

**POST** `/api/v1/checkout`
- Creates an order from cart
- Request body:
  ```json
  {
    "customer_email": "user@example.com",
    "customer_name": "John Doe",
    "shipping_address_line1": "123 Main St",
    "shipping_city": "New York",
    "shipping_state": "NY",
    "shipping_zip": "10001",
    "shipping_country": "US"
  }
  ```
- Response: Order object with order_number
- Note: Clears cart session after successful order

**GET** `/api/v1/order/{orderNumber}`
- Returns order details
- Response: Order object with items

## Template System Integration

The template system automatically makes e-commerce data available to your templates based on the `apiEndpoint` in your template config.

### Example Template Configs

**Landing Page** (show collections):
```json
{
  "name": "store-landing",
  "path": "/",
  "apiEndpoint": "/api/v1/collections",
  "cacheTime": 3600
}
```

In your template (`store-landing.tpl`):
```html
{{ range .Collections }}
  <a href="/collection/{{ .Slug }}">
    <h3>{{ .Name }}</h3>
    <img src="{{ .Image.URL }}" alt="{{ .Image.AltText }}">
    <p>{{ .ProductCount }} products</p>
  </a>
{{ end }}
```

**Store Page** (all products):
```json
{
  "name": "store",
  "path": "/store",
  "apiEndpoint": "/api/v1/products",
  "apiCount": 12,
  "cacheTime": 300
}
```

In your template (`store.tpl`):
```html
{{ range .Products }}
  <div class="product">
    <a href="/product/{{ .Slug }}">
      <h3>{{ .Name }}</h3>
      {{ if .Images }}
        <img src="{{ (index .Images 0).Image.URL }}" alt="{{ .Name }}">
      {{ end }}
      <p class="price">${{ .Price }}</p>
      {{ if gt .CompareAtPrice .Price }}
        <p class="compare">${{ .CompareAtPrice }}</p>
      {{ end }}
    </a>
  </div>
{{ end }}
```

**Collection Page**:
```json
{
  "name": "collection",
  "path": "/collection/{slug}",
  "apiEndpoint": "/api/v1/collection/{slug}/products",
  "apiCount": 12,
  "cacheTime": 300
}
```

In your template (`collection.tpl`):
```html
<h1>{{ .Collection.Name }}</h1>
<p>{{ .Collection.Description }}</p>

{{ range .Products }}
  <div class="product">
    <!-- Same as store page -->
  </div>
{{ end }}
```

**Product Page**:
```json
{
  "name": "product",
  "path": "/product/{slug}",
  "apiEndpoint": "/api/v1/product/{slug}",
  "noCache": true
}
```

In your template (`product.tpl`):
```html
<h1>{{ .Product.Name }}</h1>
<p>{{ .Product.Description }}</p>

<!-- Product Images -->
<div class="gallery">
  {{ range .Product.Images }}
    <img src="{{ .Image.URL }}" alt="{{ .Image.AltText }}">
  {{ end }}
</div>

<!-- Price -->
<div class="price">
  ${{ .Product.Price }}
  {{ if gt .Product.CompareAtPrice .Product.Price }}
    <span class="compare">${{ .Product.CompareAtPrice }}</span>
  {{ end }}
</div>

<!-- Variants -->
{{ if .Product.Variants }}
  <select id="variant-select">
    {{ range .Product.Variants }}
      <option value="{{ .ID }}" data-price="{{ .Price }}">
        {{ .Title }} - ${{ .Price }}
      </option>
    {{ end }}
  </select>
{{ end }}

<!-- Add to Cart Button -->
<button id="add-to-cart"
        data-product-id="{{ .Product.ID }}">
  Add to Cart
</button>

<!-- Your JavaScript will handle the API call -->
<script>
document.getElementById('add-to-cart').addEventListener('click', function() {
  const productId = this.dataset.productId;
  const variantId = document.getElementById('variant-select')?.value || 0;

  fetch('/api/v1/cart/add', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      product_id: parseInt(productId),
      variant_id: parseInt(variantId),
      quantity: 1
    })
  })
  .then(res => res.json())
  .then(cart => {
    alert('Added to cart!');
    // Update cart UI
  });
});
</script>
```

**Cart Page**:
```json
{
  "name": "cart",
  "path": "/cart",
  "apiEndpoint": "/api/v1/cart",
  "noCache": true
}
```

In your template (`cart.tpl`):
```html
<h1>Shopping Cart</h1>

{{ if .Cart.Items }}
  {{ range .Cart.Items }}
    <div class="cart-item">
      <h3>{{ .Product.Name }}</h3>
      {{ if .Variant.Title }}
        <p>{{ .Variant.Title }}</p>
      {{ end }}
      <p>Price: ${{ .Price }}</p>
      <p>Quantity: {{ .Quantity }}</p>
      <p>Total: ${{ .Total }}</p>

      <button class="update-qty" data-item-id="{{ .ID }}">Update</button>
      <button class="remove-item" data-item-id="{{ .ID }}">Remove</button>
    </div>
  {{ end }}

  <div class="cart-total">
    <h3>Subtotal: ${{ .Cart.Subtotal }}</h3>
  </div>

  <a href="/checkout">Proceed to Checkout</a>
{{ else }}
  <p>Your cart is empty</p>
{{ end }}
```

**Checkout Page**:
```json
{
  "name": "checkout",
  "path": "/checkout",
  "noCache": true
}
```

In your template (`checkout.tpl`):
```html
<h1>Checkout</h1>

<form id="checkout-form">
  <input type="email" name="customer_email" placeholder="Email" required>
  <input type="text" name="customer_name" placeholder="Full Name" required>
  <input type="text" name="shipping_address_line1" placeholder="Address" required>
  <input type="text" name="shipping_city" placeholder="City" required>
  <input type="text" name="shipping_state" placeholder="State" required>
  <input type="text" name="shipping_zip" placeholder="ZIP Code" required>
  <input type="text" name="shipping_country" value="US" required>

  <button type="submit">Place Order</button>
</form>

<script>
document.getElementById('checkout-form').addEventListener('submit', function(e) {
  e.preventDefault();
  const formData = new FormData(this);
  const data = Object.fromEntries(formData);

  fetch('/api/v1/checkout', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  })
  .then(res => res.json())
  .then(order => {
    // Redirect to confirmation page
    window.location.href = '/order/' + order.order_number;
  });
});
</script>
```

**Order Confirmation**:
```json
{
  "name": "order-confirmation",
  "path": "/order/{orderNumber}",
  "apiEndpoint": "/api/v1/order/{orderNumber}",
  "noCache": true
}
```

In your template (`order-confirmation.tpl`):
```html
<h1>Order Confirmed!</h1>
<p>Order Number: {{ .Order.OrderNumber }}</p>
<p>Customer: {{ .Order.CustomerName }}</p>
<p>Email: {{ .Order.CustomerEmail }}</p>

<h2>Order Items</h2>
{{ range .Order.Items }}
  <div>
    <p>{{ .ProductName }} {{ if .VariantTitle }}({{ .VariantTitle }}){{ end }}</p>
    <p>Quantity: {{ .Quantity }} × ${{ .Price }} = ${{ .Total }}</p>
  </div>
{{ end }}

<h3>Subtotal: ${{ .Order.Subtotal }}</h3>
<h3>Tax: ${{ .Order.Tax }}</h3>
<h3>Shipping: ${{ .Order.ShippingCost }}</h3>
<h2>Total: ${{ .Order.Total }}</h2>
```

## Session Management

The cart uses cookie-based sessions automatically:
- Cookie name: `stencil_cart_id`
- Expires: 7 days
- HttpOnly: true
- Sessions are created automatically when adding first item to cart
- Cart data is stored in the database, cookie only stores the session ID

## Data Structures

### Product
```go
{
  ID: 1,
  Name: "Cool T-Shirt",
  Slug: "cool-t-shirt",
  Description: "A really cool shirt",
  Price: 29.99,
  CompareAtPrice: 39.99,
  SKU: "SHIRT-001",
  InventoryQuantity: 50,
  Status: "published",
  Featured: true,
  Images: [...],
  Variants: [...],
  Collections: [...]
}
```

### Cart
```go
{
  ID: "session-id-here",
  Items: [
    {
      ID: 1,
      ProductID: 123,
      VariantID: 456,
      Product: {...},
      Variant: {...},
      Quantity: 2,
      Price: 29.99,
      Total: 59.98
    }
  ],
  Subtotal: 59.98
}
```

### Order
```go
{
  ID: 1,
  OrderNumber: "ORD-1234567890",
  CustomerEmail: "user@example.com",
  CustomerName: "John Doe",
  Subtotal: 59.98,
  Tax: 4.80,
  ShippingCost: 10.00,
  Total: 74.78,
  PaymentStatus: "pending",
  Items: [...]
}
```

## Next Steps

1. **Start the server** - E-commerce tables will be created automatically in your website's database
2. **Populate your products** - Insert products, collections, variants, and images into your database
3. **Create your templates** - Build your store, product, cart, and checkout pages
4. **Add JavaScript** - Use the API endpoints to handle cart operations
5. **(Optional) Add payment processing** - Integrate Stripe or another payment gateway

## Payment Integration (Future)

To add payment processing:

1. Add Stripe dependency: `go get github.com/stripe/stripe-go/v76`
2. Create a payment handler that:
   - Creates a Stripe Payment Intent
   - Returns client_secret to frontend
   - Handles webhook for payment confirmation
   - Updates order payment_status

This is not implemented yet but the order structure supports it via the `stripe_payment_intent_id` field.

## Notes

- Tax rate is currently hardcoded to 8% in `CreateOrder` - make this configurable
- Shipping cost is currently flat $10 - make this configurable
- No inventory tracking on purchase - add this if needed
- Cart sessions expire after 7 days - clean up with cron job
- All prices are in USD - add currency support if needed
