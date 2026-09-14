package payment

import (
	"context"
	"sovera-core-api/internal/payment/faspay"
)

type FaspayAdapter struct {
	client *faspay.Client
}

func NewFaspayAdapter(merchantID, merchantKey string, isProduction bool) *FaspayAdapter {
	return &FaspayAdapter{
		client: faspay.NewClient(merchantID, merchantKey, isProduction),
	}
}

func (a *FaspayAdapter) ProviderName() string {
	return "faspay"
}

func (a *FaspayAdapter) IsConfigured() bool {
	return a.client.IsConfigured()
}

func (a *FaspayAdapter) CreateCheckout(ctx context.Context, payload *CheckoutPayload) (*CheckoutResult, error) {
	resp, err := a.client.CreateCheckout(ctx, payload.OrderID, payload.GrossAmount, payload.ItemName, payload.CustomerName)
	if err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Provider:    "faspay",
		Token:       resp.PaymentToken,
		RedirectURL: resp.PaymentURL,
	}, nil
}

func (a *FaspayAdapter) VerifyWebhookSignature(orderID, statusCode, grossAmount, signature string) bool {
	return a.client.VerifyWebhookSignature(orderID, statusCode, grossAmount, signature)
}

func (a *FaspayAdapter) GetClient() *faspay.Client {
	return a.client
}
