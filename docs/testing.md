# Testing Guide

This guide covers testing practices for MCP Hub, including the built-in test server for automated testing.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/testserver

# Run tests with race detection
go test -race ./...
```

## Test Server

The `internal/testserver` package provides a simple MCP server for automated testing. It supports both HTTP and stdio transports.

### Available Test Tools

| Tool   | Input                    | Output               | Purpose                 |
| ------ | ------------------------ | -------------------- | ----------------------- |
| `echo` | `{message: string}`      | Returns the message  | Basic roundtrip testing |
| `add`  | `{a: number, b: number}` | Returns `a + b`      | Parameter validation    |
| `fail` | `{message?: string}`     | Always returns error | Error handling tests    |

### Usage in Tests

#### HTTP Transport

Use `testserver.StartHTTP(t)` for HTTP-based tests. The server is automatically cleaned up when the test completes.

```go
package mypackage

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/vaayne/mcphub/internal/testserver"
)

func TestWithHTTPServer(t *testing.T) {
    // Start test server - automatically cleaned up
    url := testserver.StartHTTP(t)

    // Create MCP client
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
    if err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    defer session.Close()

    // Call echo tool
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name: "echo",
        Arguments: map[string]any{
            "message": "hello",
        },
    })
    if err != nil {
        t.Fatalf("tool call failed: %v", err)
    }

    // Verify result
    textContent := result.Content[0].(*mcp.TextContent)
    if textContent.Text != "hello" {
        t.Errorf("expected 'hello', got %q", textContent.Text)
    }
}
```

#### Stdio Transport

For stdio-based tests, use `testserver.StdioCmd()` to get the command configuration:

```go
package mypackage

import (
    "testing"

    "github.com/vaayne/mcphub/internal/config"
    "github.com/vaayne/mcphub/internal/testserver"
)

func TestWithStdioServer(t *testing.T) {
    cmd, args := testserver.StdioCmd()

    // Use with config.MCPServer
    serverCfg := config.MCPServer{
        Transport: "stdio",
        Command:   cmd,
        Args:      args,
    }

    // Connect using your client manager...
}
```

### Manual Testing

Build and run the test server binary for manual testing:

```bash
# Build the test server
go build -o bin/testserver ./cmd/testserver

# Run as HTTP server
./bin/testserver -http :8080
# Server listens at http://127.0.0.1:8080/mcp

# Run as stdio server (for piping)
./bin/testserver
```

### Programmatic Server Control

For more control, create a server instance directly:

```go
package mypackage

import (
    "context"
    "testing"

    "github.com/vaayne/mcphub/internal/testserver"
)

func TestCustomServer(t *testing.T) {
    srv := testserver.New()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start HTTP server on random port
    addr, err := srv.RunHTTP(ctx, "127.0.0.1:0")
    if err != nil {
        t.Fatalf("failed to start server: %v", err)
    }

    url := "http://" + addr + "/mcp"
    // Use url for testing...
}
```

## Test Patterns

### Testing Error Handling

Use the `fail` tool to test error handling:

```go
func TestErrorHandling(t *testing.T) {
    url := testserver.StartHTTP(t)
    // ... setup client ...

    // Test default error message
    _, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name:      "fail",
        Arguments: map[string]any{},
    })
    if err == nil {
        t.Error("expected error, got nil")
    }
    if !strings.Contains(err.Error(), "intentional failure") {
        t.Errorf("unexpected error message: %v", err)
    }

    // Test custom error message
    _, err = session.CallTool(ctx, &mcp.CallToolParams{
        Name: "fail",
        Arguments: map[string]any{
            "message": "custom error",
        },
    })
    if !strings.Contains(err.Error(), "custom error") {
        t.Errorf("unexpected error message: %v", err)
    }
}
```

### Testing Tool Discovery

```go
func TestToolDiscovery(t *testing.T) {
    url := testserver.StartHTTP(t)
    // ... setup client ...

    result, err := session.ListTools(ctx, nil)
    if err != nil {
        t.Fatalf("ListTools failed: %v", err)
    }

    // Verify expected tools exist
    toolNames := make(map[string]bool)
    for _, tool := range result.Tools {
        toolNames[tool.Name] = true
    }

    expected := []string{"echo", "add", "fail"}
    for _, name := range expected {
        if !toolNames[name] {
            t.Errorf("expected tool %q not found", name)
        }
    }
}
```

### Testing with Timeouts

```go
func TestWithTimeout(t *testing.T) {
    url := testserver.StartHTTP(t)
    // ... setup client ...

    // Use short timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    _, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name: "echo",
        Arguments: map[string]any{
            "message": "test",
        },
    })
    // Handle timeout or success...
}
```

## Integration Tests

The `internal/server/integration_test.go` contains comprehensive integration tests using the mock server from `internal/testing`. These tests cover:

- Mock server basic functionality
- Multiple tools handling
- JavaScript execution with tool calls
- Concurrent tool calls
- Context cancellation
- Namespace collision handling
- Built-in tool validation

## Best Practices

1. **Use `t.Helper()`** in test helper functions for better error reporting
2. **Use `t.Cleanup()`** for resource cleanup instead of `defer`
3. **Set timeouts** on contexts to prevent hanging tests
4. **Use table-driven tests** for testing multiple inputs
5. **Test error cases** not just happy paths
6. **Use `t.Parallel()`** for independent tests to speed up test runs

## Troubleshooting

### Port Already in Use

If you see "address already in use" errors, ensure you're using port `0` for random port assignment:

```go
addr, err := srv.RunHTTP(ctx, "127.0.0.1:0")
```

### Test Timeouts

If tests hang, ensure you're:

- Setting context timeouts
- Calling `cancel()` on context
- Closing sessions with `session.Close()`

### Flaky Tests

For flaky tests:

- Increase timeouts for slow CI environments
- Use `t.Parallel()` carefully with shared resources
- Ensure proper cleanup between tests
