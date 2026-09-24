package repository

import (
	"context"
	"fmt"

	"sovera-core-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByEmail retrieves a user by their email address (used for login).
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, org_id, email, password_hash, full_name, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = true
		LIMIT 1;
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.PasswordHash,
		&u.FullName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

// FindByID retrieves a user by their UUID (used for /auth/me).
func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, org_id, email, password_hash, full_name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = true
		LIMIT 1;
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.OrgID, &u.Email, &u.PasswordHash,
		&u.FullName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

// Create inserts a new user into the database and returns the created user.
func (r *UserRepository) Create(ctx context.Context, u model.User) (*model.User, error) {
	query := `
		INSERT INTO users (org_id, email, password_hash, full_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id, org_id, email, password_hash, full_name, role, is_active, created_at, updated_at;
	`
	var created model.User
	err := r.pool.QueryRow(ctx, query, u.OrgID, u.Email, u.PasswordHash, u.FullName, u.Role).Scan(
		&created.ID, &created.OrgID, &created.Email, &created.PasswordHash,
		&created.FullName, &created.Role, &created.IsActive, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &created, nil
}

type UserWithOrgItem struct {
	model.User
	OrgName    string           `json:"org_name" db:"org_name"`
	TenantType model.TenantType `json:"tenant_type" db:"tenant_type"`
	CompanyID  *string          `json:"company_id,omitempty" db:"company_id"`
}

// FindUserWithTenantByEmail retrieves user along with tenant organization details.
func (r *UserRepository) FindUserWithTenantByEmail(ctx context.Context, email string) (*UserWithOrgItem, error) {
	query := `
		SELECT 
			u.id, u.org_id, u.email, u.password_hash, u.full_name, u.role, u.is_active, u.created_at, u.updated_at,
			COALESCE(o.name, 'System') AS org_name,
			COALESCE(o.org_type, 'ORGANIZATION') AS tenant_type,
			o.company_id
		FROM users u
		LEFT JOIN organizations o ON o.id = u.org_id
		WHERE u.email = $1 AND u.is_active = true
		LIMIT 1;
	`
	var item UserWithOrgItem
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&item.ID, &item.OrgID, &item.Email, &item.PasswordHash, &item.FullName, &item.Role,
		&item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		&item.OrgName, &item.TenantType, &item.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("user with tenant not found: %w", err)
	}
	return &item, nil
}

// FindUserWithTenantByID retrieves user along with tenant organization details by User ID.
func (r *UserRepository) FindUserWithTenantByID(ctx context.Context, id string) (*UserWithOrgItem, error) {
	query := `
		SELECT 
			u.id, u.org_id, u.email, u.password_hash, u.full_name, u.role, u.is_active, u.created_at, u.updated_at,
			COALESCE(o.name, 'System') AS org_name,
			COALESCE(o.org_type, 'ORGANIZATION') AS tenant_type,
			o.company_id
		FROM users u
		LEFT JOIN organizations o ON o.id = u.org_id
		WHERE u.id = $1 AND u.is_active = true
		LIMIT 1;
	`
	var item UserWithOrgItem
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.OrgID, &item.Email, &item.PasswordHash, &item.FullName, &item.Role,
		&item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		&item.OrgName, &item.TenantType, &item.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("user with tenant not found: %w", err)
	}
	return &item, nil
}

// ListAllUsers retrieves user accounts across all tenant organizations.
func (r *UserRepository) ListAllUsers(ctx context.Context, page, pageSize int) ([]UserWithOrgItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT 
			u.id, u.org_id, u.email, u.full_name, u.role, u.is_active, u.created_at, u.updated_at,
			COALESCE(o.name, 'System') AS org_name
		FROM users u
		LEFT JOIN organizations o ON o.id = u.org_id
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	items := []UserWithOrgItem{}
	for rows.Next() {
		var item UserWithOrgItem
		if err := rows.Scan(
			&item.ID, &item.OrgID, &item.Email, &item.FullName, &item.Role, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt, &item.OrgName,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}

