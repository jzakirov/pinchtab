package orchestrator

import (
	"testing"
	"time"

	"github.com/pinchtab/pinchtab/internal/config"
)

func newTestOrchestrator() *Orchestrator {
	return &Orchestrator{
		runtimeCfg: &config.RuntimeConfig{
			Token: "test-secret-token",
		},
	}
}

func TestPresignedURL_SignAndVerify(t *testing.T) {
	o := newTestOrchestrator()

	token, err := o.signPayload("instance-123", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	payload, err := o.verifyPayload(token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if payload.InstanceID != "instance-123" {
		t.Fatalf("expected instance-123, got %s", payload.InstanceID)
	}
}

func TestPresignedURL_Expired(t *testing.T) {
	o := newTestOrchestrator()

	token, err := o.signPayload("instance-123", time.Now().Add(-time.Second))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	_, err = o.verifyPayload(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestPresignedURL_Tampered(t *testing.T) {
	o := newTestOrchestrator()

	token, err := o.signPayload("instance-123", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	// Tamper with instance ID
	tampered := "instance-456" + token[len("instance-123"):]
	_, err = o.verifyPayload(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestPresignedURL_DifferentSecret(t *testing.T) {
	o1 := newTestOrchestrator()
	o2 := &Orchestrator{
		runtimeCfg: &config.RuntimeConfig{
			Token: "different-secret",
		},
	}

	token, err := o1.signPayload("instance-123", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	_, err = o2.verifyPayload(token)
	if err == nil {
		t.Fatal("expected error for different secret")
	}
}

func TestPresignedURL_InvalidFormat(t *testing.T) {
	o := newTestOrchestrator()

	for _, bad := range []string{"", "abc", "a:b", "a:b:c:d"} {
		_, err := o.verifyPayload(bad)
		if err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestPresignedURL_RequiresConfiguredToken(t *testing.T) {
	o := &Orchestrator{runtimeCfg: &config.RuntimeConfig{}}

	if _, err := o.signPayload("instance-123", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected signing to fail without configured token")
	}
	if _, err := o.verifyPayload("instance-123:1:deadbeef"); err == nil {
		t.Fatal("expected verification to fail without configured token")
	}
}
