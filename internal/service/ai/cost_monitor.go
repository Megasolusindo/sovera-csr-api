package ai

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrTaskCostCapExceeded   = errors.New("circuit breaker triggered: task cost exceeded $0.50 USD hard cap")
	ErrTenantBudgetExceeded  = errors.New("circuit breaker triggered: tenant daily budget exceeded $10.00 USD hard cap")
	ErrCircuitBreakerTripped = errors.New("circuit breaker is open: agent operations currently halted")
)

const (
	MaxTaskCostUSD   = 0.50  // $0.50 per task
	MaxTenantDailyUSD = 10.00 // $10.00 per tenant per day
)

// CostMonitor handles real-time token cost tracking and hard-cap circuit breaking.
type CostMonitor struct {
	mu            sync.RWMutex
	taskCosts     map[string]float64            // taskID -> total USD cost
	tenantDaily   map[string]map[string]float64 // tenantID -> YYYY-MM-DD -> total USD cost
	isCircuitOpen bool
}

func NewCostMonitor() *CostMonitor {
	return &CostMonitor{
		taskCosts:   make(map[string]float64),
		tenantDaily: make(map[string]map[string]float64),
	}
}

// TrackUsage records cost for a task run and evaluates circuit breaker thresholds
func (c *CostMonitor) TrackUsage(taskID, tenantID string, costUSD float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isCircuitOpen {
		return ErrCircuitBreakerTripped
	}

	// 1. Update Task Cost
	currentTaskCost := c.taskCosts[taskID] + costUSD
	c.taskCosts[taskID] = currentTaskCost

	if currentTaskCost > MaxTaskCostUSD {
		return fmt.Errorf("%w (accumulated: $%.4f)", ErrTaskCostCapExceeded, currentTaskCost)
	}

	// 2. Update Tenant Daily Cost (if tenantID provided)
	if tenantID != "" {
		today := time.Now().UTC().Format("2006-01-02")
		if c.tenantDaily[tenantID] == nil {
			c.tenantDaily[tenantID] = make(map[string]float64)
		}
		currentTenantCost := c.tenantDaily[tenantID][today] + costUSD
		c.tenantDaily[tenantID][today] = currentTenantCost

		if currentTenantCost > MaxTenantDailyUSD {
			return fmt.Errorf("%w (tenant %s accumulated: $%.4f)", ErrTenantBudgetExceeded, tenantID, currentTenantCost)
		}
	}

	return nil
}

// CheckLimit verifies whether a task can proceed without exceeding limits
func (c *CostMonitor) CheckLimit(taskID, tenantID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isCircuitOpen {
		return ErrCircuitBreakerTripped
	}

	if cost, exists := c.taskCosts[taskID]; exists && cost >= MaxTaskCostUSD {
		return ErrTaskCostCapExceeded
	}

	if tenantID != "" {
		today := time.Now().UTC().Format("2006-01-02")
		if tenantCosts, exists := c.tenantDaily[tenantID]; exists {
			if cost, ok := tenantCosts[today]; ok && cost >= MaxTenantDailyUSD {
				return ErrTenantBudgetExceeded
			}
		}
	}

	return nil
}

// SetGlobalCircuitBreaker manually trips or resets the global kill switch
func (c *CostMonitor) SetGlobalCircuitBreaker(open bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isCircuitOpen = open
}

func (c *CostMonitor) IsCircuitOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCircuitOpen
}
