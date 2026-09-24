package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sovera-core-api/internal/model"
)

type OrganizationAIRepository struct {
	dbPool *pgxpool.Pool
}

func NewOrganizationAIRepository(dbPool *pgxpool.Pool) *OrganizationAIRepository {
	return &OrganizationAIRepository{dbPool: dbPool}
}

// CreateConversation initializes a new thread for a user within an organization
func (r *OrganizationAIRepository) CreateConversation(ctx context.Context, orgID, userID uuid.UUID, title string) (*model.OrganizationAIConversation, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	if strings.TrimSpace(title) == "" {
		title = "Percakapan Baru"
	}

	query := `
		INSERT INTO organization_ai_conversations (org_id, user_id, title, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		RETURNING id, org_id, user_id, title, status, created_at, updated_at;
	`

	var conv model.OrganizationAIConversation
	err := r.dbPool.QueryRow(ctx, query, orgID, userID, title).Scan(
		&conv.ID,
		&conv.OrgID,
		&conv.UserID,
		&conv.Title,
		&conv.Status,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create organization conversation: %w", err)
	}

	return &conv, nil
}

// GetConversationByID fetches thread metadata with double security filters (org_id AND user_id)
func (r *OrganizationAIRepository) GetConversationByID(ctx context.Context, orgID, userID, conversationID uuid.UUID) (*model.OrganizationAIConversation, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	query := `
		SELECT id, org_id, user_id, title, status, created_at, updated_at
		FROM organization_ai_conversations
		WHERE id = $1 AND org_id = $2 AND user_id = $3 AND status != 'archived';
	`

	var conv model.OrganizationAIConversation
	err := r.dbPool.QueryRow(ctx, query, conversationID, orgID, userID).Scan(
		&conv.ID,
		&conv.OrgID,
		&conv.UserID,
		&conv.Title,
		&conv.Status,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("conversation not found or access denied: %w", err)
	}

	return &conv, nil
}

// ListConversationsByUser lists all active threads for a logged-in user in an organization
func (r *OrganizationAIRepository) ListConversationsByUser(ctx context.Context, orgID, userID uuid.UUID) ([]model.OrganizationAIConversation, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	query := `
		SELECT id, org_id, user_id, title, status, created_at, updated_at
		FROM organization_ai_conversations
		WHERE org_id = $1 AND user_id = $2 AND status != 'archived'
		ORDER BY updated_at DESC;
	`

	rows, err := r.dbPool.Query(ctx, query, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []model.OrganizationAIConversation
	for rows.Next() {
		var c model.OrganizationAIConversation
		if err := rows.Scan(&c.ID, &c.OrgID, &c.UserID, &c.Title, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}

	return conversations, nil
}

// ListConversationsByOrgAdmin lists all organization threads for admin oversight (read-only)
func (r *OrganizationAIRepository) ListConversationsByOrgAdmin(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationAIConversation, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	query := `
		SELECT id, org_id, user_id, title, status, created_at, updated_at
		FROM organization_ai_conversations
		WHERE org_id = $1
		ORDER BY updated_at DESC;
	`

	rows, err := r.dbPool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list admin conversations: %w", err)
	}
	defer rows.Close()

	var conversations []model.OrganizationAIConversation
	for rows.Next() {
		var c model.OrganizationAIConversation
		if err := rows.Scan(&c.ID, &c.OrgID, &c.UserID, &c.Title, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}

	return conversations, nil
}

// ArchiveConversation soft-deletes a conversation thread
func (r *OrganizationAIRepository) ArchiveConversation(ctx context.Context, orgID, userID, conversationID uuid.UUID) error {
	if r.dbPool == nil {
		return fmt.Errorf("dbPool is not initialized")
	}

	query := `
		UPDATE organization_ai_conversations
		SET status = 'archived', updated_at = NOW()
		WHERE id = $1 AND org_id = $2 AND user_id = $3;
	`

	cmd, err := r.dbPool.Exec(ctx, query, conversationID, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found or unauthorized")
	}

	return nil
}

// CreateChatLog inserts an audit chat interaction record
func (r *OrganizationAIRepository) CreateChatLog(ctx context.Context, log model.OrganizationAIChatLog) error {
	if r.dbPool == nil {
		return fmt.Errorf("dbPool is not initialized")
	}

	toolsJSON, err := json.Marshal(log.ToolsCalled)
	if err != nil {
		toolsJSON = []byte("[]")
	}

	query := `
		INSERT INTO organization_ai_chat_logs (
			org_id, user_id, conversation_id, message, reply, tools_called, latency_ms, model_name, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW()
		);
	`

	_, err = r.dbPool.Exec(ctx, query,
		log.OrgID,
		log.UserID,
		log.ConversationID,
		log.Message,
		log.Reply,
		toolsJSON,
		log.LatencyMS,
		log.ModelName,
	)

	if err != nil {
		return fmt.Errorf("failed to create chat log: %w", err)
	}

	// Update conversation updated_at time
	_, _ = r.dbPool.Exec(ctx, `UPDATE organization_ai_conversations SET updated_at = NOW() WHERE id = $1`, log.ConversationID)

	return nil
}

// GetConversationHistory retrieves message history for a specific thread owned by the user
func (r *OrganizationAIRepository) GetConversationHistory(ctx context.Context, orgID, userID, conversationID uuid.UUID) ([]model.OrganizationAIChatLog, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	query := `
		SELECT id, org_id, user_id, conversation_id, message, reply, tools_called, latency_ms, model_name, created_at
		FROM organization_ai_chat_logs
		WHERE org_id = $1 AND user_id = $2 AND conversation_id = $3
		ORDER BY created_at ASC;
	`

	rows, err := r.dbPool.Query(ctx, query, orgID, userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}
	defer rows.Close()

	var logs []model.OrganizationAIChatLog
	for rows.Next() {
		var l model.OrganizationAIChatLog
		var toolsBytes []byte
		if err := rows.Scan(
			&l.ID, &l.OrgID, &l.UserID, &l.ConversationID, &l.Message, &l.Reply, &toolsBytes, &l.LatencyMS, &l.ModelName, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(toolsBytes) > 0 {
			_ = json.Unmarshal(toolsBytes, &l.ToolsCalled)
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// GetAdminConversationHistory retrieves message history for an organization thread (Admin read-only)
func (r *OrganizationAIRepository) GetAdminConversationHistory(ctx context.Context, orgID, conversationID uuid.UUID) ([]model.OrganizationAIChatLog, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	query := `
		SELECT id, org_id, user_id, conversation_id, message, reply, tools_called, latency_ms, model_name, created_at
		FROM organization_ai_chat_logs
		WHERE org_id = $1 AND conversation_id = $2
		ORDER BY created_at ASC;
	`

	rows, err := r.dbPool.Query(ctx, query, orgID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin conversation history: %w", err)
	}
	defer rows.Close()

	var logs []model.OrganizationAIChatLog
	for rows.Next() {
		var l model.OrganizationAIChatLog
		var toolsBytes []byte
		if err := rows.Scan(
			&l.ID, &l.OrgID, &l.UserID, &l.ConversationID, &l.Message, &l.Reply, &toolsBytes, &l.LatencyMS, &l.ModelName, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(toolsBytes) > 0 {
			_ = json.Unmarshal(toolsBytes, &l.ToolsCalled)
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// SearchSignalsByOrg searches signals (shared market signals & org-specific data)
func (r *OrganizationAIRepository) SearchSignalsByOrg(ctx context.Context, orgID uuid.UUID, queryStr string) ([]model.CorporateSignal, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	searchPattern := "%" + strings.TrimSpace(queryStr) + "%"

	sqlQuery := `
		SELECT id, company_name, COALESCE(industry_sector, ''), source_type, COALESCE(source_url, ''), COALESCE(summary, ''), COALESCE(extracted_pillar, ''), COALESCE(target_regions, '{}'), COALESCE(estimated_budget_signal, 0), COALESCE(trigger_event, ''), COALESCE(intent_score, 0), COALESCE(content_hash, ''), COALESCE(published_date, NOW()::date), created_at
		FROM public_corporate_signals
		WHERE company_name ILIKE $1 OR summary ILIKE $1 OR extracted_pillar ILIKE $1 OR trigger_event ILIKE $1
		ORDER BY intent_score DESC, created_at DESC
		LIMIT 15;
	`

	rows, err := r.dbPool.Query(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search signals: %w", err)
	}
	defer rows.Close()

	var results []model.CorporateSignal
	for rows.Next() {
		var s model.CorporateSignal
		var sourceType string
		if err := rows.Scan(
			&s.ID, &s.CompanyName, &s.IndustrySector, &sourceType, &s.SourceURL, &s.Summary, &s.ExtractedPillar, &s.TargetRegions, &s.EstimatedBudgetSignal, &s.TriggerEvent, &s.IntentScore, &s.ContentHash, &s.PublishedDate, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.SourceType = sourceType
		results = append(results, s)
	}

	return results, nil
}

// ListWatchlistByOrg retrieves active monitoring watchlist
func (r *OrganizationAIRepository) ListWatchlistByOrg(ctx context.Context, orgID uuid.UUID) ([]model.AICompanyWatchlist, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	sqlQuery := `
		SELECT id, company_id, company_name, monitoring_keywords, check_interval_hours, last_monitored_at, is_active, created_at, updated_at
		FROM ai_company_watchlist
		WHERE is_active = true
		ORDER BY company_name ASC;
	`

	rows, err := r.dbPool.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list watchlist: %w", err)
	}
	defer rows.Close()

	var list []model.AICompanyWatchlist
	for rows.Next() {
		var item model.AICompanyWatchlist
		if err := rows.Scan(
			&item.ID, &item.CompanyID, &item.CompanyName, &item.MonitoringKeywords, &item.CheckIntervalHours, &item.LastMonitoredAt, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}

// SearchCompaniesPublic searches master company database (public data)
func (r *OrganizationAIRepository) SearchCompaniesPublic(ctx context.Context, queryStr string) ([]model.Company, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	pattern := "%" + strings.TrimSpace(queryStr) + "%"

	sqlQuery := `
		SELECT c.id::text, c.name, c.legal_name, c.slug, c.industry_sector, c.company_type, c.website, c.is_public, c.ticker, c.created_at, c.updated_at
		FROM company.companies c
		WHERE c.name ILIKE $1 OR c.ticker ILIKE $1 OR c.industry_sector ILIKE $1
		ORDER BY c.is_public DESC, c.name ASC
		LIMIT 15;
	`

	rows, err := r.dbPool.Query(ctx, sqlQuery, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search public companies: %w", err)
	}
	defer rows.Close()

	var companies []model.Company
	for rows.Next() {
		var c model.Company
		if err := rows.Scan(
			&c.ID, &c.Name, &c.LegalName, &c.Slug, &c.IndustrySector, &c.CompanyType, &c.Website, &c.IsPublic, &c.Ticker, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}

	return companies, nil
}

// ListOrganizationPrograms retrieves all institution programs belonging to the tenant organization inside an RLS-enforced transaction
func (r *OrganizationAIRepository) ListOrganizationPrograms(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationProgramInfo, error) {
	if r.dbPool == nil {
		return nil, fmt.Errorf("dbPool is not initialized")
	}

	var programs []model.OrganizationProgramInfo

	err := WithTenantContext(ctx, r.dbPool, orgID.String(), func(tx pgx.Tx) error {
		sqlQuery := `
			SELECT 
				id::text, org_id::text, title, COALESCE(description, ''), 
				COALESCE(primary_cluster, 'COMMUNITY_DEVELOPMENT'), ARRAY_TO_STRING(COALESCE(target_sdgs, '{}'), ','),
				COALESCE(asnaf_category, ''), COALESCE(esg_pillar, 'SOCIAL'), COALESCE(target_beneficiaries::text, COALESCE(target_beneficiaries_desc, '')),
				created_at::text
			FROM ngo_managed_programs
			WHERE org_id = $1
			ORDER BY created_at DESC;
		`

		rows, err := tx.Query(ctx, sqlQuery, orgID)
		if err != nil {
			return fmt.Errorf("failed to query institution programs: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p model.OrganizationProgramInfo
			var sdgsJoin string
			if err := rows.Scan(
				&p.ID, &p.OrgID, &p.Title, &p.Description,
				&p.PrimaryCluster, &sdgsJoin,
				&p.AsnafCategory, &p.ESGPillar, &p.TargetBeneficiaries,
				&p.CreatedAt,
			); err != nil {
				return fmt.Errorf("failed to scan program row: %w", err)
			}
			if sdgsJoin != "" {
				p.TargetSDGs = strings.Split(sdgsJoin, ",")
			} else {
				p.TargetSDGs = []string{}
			}
			programs = append(programs, p)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list organization programs: %w", err)
	}

	return programs, nil
}

