// Package mygoword provides a simple logging utility for Go applications.
package mygoword

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// DEBUG level for detailed debugging information.
	DEBUG LogLevel = iota
	// INFO level for general operational information.
	INFO
	// WARN level for warning messages that may indicate potential issues.
	WARN
	// ERROR level for error conditions that should be investigated.
	ERROR
	// FATAL level for severe errors that will cause the program to exit.
	FATAL
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger represents a logging instance with configurable output and level.
type Logger struct {
	mu     sync.Mutex
	level  LogLevel
	output io.Writer
}

// New creates a new Logger with the specified log level and output writer.
// If output is nil, os.Stdout is used.
func New(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		level:  level,
		output: output,
	}
}

// Default returns a new Logger with INFO level and stdout output.
func Default() *Logger {
	return New(INFO, os.Stdout)
}

// SetLevel changes the log level of the logger.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput changes the output writer of the logger.
func (l *Logger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// log writes a log message with the specified level and format.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)

	_, _ = fmt.Fprint(l.output, logEntry)
}

// Debug logs a debug level message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info level message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning level message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error level message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal logs a fatal level message and exits the program with status 1.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

// WithFields creates a new logger with additional context fields.
// This is a simplified version - in a real implementation, you might want
// to use structured logging with key-value pairs.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// For simplicity, we'll just return the same logger
	// In a real implementation, you would create a new logger with context
	return l
}