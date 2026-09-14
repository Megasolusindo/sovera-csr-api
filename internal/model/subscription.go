package model

import "time"

type SubscriptionPlan struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"` // FREE_TRIAL, PRO, ENTERPRISE
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	PriceMonthly  float64   `json:"price_monthly"`
	PriceYearly   float64   `json:"price_yearly"`
	CrawlQuota    int       `json:"crawl_quota"`
	AIQueryQuota  int       `json:"ai_query_quota"`
	MaxUserSeats  int       `json:"max_user_seats"`
	TargetPersona string    `json:"target_persona"` // ORGANIZATION, CORPORATE
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TenantSubscription struct {
	ID                     string           `json:"id"`
	OrgID                  string           `json:"org_id"`
	PlanID                 string           `json:"plan_id"`
	Plan                   *SubscriptionPlan `json:"plan,omitempty"`
	Status                 string           `json:"status"` // ACTIVE, EXPIRED, CANCELLED, PAST_DUE
	BillingCycle           string           `json:"billing_cycle"` // MONTHLY, YEARLY
	CurrentPeriodStart     time.Time        `json:"current_period_start"`
	CurrentPeriodEnd       time.Time        `json:"current_period_end"`
	AutoRenew              bool             `json:"auto_renew"`
	MidtransSubscriptionID *string          `json:"midtrans_subscription_id,omitempty"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
}

type BillingInvoice struct {
	ID            string           `json:"id"`
	InvoiceNumber string           `json:"invoice_number"`
	OrgID         string           `json:"org_id"`
	PlanID        string           `json:"plan_id"`
	Plan          *SubscriptionPlan `json:"plan,omitempty"`
	Amount        float64          `json:"amount"`
	Status        string           `json:"status"` // PENDING, PAID, FAILED, EXPIRED
	BillingCycle  string           `json:"billing_cycle"`
	DueDate       time.Time        `json:"due_date"`
	PaidAt        *time.Time       `json:"paid_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type PaymentTransaction struct {
	ID               string    `json:"id"`
	InvoiceID        string    `json:"invoice_id"`
	OrgID            string    `json:"org_id"`
	Provider         string    `json:"provider"`
	OrderID          string    `json:"order_id"`
	GrossAmount      float64   `json:"gross_amount"`
	PaymentType      *string   `json:"payment_type,omitempty"`
	SnapToken        *string   `json:"snap_token,omitempty"`
	SnapRedirectURL  *string   `json:"snap_redirect_url,omitempty"`
	Status           string    `json:"status"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SubscriptionUsageResponse struct {
	Subscription *TenantSubscription `json:"subscription"`
	Plan         *SubscriptionPlan   `json:"plan"`
	Usage        struct {
		CrawlsUsed   int `json:"crawls_used"`
		CrawlQuota   int `json:"crawl_quota"`
		AIQueries    int `json:"ai_queries_used"`
		AIQueryQuota int `json:"ai_query_quota"`
		ActiveSeats  int `json:"active_seats"`
		MaxSeats     int `json:"max_seats"`
	} `json:"usage"`
}
