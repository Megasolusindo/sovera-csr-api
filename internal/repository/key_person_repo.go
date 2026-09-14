package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type KeyPersonRepository struct {
	pool *pgxpool.Pool
}

func NewKeyPersonRepository(pool *pgxpool.Pool) *KeyPersonRepository {
	return &KeyPersonRepository{pool: pool}
}

// CreateKeyPerson inserts a new key person.
func (r *KeyPersonRepository) CreateKeyPerson(ctx context.Context, person *model.CompanyKeyPerson) (*model.CompanyKeyPerson, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO company_key_persons (
			company_id, full_name, normalized_name, current_title, role_category,
			linkedin_url, twitter_handle, instagram_handle, is_decision_maker, is_monitored
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		RETURNING 
			id::text, company_id::text, full_name, normalized_name, current_title,
			role_category, linkedin_url, twitter_handle, instagram_handle,
			is_decision_maker, is_monitored, created_at, updated_at;
	`

	var kp model.CompanyKeyPerson
	err := r.pool.QueryRow(ctx, query,
		person.CompanyID, person.FullName, person.NormalizedName, person.CurrentTitle, person.RoleCategory,
		person.LinkedInURL, person.TwitterHandle, person.InstagramHandle, person.IsDecisionMaker, person.IsMonitored,
	).Scan(
		&kp.ID, &kp.CompanyID, &kp.FullName, &kp.NormalizedName, &kp.CurrentTitle,
		&kp.RoleCategory, &kp.LinkedInURL, &kp.TwitterHandle, &kp.InstagramHandle,
		&kp.IsDecisionMaker, &kp.IsMonitored, &kp.CreatedAt, &kp.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create key person: %w", err)
	}

	return &kp, nil
}

// UpdateKeyPerson updates an existing key person profile.
func (r *KeyPersonRepository) UpdateKeyPerson(ctx context.Context, person *model.CompanyKeyPerson) (*model.CompanyKeyPerson, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE company_key_persons SET
			full_name = $2,
			normalized_name = $3,
			current_title = $4,
			role_category = $5,
			linkedin_url = $6,
			twitter_handle = $7,
			instagram_handle = $8,
			is_decision_maker = $9,
			is_monitored = $10,
			updated_at = NOW()
		WHERE id::text = $1
		RETURNING 
			id::text, company_id::text, full_name, normalized_name, current_title,
			role_category, linkedin_url, twitter_handle, instagram_handle,
			is_decision_maker, is_monitored, created_at, updated_at;
	`

	var kp model.CompanyKeyPerson
	err := r.pool.QueryRow(ctx, query,
		person.ID, person.FullName, person.NormalizedName, person.CurrentTitle, person.RoleCategory,
		person.LinkedInURL, person.TwitterHandle, person.InstagramHandle, person.IsDecisionMaker, person.IsMonitored,
	).Scan(
		&kp.ID, &kp.CompanyID, &kp.FullName, &kp.NormalizedName, &kp.CurrentTitle,
		&kp.RoleCategory, &kp.LinkedInURL, &kp.TwitterHandle, &kp.InstagramHandle,
		&kp.IsDecisionMaker, &kp.IsMonitored, &kp.CreatedAt, &kp.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update key person: %w", err)
	}

	return &kp, nil
}

// GetKeyPersonsByCompanyID retrieves all key persons for a company.
func (r *KeyPersonRepository) GetKeyPersonsByCompanyID(ctx context.Context, companyID string) ([]model.CompanyKeyPerson, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, company_id::text, full_name, normalized_name, current_title,
			role_category, linkedin_url, twitter_handle, instagram_handle,
			is_decision_maker, is_monitored, created_at, updated_at
		FROM company_key_persons
		WHERE company_id::text = $1
		ORDER BY is_decision_maker DESC, created_at ASC;
	`

	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list key persons for company %s: %w", companyID, err)
	}
	defer rows.Close()

	var result []model.CompanyKeyPerson
	for rows.Next() {
		var kp model.CompanyKeyPerson
		err := rows.Scan(
			&kp.ID, &kp.CompanyID, &kp.FullName, &kp.NormalizedName, &kp.CurrentTitle,
			&kp.RoleCategory, &kp.LinkedInURL, &kp.TwitterHandle, &kp.InstagramHandle,
			&kp.IsDecisionMaker, &kp.IsMonitored, &kp.CreatedAt, &kp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key person row: %w", err)
		}
		result = append(result, kp)
	}

	return result, nil
}

// GetKeyPersonByID retrieves a single key person by ID.
func (r *KeyPersonRepository) GetKeyPersonByID(ctx context.Context, id string) (*model.CompanyKeyPerson, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, company_id::text, full_name, normalized_name, current_title,
			role_category, linkedin_url, twitter_handle, instagram_handle,
			is_decision_maker, is_monitored, created_at, updated_at
		FROM company_key_persons
		WHERE id::text = $1
		LIMIT 1;
	`

	var kp model.CompanyKeyPerson
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&kp.ID, &kp.CompanyID, &kp.FullName, &kp.NormalizedName, &kp.CurrentTitle,
		&kp.RoleCategory, &kp.LinkedInURL, &kp.TwitterHandle, &kp.InstagramHandle,
		&kp.IsDecisionMaker, &kp.IsMonitored, &kp.CreatedAt, &kp.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("key person not found: %w", err)
	}

	return &kp, nil
}

// DeleteKeyPerson removes a key person profile.
func (r *KeyPersonRepository) DeleteKeyPerson(ctx context.Context, id string) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM company_key_persons WHERE id::text = $1;`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete key person: %w", err)
	}

	return nil
}

// CreateSocialSignal inserts a social signal.
func (r *KeyPersonRepository) CreateSocialSignal(ctx context.Context, signal *model.KeyPersonSocialSignal) (*model.KeyPersonSocialSignal, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO key_person_social_signals (
			person_id, company_id, platform, post_url, post_text,
			posted_at, matched_keywords, sentiment, ai_summary, is_actionable
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (post_url) WHERE post_url IS NOT NULL DO UPDATE SET
			matched_keywords = EXCLUDED.matched_keywords,
			sentiment = EXCLUDED.sentiment,
			ai_summary = EXCLUDED.ai_summary,
			is_actionable = EXCLUDED.is_actionable
		RETURNING 
			id::text, person_id::text, company_id::text, platform, post_url,
			post_text, posted_at, COALESCE(matched_keywords, '{}'), sentiment,
			ai_summary, is_actionable, created_at;
	`

	var s model.KeyPersonSocialSignal
	err := r.pool.QueryRow(ctx, query,
		signal.PersonID, signal.CompanyID, signal.Platform, signal.PostURL, signal.PostText,
		signal.PostedAt, signal.MatchedKeywords, signal.Sentiment, signal.AISummary, signal.IsActionable,
	).Scan(
		&s.ID, &s.PersonID, &s.CompanyID, &s.Platform, &s.PostURL,
		&s.PostText, &s.PostedAt, &s.MatchedKeywords, &s.Sentiment,
		&s.AISummary, &s.IsActionable, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create key person social signal: %w", err)
	}

	return &s, nil
}

// GetSocialSignalsByCompanyID retrieves social signals for a company.
func (r *KeyPersonRepository) GetSocialSignalsByCompanyID(ctx context.Context, companyID string, limit, offset int) ([]model.KeyPersonSocialSignal, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT 
			id::text, person_id::text, company_id::text, platform, post_url,
			post_text, posted_at, COALESCE(matched_keywords, '{}'), sentiment,
			ai_summary, is_actionable, created_at
		FROM key_person_social_signals
		WHERE company_id::text = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query social signals for company %s: %w", companyID, err)
	}
	defer rows.Close()

	var result []model.KeyPersonSocialSignal
	for rows.Next() {
		var s model.KeyPersonSocialSignal
		err := rows.Scan(
			&s.ID, &s.PersonID, &s.CompanyID, &s.Platform, &s.PostURL,
			&s.PostText, &s.PostedAt, &s.MatchedKeywords, &s.Sentiment,
			&s.AISummary, &s.IsActionable, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan social signal row: %w", err)
		}
		result = append(result, s)
	}

	return result, nil
}

// GetActionableSignals retrieves signals marked as actionable for outreach generation.
func (r *KeyPersonRepository) GetActionableSignals(ctx context.Context, companyID string) ([]model.KeyPersonSocialSignal, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT 
			id::text, person_id::text, company_id::text, platform, post_url,
			post_text, posted_at, COALESCE(matched_keywords, '{}'), sentiment,
			ai_summary, is_actionable, created_at
		FROM key_person_social_signals
		WHERE company_id::text = $1 AND is_actionable = TRUE
		ORDER BY created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query actionable signals for company %s: %w", companyID, err)
	}
	defer rows.Close()

	var result []model.KeyPersonSocialSignal
	for rows.Next() {
		var s model.KeyPersonSocialSignal
		err := rows.Scan(
			&s.ID, &s.PersonID, &s.CompanyID, &s.Platform, &s.PostURL,
			&s.PostText, &s.PostedAt, &s.MatchedKeywords, &s.Sentiment,
			&s.AISummary, &s.IsActionable, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan actionable social signal row: %w", err)
		}
		result = append(result, s)
	}

	return result, nil
}
