package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	pid           = os.Getpid()
	levelStrings  = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

// Logger provides structured logging with timestamp, PID, and function name
type Logger struct {
	minLevel LogLevel
}

// NewLogger creates a new logger instance
func NewLogger(minLevel LogLevel) *Logger {
	return &Logger{minLevel: minLevel}
}

// Default logger instance (INFO level)
var defaultLogger = NewLogger(INFO)

// getFunctionName extracts the calling function name
func getFunctionName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fullName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(fullName, "/")
	name := parts[len(parts)-1]

	// Remove package path prefix, keep last segment
	if idx := strings.LastIndex(name, "."); idx != -1 {
		return name[idx+1:]
	}
	return name
}

// formatMessage creates the log message with timestamp, PID, function name
func formatMessage(level LogLevel, funcName, message string, context map[string]interface{}) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	levelStr := levelStrings[level]

	var contextStr string
	if len(context) > 0 {
		var pairs []string
		for k, v := range context {
			pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
		}
		contextStr = " | " + strings.Join(pairs, " ")
	}

	return fmt.Sprintf("[%s] [PID:%d] [%s] %s: %s%s",
		timestamp, pid, funcName, levelStr, message, contextStr)
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, message string, context map[string]interface{}) {
	if level < l.minLevel {
		return
	}

	funcName := getFunctionName(3) // Skip: log -> Debug/Info/Warn/Error -> actual caller
	msg := formatMessage(level, funcName, message, context)

	if level >= ERROR {
		fmt.Fprintln(os.Stderr, msg)
	} else {
		fmt.Fprintln(os.Stdout, msg)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(DEBUG, message, ctx)
}

// Info logs an info message
func (l *Logger) Info(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(INFO, message, ctx)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(WARN, message, ctx)
}

// Error logs an error message
func (l *Logger) Error(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(ERROR, message, ctx)
}

// Package-level convenience functions using default logger

// Debug logs a debug message using the default logger
func Debug(message string, context ...map[string]interface{}) {
	defaultLogger.Debug(message, context...)
}

// Info logs an info message using the default logger
func Info(message string, context ...map[string]interface{}) {
	defaultLogger.Info(message, context...)
}

// Warn logs a warning message using the default logger
func Warn(message string, context ...map[string]interface{}) {
	defaultLogger.Warn(message, context...)
}

// Error logs an error message using the default logger
func Error(message string, context ...map[string]interface{}) {
	defaultLogger.Error(message, context...)
}

// SetMinLevel sets the minimum log level for the default logger
func SetMinLevel(level LogLevel) {
	defaultLogger.minLevel = level
}
