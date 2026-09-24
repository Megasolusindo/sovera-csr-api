package ai

import (
	"errors"
	"fmt"
	"testing"
)

func TestCostMonitor_TaskLimit(t *testing.T) {
	cm := NewCostMonitor()

	// Track under limit
	err := cm.TrackUsage("task_1", "tenant_a", 0.30)
	if err != nil {
		t.Fatalf("unexpected error tracking cost: %v", err)
	}

	// Exceeding $0.50 task cap
	err = cm.TrackUsage("task_1", "tenant_a", 0.25)
	if !errors.Is(err, ErrTaskCostCapExceeded) {
		t.Fatalf("expected ErrTaskCostCapExceeded, got: %v", err)
	}
}

func TestCostMonitor_TenantDailyLimit(t *testing.T) {
	cm := NewCostMonitor()

	// Track multi-task usage for tenant up to $10.00 (20 tasks of $0.50)
	for i := 0; i < 20; i++ {
		taskID := fmt.Sprintf("task_%d", i)
		if err := cm.TrackUsage(taskID, "tenant_b", 0.50); err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
	}

	// Next task (21st task) exceeds daily tenant budget ($10.00)
	err := cm.TrackUsage("task_21", "tenant_b", 0.50)
	if !errors.Is(err, ErrTenantBudgetExceeded) {
		t.Fatalf("expected ErrTenantBudgetExceeded, got: %v", err)
	}
}

func TestCostMonitor_GlobalKillSwitch(t *testing.T) {
	cm := NewCostMonitor()
	cm.SetGlobalCircuitBreaker(true)

	err := cm.TrackUsage("task_1", "tenant_c", 0.01)
	if !errors.Is(err, ErrCircuitBreakerTripped) {
		t.Fatalf("expected ErrCircuitBreakerTripped, got: %v", err)
	}
}
