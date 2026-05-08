package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

var globalLogger *Logger

func Init() {
	globalLogger = New()
}

func New() *Logger {
	log := logrus.New()

	log.SetOutput(os.Stdout)

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

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

func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

func WithFields(fields map[string]interface{}) *logrus.Entry {
	if globalLogger == nil {
		Init()
	}
	return globalLogger.Logger.WithFields(fields)
}

func Info(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Infof(format, args...)
}

func Error(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Errorf(format, args...)
}

func Warn(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Warnf(format, args...)
}

func Fatal(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	if globalLogger == nil {
		Init()
	}
	globalLogger.Debugf(format, args...)
}
