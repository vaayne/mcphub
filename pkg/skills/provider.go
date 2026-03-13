package skills

import (
	"context"
	"net/http"
	"sync"
)

// ProviderMatch represents the result of matching a URL to a provider.
type ProviderMatch struct {
	Matches          bool
	SourceIdentifier string
}

// HostProvider interface for HTTP-based skill providers.
// Implementations: Mintlify, HuggingFace, Direct URL.
// Note: Git-based sources (GitHub, GitLab) use FetchGitSkill(), not this interface.
type HostProvider interface {
	// ID returns a unique identifier for this provider (e.g., "mintlify", "huggingface")
	ID() string

	// DisplayName returns a human-readable name for this provider
	DisplayName() string

	// Match checks if a URL belongs to this provider
	Match(url string) ProviderMatch

	// FetchSkill fetches and parses a skill from the given URL
	FetchSkill(ctx context.Context, url string, client HTTPClient) (*RemoteSkill, error)

	// ToRawURL converts a user-facing URL to a raw content URL
	ToRawURL(url string) string

	// GetSourceIdentifier returns a stable identifier for telemetry/storage
	GetSourceIdentifier(url string) string
}

// HTTPClient interface for HTTP operations (allows mocking in tests)
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ProviderRegistry manages registered HTTP providers.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers []HostProvider
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make([]HostProvider, 0),
	}
}

// Register adds a provider to the registry.
func (r *ProviderRegistry) Register(p HostProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, p)
}

// FindProvider finds a provider that matches the given URL.
// Returns nil if no provider matches.
func (r *ProviderRegistry) FindProvider(url string) HostProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.providers {
		if match := p.Match(url); match.Matches {
			return p
		}
	}
	return nil
}

// Providers returns all registered providers.
func (r *ProviderRegistry) Providers() []HostProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]HostProvider, len(r.providers))
	copy(result, r.providers)
	return result
}

// DefaultRegistry is the global provider registry.
var DefaultRegistry = NewProviderRegistry()

// RegisterProvider adds a provider to the default registry.
func RegisterProvider(p HostProvider) {
	DefaultRegistry.Register(p)
}

// FindProvider finds a provider in the default registry.
func FindProvider(url string) HostProvider {
	return DefaultRegistry.FindProvider(url)
}
