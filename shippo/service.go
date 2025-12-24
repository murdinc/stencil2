package shippo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.goshippo.com"
)

// Client represents a Shippo API client
type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Shippo client
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Address represents a shipping address
type Address struct {
	Name    string `json:"name"`
	Street1 string `json:"street1"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

// Parcel represents package dimensions and weight
type Parcel struct {
	Length       string `json:"length"`        // inches (as string)
	Width        string `json:"width"`         // inches (as string)
	Height       string `json:"height"`        // inches (as string)
	DistanceUnit string `json:"distance_unit"` // "in" or "cm"
	Weight       string `json:"weight"`        // pounds or kg (as string)
	MassUnit     string `json:"mass_unit"`     // "lb" or "kg"
}

// Shipment represents a shipment request
type Shipment struct {
	AddressFrom Address `json:"address_from"`
	AddressTo   Address `json:"address_to"`
	Parcels     []Parcel `json:"parcels"`
}

// Rate represents a shipping rate option
type Rate struct {
	ObjectID         string  `json:"object_id"`
	Amount           string  `json:"amount"`
	Currency         string  `json:"currency"`
	Provider         string  `json:"provider"`
	ProviderImage75  string  `json:"provider_image_75"`
	ProviderImage200 string  `json:"provider_image_200"`
	ServiceLevel     struct {
		Name  string `json:"name"`
		Token string `json:"token"`
	} `json:"servicelevel"`
	EstimatedDays int    `json:"estimated_days"`
	DurationTerms string `json:"duration_terms"`
}

// ShipmentResponse represents the response from creating a shipment
type ShipmentResponse struct {
	ObjectID    string `json:"object_id"`
	Status      string `json:"status"`
	Rates       []Rate `json:"rates"`
	Messages    []struct {
		Source string `json:"source"`
		Text   string `json:"text"`
	} `json:"messages"`
}

// Transaction represents a label purchase
type Transaction struct {
	ObjectID       string `json:"object_id"`
	Status         string `json:"status"`
	Rate           string `json:"rate"`
	TrackingNumber string `json:"tracking_number"`
	LabelURL       string `json:"label_url"`
	CommercialInvoiceURL string `json:"commercial_invoice_url,omitempty"`
	Metadata       string `json:"metadata,omitempty"`
}

// AddressValidation represents the validation response for an address
type AddressValidation struct {
	IsValid      bool   `json:"is_valid"`
	IsComplete   bool   `json:"is_complete"`
	IsResidential *bool `json:"is_residential"`
	Messages     []struct {
		Source string `json:"source"`
		Type   string `json:"type"`
		Code   string `json:"code"`
		Text   string `json:"text"`
	} `json:"messages"`
}

// AddressResponse represents a validated address
type AddressResponse struct {
	ObjectID    string             `json:"object_id"`
	Name        string             `json:"name"`
	Street1     string             `json:"street1"`
	Street2     string             `json:"street2"`
	City        string             `json:"city"`
	State       string             `json:"state"`
	Zip         string             `json:"zip"`
	Country     string             `json:"country"`
	Phone       string             `json:"phone"`
	Email       string             `json:"email"`
	IsComplete  bool               `json:"is_complete"`
	Validation  AddressValidation  `json:"validation_results"`
}

// TrackingStatus represents the current tracking status
type TrackingStatus struct {
	Status        string `json:"status"`
	StatusDetails string `json:"status_details"`
	StatusDate    string `json:"status_date"`
	Location      struct {
		City    string `json:"city"`
		State   string `json:"state"`
		Zip     string `json:"zip"`
		Country string `json:"country"`
	} `json:"location"`
}

// TrackingResponse represents package tracking information
type TrackingResponse struct {
	Carrier          string           `json:"carrier"`
	TrackingNumber   string           `json:"tracking_number"`
	TrackingStatus   TrackingStatus   `json:"tracking_status"`
	TrackingHistory  []TrackingStatus `json:"tracking_history"`
	ETA              string           `json:"eta"`
	OriginalETA      string           `json:"original_eta"`
	ServiceLevel     struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	} `json:"servicelevel"`
	Metadata         string           `json:"metadata"`
}

// GetRates gets shipping rates for a shipment
func (c *Client) GetRates(from, to Address, parcel Parcel) (*ShipmentResponse, error) {
	shipment := Shipment{
		AddressFrom: from,
		AddressTo:   to,
		Parcels:     []Parcel{parcel},
	}

	body, err := json.Marshal(shipment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shipment: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/shipments/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "ShippoToken "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shippo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var shipmentResp ShipmentResponse
	if err := json.Unmarshal(respBody, &shipmentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &shipmentResp, nil
}

// PurchaseLabel purchases a shipping label using a rate ID
func (c *Client) PurchaseLabel(rateID string, labelFileType string) (*Transaction, error) {
	if labelFileType == "" {
		labelFileType = "PDF"
	}

	reqBody := map[string]interface{}{
		"rate":            rateID,
		"label_file_type": labelFileType,
		"async":           false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/transactions/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "ShippoToken "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shippo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var transaction Transaction
	if err := json.Unmarshal(respBody, &transaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if transaction.Status != "SUCCESS" {
		return nil, fmt.Errorf("label purchase failed with status: %s", transaction.Status)
	}

	return &transaction, nil
}

// ValidateAddress validates a shipping address
func (c *Client) ValidateAddress(addr Address) (*AddressResponse, error) {
	// Build request body with validate flag
	reqBody := map[string]interface{}{
		"name":     addr.Name,
		"street1":  addr.Street1,
		"street2":  addr.Street2,
		"city":     addr.City,
		"state":    addr.State,
		"zip":      addr.Zip,
		"country":  addr.Country,
		"email":    addr.Email,
		"phone":    addr.Phone,
		"validate": true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal address: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/addresses/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "ShippoToken "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shippo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var addrResp AddressResponse
	if err := json.Unmarshal(respBody, &addrResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &addrResp, nil
}

// GetTracking retrieves tracking information for a package
func (c *Client) GetTracking(carrier, trackingNumber string) (*TrackingResponse, error) {
	// Register tracking with POST first (creates tracking object)
	reqBody := map[string]interface{}{
		"carrier":         carrier,
		"tracking_number": trackingNumber,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tracking request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/tracks/", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "ShippoToken "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shippo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var trackingResp TrackingResponse
	if err := json.Unmarshal(respBody, &trackingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &trackingResp, nil
}
