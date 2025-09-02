package logger

import (
	"fmt"
	"os"
)

// LogrusCompatible defines the interface we need to mimic from logrus
type LogrusCompatible interface {
	WithField(key string, value interface{}) LogrusEntryCompatible
	WithFields(fields map[string]interface{}) LogrusEntryCompatible 
	WithError(err error) LogrusEntryCompatible
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type LogrusEntryCompatible interface {
	WithField(key string, value interface{}) LogrusEntryCompatible
	WithFields(fields map[string]interface{}) LogrusEntryCompatible
	WithError(err error) LogrusEntryCompatible
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// LogrusAdapter wraps our structured logger to implement LogrusCompatible interface
// This is a compatibility layer to help with gradual migration from logrus to slog
type LogrusAdapter struct {
	logger Interface
}

// NewLogrusAdapter creates a new adapter that makes our logger compatible with logrus
func NewLogrusAdapter(logger Interface) *LogrusAdapter {
	return &LogrusAdapter{logger: logger}
}

// Compatibility methods to match LogrusCompatible interface

func (l *LogrusAdapter) WithField(key string, value interface{}) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithField(key, value)}
}

func (l *LogrusAdapter) WithFields(fields map[string]interface{}) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithFields(fields)}
}

func (l *LogrusAdapter) WithError(err error) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithError(err)}
}

func (l *LogrusAdapter) Debug(args ...interface{}) {
	l.logger.Debug(toString(args...))
}

func (l *LogrusAdapter) Info(args ...interface{}) {
	l.logger.Info(toString(args...))
}

func (l *LogrusAdapter) Warn(args ...interface{}) {
	l.logger.Warn(toString(args...))
}

func (l *LogrusAdapter) Error(args ...interface{}) {
	l.logger.Error(toString(args...))
}

func (l *LogrusAdapter) Fatal(args ...interface{}) {
	l.logger.Error(toString(args...))
	os.Exit(1)
}

func (l *LogrusAdapter) Panic(args ...interface{}) {
	msg := toString(args...)
	l.logger.Error(msg)
	panic(msg)
}

func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *LogrusAdapter) Fatalf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
	os.Exit(1)
}

func (l *LogrusAdapter) Panicf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
	panic("Panic error")
}

// LogrusEntryAdapter wraps our structured logger to implement LogrusEntryCompatible
type LogrusEntryAdapter struct {
	logger Interface
}

func (l *LogrusEntryAdapter) WithField(key string, value interface{}) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithField(key, value)}
}

func (l *LogrusEntryAdapter) WithFields(fields map[string]interface{}) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithFields(fields)}
}

func (l *LogrusEntryAdapter) WithError(err error) LogrusEntryCompatible {
	return &LogrusEntryAdapter{logger: l.logger.WithError(err)}
}

func (l *LogrusEntryAdapter) Debug(args ...interface{}) {
	l.logger.Debug(toString(args...))
}

func (l *LogrusEntryAdapter) Info(args ...interface{}) {
	l.logger.Info(toString(args...))
}

func (l *LogrusEntryAdapter) Warn(args ...interface{}) {
	l.logger.Warn(toString(args...))
}

func (l *LogrusEntryAdapter) Error(args ...interface{}) {
	l.logger.Error(toString(args...))
}

func (l *LogrusEntryAdapter) Fatal(args ...interface{}) {
	l.logger.Error(toString(args...))
	os.Exit(1)
}

func (l *LogrusEntryAdapter) Panic(args ...interface{}) {
	msg := toString(args...)
	l.logger.Error(msg)
	panic(msg)
}

func (l *LogrusEntryAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *LogrusEntryAdapter) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *LogrusEntryAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *LogrusEntryAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *LogrusEntryAdapter) Fatalf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
	os.Exit(1)
}

func (l *LogrusEntryAdapter) Panicf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
	panic("Panic error")
}

// toString converts args to string similar to logrus behavior
func toString(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		if s, ok := args[0].(string); ok {
			return s
		}
	}
	return fmt.Sprint(args...)
}