package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/pkg/phoneverifier"
)

type OrganizationItem struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	OrgType          string    `json:"org_type" db:"org_type"`
	SubscriptionTier string    `json:"subscription_tier" db:"subscription_tier"`
	AccountStatus    string    `json:"account_status" db:"account_status"`
	UsersCount       int       `json:"users_count" db:"users_count"`
	ContactName      string    `json:"contact_name" db:"contact_name"`
	ContactEmail     string    `json:"contact_email" db:"contact_email"`
	ContactPhone     string    `json:"contact_phone" db:"contact_phone"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// ListOrganizations retrieves tenant organizations with user count aggregation.
func (r *OrganizationRepository) ListOrganizations(ctx context.Context, page, pageSize int, search string) ([]OrganizationItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (o.name ILIKE $%d OR o.id::text ILIKE $%d OR COALESCE(o.org_type, '') ILIKE $%d OR COALESCE(o.contact_name, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations o %s", baseWhere)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count organizations: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			o.id,
			o.name,
			COALESCE(o.org_type, 'HUMANITARIAN_NGO') AS org_type,
			COALESCE(o.subscription_tier, 'PRO') AS subscription_tier,
			COALESCE(o.account_status, 'PROSPECT') AS account_status,
			COUNT(u.id) AS users_count,
			COALESCE(o.contact_name, '') AS contact_name,
			COALESCE(o.contact_email, '') AS contact_email,
			COALESCE(o.contact_phone, '') AS contact_phone,
			o.created_at,
			o.updated_at
		FROM organizations o
		LEFT JOIN users u ON u.org_id = o.id
		%s
		GROUP BY o.id
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d;
	`, baseWhere, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	items := []OrganizationItem{}
	for rows.Next() {
		var item OrganizationItem
		if err := rows.Scan(
			&item.ID, &item.Name, &item.OrgType, &item.SubscriptionTier,
			&item.AccountStatus, &item.UsersCount,
			&item.ContactName, &item.ContactEmail, &item.ContactPhone,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan organization row: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}

// Create inserts a new organization into the database.
func (r *OrganizationRepository) Create(ctx context.Context, name, orgType, tier string) (*OrganizationItem, error) {
	query := `
		INSERT INTO organizations (name, org_type, subscription_tier)
		VALUES ($1, $2, $3)
		RETURNING id, name, org_type, subscription_tier, created_at, updated_at;
	`
	var item OrganizationItem
	err := r.pool.QueryRow(ctx, query, name, orgType, tier).Scan(
		&item.ID, &item.Name, &item.OrgType, &item.SubscriptionTier, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	item.AccountStatus = "ACTIVE"
	item.UsersCount = 0
	return &item, nil
}

// UpdateOrganizationInput holds fields to update an organization.
type UpdateOrganizationInput struct {
	Name             string `json:"name"`
	OrgType          string `json:"org_type"`
	SubscriptionTier string `json:"subscription_tier"`
	AccountStatus    string `json:"account_status"`
	ContactName      string `json:"contact_name"`
	ContactEmail     string `json:"contact_email"`
	ContactPhone     string `json:"contact_phone"`
}

// Update modifies an existing organization record in the database.
func (r *OrganizationRepository) Update(ctx context.Context, id string, input UpdateOrganizationInput) (*OrganizationItem, error) {
	if input.ContactPhone != "" {
		ok, normalized, err := phoneverifier.DefaultVerifier.Verify(input.ContactPhone)
		if !ok {
			return nil, fmt.Errorf("invalid contact phone format: %w", err)
		}
		input.ContactPhone = normalized
	}

	query := `
		UPDATE organizations
		SET 
			name = COALESCE(NULLIF($2, ''), name),
			org_type = COALESCE(NULLIF($3, ''), org_type),
			subscription_tier = COALESCE(NULLIF($4, ''), subscription_tier),
			account_status = COALESCE(NULLIF($5, ''), account_status),
			contact_name = COALESCE(NULLIF($6, ''), contact_name),
			contact_email = COALESCE(NULLIF($7, ''), contact_email),
			contact_phone = COALESCE(NULLIF($8, ''), contact_phone),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, COALESCE(org_type, 'HUMANITARIAN_NGO'), COALESCE(subscription_tier, 'PRO'), COALESCE(account_status, 'PROSPECT'), COALESCE(contact_name, ''), COALESCE(contact_email, ''), COALESCE(contact_phone, ''), created_at, updated_at;
	`
	var item OrganizationItem
	err := r.pool.QueryRow(ctx, query, id, input.Name, input.OrgType, input.SubscriptionTier, input.AccountStatus, input.ContactName, input.ContactEmail, input.ContactPhone).Scan(
		&item.ID, &item.Name, &item.OrgType, &item.SubscriptionTier, &item.AccountStatus, &item.ContactName, &item.ContactEmail, &item.ContactPhone, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	return &item, nil
}

