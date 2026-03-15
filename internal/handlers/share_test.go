package handlers

import (
	"testing"
	"time"
)

func TestShareTokenStore_IssueAndValidate(t *testing.T) {
	store := NewShareTokenStore()

	token, err := store.Issue("tab-123", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if token.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if token.TabID != "tab-123" {
		t.Fatalf("expected tabId tab-123, got %s", token.TabID)
	}

	// Validate should succeed
	tabID, ok := store.Validate(token.Token)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if tabID != "tab-123" {
		t.Fatalf("expected tab-123, got %s", tabID)
	}

	// Invalid token should fail
	_, ok = store.Validate("nonexistent")
	if ok {
		t.Fatal("expected invalid token to fail validation")
	}
}

func TestShareTokenStore_Revoke(t *testing.T) {
	store := NewShareTokenStore()

	token, _ := store.Issue("tab-1", time.Hour)

	// Should be valid before revocation
	_, ok := store.Validate(token.Token)
	if !ok {
		t.Fatal("expected valid before revoke")
	}

	store.Revoke(token.Token)

	// Should be invalid after revocation
	_, ok = store.Validate(token.Token)
	if ok {
		t.Fatal("expected invalid after revoke")
	}
}

func TestShareTokenStore_Expiry(t *testing.T) {
	store := NewShareTokenStore()

	token, _ := store.Issue("tab-1", 1*time.Millisecond)

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	_, ok := store.Validate(token.Token)
	if ok {
		t.Fatal("expected expired token to be invalid")
	}
}

func TestInputEvent_MouseButton(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"left", "left"},
		{"middle", "middle"},
		{"right", "right"},
		{"", "left"}, // default
		{"unknown", "left"},
	}
	for _, tt := range tests {
		btn := toMouseButton(tt.input)
		if string(btn) != tt.expected {
			t.Errorf("toMouseButton(%q) = %q, want %q", tt.input, btn, tt.expected)
		}
	}
}
