package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sovera-core-api/internal/model"
)

type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (r *SubscriptionRepository) ListPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	query := `
		SELECT id, code, name, COALESCE(description, ''), price_monthly, price_yearly, 
		       crawl_quota, ai_query_quota, max_user_seats, COALESCE(target_persona, 'ORGANIZATION'), is_active, created_at, updated_at
		FROM subscription_plans
		WHERE is_active = TRUE
		ORDER BY price_monthly ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription plans: %w", err)
	}
	defer rows.Close()

	var plans []model.SubscriptionPlan
	for rows.Next() {
		var p model.SubscriptionPlan
		err := rows.Scan(
			&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly,
			&p.CrawlQuota, &p.AIQueryQuota, &p.MaxUserSeats, &p.TargetPersona, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *SubscriptionRepository) GetPlanByID(ctx context.Context, planID string) (*model.SubscriptionPlan, error) {
	query := `
		SELECT id, code, name, COALESCE(description, ''), price_monthly, price_yearly, 
		       crawl_quota, ai_query_quota, max_user_seats, COALESCE(target_persona, 'ORGANIZATION'), is_active, created_at, updated_at
		FROM subscription_plans
		WHERE id::text = $1 OR code = $1`

	var p model.SubscriptionPlan
	err := r.pool.QueryRow(ctx, query, planID).Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly,
		&p.CrawlQuota, &p.AIQueryQuota, &p.MaxUserSeats, &p.TargetPersona, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SubscriptionRepository) GetPlanByCode(ctx context.Context, code string) (*model.SubscriptionPlan, error) {
	query := `
		SELECT id, code, name, COALESCE(description, ''), price_monthly, price_yearly, 
		       crawl_quota, ai_query_quota, max_user_seats, COALESCE(target_persona, 'ORGANIZATION'), is_active, created_at, updated_at
		FROM subscription_plans
		WHERE code = $1`

	var p model.SubscriptionPlan
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly,
		&p.CrawlQuota, &p.AIQueryQuota, &p.MaxUserSeats, &p.TargetPersona, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SubscriptionRepository) UpsertPlan(ctx context.Context, p model.SubscriptionPlan) error {
	query := `
		INSERT INTO subscription_plans (code, name, description, target_persona, price_monthly, price_yearly, crawl_quota, ai_query_quota, max_user_seats, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			target_persona = EXCLUDED.target_persona,
			price_monthly = EXCLUDED.price_monthly,
			price_yearly = EXCLUDED.price_yearly,
			crawl_quota = EXCLUDED.crawl_quota,
			ai_query_quota = EXCLUDED.ai_query_quota,
			max_user_seats = EXCLUDED.max_user_seats,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()`

	persona := p.TargetPersona
	if persona == "" {
		persona = "ORGANIZATION"
	}

	_, err := r.pool.Exec(ctx, query,
		p.Code, p.Name, p.Description, persona,
		p.PriceMonthly, p.PriceYearly, p.CrawlQuota, p.AIQueryQuota, p.MaxUserSeats, p.IsActive,
	)
	return err
}

func (r *SubscriptionRepository) GetTenantSubscription(ctx context.Context, orgID string) (*model.TenantSubscription, error) {
	query := `
		SELECT ts.id, ts.org_id, ts.plan_id, ts.status, ts.billing_cycle, 
		       ts.current_period_start, ts.current_period_end, ts.auto_renew, 
		       ts.midtrans_subscription_id, ts.created_at, ts.updated_at,
		       sp.id, sp.code, sp.name, COALESCE(sp.description, ''), sp.price_monthly, sp.price_yearly,
		       sp.crawl_quota, sp.ai_query_quota, sp.max_user_seats, sp.is_active, sp.created_at, sp.updated_at
		FROM tenant_subscriptions ts
		JOIN subscription_plans sp ON ts.plan_id = sp.id
		WHERE ts.org_id = $1`

	var ts model.TenantSubscription
	var sp model.SubscriptionPlan
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&ts.ID, &ts.OrgID, &ts.PlanID, &ts.Status, &ts.BillingCycle,
		&ts.CurrentPeriodStart, &ts.CurrentPeriodEnd, &ts.AutoRenew,
		&ts.MidtransSubscriptionID, &ts.CreatedAt, &ts.UpdatedAt,
		&sp.ID, &sp.Code, &sp.Name, &sp.Description, &sp.PriceMonthly, &sp.PriceYearly,
		&sp.CrawlQuota, &sp.AIQueryQuota, &sp.MaxUserSeats, &sp.IsActive, &sp.CreatedAt, &sp.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Auto create free trial if missing
			plan, errPlan := r.GetPlanByCode(ctx, "PRO")
			if errPlan != nil {
				return nil, err
			}
			return r.UpsertTenantSubscription(ctx, orgID, plan.ID, "MONTHLY", 30)
		}
		return nil, err
	}
	ts.Plan = &sp
	return &ts, nil
}

func (r *SubscriptionRepository) UpsertTenantSubscription(ctx context.Context, orgID, planID, cycle string, durationDays int) (*model.TenantSubscription, error) {
	query := `
		INSERT INTO tenant_subscriptions (org_id, plan_id, status, billing_cycle, current_period_start, current_period_end)
		VALUES ($1, $2, 'ACTIVE', $3, NOW(), NOW() + ($4 || ' days')::INTERVAL)
		ON CONFLICT (org_id) DO UPDATE 
		SET plan_id = EXCLUDED.plan_id,
		    status = 'ACTIVE',
		    billing_cycle = EXCLUDED.billing_cycle,
		    current_period_start = NOW(),
		    current_period_end = NOW() + ($4 || ' days')::INTERVAL,
		    updated_at = NOW()
		RETURNING id, org_id, plan_id, status, billing_cycle, current_period_start, current_period_end, auto_renew, created_at, updated_at`

	var ts model.TenantSubscription
	err := r.pool.QueryRow(ctx, query, orgID, planID, cycle, durationDays).Scan(
		&ts.ID, &ts.OrgID, &ts.PlanID, &ts.Status, &ts.BillingCycle,
		&ts.CurrentPeriodStart, &ts.CurrentPeriodEnd, &ts.AutoRenew, &ts.CreatedAt, &ts.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert tenant subscription: %w", err)
	}

	// Update organization subscription_tier
	plan, errPlan := r.GetPlanByID(ctx, planID)
	if errPlan == nil {
		_, _ = r.pool.Exec(ctx, "UPDATE organizations SET subscription_tier = $1, account_status = 'ACTIVE' WHERE id = $2", plan.Code, orgID)
		ts.Plan = plan
	}

	return &ts, nil
}

func (r *SubscriptionRepository) CreateInvoice(ctx context.Context, inv *model.BillingInvoice) (*model.BillingInvoice, error) {
	query := `
		INSERT INTO billing_invoices (invoice_number, org_id, plan_id, amount, status, billing_cycle, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query, inv.InvoiceNumber, inv.OrgID, inv.PlanID, inv.Amount, inv.Status, inv.BillingCycle, inv.DueDate).Scan(
		&inv.ID, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing invoice: %w", err)
	}
	return inv, nil
}

func (r *SubscriptionRepository) ListInvoicesByOrg(ctx context.Context, orgID string) ([]model.BillingInvoice, error) {
	query := `
		SELECT bi.id, bi.invoice_number, bi.org_id, bi.plan_id, bi.amount, bi.status, bi.billing_cycle, bi.due_date, bi.paid_at, bi.created_at, bi.updated_at,
		       sp.code, sp.name
		FROM billing_invoices bi
		JOIN subscription_plans sp ON bi.plan_id = sp.id
		WHERE bi.org_id = $1
		ORDER BY bi.created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []model.BillingInvoice
	for rows.Next() {
		var bi model.BillingInvoice
		var pCode, pName string
		err := rows.Scan(
			&bi.ID, &bi.InvoiceNumber, &bi.OrgID, &bi.PlanID, &bi.Amount, &bi.Status, &bi.BillingCycle, &bi.DueDate, &bi.PaidAt, &bi.CreatedAt, &bi.UpdatedAt,
			&pCode, &pName,
		)
		if err != nil {
			return nil, err
		}
		bi.Plan = &model.SubscriptionPlan{Code: pCode, Name: pName}
		invoices = append(invoices, bi)
	}
	return invoices, nil
}

func (r *SubscriptionRepository) CreatePaymentTransaction(ctx context.Context, tx *model.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions (invoice_id, org_id, provider, order_id, gross_amount, snap_token, snap_redirect_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query, tx.InvoiceID, tx.OrgID, tx.Provider, tx.OrderID, tx.GrossAmount, tx.SnapToken, tx.SnapRedirectURL, tx.Status).Scan(
		&tx.ID, &tx.CreatedAt, &tx.UpdatedAt,
	)
}

func (r *SubscriptionRepository) GetPaymentTransactionByOrderID(ctx context.Context, orderID string) (*model.PaymentTransaction, error) {
	query := `
		SELECT id, invoice_id, org_id, provider, order_id, gross_amount, payment_type, snap_token, snap_redirect_url, status, paid_at, created_at, updated_at
		FROM payment_transactions
		WHERE order_id = $1`

	var tx model.PaymentTransaction
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&tx.ID, &tx.InvoiceID, &tx.OrgID, &tx.Provider, &tx.OrderID, &tx.GrossAmount, &tx.PaymentType, &tx.SnapToken, &tx.SnapRedirectURL, &tx.Status, &tx.PaidAt, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *SubscriptionRepository) LogWebhookInbound(ctx context.Context, source, externalID string, payload interface{}, signature string) (string, error) {
	payloadJSON, _ := json.Marshal(payload)
	query := `
		INSERT INTO webhook_inbound_logs (source, external_id, payload, signature_hash, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, query, source, externalID, payloadJSON, signature).Scan(&id)
	return id, err
}

func (r *SubscriptionRepository) UpdateWebhookStatus(ctx context.Context, logID, status, errMsg string) error {
	query := `
		UPDATE webhook_inbound_logs 
		SET status = $1, error_message = $2, processed_at = NOW() 
		WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, errMsg, logID)
	return err
}

func (r *SubscriptionRepository) SettlePaymentAndActivateSubscription(ctx context.Context, orderID, paymentType string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get payment transaction
	var txID, invID, orgID string
	var amount float64
	err = tx.QueryRow(ctx, "SELECT id, invoice_id, org_id, gross_amount FROM payment_transactions WHERE order_id = $1", orderID).Scan(&txID, &invID, &orgID, &amount)
	if err != nil {
		return fmt.Errorf("transaction not found for order_id %s: %w", orderID, err)
	}

	now := time.Now()
	// 2. Update payment transaction
	_, err = tx.Exec(ctx, "UPDATE payment_transactions SET status = 'SETTLEMENT', payment_type = $1, paid_at = $2, updated_at = $2 WHERE id = $3", paymentType, now, txID)
	if err != nil {
		return fmt.Errorf("failed to update payment_transactions: %w", err)
	}

	// 3. Update invoice
	var planID, billingCycle string
	err = tx.QueryRow(ctx, "UPDATE billing_invoices SET status = 'PAID', paid_at = $1, updated_at = $1 WHERE id = $2 RETURNING plan_id, billing_cycle", now, invID).Scan(&planID, &billingCycle)
	if err != nil {
		return fmt.Errorf("failed to update billing_invoices: %w", err)
	}

	durationDays := 30
	if billingCycle == "YEARLY" {
		durationDays = 365
	}

	// 4. Update tenant_subscriptions
	_, err = tx.Exec(ctx, `
		INSERT INTO tenant_subscriptions (org_id, plan_id, status, billing_cycle, current_period_start, current_period_end)
		VALUES ($1, $2, 'ACTIVE', $3, NOW(), NOW() + ($4 || ' days')::INTERVAL)
		ON CONFLICT (org_id) DO UPDATE 
		SET plan_id = EXCLUDED.plan_id,
		    status = 'ACTIVE',
		    billing_cycle = EXCLUDED.billing_cycle,
		    current_period_start = NOW(),
		    current_period_end = NOW() + ($4 || ' days')::INTERVAL,
		    updated_at = NOW()`, orgID, planID, billingCycle, durationDays)
	if err != nil {
		return fmt.Errorf("failed to update tenant_subscriptions: %w", err)
	}

	// 5. Update organizations account_status & tier
	var planCode string
	_ = tx.QueryRow(ctx, "SELECT code FROM subscription_plans WHERE id = $1", planID).Scan(&planCode)
	if planCode != "" {
		_, _ = tx.Exec(ctx, "UPDATE organizations SET subscription_tier = $1, account_status = 'ACTIVE', updated_at = NOW() WHERE id = $2", planCode, orgID)
	}

	return tx.Commit(ctx)
}
