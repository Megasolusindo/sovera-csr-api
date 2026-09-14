package keyperson

import (
	"context"
	"fmt"
	"strings"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/repository"
)

type KeyPersonService struct {
	repo *repository.KeyPersonRepository
}

func NewKeyPersonService(repo *repository.KeyPersonRepository) *KeyPersonService {
	return &KeyPersonService{
		repo: repo,
	}
}

// CreateKeyPerson normalizes input and saves a new key person.
func (s *KeyPersonService) CreateKeyPerson(ctx context.Context, person *model.CompanyKeyPerson) (*model.CompanyKeyPerson, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}

	if person.FullName == "" {
		return nil, fmt.Errorf("full_name is required")
	}

	if person.CompanyID == "" {
		return nil, fmt.Errorf("company_id is required")
	}

	// Normalize name
	person.NormalizedName = normalizeName(person.FullName)

	return s.repo.CreateKeyPerson(ctx, person)
}

// UpdateKeyPerson updates an existing key person profile.
func (s *KeyPersonService) UpdateKeyPerson(ctx context.Context, person *model.CompanyKeyPerson) (*model.CompanyKeyPerson, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}

	if person.ID == "" {
		return nil, fmt.Errorf("key person ID is required")
	}

	if person.FullName != "" {
		person.NormalizedName = normalizeName(person.FullName)
	}

	return s.repo.UpdateKeyPerson(ctx, person)
}

// GetKeyPersonsByCompanyID retrieves key persons associated with a company.
func (s *KeyPersonService) GetKeyPersonsByCompanyID(ctx context.Context, companyID string) ([]model.CompanyKeyPerson, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}
	return s.repo.GetKeyPersonsByCompanyID(ctx, companyID)
}

// GetKeyPersonByID retrieves a single key person.
func (s *KeyPersonService) GetKeyPersonByID(ctx context.Context, id string) (*model.CompanyKeyPerson, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}
	return s.repo.GetKeyPersonByID(ctx, id)
}

// DeleteKeyPerson deletes a key person by ID.
func (s *KeyPersonService) DeleteKeyPerson(ctx context.Context, id string) error {
	if s.repo == nil {
		return fmt.Errorf("key person repository is nil")
	}
	return s.repo.DeleteKeyPerson(ctx, id)
}

// IngestSocialSignal validates and saves a social or news signal.
func (s *KeyPersonService) IngestSocialSignal(ctx context.Context, signal *model.KeyPersonSocialSignal) (*model.KeyPersonSocialSignal, error) {
	if signal == nil {
		return nil, fmt.Errorf("signal is nil")
	}

	if signal.CompanyID == "" {
		return nil, fmt.Errorf("company_id is required")
	}

	if signal.PostText == "" {
		return nil, fmt.Errorf("post_text is required")
	}

	// Auto-evaluate actionability if matched keywords contain actionable intent
	if len(signal.MatchedKeywords) > 0 {
		for _, kw := range signal.MatchedKeywords {
			switch kw {
			case "bantuan", "beasiswa", "stunting", "air bersih", "bencana", "hibah", "kemitraan", "proposal":
				signal.IsActionable = true
			}
		}
	}

	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}

	return s.repo.CreateSocialSignal(ctx, signal)
}

// GetSocialSignalsByCompanyID retrieves social signals for a company.
func (s *KeyPersonService) GetSocialSignalsByCompanyID(ctx context.Context, companyID string, limit, offset int) ([]model.KeyPersonSocialSignal, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}
	return s.repo.GetSocialSignalsByCompanyID(ctx, companyID, limit, offset)
}

// GetActionableSignals retrieves signals marked as actionable for outreach.
func (s *KeyPersonService) GetActionableSignals(ctx context.Context, companyID string) ([]model.KeyPersonSocialSignal, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("key person repository is nil")
	}
	return s.repo.GetActionableSignals(ctx, companyID)
}

func normalizeName(input string) string {
	cleaned := strings.ToLower(strings.TrimSpace(input))
	return strings.Join(strings.Fields(cleaned), " ")
}
