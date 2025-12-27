package twilio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL = "https://api.twilio.com/2010-04-01"
)

// Client represents a Twilio API client
type Client struct {
	AccountSID string
	AuthToken  string
	FromPhone  string
	HTTPClient *http.Client
}

// NewClient creates a new Twilio client
func NewClient(accountSID, authToken, fromPhone string) *Client {
	return &Client{
		AccountSID: accountSID,
		AuthToken:  authToken,
		FromPhone:  fromPhone,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerificationResponse represents a verification code response
type VerificationResponse struct {
	Code      string `json:"code"`
	ExpiresAt time.Time
}

// SMSResponse represents the response from sending an SMS
type SMSResponse struct {
	SID         string `json:"sid"`
	DateCreated string `json:"date_created"`
	DateUpdated string `json:"date_updated"`
	DateSent    string `json:"date_sent"`
	AccountSID  string `json:"account_sid"`
	To          string `json:"to"`
	From        string `json:"from"`
	Body        string `json:"body"`
	Status      string `json:"status"`
	Direction   string `json:"direction"`
	Price       string `json:"price"`
	PriceUnit   string `json:"price_unit"`
	ErrorCode   string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// doRequest is a helper method to make HTTP requests to Twilio API
func (c *Client) doRequest(method, endpoint string, formData url.Values, result interface{}) error {
	req, err := http.NewRequest(method, baseURL+endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.AccountSID, c.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("twilio API error (status %d): %s", resp.StatusCode, errorResp.Message)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// SendSMS sends an SMS message to a phone number
func (c *Client) SendSMS(to, message string) (*SMSResponse, error) {
	formData := url.Values{}
	formData.Set("To", to)
	formData.Set("From", c.FromPhone)
	formData.Set("Body", message)

	endpoint := fmt.Sprintf("/Accounts/%s/Messages.json", c.AccountSID)

	var smsResp SMSResponse
	if err := c.doRequest("POST", endpoint, formData, &smsResp); err != nil {
		return nil, err
	}

	return &smsResp, nil
}

// SendVerificationCode sends a verification code to a phone number
func (c *Client) SendVerificationCode(to, code string) error {
	message := fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in 10 minutes.", code)

	_, err := c.SendSMS(to, message)
	return err
}

// SendBulkSMS sends an SMS to multiple phone numbers
// Returns a map of phone numbers to their status (success/error message)
func (c *Client) SendBulkSMS(phoneNumbers []string, message string) (map[string]string, error) {
	results := make(map[string]string)

	for _, phone := range phoneNumbers {
		resp, err := c.SendSMS(phone, message)
		if err != nil {
			results[phone] = fmt.Sprintf("Error: %v", err)
		} else {
			results[phone] = fmt.Sprintf("Success: %s", resp.Status)
		}
	}

	return results, nil
}

// FormatPhoneNumber formats a phone number for Twilio (E.164 format)
// Example: countryCode="+1", phone="4155551234" -> "+14155551234"
func FormatPhoneNumber(countryCode, phone string) string {
	// Remove any non-numeric characters from phone
	cleanPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Ensure country code starts with +
	if !strings.HasPrefix(countryCode, "+") {
		countryCode = "+" + countryCode
	}

	return countryCode + cleanPhone
}
