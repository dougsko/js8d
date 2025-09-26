package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dougsko/js8d/pkg/config"
	"gopkg.in/lumberjack.v2"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string log level
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger provides structured logging functionality
type Logger struct {
	level         LogLevel
	fileLogger    *log.Logger
	consoleLogger *log.Logger
	structured    bool
	rotatingFile  *lumberjack.Logger
}

// NewLogger creates a new logger from configuration
func NewLogger(cfg *config.Config) (*Logger, error) {
	logger := &Logger{
		level:      ParseLogLevel(cfg.Logging.Level),
		structured: cfg.Logging.Structured,
	}

	// Setup file logging with rotation (only if file path is specified)
	if cfg.Logging.File != "" && cfg.Logging.File != "/var/log/js8d/js8d.log" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(cfg.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Setup rotating file logger
		logger.rotatingFile = &lumberjack.Logger{
			Filename:   cfg.Logging.File,
			MaxSize:    cfg.Logging.MaxSize,    // megabytes
			MaxBackups: cfg.Logging.MaxBackups, // number of backups
			MaxAge:     cfg.Logging.MaxAge,     // days
			Compress:   cfg.Logging.Compress,   // compress old files
		}

		logger.fileLogger = log.New(logger.rotatingFile, "", 0)
	}

	// Setup console logging (enabled by config or when no file logging)
	if cfg.Logging.Console || logger.fileLogger == nil {
		logger.consoleLogger = log.New(os.Stdout, "", 0)
	}

	return logger, nil
}

// Close closes the logger and any open files
func (l *Logger) Close() error {
	if l.rotatingFile != nil {
		return l.rotatingFile.Close()
	}
	return nil
}

// shouldLog checks if a message should be logged at the given level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// formatMessage formats a log message
func (l *Logger) formatMessage(level LogLevel, component, message string, fields map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	if l.structured {
		// JSON-like structured format
		fieldsStr := ""
		if len(fields) > 0 {
			var parts []string
			for k, v := range fields {
				parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
			}
			fieldsStr = fmt.Sprintf(" {%s}", strings.Join(parts, ","))
		}
		return fmt.Sprintf(`{"time":"%s","level":"%s","component":"%s","message":"%s"%s}`,
			timestamp, level.String(), component, message, fieldsStr)
	} else {
		// Human-readable format
		fieldsStr := ""
		if len(fields) > 0 {
			var parts []string
			for k, v := range fields {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			fieldsStr = fmt.Sprintf(" [%s]", strings.Join(parts, " "))
		}
		return fmt.Sprintf("%s [%s] %s: %s%s",
			timestamp, level.String(), component, message, fieldsStr)
	}
}

// log writes a log message
func (l *Logger) log(level LogLevel, component, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	formatted := l.formatMessage(level, component, message, fields)

	if l.fileLogger != nil {
		l.fileLogger.Println(formatted)
	}

	if l.consoleLogger != nil {
		l.consoleLogger.Println(formatted)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(component, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, component, message, f)
}

// Info logs an info message
func (l *Logger) Info(component, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, component, message, f)
}

// Warn logs a warning message
func (l *Logger) Warn(component, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, component, message, f)
}

// Error logs an error message
func (l *Logger) Error(component, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, component, message, f)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(component, format string, args ...interface{}) {
	l.Debug(component, fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(component, format string, args ...interface{}) {
	l.Info(component, fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(component, format string, args ...interface{}) {
	l.Warn(component, fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(component, format string, args ...interface{}) {
	l.Error(component, fmt.Sprintf(format, args...))
}

// WithFields creates a logger with predefined fields
func (l *Logger) WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger is a logger with predefined fields
type FieldLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// Debug logs a debug message with predefined fields
func (fl *FieldLogger) Debug(component, message string) {
	fl.logger.log(LevelDebug, component, message, fl.fields)
}

// Info logs an info message with predefined fields
func (fl *FieldLogger) Info(component, message string) {
	fl.logger.log(LevelInfo, component, message, fl.fields)
}

// Warn logs a warning message with predefined fields
func (fl *FieldLogger) Warn(component, message string) {
	fl.logger.log(LevelWarn, component, message, fl.fields)
}

// Error logs an error message with predefined fields
func (fl *FieldLogger) Error(component, message string) {
	fl.logger.log(LevelError, component, message, fl.fields)
}

// Debugf logs a formatted debug message with predefined fields
func (fl *FieldLogger) Debugf(component, format string, args ...interface{}) {
	fl.logger.log(LevelDebug, component, fmt.Sprintf(format, args...), fl.fields)
}

// Infof logs a formatted info message with predefined fields
func (fl *FieldLogger) Infof(component, format string, args ...interface{}) {
	fl.logger.log(LevelInfo, component, fmt.Sprintf(format, args...), fl.fields)
}

// Warnf logs a formatted warning message with predefined fields
func (fl *FieldLogger) Warnf(component, format string, args ...interface{}) {
	fl.logger.log(LevelWarn, component, fmt.Sprintf(format, args...), fl.fields)
}

// Errorf logs a formatted error message with predefined fields
func (fl *FieldLogger) Errorf(component, format string, args ...interface{}) {
	fl.logger.log(LevelError, component, fmt.Sprintf(format, args...), fl.fields)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(cfg *config.Config) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Fallback to console logging if not initialized
		globalLogger = &Logger{
			level:         LevelInfo,
			consoleLogger: log.New(os.Stdout, "", 0),
		}
	}
	return globalLogger
}

// CloseGlobalLogger closes the global logger
func CloseGlobalLogger() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// Convenience functions for global logger
func Debug(component, message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Debug(component, message, fields...)
}

func Info(component, message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Info(component, message, fields...)
}

func Warn(component, message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Warn(component, message, fields...)
}

func Error(component, message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Error(component, message, fields...)
}

func Debugf(component, format string, args ...interface{}) {
	GetGlobalLogger().Debugf(component, format, args...)
}

func Infof(component, format string, args ...interface{}) {
	GetGlobalLogger().Infof(component, format, args...)
}

func Warnf(component, format string, args ...interface{}) {
	GetGlobalLogger().Warnf(component, format, args...)
}

func Errorf(component, format string, args ...interface{}) {
	GetGlobalLogger().Errorf(component, format, args...)
}