package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Interface defines the logging interface used throughout the application
type Interface interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	WithField(key string, value interface{}) Interface
	WithFields(fields map[string]interface{}) Interface
	WithError(err error) Interface
}

// Logger wraps slog.Logger to provide a common interface across the application
type Logger struct {
	*slog.Logger
}

// Config contains logging configuration
type Config struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"` // "json" or "text"
	Output     string `yaml:"output"` // "stdout", "stderr", or file path
	File       string `yaml:"file"`   // deprecated, use Output instead
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// New creates a new structured logger with the given configuration
func New(config Config) (*Logger, error) {
	// Determine log level
	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", config.Level, err)
	}

	// Determine output destination
	var output io.Writer
	switch config.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// If it's not stdout/stderr, treat it as a file path
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %w", config.Output, err)
		}
		output = file
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source info for debug level
	}

	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text", "":
		handler = slog.NewTextHandler(output, opts)
	default:
		return nil, fmt.Errorf("unsupported log format '%s'", config.Format)
	}

	logger := slog.New(handler)
	return &Logger{Logger: logger}, nil
}

// parseLevel converts a string level to slog.Level
func parseLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown level: %s", levelStr)
	}
}

// WithFields returns a logger with the given fields added to all log entries
func (l *Logger) WithFields(fields map[string]interface{}) Interface {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithField returns a logger with a single field added to all log entries
func (l *Logger) WithField(key string, value interface{}) Interface {
	return &Logger{Logger: l.Logger.With(key, value)}
}

// WithError returns a logger with error field added
func (l *Logger) WithError(err error) Interface {
	return &Logger{Logger: l.Logger.With("error", err)}
}

// Compatibility methods for easier migration from logrus

// Debug logs at debug level with string message and args
func (l *Logger) Debug(msg string, args ...interface{}) {
	if len(args) == 0 {
		l.Logger.Debug(msg)
	} else {
		l.Logger.Debug(msg, args...)
	}
}

// Info logs at info level with string message and args
func (l *Logger) Info(msg string, args ...interface{}) {
	if len(args) == 0 {
		l.Logger.Info(msg)
	} else {
		l.Logger.Info(msg, args...)
	}
}

// Warn logs at warn level with string message and args
func (l *Logger) Warn(msg string, args ...interface{}) {
	if len(args) == 0 {
		l.Logger.Warn(msg)
	} else {
		l.Logger.Warn(msg, args...)
	}
}

// Error logs at error level with string message and args
func (l *Logger) Error(msg string, args ...interface{}) {
	if len(args) == 0 {
		l.Logger.Error(msg)
	} else {
		l.Logger.Error(msg, args...)
	}
}

// Debugf logs at debug level with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

// Infof logs at info level with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

// Warnf logs at warn level with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs at error level with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

// Fatalf logs at error level and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Panicf logs at error level and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.Logger.Error(msg)
	panic(msg)
}

// Default creates a default logger for the application
func Default() *Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{Logger: slog.New(handler)}
}

// SetDefault sets the default logger for slog package
func SetDefault(logger *Logger) {
	slog.SetDefault(logger.Logger)
}
