// Package midtrans provides HTTP client integration with Midtrans Payment Gateway (Snap API & Webhooks).
package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	snapSandboxBase = "https://app.sandbox.midtrans.com/snap/v1"
	snapProdBase    = "https://app.midtrans.com/snap/v1"
	apiSandboxBase  = "https://api.sandbox.midtrans.com"
	apiProdBase     = "https://api.midtrans.com"
)

type Client struct {
	serverKey    string
	clientKey    string
	isProduction bool
	httpClient   *http.Client
}

func NewClient(serverKey, clientKey string, isProduction bool) *Client {
	return &Client{
		serverKey:    serverKey,
		clientKey:    clientKey,
		isProduction: isProduction,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func NewAdapter(serverKey, clientKey string, isProduction bool) *Client {
	return NewClient(serverKey, clientKey, isProduction)
}

func (c *Client) ProviderName() string {
	return "midtrans"
}

func (c *Client) IsConfigured() bool {
	return c.serverKey != ""
}

func (c *Client) snapBase() string {
	if c.isProduction {
		return snapProdBase
	}
	return snapSandboxBase
}

func (c *Client) apiBase() string {
	if c.isProduction {
		return apiProdBase
	}
	return apiSandboxBase
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.serverKey+":"))
}

// TransactionDetails defines order_id and gross_amount (in IDR).
type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type CustomerDetails struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

type ItemDetail struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type SnapRequest struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`
	CustomerDetails    *CustomerDetails   `json:"customer_details,omitempty"`
	ItemDetails        []ItemDetail       `json:"item_details,omitempty"`
	EnabledPayments    []string           `json:"enabled_payments,omitempty"`
}

type SnapResponse struct {
	Token       string   `json:"token"`
	RedirectURL string   `json:"redirect_url"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

// CreateSnapTransaction requests a new Snap payment token and redirect URL.
func (c *Client) CreateSnapTransaction(ctx context.Context, req *SnapRequest) (*SnapResponse, error) {
	if !c.IsConfigured() {
		// Mock response for development mode when MIDTRANS_SERVER_KEY is not configured
		mockToken := fmt.Sprintf("MOCK-SNAP-TOKEN-%d", time.Now().Unix())
		return &SnapResponse{
			Token:       mockToken,
			RedirectURL: fmt.Sprintf("https://app.sandbox.midtrans.com/snap/v2/vtweb/%s", mockToken),
		}, nil
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snap request: %w", err)
	}

	url := c.snapBase() + "/transactions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", c.authHeader())

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request to midtrans failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("midtrans error (status %d): %s", res.StatusCode, string(respBody))
	}

	var snapResp SnapResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return nil, fmt.Errorf("failed to decode snap response: %w", err)
	}

	return &snapResp, nil
}

// WebhookNotification represents payload pushed by Midtrans to the notification endpoint.
type WebhookNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status,omitempty"`
	Currency          string `json:"currency,omitempty"`
}

// VerifySignature checks SHA512(order_id + status_code + gross_amount + server_key).
func (c *Client) VerifySignature(n *WebhookNotification) bool {
	if c.serverKey == "" {
		return false
	}
	raw := n.OrderID + n.StatusCode + n.GrossAmount + c.serverKey
	sum := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(n.SignatureKey)) == 1
}

func (n *WebhookNotification) IsSuccess() bool {
	switch n.TransactionStatus {
	case "settlement":
		return true
	case "capture":
		return n.FraudStatus == "accept" || n.FraudStatus == ""
	}
	return false
}

func (n *WebhookNotification) IsFailed() bool {
	switch n.TransactionStatus {
	case "deny", "cancel", "expire":
		return true
	}
	return false
}
