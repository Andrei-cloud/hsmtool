package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents a logging level.
type Level int

const (
	// DEBUG level for detailed information.
	DEBUG Level = iota
	// INFO level for general information.
	INFO
	// ERROR level for error conditions.
	ERROR
)

// String returns the string representation of a log level.
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Entry represents a log entry.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Event     string    `json:"event"`
	Status    string    `json:"status"`
	Details   string    `json:"details,omitempty"`
}

// Logger handles application logging.
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	level    Level
	callback func(Entry)
}

// NewLogger creates a new logger instance.
func NewLogger(logPath string, level Level, callback func(Entry)) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	return &Logger{
		file:     file,
		level:    level,
		callback: callback,
	}, nil
}

// Log writes a log entry.
func (l *Logger) Log(level Level, event, status, details string) {
	if level < l.level {
		return
	}

	entry := Entry{
		Timestamp: time.Now(),
		Level:     level,
		Event:     event,
		Status:    status,
		Details:   details,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to file.
	data, err := json.Marshal(entry)
	if err == nil {
		fmt.Fprintln(l.file, string(data))
	}

	// Call callback if set.
	if l.callback != nil {
		l.callback(entry)
	}
}

// Debug logs a debug level message.
func (l *Logger) Debug(event, status, details string) {
	l.Log(DEBUG, event, status, details)
}

// Info logs an info level message.
func (l *Logger) Info(event, status, details string) {
	l.Log(INFO, event, status, details)
}

// Error logs an error level message.
func (l *Logger) Error(event, status, details string) {
	l.Log(ERROR, event, status, details)
}

// Close closes the logger.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// SetLevel changes the logging level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetEntries retrieves log entries with optional filtering.
func (l *Logger) GetEntries(start, end time.Time, filter string) ([]Entry, error) {
	// TODO: Implement log retrieval and filtering.
	return nil, fmt.Errorf("not implemented")
}
