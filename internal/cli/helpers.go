package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/vaayne/mcphub/internal/logging"
	"github.com/vaayne/mcphub/internal/toolname"

	ucli "github.com/urfave/cli/v3"
)

// createRemoteClient creates a RemoteClient from command flags
func createRemoteClient(ctx context.Context, cmd *ucli.Command) (*RemoteClient, error) {
	url := cmd.String("url")
	transport := cmd.String("transport")
	timeout := cmd.Int("timeout")
	headers := cmd.StringSlice("header")
	verbose := cmd.Bool("verbose")
	logFile := cmd.String("log-file")

	// Default to http for remote commands
	if transport == "" {
		transport = "http"
	}

	// Initialize logging
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return nil, err
	}

	opts := RemoteClientOpts{
		ServerURL: url,
		Transport: transport,
		Headers:   parseHeaders(headers),
		Timeout:   int(timeout),
		Logger:    logging.Logger,
	}

	return NewRemoteClient(ctx, opts)
}

func createConfigClient(ctx context.Context, cmd *ucli.Command) (*ConfigClient, error) {
	configPath := cmd.String("config")
	timeout := cmd.Int("timeout")
	logger := getLogger(cmd)

	return NewConfigClient(ctx, configPath, logger, time.Duration(timeout)*time.Second)
}

// parseHeaders parses headers from []string in format "Key: Value" into map[string]string.
// Malformed headers (without ":") are silently skipped.
// Header values can contain environment variables that are expanded (e.g., $TOKEN or ${TOKEN}).
func parseHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		// Malformed headers without ":" are silently skipped
	}
	return result
}

// getLogger returns a configured logger based on command flags
func getLogger(cmd *ucli.Command) *slog.Logger {
	verbose := cmd.Bool("verbose")
	logFile := cmd.String("log-file")

	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return logging.NopLogger() // Safe fallback
	}

	return logging.Logger
}

// ensureNamespacedToolName validates that a tool name is in namespaced format for config mode
func ensureNamespacedToolName(name string) error {
	if !toolname.IsNamespaced(name) {
		return fmt.Errorf("tool name must include server prefix (server__tool) when using --config")
	}
	return nil
}

// getStdioCommand extracts the stdio command from os.Args after the "--" separator.
// Returns the command slice and an error if --stdio is used without a command.
func getStdioCommand() ([]string, error) {
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--stdio requires a command after --")
			}
			return args[i+1:], nil
		}
	}
	return nil, fmt.Errorf("--stdio requires -- followed by a command (e.g., mh --stdio list -- npx @mcp/server)")
}

// getStdioCommandLength returns the number of args that belong to the stdio command
// (everything after "--" in os.Args). Returns 0 if no "--" is found.
func getStdioCommandLength() int {
	for i, arg := range os.Args {
		if arg == "--" {
			return len(os.Args) - i - 1
		}
	}
	return 0
}

// filterArgsBeforeDash returns only the args that come before "--" separator.
// Since Cobra strips "--" from the args it passes to commands, we need to
// calculate how many args belong to the stdio command and exclude them.
func filterArgsBeforeDash(args []string) []string {
	stdioLen := getStdioCommandLength()
	if stdioLen == 0 {
		return args
	}
	// Remove the last stdioLen args (they belong to stdio command)
	if stdioLen >= len(args) {
		return []string{}
	}
	return args[:len(args)-stdioLen]
}

// createStdioClientFromCmd creates a StdioClient using command flags and the stdio command from os.Args
func createStdioClientFromCmd(ctx context.Context, cmd *ucli.Command) (*StdioClient, error) {
	timeout := cmd.Int("timeout")
	verbose := cmd.Bool("verbose")
	logFile := cmd.String("log-file")

	stdioCmd, err := getStdioCommand()
	if err != nil {
		return nil, err
	}

	return createStdioClient(ctx, stdioCmd, int(timeout), verbose, logFile)
}
