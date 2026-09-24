package ai

import (
	"testing"
)

func TestSignedTaskContext_Success(t *testing.T) {
	nonceStore := NewInMemoryNonceStore()
	mgr, err := NewTaskSecurityManager("super-secret-key-12345", nonceStore)
	if err != nil {
		t.Fatalf("unexpected error creating security manager: %v", err)
	}

	token, stc, err := mgr.CreateSignedContext("task_123", "run_456", "org_789", ScopeTenant, []string{"search_companies"})
	if err != nil {
		t.Fatalf("failed to create signed context: %v", err)
	}

	if token == "" || stc == nil {
		t.Fatalf("token or stc is empty")
	}

	parsed, err := mgr.VerifyAndParseContext(token)
	if err != nil {
		t.Fatalf("failed to parse valid token: %v", err)
	}

	if parsed.TaskID != "task_123" || parsed.RunID != "run_456" || parsed.OrgID != "org_789" || parsed.ScopeType != ScopeTenant {
		t.Errorf("parsed fields mismatch: %+v", parsed)
	}
}

func TestSignedTaskContext_GlobalScopeIsolation(t *testing.T) {
	nonceStore := NewInMemoryNonceStore()
	mgr, _ := NewTaskSecurityManager("super-secret-key-12345", nonceStore)

	// Attempting to pass orgID to GLOBAL scope must sanitize orgID to empty string
	token, stc, err := mgr.CreateSignedContext("task_999", "run_999", "secret_org_leak", ScopeGlobal, []string{"web_fetch"})
	if err != nil {
		t.Fatalf("failed to create signed context: %v", err)
	}

	if stc.OrgID != "" {
		t.Errorf("expected orgID to be empty for GLOBAL scope, got: %s", stc.OrgID)
	}

	parsed, err := mgr.VerifyAndParseContext(token)
	if err != nil {
		t.Fatalf("failed to verify global context: %v", err)
	}
	if parsed.OrgID != "" {
		t.Errorf("parsed orgID should be empty for GLOBAL scope")
	}
}

func TestSignedTaskContext_ReplayAttack(t *testing.T) {
	nonceStore := NewInMemoryNonceStore()
	mgr, _ := NewTaskSecurityManager("super-secret-key-12345", nonceStore)

	token, _, err := mgr.CreateSignedContext("task_replay", "run_1", "", ScopeGlobal, nil)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// First verification -> Should pass
	_, err = mgr.VerifyAndParseContext(token)
	if err != nil {
		t.Fatalf("first verification failed: %v", err)
	}

	// Second verification with same token (nonce replay) -> Must fail!
	_, err = mgr.VerifyAndParseContext(token)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected on second verification, got: %v", err)
	}
}

func TestSignedTaskContext_TamperedToken(t *testing.T) {
	nonceStore := NewInMemoryNonceStore()
	mgr, _ := NewTaskSecurityManager("super-secret-key-12345", nonceStore)

	token, _, _ := mgr.CreateSignedContext("task_tamper", "run_1", "org_1", ScopeTenant, nil)
	tamperedToken := token + "tampered"

	_, err := mgr.VerifyAndParseContext(tamperedToken)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for tampered token, got: %v", err)
	}
}
