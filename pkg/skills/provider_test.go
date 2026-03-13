package skills

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements HostProvider for testing
type mockProvider struct {
	id          string
	displayName string
	matchFunc   func(string) ProviderMatch
}

func (m *mockProvider) ID() string          { return m.id }
func (m *mockProvider) DisplayName() string { return m.displayName }
func (m *mockProvider) Match(url string) ProviderMatch {
	if m.matchFunc != nil {
		return m.matchFunc(url)
	}
	return ProviderMatch{Matches: false}
}
func (m *mockProvider) FetchSkill(ctx context.Context, url string, client HTTPClient) (*RemoteSkill, error) {
	return nil, nil
}
func (m *mockProvider) ToRawURL(url string) string            { return url }
func (m *mockProvider) GetSourceIdentifier(url string) string { return m.id }

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Initially empty
	assert.Nil(t, registry.FindProvider("https://example.com"))
	assert.Empty(t, registry.Providers())

	// Register a provider
	provider1 := &mockProvider{
		id:          "test1",
		displayName: "Test Provider 1",
		matchFunc: func(url string) ProviderMatch {
			if url == "https://test1.com/skill.md" {
				return ProviderMatch{Matches: true, SourceIdentifier: "test1/com"}
			}
			return ProviderMatch{Matches: false}
		},
	}
	registry.Register(provider1)

	// Find registered provider
	found := registry.FindProvider("https://test1.com/skill.md")
	require.NotNil(t, found)
	assert.Equal(t, "test1", found.ID())

	// No match returns nil
	assert.Nil(t, registry.FindProvider("https://other.com/skill.md"))

	// Providers() returns copy
	providers := registry.Providers()
	assert.Len(t, providers, 1)
	assert.Equal(t, "test1", providers[0].ID())
}

func TestProviderRegistryPriority(t *testing.T) {
	registry := NewProviderRegistry()

	// First registered provider wins on conflict
	provider1 := &mockProvider{
		id: "first",
		matchFunc: func(url string) ProviderMatch {
			return ProviderMatch{Matches: true}
		},
	}
	provider2 := &mockProvider{
		id: "second",
		matchFunc: func(url string) ProviderMatch {
			return ProviderMatch{Matches: true}
		},
	}

	registry.Register(provider1)
	registry.Register(provider2)

	// First registered wins
	found := registry.FindProvider("https://any.com")
	require.NotNil(t, found)
	assert.Equal(t, "first", found.ID())
}
