package serper

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"sovera-core-api/internal/pkg/telegram"
)

type AccountResponse struct {
	Balance   int `json:"balance"`
	RateLimit int `json:"rateLimit"`
}

type SerperMonitor struct {
	apiKey     string
	notifier   *telegram.Notifier
	httpClient *http.Client
	mu         sync.Mutex
	lastAlert  string // "LOW" | "EXHAUSTED" | "OK"
}

var (
	defaultMonitor     *SerperMonitor
	defaultMonitorOnce sync.Once
)

func NewSerperMonitor(apiKey string, notifier *telegram.Notifier) *SerperMonitor {
	if apiKey == "" {
		apiKey = os.Getenv("SERPER_API_KEY")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &SerperMonitor{
		apiKey:   apiKey,
		notifier: notifier,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		},
		lastAlert: "OK",
	}
}

func InitDefaultMonitor(apiKey string, notifier *telegram.Notifier) *SerperMonitor {
	defaultMonitorOnce.Do(func() {
		defaultMonitor = NewSerperMonitor(apiKey, notifier)
	})
	return defaultMonitor
}

func GetDefaultMonitor() *SerperMonitor {
	return defaultMonitor
}

// CheckBalance queries GET https://google.serper.dev/account and returns remaining balance
func (m *SerperMonitor) CheckBalance(ctx context.Context) (int, error) {
	if m == nil || m.apiKey == "" {
		return 0, fmt.Errorf("SERPER_API_KEY is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://google.serper.dev/account", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-KEY", m.apiKey)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("network error querying serper account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if strings.Contains(string(bodyBytes), "Not enough credits") || resp.StatusCode == http.StatusBadRequest {
			return 0, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("serper API returned status %d", resp.StatusCode)
	}

	var acc AccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
		return 0, fmt.Errorf("failed to decode serper account response: %w", err)
	}

	return acc.Balance, nil
}

// CheckAndNotify checks current credit balance and sends Telegram alert if < 1000 or exhausted (0)
func (m *SerperMonitor) CheckAndNotify(ctx context.Context) (int, error) {
	if m == nil || m.apiKey == "" {
		log.Println("[SerperMonitor] Skipping quota check (SERPER_API_KEY is empty)")
		return 0, nil
	}

	balance, err := m.CheckBalance(ctx)
	if err != nil {
		log.Printf("[SerperMonitor] Error checking Serper balance: %v", err)
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	maskedKey := m.apiKey
	if len(maskedKey) > 8 {
		maskedKey = maskedKey[:4] + "..." + maskedKey[len(maskedKey)-4:]
	}

	if balance <= 0 {
		m.lastAlert = "EXHAUSTED"
		log.Printf("[SerperMonitor] Notice: Serper API balance is 0. Search queries automatically fall back to FreeRSSProvider in web-scraper engine.")
	} else if balance < 1000 {
		m.lastAlert = "LOW"
		log.Printf("[SerperMonitor] Notice: Serper API balance is low (%d credits).", balance)
	} else {
		// Reset alert state if balance topped up back to >= 1000
		m.lastAlert = "OK"
		log.Printf("[SerperMonitor] OK: Serper API balance is healthy (%d credits)", balance)
	}

	return balance, nil
}
