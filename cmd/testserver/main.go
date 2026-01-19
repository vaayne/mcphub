// Command testserver runs the MCP test server for automated testing.
// It supports both stdio and HTTP transports.
//
// Usage:
//
//	testserver              # Run as stdio server
//	testserver -http :8080  # Run as HTTP server on port 8080
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/vaayne/mcpx/internal/testserver"
)

func main() {
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, runs as stdio server")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	srv := testserver.New()

	if *httpAddr != "" {
		addr, err := srv.RunHTTP(ctx, *httpAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Test server listening on http://%s/mcp\n", addr)

		// Wait for context cancellation
		<-ctx.Done()
	} else {
		if err := srv.RunStdio(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}
