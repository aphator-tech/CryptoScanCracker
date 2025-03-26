package utils

import (
        "fmt"
        "log"
        "os"
        "strings"
        "time"
        
        "github.com/fatih/color"
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

// log formats and writes a log message with colors
func (l *Logger) log(level, message string) {
        timestamp := time.Now().Format("2006-01-02 15:04:05")
        
        // Format the timestamp
        timestampStr := fmt.Sprintf("[%s]", timestamp)
        
        // Apply color based on log level
        var levelStr string
        
        switch level {
        case "DEBUG":
                levelStr = color.CyanString("DEBUG")
        case "INFO":
                levelStr = color.GreenString("INFO")
        case "WARN":
                levelStr = color.YellowString("WARN")
        case "ERROR":
                levelStr = color.RedString("ERROR")
        default:
                levelStr = level
        }
        
        // Format final message
        logMessage := fmt.Sprintf("%s %s: %s", timestampStr, levelStr, message)
        
        // For ERROR level, make the whole message red for high visibility
        if level == "ERROR" {
                logMessage = color.RedString("%s %s: %s", timestampStr, levelStr, message)
        }
        
        // Print the formatted message
        fmt.Println(logMessage)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(levelStr string) {
        l.level = parseLogLevel(levelStr)
}

// PrintBanner prints a colorful banner message at startup
func (l *Logger) PrintBanner(appName, version string) {
        // Create colored strings
        appNameColored := color.HiGreenString(appName)
        versionColored := color.HiYellowString("v%s", version)
        separator := color.CyanString("====================================")
        
        // Format the banner
        banner := fmt.Sprintf(`
%s
  %s %s
%s
`, separator, appNameColored, versionColored, separator)
        
        fmt.Println(banner)
}
