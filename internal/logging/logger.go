package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"win-service-manager/internal/config"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// ToSlogLevel converts LogLevel to slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LoggerConfig contains logger configuration
type LoggerConfig struct {
	Level         LogLevel
	File          string
	EnableConsole bool
	MaxSize       int64
	MaxBackups    int
}

// LoggerSetup sets up the application logger
func SetupLogger(cfg *config.Config) (*slog.Logger, error) {
	logLevel := ParseLogLevel(cfg.Logging.Level)
	
	var writers []io.Writer
	
	// Add console output if enabled
	if cfg.Logging.EnableConsole {
		writers = append(writers, os.Stdout)
	}
	
	// Add file output if specified
	if cfg.Logging.File != "" {
		fileWriter, err := setupFileLogger(cfg.Logging.File, int64(cfg.Logging.MaxSize)*1024*1024)
		if err != nil {
			return nil, fmt.Errorf("failed to setup file logger: %w", err)
		}
		writers = append(writers, fileWriter)
	}
	
	// If no writers are configured, default to stdout
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	
	// Create multi-writer
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = io.MultiWriter(writers...)
	}
	
	// Create structured logger options
	opts := &slog.HandlerOptions{
		Level: logLevel.ToSlogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			return a
		},
	}
	
	// Create handler based on output type
	var handler slog.Handler
	if cfg.Logging.EnableConsole && cfg.Logging.File == "" {
		// Console only - use text handler for better readability
		handler = slog.NewTextHandler(writer, opts)
	} else {
		// File output or mixed - use JSON handler for structured logs
		handler = slog.NewJSONHandler(writer, opts)
	}
	
	logger := slog.New(handler)
	
	// Set as default logger
	slog.SetDefault(logger)
	
	return logger, nil
}

// setupFileLogger creates a file writer with basic rotation
func setupFileLogger(logFile string, maxSize int64) (io.Writer, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Check if we need to rotate the log file
	if err := rotateLogIfNeeded(logFile, maxSize); err != nil {
		return nil, fmt.Errorf("failed to rotate log file: %w", err)
	}
	
	// Open log file for appending
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	
	return file, nil
}

// rotateLogIfNeeded rotates the log file if it exceeds the maximum size
func rotateLogIfNeeded(logFile string, maxSize int64) error {
	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, no need to rotate
		}
		return err
	}
	
	if info.Size() < maxSize {
		return nil // File is under the size limit
	}
	
	// Rotate the file
	timestamp := time.Now().Format("20060102-150405")
	rotatedFile := fmt.Sprintf("%s.%s", logFile, timestamp)
	
	if err := os.Rename(logFile, rotatedFile); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}
	
	return nil
}

// LogApplicationStart logs application startup information
func LogApplicationStart(logger *slog.Logger, version string) {
	logger.Info("application starting",
		"version", version,
		"pid", os.Getpid(),
		"args", os.Args,
	)
}

// LogApplicationStop logs application shutdown information
func LogApplicationStop(logger *slog.Logger, reason string) {
	logger.Info("application stopping",
		"reason", reason,
		"pid", os.Getpid(),
	)
}

// LogConfigLoaded logs configuration loading information
func LogConfigLoaded(logger *slog.Logger, configPath string) {
	logger.Info("configuration loaded",
		"config_path", configPath,
	)
}

// LogServiceOperation logs service operations
func LogServiceOperation(logger *slog.Logger, operation, serviceName string, success bool, err error) {
	if success {
		logger.Info("service operation completed",
			"operation", operation,
			"service", serviceName,
		)
	} else {
		logger.Error("service operation failed",
			"operation", operation,
			"service", serviceName,
			"error", err,
		)
	}
}

// LogError logs errors with context
func LogError(logger *slog.Logger, message string, err error, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs)*2+2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	args = append(args, "error", err)
	
	logger.Error(message, args...)
}

// LogPerformance logs performance metrics
func LogPerformance(logger *slog.Logger, operation string, duration time.Duration, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs)*2+4)
	args = append(args, "operation", operation, "duration_ms", duration.Milliseconds())
	
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	
	logger.Debug("performance metric", args...)
}

// ContextLogger wraps slog.Logger with context information
type ContextLogger struct {
	logger *slog.Logger
	attrs  []slog.Attr
}

// NewContextLogger creates a new context logger
func NewContextLogger(logger *slog.Logger, attrs ...slog.Attr) *ContextLogger {
	return &ContextLogger{
		logger: logger,
		attrs:  attrs,
	}
}

// With adds attributes to the context logger
func (cl *ContextLogger) With(attrs ...slog.Attr) *ContextLogger {
	newAttrs := make([]slog.Attr, len(cl.attrs)+len(attrs))
	copy(newAttrs, cl.attrs)
	copy(newAttrs[len(cl.attrs):], attrs)
	
	return &ContextLogger{
		logger: cl.logger,
		attrs:  newAttrs,
	}
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(message string, attrs ...slog.Attr) {
	allAttrs := append(cl.attrs, attrs...)
	args := make([]any, 0, len(allAttrs)*2)
	for _, attr := range allAttrs {
		args = append(args, attr.Key, attr.Value)
	}
	cl.logger.Debug(message, args...)
}

// Info logs an info message with context
func (cl *ContextLogger) Info(message string, attrs ...slog.Attr) {
	allAttrs := append(cl.attrs, attrs...)
	args := make([]any, 0, len(allAttrs)*2)
	for _, attr := range allAttrs {
		args = append(args, attr.Key, attr.Value)
	}
	cl.logger.Info(message, args...)
}

// Warn logs a warning message with context
func (cl *ContextLogger) Warn(message string, attrs ...slog.Attr) {
	allAttrs := append(cl.attrs, attrs...)
	args := make([]any, 0, len(allAttrs)*2)
	for _, attr := range allAttrs {
		args = append(args, attr.Key, attr.Value)
	}
	cl.logger.Warn(message, args...)
}

// Error logs an error message with context
func (cl *ContextLogger) Error(message string, attrs ...slog.Attr) {
	allAttrs := append(cl.attrs, attrs...)
	args := make([]any, 0, len(allAttrs)*2)
	for _, attr := range allAttrs {
		args = append(args, attr.Key, attr.Value)
	}
	cl.logger.Error(message, args...)
}