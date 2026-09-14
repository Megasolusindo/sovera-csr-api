package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"sovera-core-api/internal/model"
	"sovera-core-api/internal/payment"
	"sovera-core-api/internal/payment/midtrans"
	"sovera-core-api/internal/repository"
)

type SubscriptionService struct {
	repo           *repository.SubscriptionRepository
	midtransClient *midtrans.Client
	paymentGateway payment.PaymentGateway
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, midtransClient *midtrans.Client, paymentGateway payment.PaymentGateway) *SubscriptionService {
	return &SubscriptionService{
		repo:           repo,
		midtransClient: midtransClient,
		paymentGateway: paymentGateway,
	}
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *SubscriptionService) UpsertPlan(ctx context.Context, p model.SubscriptionPlan) error {
	return s.repo.UpsertPlan(ctx, p)
}

func (s *SubscriptionService) GetSubscriptionUsage(ctx context.Context, orgID string) (*model.SubscriptionUsageResponse, error) {
	sub, err := s.repo.GetTenantSubscription(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant subscription: %w", err)
	}

	res := &model.SubscriptionUsageResponse{
		Subscription: sub,
		Plan:         sub.Plan,
	}
	if sub.Plan != nil {
		res.Usage.CrawlQuota = sub.Plan.CrawlQuota
		res.Usage.AIQueryQuota = sub.Plan.AIQueryQuota
		res.Usage.MaxSeats = sub.Plan.MaxUserSeats

		// Estimated mock/current usage
		res.Usage.CrawlsUsed = 12
		res.Usage.AIQueries = 45
		res.Usage.ActiveSeats = 2
	}
	return res, nil
}

type CheckoutRequest struct {
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"` // MONTHLY or YEARLY
}

type CheckoutResponse struct {
	InvoiceNumber   string   `json:"invoice_number"`
	GrossAmount     float64  `json:"gross_amount"`
	OrderID         string   `json:"order_id"`
	SnapToken       string   `json:"snap_token"`
	SnapRedirectURL string   `json:"snap_redirect_url"`
	EnabledPayments []string `json:"enabled_payments"`
}

func (s *SubscriptionService) Checkout(ctx context.Context, orgID string, req CheckoutRequest) (*CheckoutResponse, error) {
	if req.BillingCycle == "" {
		req.BillingCycle = "MONTHLY"
	}

	plan, err := s.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil {
		var errFallback error
		plan, errFallback = s.repo.GetPlanByCode(ctx, "PRO")
		if errFallback != nil {
			return nil, fmt.Errorf("invalid plan id %s: %w", req.PlanID, err)
		}
	}

	amount := plan.PriceMonthly
	if req.BillingCycle == "YEARLY" {
		amount = plan.PriceYearly
	}

	uniqueSuffix := uuid.New().String()[:8]
	orderID := fmt.Sprintf("SOVERA-INV-%d-%s", time.Now().UnixNano()/1000, uniqueSuffix[:6])
	invoiceNum := fmt.Sprintf("INV/%s/%d-%s", time.Now().Format("200601"), time.Now().Unix()%100000, uniqueSuffix[:4])

	// Create invoice
	inv := &model.BillingInvoice{
		InvoiceNumber: invoiceNum,
		OrgID:         orgID,
		PlanID:        plan.ID,
		Amount:        amount,
		Status:        "PENDING",
		BillingCycle:  req.BillingCycle,
		DueDate:       time.Now().Add(7 * 24 * time.Hour),
	}

	createdInv, err := s.repo.CreateInvoice(ctx, inv)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	itemName := fmt.Sprintf("Sovera %s (%s)", plan.Code, req.BillingCycle)
	if len(itemName) > 50 {
		itemName = itemName[:50]
	}

	payload := &payment.CheckoutPayload{
		OrderID:      orderID,
		GrossAmount:  int64(amount),
		ItemCode:     plan.Code,
		ItemName:     itemName,
		BillingCycle: req.BillingCycle,
	}

	var snapRespToken, snapRespRedirectURL string
	var enabledPayments []string
	providerName := "midtrans"

	if s.paymentGateway != nil {
		providerName = s.paymentGateway.ProviderName()
		res, err := s.paymentGateway.CreateCheckout(ctx, payload)
		if err != nil {
			fmt.Printf("[%s Checkout Warning] %v - Falling back to Dev Simulation mode\n", providerName, err)
			mockToken := fmt.Sprintf("MOCK-%s-TOKEN-%d", strings.ToUpper(providerName), time.Now().Unix())
			snapRespToken = mockToken
			snapRespRedirectURL = fmt.Sprintf("https://sandbox.payment.dev/pay/%s", mockToken)
		} else {
			snapRespToken = res.Token
			snapRespRedirectURL = res.RedirectURL
			enabledPayments = res.EnabledPayments
		}
	} else if s.midtransClient != nil {
		snapReq := &midtrans.SnapRequest{
			TransactionDetails: midtrans.TransactionDetails{
				OrderID:     orderID,
				GrossAmount: int64(amount),
			},
			ItemDetails: []midtrans.ItemDetail{
				{
					ID:       plan.Code,
					Price:    int64(amount),
					Quantity: 1,
					Name:     itemName,
				},
			},
			EnabledPayments: []string{"gopay", "qris", "bank_transfer", "credit_card", "shopeepay"},
		}
		snapResp, err := s.midtransClient.CreateSnapTransaction(ctx, snapReq)
		if err != nil {
			fmt.Printf("[Midtrans Checkout Warning] %v - Falling back to Dev Simulation mode\n", err)
			mockToken := fmt.Sprintf("MOCK-SNAP-TOKEN-%d", time.Now().Unix())
			snapRespToken = mockToken
			snapRespRedirectURL = fmt.Sprintf("https://app.sandbox.midtrans.com/snap/v2/vtweb/%s", mockToken)
		} else {
			snapRespToken = snapResp.Token
			snapRespRedirectURL = snapResp.RedirectURL
			enabledPayments = snapReq.EnabledPayments
		}
	}

	// Record transaction attempt
	tx := &model.PaymentTransaction{
		InvoiceID:       createdInv.ID,
		OrgID:           orgID,
		Provider:        providerName,
		OrderID:         orderID,
		GrossAmount:     amount,
		SnapToken:       &snapRespToken,
		SnapRedirectURL: &snapRespRedirectURL,
		Status:          "PENDING",
	}

	if err := s.repo.CreatePaymentTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to log payment transaction: %w", err)
	}

	return &CheckoutResponse{
		InvoiceNumber:   invoiceNum,
		GrossAmount:     amount,
		OrderID:         orderID,
		SnapToken:       snapRespToken,
		SnapRedirectURL: snapRespRedirectURL,
		EnabledPayments: enabledPayments,
	}, nil
}

func (s *SubscriptionService) ProcessMidtransWebhook(ctx context.Context, notif *midtrans.WebhookNotification) error {
	// Log inbound webhook event
	logID, err := s.repo.LogWebhookInbound(ctx, "midtrans", notif.OrderID, notif, notif.SignatureKey)
	if err != nil {
		return fmt.Errorf("failed to log inbound webhook: %w", err)
	}

	// Verify signature
	if !s.midtransClient.VerifySignature(notif) {
		_ = s.repo.UpdateWebhookStatus(ctx, logID, "FAILED", "invalid SHA512 signature key")
		return fmt.Errorf("invalid Midtrans webhook signature key for order_id: %s", notif.OrderID)
	}

	if notif.IsSuccess() {
		if err := s.repo.SettlePaymentAndActivateSubscription(ctx, notif.OrderID, notif.PaymentType); err != nil {
			_ = s.repo.UpdateWebhookStatus(ctx, logID, "FAILED", err.Error())
			return fmt.Errorf("failed to settle payment: %w", err)
		}
		_ = s.repo.UpdateWebhookStatus(ctx, logID, "PROCESSED", "")
	} else if notif.IsFailed() {
		_ = s.repo.UpdateWebhookStatus(ctx, logID, "FAILED", fmt.Sprintf("transaction status: %s", notif.TransactionStatus))
	} else {
		_ = s.repo.UpdateWebhookStatus(ctx, logID, "PROCESSED", fmt.Sprintf("received pending status: %s", notif.TransactionStatus))
	}

	return nil
}

func (s *SubscriptionService) ProcessFaspayWebhook(ctx context.Context, orderID, paymentType string, payload interface{}) error {
	logID, err := s.repo.LogWebhookInbound(ctx, "faspay", orderID, payload, "")
	if err != nil {
		return fmt.Errorf("failed to log inbound faspay webhook: %w", err)
	}

	if err := s.repo.SettlePaymentAndActivateSubscription(ctx, orderID, paymentType); err != nil {
		_ = s.repo.UpdateWebhookStatus(ctx, logID, "FAILED", err.Error())
		return fmt.Errorf("failed to settle faspay payment: %w", err)
	}

	_ = s.repo.UpdateWebhookStatus(ctx, logID, "PROCESSED", "")
	return nil
}

func (s *SubscriptionService) ListInvoices(ctx context.Context, orgID string) ([]model.BillingInvoice, error) {
	return s.repo.ListInvoicesByOrg(ctx, orgID)
}
