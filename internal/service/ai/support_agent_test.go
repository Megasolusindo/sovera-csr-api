package ai

import (
	"context"
	"strings"
	"testing"
)

func TestSupportAgent_Success(t *testing.T) {
	cm := NewCostMonitor()
	sa := NewSupportAgent(nil, cm)

	stc := &SignedTaskContext{
		TaskID:    "task_support_1",
		OrgID:     "org_ngo_99",
		ScopeType: ScopeTenant,
	}

	res, err := sa.RunSupportAssistant(context.Background(), stc, "Cari sinyal CSR BSI 2026")
	if err != nil {
		t.Fatalf("unexpected error running support assistant: %v", err)
	}

	if res.OrgID != "org_ngo_99" {
		t.Errorf("expected OrgID org_ngo_99, got: %s", res.OrgID)
	}
	if len(res.ToolsUsed) != 2 {
		t.Errorf("expected 2 tools used, got %d", len(res.ToolsUsed))
	}
}

func TestSupportAgent_SideEffectBlocked(t *testing.T) {
	cm := NewCostMonitor()
	sa := NewSupportAgent(nil, cm)

	stc := &SignedTaskContext{
		TaskID:    "task_support_2",
		OrgID:     "org_ngo_99",
		ScopeType: ScopeTenant,
	}

	// Requesting side effect like trigger_webhook must be blocked!
	_, err := sa.RunSupportAssistant(context.Background(), stc, "Tolong trigger_webhook ke email massal")
	if err == nil || !strings.Contains(err.Error(), "external side effects") {
		t.Fatalf("expected ErrSupportSideEffectBlocked, got: %v", err)
	}
}

func TestSupportAgent_GlobalScopeRejection(t *testing.T) {
	sa := NewSupportAgent(nil, nil)

	stc := &SignedTaskContext{
		TaskID:    "task_support_3",
		ScopeType: ScopeGlobal,
	}

	_, err := sa.RunSupportAssistant(context.Background(), stc, "Cari sinyal CSR")
	if err == nil || !strings.Contains(err.Error(), "requires ScopeTenant") {
		t.Fatalf("expected ScopeTenant requirement error, got: %v", err)
	}
}
