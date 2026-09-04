package hitl

import (
	"strings"
	"testing"
	"time"
)

func TestHITL_RequestAndPop(t *testing.T) {
	mgr := NewManager(10, time.Hour)

	details := map[string]any{
		"to":      "boss@mail.ru",
		"subject": "Status Report",
	}

	resp := mgr.Request("mail_send", details)
	if !strings.Contains(resp, "ACTION BLOCKED BY HITL") {
		t.Fatalf("expected blocked response, got: %s", resp)
	}

	// Extract token
	idx := strings.Index(resp, "token='")
	if idx == -1 {
		t.Fatalf("token not found in response: %s", resp)
	}
	token := resp[idx+7 : idx+7+32]

	if mgr.Len() != 1 {
		t.Fatalf("expected 1 pending action, got %d", mgr.Len())
	}

	act, err := mgr.Pop(token)
	if err != nil {
		t.Fatalf("unexpected error popping token: %v", err)
	}

	if act.Type != "mail_send" {
		t.Errorf("expected mail_send, got %s", act.Type)
	}
	if act.Details["to"] != "boss@mail.ru" {
		t.Errorf("expected boss@mail.ru, got %v", act.Details["to"])
	}

	// Second pop must fail (single-use token)
	_, err = mgr.Pop(token)
	if err == nil {
		t.Fatal("expected error on duplicate pop, got nil")
	}

	if mgr.Len() != 0 {
		t.Fatalf("expected 0 pending actions, got %d", mgr.Len())
	}
}

func TestHITL_Expiration(t *testing.T) {
	mgr := NewManager(10, 20*time.Millisecond)

	resp := mgr.Request("dav_delete", map[string]any{"path": "/old_file.txt"})
	idx := strings.Index(resp, "token='")
	token := resp[idx+7 : idx+7+32]

	time.Sleep(30 * time.Millisecond)

	cleaned := mgr.CleanupExpired()
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned action, got %d", cleaned)
	}

	_, err := mgr.Pop(token)
	if err == nil {
		t.Fatal("expected error popping expired token, got nil")
	}
}

func TestHITL_MaxCapacityEviction(t *testing.T) {
	mgr := NewManager(2, time.Hour)

	resp1 := mgr.Request("act1", map[string]any{"id": 1})
	idx1 := strings.Index(resp1, "token='")
	tok1 := resp1[idx1+7 : idx1+7+32]

	time.Sleep(5 * time.Millisecond)
	resp2 := mgr.Request("act2", map[string]any{"id": 2})
	idx2 := strings.Index(resp2, "token='")
	tok2 := resp2[idx2+7 : idx2+7+32]

	time.Sleep(5 * time.Millisecond)
	resp3 := mgr.Request("act3", map[string]any{"id": 3})
	idx3 := strings.Index(resp3, "token='")
	tok3 := resp3[idx3+7 : idx3+7+32]

	// tok1 should have been evicted because maxPending=2
	_, err := mgr.Pop(tok1)
	if err == nil {
		t.Error("expected tok1 to be evicted, but it was found")
	}

	// tok2 and tok3 should still exist
	act2, err := mgr.Pop(tok2)
	if err != nil || act2.Type != "act2" {
		t.Errorf("tok2 should be present, err: %v", err)
	}
	act3, err := mgr.Pop(tok3)
	if err != nil || act3.Type != "act3" {
		t.Errorf("tok3 should be present, err: %v", err)
	}
}

func TestHITL_Defaults(t *testing.T) {
	mgr := NewManager(0, 0)
	if mgr.maxPending != DefaultMaxPending {
		t.Errorf("expected default max pending %d, got %d", DefaultMaxPending, mgr.maxPending)
	}
	if mgr.ttl != DefaultTTL {
		t.Errorf("expected default TTL %v, got %v", DefaultTTL, mgr.ttl)
	}
}
