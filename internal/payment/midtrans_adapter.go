package payment

import (
	"context"
	"sovera-core-api/internal/payment/midtrans"
)

type MidtransAdapter struct {
	client *midtrans.Client
}

func NewMidtransAdapter(serverKey, clientKey string, isProduction bool) *MidtransAdapter {
	return &MidtransAdapter{
		client: midtrans.NewClient(serverKey, clientKey, isProduction),
	}
}

func (a *MidtransAdapter) ProviderName() string {
	return "midtrans"
}

func (a *MidtransAdapter) IsConfigured() bool {
	return a.client.IsConfigured()
}

func (a *MidtransAdapter) CreateCheckout(ctx context.Context, payload *CheckoutPayload) (*CheckoutResult, error) {
	enabledPayments := []string{"gopay", "qris", "bank_transfer", "credit_card", "shopeepay"}
	snapReq := &midtrans.SnapRequest{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:     payload.OrderID,
			GrossAmount: payload.GrossAmount,
		},
		ItemDetails: []midtrans.ItemDetail{
			{
				ID:       payload.ItemCode,
				Price:    payload.GrossAmount,
				Quantity: 1,
				Name:     payload.ItemName,
			},
		},
		EnabledPayments: enabledPayments,
	}

	snapResp, err := a.client.CreateSnapTransaction(ctx, snapReq)
	if err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Provider:        "midtrans",
		Token:           snapResp.Token,
		RedirectURL:     snapResp.RedirectURL,
		EnabledPayments: enabledPayments,
	}, nil
}

func (a *MidtransAdapter) VerifyWebhookSignature(orderID, statusCode, grossAmount, signature string) bool {
	n := &midtrans.WebhookNotification{
		OrderID:      orderID,
		StatusCode:   statusCode,
		GrossAmount:  grossAmount,
		SignatureKey: signature,
	}
	return a.client.VerifySignature(n)
}

func (a *MidtransAdapter) GetClient() *midtrans.Client {
	return a.client
}
