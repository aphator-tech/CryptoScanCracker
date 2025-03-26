package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// LogLevelDebug is for debug messages
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is for informational messages
	LogLevelInfo
	// LogLevelWarn is for warning messages
	LogLevelWarn
	// LogLevelError is for error messages
	LogLevelError
)

// Logger provides structured logging functionality
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new logger with the specified log level
func NewLogger(levelStr string) *Logger {
	level := parseLogLevel(levelStr)
	
	logger := log.New(os.Stdout, "", 0)
	
	return &Logger{
		level:  level,
		logger: logger,
	}
}

// parseLogLevel parses a string log level into a LogLevel value
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	if l.level <= LogLevelDebug {
		l.log("DEBUG", message)
	}
}

// Info logs an informational message
func (l *Logger) Info(message string) {
	if l.level <= LogLevelInfo {
		l.log("INFO", message)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	if l.level <= LogLevelWarn {
		l.log("WARN", message)
	}
}

// Error logs an error message
func (l *Logger) Error(message string) {
	if l.level <= LogLevelError {
		l.log("ERROR", message)
	}
}

// log formats and writes a log message
func (l *Logger) log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.logger.Printf("[%s] %s: %s", timestamp, level, message)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(levelStr string) {
	l.level = parseLogLevel(levelStr)
}

// PrintBanner prints a banner message at startup
func (l *Logger) PrintBanner(appName, version string) {
	banner := fmt.Sprintf(`
====================================
  %s v%s
====================================
`, appName, version)
	
	fmt.Println(banner)
}
