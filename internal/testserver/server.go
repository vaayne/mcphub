// Package testserver provides a simple MCP test server for automated testing.
// It can run as either stdio or HTTP server.
package testserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is a simple MCP test server with predefined tools.
type Server struct {
	server *mcp.Server
}

// New creates a new test server with predefined tools.
func New() *Server {
	s := &Server{}

	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	s.registerTools()

	return s
}

// registerTools registers the predefined test tools.
func (s *Server) registerTools() {
	// Echo tool - returns input as-is
	s.server.AddTool(&mcp.Tool{
		Name:        "echo",
		Description: "Returns the input message",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Message to echo",
				},
			},
			"required": []string{"message"},
		},
	}, s.handleEcho)

	// Add tool - returns sum of two numbers
	s.server.AddTool(&mcp.Tool{
		Name:        "add",
		Description: "Returns the sum of two numbers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{
					"type":        "number",
					"description": "First number",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "Second number",
				},
			},
			"required": []string{"a", "b"},
		},
	}, s.handleAdd)

	// Fail tool - always returns an error
	s.server.AddTool(&mcp.Tool{
		Name:        "fail",
		Description: "Always returns an error (for testing error handling)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Error message to return",
				},
			},
		},
	}, s.handleFail)
}

func (s *Server) handleEcho(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: args.Message},
		},
	}, nil
}

func (s *Server) handleAdd(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	result := args.A + args.B
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("%v", result)},
		},
	}, nil
}

func (s *Server) handleFail(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if len(req.Params.Arguments) > 0 {
		_ = json.Unmarshal(req.Params.Arguments, &args)
	}

	msg := "intentional failure"
	if args.Message != "" {
		msg = args.Message
	}

	return nil, fmt.Errorf("%s", msg)
}

// RunStdio runs the server using stdio transport.
// This is typically called from a test binary or main function.
func (s *Server) RunStdio(ctx context.Context) error {
	transport := &mcp.StdioTransport{}
	return s.server.Run(ctx, transport)
}

// RunHTTP runs the server as an HTTP server on the given address.
// Returns the actual address the server is listening on.
func (s *Server) RunHTTP(ctx context.Context, addr string) (string, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to listen: %w", err)
	}

	actualAddr := listener.Addr().String()

	httpServer := &http.Server{
		Handler: mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return s.server },
			nil,
		),
	}

	// Start server in goroutine
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	// Handle shutdown
	go func() {
		<-ctx.Done()
		_ = httpServer.Close()
	}()

	return actualAddr, nil
}

// StartHTTP starts the test server as HTTP and returns the URL.
// The server is automatically stopped when the test completes.
func StartHTTP(t *testing.T) string {
	t.Helper()

	srv := New()
	ctx, cancel := context.WithCancel(context.Background())

	addr, err := srv.RunHTTP(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}

	t.Cleanup(func() {
		cancel()
	})

	return fmt.Sprintf("http://%s/mcp", addr)
}

// StdioCmd returns the command and args needed to run the test server in stdio mode.
// This requires the testserver binary to be built first (go build ./cmd/testserver).
// The returned command can be used with config.MCPServer.
func StdioCmd() (command string, args []string) {
	return "go", []string{"run", "./cmd/testserver"}
}
