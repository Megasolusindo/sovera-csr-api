package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AnonymizedTopicDemand struct {
	Sector      string   `json:"sector"`
	Keywords    []string `json:"keywords"`
	DemandCount int      `json:"demand_count"`
}

type GlobalResearchTask struct {
	TaskID    string    `json:"task_id"`
	Sector    string    `json:"sector"`
	Topic     string    `json:"topic"`
	Priority  int       `json:"priority"`
	ScopeType string    `json:"scope_type"` // Must always be GLOBAL
	CreatedAt time.Time `json:"created_at"`
}

// DemandAggregator aggregates NGO watchlists from multiple tenants without carrying tenant identities into research tasks.
type DemandAggregator struct{}

func NewDemandAggregator() *DemandAggregator {
	return &DemandAggregator{}
}

// DepersonalizeAndAggregate takes raw tenant watchlist items and produces anonymized global research priorities
func (da *DemandAggregator) DepersonalizeAndAggregate(ctx context.Context, rawWatchlistItems map[string][]string) ([]GlobalResearchTask, error) {
	// rawWatchlistItems: map[tenantID][]keywords
	// Step 1: Depersonalize - strip tenantID keys completely
	keywordFrequency := make(map[string]int)
	for _, keywords := range rawWatchlistItems {
		seenInTenant := make(map[string]bool)
		for _, kw := range keywords {
			clean := strings.TrimSpace(strings.ToLower(kw))
			if clean != "" && !seenInTenant[clean] {
				seenInTenant[clean] = true
				keywordFrequency[clean]++
			}
		}
	}

	// Step 2: Convert to anonymized tasks sorted by priority (frequency)
	var globalTasks []GlobalResearchTask
	taskIndex := 1
	for kw, freq := range keywordFrequency {
		priority := freq * 10
		task := GlobalResearchTask{
			TaskID:    fmt.Sprintf("global_res_%d_%d", time.Now().Unix(), taskIndex),
			Sector:    categorizeSector(kw),
			Topic:     kw,
			Priority:  priority,
			ScopeType: "GLOBAL", // Strict GLOBAL scope
			CreatedAt: time.Now().UTC(),
		}
		globalTasks = append(globalTasks, task)
		taskIndex++
	}

	return globalTasks, nil
}

func categorizeSector(kw string) string {
	lower := strings.ToLower(kw)
	if strings.Contains(lower, "pendidikan") || strings.Contains(lower, "beasiswa") || strings.Contains(lower, "sekolah") {
		return "Pendidikan"
	}
	if strings.Contains(lower, "lingkungan") || strings.Contains(lower, "sampah") || strings.Contains(lower, "pohon") || strings.Contains(lower, "emisi") {
		return "Lingkungan & Keberlanjutan"
	}
	if strings.Contains(lower, "kesehatan") || strings.Contains(lower, "stunting") || strings.Contains(lower, "rs") {
		return "Kesehatan"
	}
	if strings.Contains(lower, "umkm") || strings.Contains(lower, "ekonomi") || strings.Contains(lower, "pemberdayaan") {
		return "Pemberdayaan Ekonomi"
	}
	return "Umum"
}
