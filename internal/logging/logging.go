package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
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

// ── Ring buffer for web UI log streaming ────────────────────────────────────

// LogLine is a single buffered log entry returned by GetLogLinesSince.
type LogLine struct {
	Index   int    `json:"index"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

const ringBufferMax = 500

var (
	ringBuf    []LogLine
	ringMu     sync.Mutex
	ringCursor int
)

func appendToRing(level, message string) {
	ringMu.Lock()
	defer ringMu.Unlock()
	ringCursor++
	entry := LogLine{
		Index:   ringCursor,
		Level:   level,
		Message: message,
		Time:    time.Now().Format(time.RFC3339),
	}
	if len(ringBuf) >= ringBufferMax {
		ringBuf = ringBuf[1:]
	}
	ringBuf = append(ringBuf, entry)
}

// GetLogLinesSince returns buffered lines with index > since and the current cursor.
func GetLogLinesSince(since int) ([]LogLine, int) {
	ringMu.Lock()
	defer ringMu.Unlock()
	var result []LogLine
	for _, line := range ringBuf {
		if line.Index > since {
			result = append(result, line)
		}
	}
	return result, ringCursor
}

// EnableBuffer installs a handler that writes to both stderr and the ring buffer.
// Call this once when starting the web server.
func EnableBuffer() {
	stderrH := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: currentLevel})
	logger = slog.New(&multiHandler{primary: stderrH})
	slog.SetDefault(logger)
}

// multiHandler writes to a primary slog.Handler AND appends to the ring buffer.
type multiHandler struct {
	primary slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.primary.Enabled(ctx, level)
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	// Build human-readable message with key=value attrs for the ring buffer.
	var attrs []string
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a.Key+"="+a.Value.String())
		return true
	})
	msg := record.Message
	if len(attrs) > 0 {
		msg = msg + " [" + strings.Join(attrs, ", ") + "]"
	}
	appendToRing(record.Level.String(), msg)
	return h.primary.Handle(ctx, record)
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiHandler{primary: h.primary.WithAttrs(attrs)}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	return &multiHandler{primary: h.primary.WithGroup(name)}
}

// Recover is a deferred panic recovery helper for goroutines.
// It logs the panic and stack trace, preventing the process from crashing.
// Usage:
//
//	go func() {
//	    defer Recover("background task name")
//	    ...
//	}()
func Recover(name string) {
	if r := recover(); r != nil {
		Error("goroutine panic recovered", "name", name, "panic", r)
	}
}
