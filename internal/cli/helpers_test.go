package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "valid headers",
			input:    []string{"Authorization: Bearer token", "Content-Type: application/json"},
			expected: map[string]string{"Authorization": "Bearer token", "Content-Type": "application/json"},
		},
		{
			name:     "malformed header skipped",
			input:    []string{"Authorization: Bearer token", "invalid-header"},
			expected: map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:     "value with colon",
			input:    []string{"Header: value:with:colons"},
			expected: map[string]string{"Header": "value:with:colons"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaders(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
