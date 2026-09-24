package ai

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid task context HMAC signature")
	ErrContextExpired   = errors.New("signed task context has expired")
	ErrReplayDetected   = errors.New("nonce replay attack detected")
	ErrScopeViolation   = errors.New("task scope violation detected")
)

type TaskScopeType string

const (
	ScopeGlobal TaskScopeType = "GLOBAL"
	ScopeTenant TaskScopeType = "TENANT"
)

// SignedTaskContext represents the cryptographically signed metadata bound to an AI agent run.
type SignedTaskContext struct {
	TaskID       string        `json:"task_id"`
	RunID        string        `json:"run_id"`
	OrgID        string        `json:"org_id,omitempty"`
	ScopeType    TaskScopeType `json:"scope_type"`
	AllowedTools []string      `json:"allowed_tools"`
	Nonce        string        `json:"nonce"`
	IssuedAt     int64         `json:"issued_at"`
	ExpiresAt    int64         `json:"expires_at"`
}

type TaskSecurityManager struct {
	secretKey  []byte
	nonceStore NonceStore
	defaultTTL time.Duration
}

func NewTaskSecurityManager(secretKey string, nonceStore NonceStore) (*TaskSecurityManager, error) {
	if len(secretKey) < 16 {
		return nil, fmt.Errorf("secretKey must be at least 16 characters long")
	}
	return &TaskSecurityManager{
		secretKey:  []byte(secretKey),
		nonceStore: nonceStore,
		defaultTTL: 5 * time.Minute,
	}, nil
}

// GenerateNonce creates a secure 16-byte random hex string
func GenerateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSignedContext generates a signed token string for the task execution context
func (m *TaskSecurityManager) CreateSignedContext(taskID, runID, orgID string, scope TaskScopeType, allowedTools []string) (string, *SignedTaskContext, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return "", nil, err
	}

	now := time.Now().UTC()
	expires := now.Add(m.defaultTTL)

	// Security Hardening: Global scope tasks must NOT carry orgID
	if scope == ScopeGlobal {
		orgID = ""
	}

	stc := &SignedTaskContext{
		TaskID:       taskID,
		RunID:        runID,
		OrgID:        orgID,
		ScopeType:    scope,
		AllowedTools: allowedTools,
		Nonce:        nonce,
		IssuedAt:     now.Unix(),
		ExpiresAt:    expires.Unix(),
	}

	payloadBytes, err := json.Marshal(stc)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal task context: %w", err)
	}

	mac := hmac.New(sha256.New, m.secretKey)
	mac.Write(payloadBytes)
	signature := mac.Sum(nil)

	token := fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(payloadBytes),
		base64.RawURLEncoding.EncodeToString(signature),
	)

	return token, stc, nil
}

// VerifyAndParseContext verifies HMAC signature, TTL, nonce uniqueness, and returns context
func (m *TaskSecurityManager) VerifyAndParseContext(tokenStr string) (*SignedTaskContext, error) {
	parts := splitToken(tokenStr)
	if len(parts) != 2 {
		return nil, ErrInvalidSignature
	}
	payloadB64, sigB64 := parts[0], parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrInvalidSignature
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, m.secretKey)
	mac.Write(payloadBytes)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expectedSig) {
		return nil, ErrInvalidSignature
	}

	var stc SignedTaskContext
	if err := json.Unmarshal(payloadBytes, &stc); err != nil {
		return nil, fmt.Errorf("invalid payload json: %w", err)
	}

	// 1. Check Expiry
	if time.Now().UTC().Unix() > stc.ExpiresAt {
		return nil, ErrContextExpired
	}

	// 2. Check Global Scope Hardening (Global must never carry OrgID)
	if stc.ScopeType == ScopeGlobal && stc.OrgID != "" {
		return nil, ErrScopeViolation
	}

	// 3. Verify Nonce Replay via NonceStore
	if m.nonceStore != nil {
		valid, err := m.nonceStore.VerifyAndMarkNonce(stc.Nonce, m.defaultTTL)
		if err != nil {
			return nil, fmt.Errorf("nonce verification failed: %w", err)
		}
		if !valid {
			return nil, ErrReplayDetected
		}
	}

	return &stc, nil
}

func splitToken(token string) []string {
	idx := -1
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	return []string{token[:idx], token[idx+1:]}
}
