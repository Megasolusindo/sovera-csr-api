package faspay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	sandboxBaseURL = "https://sandbox.faspay.co.id/v1.0"
	prodBaseURL    = "https://api.faspay.co.id/v1.0"
)

type Client struct {
	merchantID   string
	merchantKey  string
	isProduction bool
	httpClient   *http.Client
}

func NewClient(merchantID, merchantKey string, isProduction bool) *Client {
	return &Client{
		merchantID:   merchantID,
		merchantKey:  merchantKey,
		isProduction: isProduction,
		httpClient: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (c *Client) ProviderName() string {
	return "faspay"
}

func (c *Client) IsConfigured() bool {
	return c.merchantID != "" && c.merchantKey != ""
}

func (c *Client) baseURL() string {
	if c.isProduction {
		return prodBaseURL
	}
	return sandboxBaseURL
}

// SNAP Amount Structure
type SNAPAmount struct {
	Value    string `json:"value"`    // e.g. "150000.00"
	Currency string `json:"currency"` // "IDR"
}

// 1. SNAP OAuth 2.0 B2B Access Token Request & Response
type SNAPAccessTokenRequest struct {
	GrantType string `json:"grantType"` // "client_credentials"
}

type SNAPAccessTokenResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccessToken     string `json:"accessToken"`
	TokenType       string `json:"tokenType"`
	ExpiresIn       string `json:"expiresIn"`
}

// 2. SNAP Virtual Account Creation Request & Response (POST /v1.0/transfer-va/create-va)
type SNAPVACreateRequest struct {
	PartnerServiceId     string                 `json:"partnerServiceId"`
	CustomerNo           string                 `json:"customerNo"`
	VirtualAccountNo     string                 `json:"virtualAccountNo"`
	VirtualAccountName   string                 `json:"virtualAccountName"`
	TrxId                string                 `json:"trxId"`
	TotalAmount          SNAPAmount             `json:"totalAmount"`
	VirtualAccountTrxType string                `json:"virtualAccountTrxType,omitempty"` // 1: Single, 2: Multiple
	ExpiredDate          string                 `json:"expiredDate,omitempty"`          // ISO 8601
	AdditionalInfo       map[string]interface{} `json:"additionalInfo,omitempty"`
}

type SNAPVADetails struct {
	PartnerServiceId    string     `json:"partnerServiceId"`
	CustomerNo          string     `json:"customerNo"`
	VirtualAccountNo    string     `json:"virtualAccountNo"`
	VirtualAccountName  string     `json:"virtualAccountName"`
	VirtualAccountToken string     `json:"virtualAccountToken,omitempty"`
	PaymentUrl          string     `json:"paymentUrl,omitempty"`
	TotalAmount         SNAPAmount `json:"totalAmount"`
}

type SNAPVACreateResponse struct {
	ResponseCode       string        `json:"responseCode"`
	ResponseMessage    string        `json:"responseMessage"`
	VirtualAccountData SNAPVADetails `json:"virtualAccountData"`
}

// 3. SNAP QRIS Generation Request & Response (POST /v1.0/qr/qr-mpm-generate)
type SNAPQRISCreateRequest struct {
	PartnerReferenceNo string                 `json:"partnerReferenceNo"`
	Amount             SNAPAmount             `json:"amount"`
	MerchantId         string                 `json:"merchantId"`
	TerminalId         string                 `json:"terminalId,omitempty"`
	AdditionalInfo     map[string]interface{} `json:"additionalInfo,omitempty"`
}

type SNAPQRISCreateResponse struct {
	ResponseCode       string `json:"responseCode"`
	ResponseMessage    string `json:"responseMessage"`
	PartnerReferenceNo string `json:"partnerReferenceNo"`
	QrContent          string `json:"qrContent,omitempty"`
	QrUrl              string `json:"qrUrl,omitempty"`
	QrToken            string `json:"qrToken,omitempty"`
}

// Legacy / Unified response for SubscriptionService checkout
type FaspayResponse struct {
	StatusCode    string `json:"status_code"`
	StatusMessage string `json:"status_message"`
	PaymentToken  string `json:"payment_token"`
	PaymentURL    string `json:"payment_url"`
}

// GenerateSNAPB2BSignature generates HMAC-SHA512 signature for OAuth B2B
func (c *Client) GenerateSNAPB2BSignature(timestamp string) string {
	raw := fmt.Sprintf("%s|%s", c.merchantID, timestamp)
	h := hmac.New(sha512.New, []byte(c.merchantKey))
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSNAPServiceSignature generates HMAC-SHA512 signature for SNAP service API calls
// Signature = HMAC-SHA512(client_secret, HTTPMethod + ":" + EndpointUrl + ":" + AccessToken + ":" + SHA256(Body) + ":" + Timestamp)
func (c *Client) GenerateSNAPServiceSignature(httpMethod, endpointPath, accessToken string, bodyBytes []byte, timestamp string) string {
	sha256Hash := sha256.Sum256(bodyBytes)
	hexBodyHash := strings.ToLower(hex.EncodeToString(sha256Hash[:]))

	stringToSign := fmt.Sprintf("%s:%s:%s:%s:%s", httpMethod, endpointPath, accessToken, hexBodyHash, timestamp)
	h := hmac.New(sha512.New, []byte(c.merchantKey))
	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}

// GetAccessToken obtains OAuth 2.0 B2B Access Token from Faspay SNAP
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	if !c.IsConfigured() {
		return "MOCK-FASPAY-ACCESS-TOKEN", nil
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	sig := c.GenerateSNAPB2BSignature(timestamp)

	reqPayload := SNAPAccessTokenRequest{GrantType: "client_credentials"}
	bodyBytes, _ := json.Marshal(reqPayload)

	url := c.baseURL() + "/access-token/b2b"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create access token request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-CLIENT-KEY", c.merchantID)
	httpReq.Header.Set("X-SIGNATURE", sig)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("faspay access token http request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read access token response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("faspay access token error (status %d): %s", res.StatusCode, string(respBody))
	}

	var tokenResp SNAPAccessTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode access token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// CreateSNAPVirtualAccount calls POST /v1.0/transfer-va/create-va
func (c *Client) CreateSNAPVirtualAccount(ctx context.Context, req *SNAPVACreateRequest) (*SNAPVACreateResponse, error) {
	if !c.IsConfigured() {
		mockToken := fmt.Sprintf("MOCK-FASPAY-VA-TOKEN-%d", time.Now().Unix())
		return &SNAPVACreateResponse{
			ResponseCode:    "2002700",
			ResponseMessage: "Successful",
			VirtualAccountData: SNAPVADetails{
				PartnerServiceId:   req.PartnerServiceId,
				VirtualAccountNo:   req.VirtualAccountNo,
				VirtualAccountName: req.VirtualAccountName,
				VirtualAccountToken: mockToken,
				PaymentUrl:         fmt.Sprintf("https://sandbox.faspay.co.id/pay/va/%s", mockToken),
				TotalAmount:        req.TotalAmount,
			},
		}, nil
	}

	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize snap va request: %w", err)
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snap va request: %w", err)
	}

	endpointPath := "/v1.0/transfer-va/create-va"
	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	sig := c.GenerateSNAPServiceSignature(http.MethodPost, endpointPath, accessToken, bodyBytes, timestamp)

	url := c.baseURL() + "/transfer-va/create-va"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create snap va request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", sig)
	httpReq.Header.Set("X-PARTNER-ID", c.merchantID)
	httpReq.Header.Set("X-EXTERNAL-ID", req.TrxId)
	httpReq.Header.Set("CHANNEL-ID", "95051")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("snap va http request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read snap va response: %w", err)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("faspay snap va error (status %d): %s", res.StatusCode, string(respBody))
	}

	var resp SNAPVACreateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode snap va response: %w", err)
	}

	return &resp, nil
}

// CreateSNAPQRIS calls POST /v1.0/qr/qr-mpm-generate
func (c *Client) CreateSNAPQRIS(ctx context.Context, req *SNAPQRISCreateRequest) (*SNAPQRISCreateResponse, error) {
	if !c.IsConfigured() {
		mockToken := fmt.Sprintf("MOCK-FASPAY-QRIS-TOKEN-%d", time.Now().Unix())
		return &SNAPQRISCreateResponse{
			ResponseCode:       "2004700",
			ResponseMessage:    "Successful",
			PartnerReferenceNo: req.PartnerReferenceNo,
			QrToken:            mockToken,
			QrUrl:              fmt.Sprintf("https://sandbox.faspay.co.id/pay/qr/%s", mockToken),
		}, nil
	}

	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize snap qris request: %w", err)
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snap qris request: %w", err)
	}

	endpointPath := "/v1.0/qr/qr-mpm-generate"
	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	sig := c.GenerateSNAPServiceSignature(http.MethodPost, endpointPath, accessToken, bodyBytes, timestamp)

	url := c.baseURL() + "/qr/qr-mpm-generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create snap qris request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", sig)
	httpReq.Header.Set("X-PARTNER-ID", c.merchantID)
	httpReq.Header.Set("X-EXTERNAL-ID", req.PartnerReferenceNo)
	httpReq.Header.Set("CHANNEL-ID", "95052")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("snap qris http request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read snap qris response: %w", err)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("faspay snap qris error (status %d): %s", res.StatusCode, string(respBody))
	}

	var resp SNAPQRISCreateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode snap qris response: %w", err)
	}

	return &resp, nil
}

// CreateCheckout satisfies PaymentGateway interface by issuing a Faspay SNAP Virtual Account checkout
func (c *Client) CreateCheckout(ctx context.Context, orderID string, amount int64, itemName, customerName string) (*FaspayResponse, error) {
	if customerName == "" {
		customerName = "Sovera Subscriber"
	}

	vaReq := &SNAPVACreateRequest{
		PartnerServiceId:   c.merchantID,
		CustomerNo:         "08123456789",
		VirtualAccountNo:   fmt.Sprintf("%s%d", c.merchantID, time.Now().Unix()%100000000),
		VirtualAccountName: customerName,
		TrxId:              orderID,
		TotalAmount: SNAPAmount{
			Value:    fmt.Sprintf("%.2f", float64(amount)),
			Currency: "IDR",
		},
		VirtualAccountTrxType: "1",
		ExpiredDate:          time.Now().Add(24 * time.Hour).Format("2006-01-02T15:04:05-07:00"),
	}

	vaResp, err := c.CreateSNAPVirtualAccount(ctx, vaReq)
	if err != nil {
		// Fallback to QRIS if VA creation fails or is not configured
		qrisReq := &SNAPQRISCreateRequest{
			PartnerReferenceNo: orderID,
			Amount: SNAPAmount{
				Value:    fmt.Sprintf("%.2f", float64(amount)),
				Currency: "IDR",
			},
			MerchantId: c.merchantID,
		}
		qrisResp, qrisErr := c.CreateSNAPQRIS(ctx, qrisReq)
		if qrisErr != nil {
			return nil, fmt.Errorf("failed both snap va (%v) and qris (%v)", err, qrisErr)
		}

		return &FaspayResponse{
			StatusCode:    qrisResp.ResponseCode,
			StatusMessage: qrisResp.ResponseMessage,
			PaymentToken:  qrisResp.QrToken,
			PaymentURL:    qrisResp.QrUrl,
		}, nil
	}

	paymentUrl := vaResp.VirtualAccountData.PaymentUrl
	if paymentUrl == "" {
		paymentUrl = fmt.Sprintf("https://sandbox.faspay.co.id/pay/va/%s", vaResp.VirtualAccountData.VirtualAccountToken)
	}

	return &FaspayResponse{
		StatusCode:    vaResp.ResponseCode,
		StatusMessage: vaResp.ResponseMessage,
		PaymentToken:  vaResp.VirtualAccountData.VirtualAccountToken,
		PaymentURL:    paymentUrl,
	}, nil
}

func (c *Client) VerifyWebhookSignature(orderID, statusCode, grossAmount, signature string) bool {
	if c.merchantKey == "" {
		return false
	}
	raw := fmt.Sprintf("%s%s%s%s", c.merchantID, orderID, grossAmount, c.merchantKey)
	hash := sha256.Sum256([]byte(raw))
	expected := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}
