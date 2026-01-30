package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vaayne/mcphub/internal/config"
	"github.com/vaayne/mcphub/internal/logging"
	"github.com/vaayne/mcphub/internal/server"

	ucli "github.com/urfave/cli/v3"
)

// ServeCmd is the serve subcommand that starts the MCP hub server
var ServeCmd = &ucli.Command{
	Name:  "serve",
	Usage: "Start the MCP hub server",
	Description: `Start the MCP hub server with the specified transport.

Transport Types:
  stdio  - Standard input/output (default, for CLI integration)
  http   - HTTP server with StreamableHTTP protocol
  sse    - HTTP server with Server-Sent Events protocol

Examples:
  # Run with stdio transport (default)
  mh serve -c config.json

  # Run with HTTP transport on port 8080
  mh serve -c config.json -t http -p 8080

  # Run with SSE transport on custom host and port
  mh serve -c config.json -t sse --host 0.0.0.0 -p 3000

  # Run with verbose logging
  mh serve -c config.json --verbose`,
	Flags:  MCPServeFlags(),
	Action: runServe,
}

func runServe(ctx context.Context, cmd *ucli.Command) error {
	configPath := cmd.String("config")
	port := cmd.Int("port")
	host := cmd.String("host")
	transport := cmd.String("transport")
	verbose := cmd.Bool("verbose")
	logFile := cmd.String("log-file")

	// Validate transport type for serve command (stdio/http/sse)
	if transport != "stdio" && transport != "http" && transport != "sse" {
		return fmt.Errorf("invalid transport type for serve: %s (must be stdio, http, or sse)", transport)
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", port)
	}

	// Determine log level based on verbose flag
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	// Initialize logging
	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	result, err := logging.InitLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if err := logging.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	logger := logging.Logger

	// Log initialization status
	if result.FileLoggingEnabled {
		logger.Info("File logging enabled", slog.String("log_file", logFile))
	} else if result.FileLoggingError != nil {
		logger.Warn("File logging disabled due to error",
			slog.String("log_file", logFile),
			slog.String("error", result.FileLoggingError.Error()),
		)
	}

	// Validate config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Error("Configuration file does not exist", slog.String("path", configPath))
		return fmt.Errorf("config file not found: %s", configPath)
	}

	logger.Info("Starting MCP Hub",
		slog.String("config", configPath),
		slog.String("transport", transport),
	)

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create server
	srv := server.NewServer(cfg, logger)

	// Create transport config
	transportCfg := server.TransportConfig{
		Type: transport,
		Host: host,
		Port: int(port),
	}

	// Setup context with cancellation
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(runCtx, transportCfg); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
		cancel()
	case err := <-errChan:
		logger.Error("Server error", slog.String("error", err.Error()))
		return err
	}

	// Graceful shutdown
	if err := srv.Stop(); err != nil {
		logger.Error("Error during shutdown", slog.String("error", err.Error()))
		return err
	}

	logger.Info("MCP Hub stopped gracefully")
	return nil
}
