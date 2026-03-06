// Package strategy defines the interface for pluggable allocation strategies.
// Strategies control how the dashboard routes requests to browser instances.
package strategy

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Orchestrator is the interface strategies need from the orchestrator.
// Avoids import cycles by depending on the interface, not the concrete type.
type Orchestrator interface {
	RegisterHandlers(mux *http.ServeMux)
	FirstRunningURL() string
}

// Strategy defines a browser allocation approach.
type Strategy interface {
	// Name returns the strategy identifier.
	Name() string

	// RegisterRoutes adds strategy-specific HTTP endpoints to the mux.
	RegisterRoutes(mux *http.ServeMux)

	// Start begins any background tasks.
	Start(ctx context.Context) error

	// Stop gracefully shuts down.
	Stop() error
}

// Factory creates a new Strategy instance.
type Factory func() Strategy

var (
	registry = make(map[string]Factory)
	mu       sync.RWMutex
)

// Register adds a strategy factory to the registry.
func Register(name string, factory Factory) error {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		return fmt.Errorf("strategy %q already registered", name)
	}
	registry[name] = factory
	return nil
}

// MustRegister is like Register but panics on duplicate name.
func MustRegister(name string, factory Factory) {
	if err := Register(name, factory); err != nil {
		panic(err)
	}
}

// New creates a strategy by name from the registry.
func New(name string) (Strategy, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown strategy: %s (available: %v)", name, Names())
	}
	return factory(), nil
}

// Names returns all registered strategy names.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
