package testserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Echo(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Call echo tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Arguments: map[string]any{
			"message": "hello world",
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "hello world", textContent.Text)
}

func TestServer_Add(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Call add tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add",
		Arguments: map[string]any{
			"a": 10,
			"b": 32,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "42", textContent.Text)
}

func TestServer_Fail(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Call fail tool without custom message
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fail",
		Arguments: map[string]any{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "intentional failure")

	// Call fail tool with custom message
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fail",
		Arguments: map[string]any{
			"message": "custom error",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom error")
}

func TestServer_ListTools(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// List tools
	result, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}

	assert.Contains(t, toolNames, "echo")
	assert.Contains(t, toolNames, "add")
	assert.Contains(t, toolNames, "fail")
}

func TestServer_AddWithFloats(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Call add tool with floats
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add",
		Arguments: map[string]any{
			"a": 1.5,
			"b": 2.5,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "4", textContent.Text)
}

func TestServer_EchoWithJSON(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Echo a JSON string
	jsonMsg := `{"key": "value", "nested": {"arr": [1, 2, 3]}}`
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Arguments: map[string]any{
			"message": jsonMsg,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, jsonMsg, textContent.Text)

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal([]byte(textContent.Text), &parsed)
	require.NoError(t, err)
}

func TestServer_EchoWithUnicode(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Echo unicode message
	unicodeMsg := "Hello 世界 🌍 مرحبا"
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Arguments: map[string]any{
			"message": unicodeMsg,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, unicodeMsg, textContent.Text)
}

func TestServer_EchoWithLargeMessage(t *testing.T) {
	url := StartHTTP(t)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err)
	defer session.Close()

	// Echo large message (64KB - reasonable for HTTP)
	largeMsg := strings.Repeat("a", 64*1024)
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Arguments: map[string]any{
			"message": largeMsg,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, len(largeMsg), len(textContent.Text))
}
