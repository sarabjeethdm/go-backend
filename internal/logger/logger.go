package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus logger
type Logger struct {
	*logrus.Logger
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init() {
	globalLogger = New()
}

// New creates a new logger instance
func New() *Logger {
	log := logrus.New()

	// Set output to stdout
	log.SetOutput(os.Stdout)

	// Set log format to JSON for structured logging
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set log level from environment or default to Info
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	return &Logger{log}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Global logging functions

// WithFields adds multiple fields to the global logger
func WithFields(fields map[string]interface{}) *logrus.Entry {
	if globalLogger == nil {
		Init()
	}
	return globalLogger.Logger.WithFields(fields)
}

// Info logs an info message
func Info(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Info(args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Infof(format, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Error(args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Errorf(format, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Warnf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Fatalf(format, args...)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Debugf(format, args...)
}
