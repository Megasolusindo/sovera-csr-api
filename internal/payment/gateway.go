package payment

import (
	"context"
	"sovera-core-api/internal/config"
)

type CheckoutPayload struct {
	OrderID     string
	GrossAmount int64
	ItemCode    string
	ItemName    string
	BillingCycle string
	CustomerEmail string
	CustomerName  string
}

type CheckoutResult struct {
	Provider        string   `json:"provider"`
	Token           string   `json:"token"`
	RedirectURL     string   `json:"redirect_url"`
	EnabledPayments []string `json:"enabled_payments,omitempty"`
}

type PaymentGateway interface {
	ProviderName() string
	IsConfigured() bool
	CreateCheckout(ctx context.Context, payload *CheckoutPayload) (*CheckoutResult, error)
	VerifyWebhookSignature(orderID, statusCode, grossAmount, signature string) bool
}

// GatewayFactory returns the active PaymentGateway provider based on config.
func NewGatewayFactory(cfg *config.Config) PaymentGateway {
	switch cfg.ActivePaymentGateway {
	case "faspay":
		return NewFaspayAdapter(cfg.FaspayMerchantID, cfg.FaspayMerchantKey, cfg.FaspayIsProduction)
	case "midtrans":
		fallthrough
	default:
		return NewMidtransAdapter(cfg.MidtransServerKey, cfg.MidtransClientKey, cfg.MidtransIsProduction)
	}
}
