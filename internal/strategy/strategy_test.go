package strategy_test

import (
	"testing"

	"github.com/pinchtab/pinchtab/internal/strategy"

	// Register strategies via init()
	_ "github.com/pinchtab/pinchtab/internal/strategy/explicit"
	_ "github.com/pinchtab/pinchtab/internal/strategy/simple"
)

func TestRegistry_ExplicitRegistered(t *testing.T) {
	s, err := strategy.New("explicit")
	if err != nil {
		t.Fatalf("explicit strategy not registered: %v", err)
	}
	if s.Name() != "explicit" {
		t.Errorf("expected name 'explicit', got %q", s.Name())
	}
}

func TestRegistry_SimpleRegistered(t *testing.T) {
	s, err := strategy.New("simple")
	if err != nil {
		t.Fatalf("simple strategy not registered: %v", err)
	}
	if s.Name() != "simple" {
		t.Errorf("expected name 'simple', got %q", s.Name())
	}
}

func TestRegistry_UnknownStrategy(t *testing.T) {
	_, err := strategy.New("nonexistent")
	if err == nil {
		t.Error("expected error for unknown strategy")
	}
}

func TestRegistry_Names(t *testing.T) {
	names := strategy.Names()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["explicit"] {
		t.Error("explicit not in names")
	}
	if !found["simple"] {
		t.Error("simple not in names")
	}
}
