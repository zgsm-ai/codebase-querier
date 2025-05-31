package scip

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	// logger is the global logger instance
	logger *log.Logger
)

// InitLogger initializes the logger
func InitLogger(logPath string) error {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger
	logger = log.New(logFile, "", log.LstdFlags)
	return nil
}

// LogIndexInfo logs an info message
func LogIndexInfo(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[INFO] "+format, args...)
	}
}

// LogIndexError logs an error message
func LogIndexError(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[ERROR] "+format, args...)
	}
}

// LogIndexDebug logs debug level message for indexing
func LogIndexDebug(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[DEBUG] "+format, args...)
	}
}
