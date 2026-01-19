package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// loggerState holds the global logger state with thread-safe access
var loggerState = struct {
	mu     sync.Mutex
	logger *slog.Logger
	file   *os.File
}{
	logger: nil,
	file:   nil,
}

// Logger is the global logger instance (deprecated: use GetLogger() for thread-safe access)
var Logger *slog.Logger

// Config holds logging configuration
type Config struct {
	// LogLevel sets the minimum log level (debug, info, warn, error)
	LogLevel slog.Level
	// LogFilePath is the path to the log file (empty to disable file logging)
	LogFilePath string
}

// InitResult contains the result of logger initialization
type InitResult struct {
	// FileLoggingEnabled indicates whether file logging was successfully enabled
	FileLoggingEnabled bool
	// FileLoggingError contains the error if file logging failed (nil if succeeded or not attempted)
	FileLoggingError error
}

// InitLogger initializes the global logger with JSON structured logging
// It always logs to stdout and optionally logs to a file if LogFilePath is set
// Returns InitResult indicating whether file logging was enabled and any errors encountered
func InitLogger(cfg Config) (*InitResult, error) {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	result := &InitResult{
		FileLoggingEnabled: false,
		FileLoggingError:   nil,
	}

	// Close previous file handle if exists to prevent file descriptor leak
	if loggerState.file != nil {
		if err := loggerState.file.Close(); err != nil {
			// Log warning but continue - we're reinitializing anyway
			if loggerState.logger != nil {
				loggerState.logger.Warn("Failed to close previous log file during reinitialization",
					slog.String("error", err.Error()),
				)
			}
		}
		loggerState.file = nil
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     cfg.LogLevel,
		AddSource: true,
	}

	var writers []io.Writer
	writers = append(writers, os.Stdout)

	// Try to create file writer if log file path is specified
	if cfg.LogFilePath != "" {
		file, err := createLogFile(cfg.LogFilePath)
		if err != nil {
			// Log warning to stdout and continue without file logging
			tempLogger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
			tempLogger.Warn("Failed to initialize file logging, continuing with stdout only",
				slog.String("log_file", cfg.LogFilePath),
				slog.String("error", err.Error()),
			)
			result.FileLoggingError = err
		} else {
			writers = append(writers, file)
			loggerState.file = file
			result.FileLoggingEnabled = true
		}
	}

	// Create multi-writer for all destinations
	multiWriter := io.MultiWriter(writers...)

	// Create JSON handler with multi-writer
	handler := slog.NewJSONHandler(multiWriter, opts)

	// Create logger
	loggerState.logger = slog.New(handler)
	Logger = loggerState.logger

	return result, nil
}

// validateLogFilePath validates and sanitizes the log file path
func validateLogFilePath(path string) (string, error) {
	// Clean the path to normalize it (removes . and .. where possible)
	cleanPath := filepath.Clean(path)

	// Convert to absolute path for security and clarity
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	return absPath, nil
}

// createLogFile creates or opens a log file with proper permissions
func createLogFile(logFilePath string) (*os.File, error) {
	// Validate and sanitize the file path
	cleanPath, err := validateLogFilePath(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid log file path: %w", err)
	}

	// Try to open/create the log file with restricted permissions (owner-only read/write)
	file, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// Sync flushes any buffered log entries
func Sync() error {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	// slog doesn't have a Sync method, but we can sync the file if it exists
	if loggerState.file != nil {
		return loggerState.file.Sync()
	}
	return nil
}

// GetLogger returns the global logger instance in a thread-safe manner
func GetLogger() *slog.Logger {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()
	return loggerState.logger
}

// WithRequestID returns a logger with the request ID field added
// This is useful for tracing requests across log entries
func WithRequestID(requestID string) *slog.Logger {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	if loggerState.logger == nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return loggerState.logger.With(slog.String("request_id", requestID))
}

// ParseLevel converts a string log level to slog.Level
func ParseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO":
		return slog.LevelInfo, nil
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}

// NopLogger returns a no-op logger that discards all output
func NopLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
