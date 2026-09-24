package ai

import (
	"context"
	"strings"
	"testing"
)

func TestSanitizeAndValidateURL_SSRFBlocked(t *testing.T) {
	blockedURLs := []string{
		"http://localhost:8080/admin",
		"http://127.0.0.1/secret",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/db",
		"http://192.168.1.1/router",
	}

	for _, raw := range blockedURLs {
		_, err := SanitizeAndValidateURL(raw)
		if err != ErrSSRFForbidden {
			t.Errorf("expected ErrSSRFForbidden for %s, got: %v", raw, err)
		}
	}
}

func TestSanitizeAndValidateURL_EgressSanitation(t *testing.T) {
	raw := "https://example.com/news?page=1&token=secret123&tenant_id=org_abc&topic=csr"
	u, err := SanitizeAndValidateURL(raw)
	if err != nil {
		t.Fatalf("unexpected error sanitizing URL: %v", err)
	}

	query := u.Query()
	if query.Get("token") != "" || query.Get("tenant_id") != "" {
		t.Errorf("sensitive parameters were not stripped: %s", u.String())
	}
	if query.Get("page") != "1" || query.Get("topic") != "csr" {
		t.Errorf("safe parameters were erroneously removed: %s", u.String())
	}
}

func TestDemandAggregator_Depersonalize(t *testing.T) {
	da := NewDemandAggregator()

	rawWatchlists := map[string][]string{
		"tenant_ngo_1": {"beasiswa pendidikan", "stunting anak"},
		"tenant_ngo_2": {"beasiswa pendidikan", "sampah plastik"},
		"tenant_ngo_3": {"beasiswa pendidikan", "stunting anak", "umkm batik"},
	}

	tasks, err := da.DepersonalizeAndAggregate(context.Background(), rawWatchlists)
	if err != nil {
		t.Fatalf("unexpected error aggregating demand: %v", err)
	}

	if len(tasks) != 4 {
		t.Fatalf("expected 4 unique aggregated tasks, got: %d", len(tasks))
	}

	// Verify highest demand task is "beasiswa pendidikan" with priority 30
	var topTask GlobalResearchTask
	for _, task := range tasks {
		if task.ScopeType != "GLOBAL" {
			t.Errorf("task scope must be GLOBAL, got %s", task.ScopeType)
		}
		if task.Topic == "beasiswa pendidikan" {
			topTask = task
		}
	}

	if topTask.Priority != 30 {
		t.Errorf("expected priority 30 for top task, got: %d", topTask.Priority)
	}
}

func TestResearchAgent_TenantScopeRejection(t *testing.T) {
	ra := NewResearchAgent(nil, nil)

	// Attempting to run research agent with TENANT scope should fail
	tenantSTC := &SignedTaskContext{
		TaskID:    "task_tenant",
		ScopeType: ScopeTenant,
		OrgID:     "org_secret",
	}

	_, err := ra.RunGlobalResearch(context.Background(), tenantSTC, "https://example.com")
	if !strings.Contains(err.Error(), "scope must be ScopeGlobal") {
		t.Fatalf("expected scope rejection error, got: %v", err)
	}
}
