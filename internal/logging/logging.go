package logging

import (
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
