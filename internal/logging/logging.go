package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// LogLevelDebug enables debug level logging
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo enables info level logging
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn enables warn level logging
	LogLevelWarn LogLevel = "warn"
	// LogLevelError enables error level logging
	LogLevelError LogLevel = "error"
)

var (
	// Default logger instance
	logger     *slog.Logger
	loggerOnce sync.Once

	// Current log level
	currentLevel = new(slog.LevelVar)
)

// Init initializes the logger with the specified level
func Init(level LogLevel) {
	loggerOnce.Do(func() {
		// Set the log level
		setLevel(level)

		// Create a JSON handler for structured logging
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: currentLevel,
		})

		// Create the logger
		logger = slog.New(handler)

		// Set as default logger
		slog.SetDefault(logger)
	})
}

// SetLevel changes the current logging level
func SetLevel(level LogLevel) {
	setLevel(level)
}

// setLevel is an internal function to set the log level
func setLevel(level LogLevel) {
	switch level {
	case LogLevelDebug:
		currentLevel.Set(slog.LevelDebug)
	case LogLevelInfo:
		currentLevel.Set(slog.LevelInfo)
	case LogLevelWarn:
		currentLevel.Set(slog.LevelWarn)
	case LogLevelError:
		currentLevel.Set(slog.LevelError)
	default:
		currentLevel.Set(slog.LevelInfo) // Default to info
	}
}

// GetLogger returns the configured logger
func GetLogger() *slog.Logger {
	if logger == nil {
		Init(LogLevelInfo) // Initialize with default level if not done yet
	}
	return logger
}

// GetLogLevel returns the current log level
func GetLogLevel() LogLevel {
	level := currentLevel.Level()
	switch {
	case level <= slog.LevelDebug:
		return LogLevelDebug
	case level <= slog.LevelInfo:
		return LogLevelInfo
	case level <= slog.LevelWarn:
		return LogLevelWarn
	default:
		return LogLevelError
	}
}

// Debug logs a debug message with the given attributes
func Debug(msg string, attrs ...any) {
	GetLogger().Debug(msg, attrs...)
}

// Info logs an info message with the given attributes
func Info(msg string, attrs ...any) {
	GetLogger().Info(msg, attrs...)
}

// Warn logs a warning message with the given attributes
func Warn(msg string, attrs ...any) {
	GetLogger().Warn(msg, attrs...)
}

// Error logs an error message with the given attributes
func Error(msg string, attrs ...any) {
	GetLogger().Error(msg, attrs...)
}

// WithWriter returns a logger that writes to the specified writer
func WithWriter(w io.Writer) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: currentLevel,
	})
	return slog.New(handler)
}

// LogHandler is a callback function for custom log handling
type LogHandler func(level, message string)

// customHandler is a slog.Handler that sends logs to a custom callback
type customHandler struct {
	handler LogHandler
	level   *slog.LevelVar
	attrs   []slog.Attr
	groups  []string
}

// newCustomHandler creates a new custom handler
func newCustomHandler(handler LogHandler, level *slog.LevelVar) *customHandler {
	return &customHandler{
		handler: handler,
		level:   level,
		attrs:   []slog.Attr{},
		groups:  []string{},
	}
}

// Enabled implements slog.Handler
func (h *customHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements slog.Handler
func (h *customHandler) Handle(_ context.Context, record slog.Record) error {
	levelStr := record.Level.String()
	message := record.Message

	// Build full message with attributes
	var attrs []string
	record.Attrs(func(a slog.Attr) bool {
		// Format attribute as key=value
		attrs = append(attrs, a.Key+"="+a.Value.String())
		return true
	})

	// Combine message with attributes
	fullMessage := message
	if len(attrs) > 0 {
		fullMessage = message + " [" + joinStrings(attrs, ", ") + "]"
	}

	// Call the custom handler
	if h.handler != nil {
		h.handler(levelStr, fullMessage)
	}

	return nil
}

// joinStrings joins string slices (helper to avoid importing strings in this context)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// WithAttrs implements slog.Handler
func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := *h
	newHandler.attrs = append([]slog.Attr{}, h.attrs...)
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return &newHandler
}

// WithGroup implements slog.Handler
func (h *customHandler) WithGroup(name string) slog.Handler {
	newHandler := *h
	newHandler.groups = append([]string{}, h.groups...)
	newHandler.groups = append(newHandler.groups, name)
	return &newHandler
}

// SetCustomHandler sets a custom log handler for TUI mode
func SetCustomHandler(handler LogHandler) {
	customH := newCustomHandler(handler, currentLevel)
	logger = slog.New(customH)
	slog.SetDefault(logger)
}

// ResetToStderr resets logging back to stderr (for non-TUI mode)
func ResetToStderr() {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: currentLevel,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)
}
