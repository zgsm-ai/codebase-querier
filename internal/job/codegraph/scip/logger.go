package scip

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// newFileLogger initializes the indexFileLogger
func newFileLogger(logDir string) (*log.Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(filepath.Join(logDir, newLogFileName()),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return log.New(logFile, "", log.LstdFlags), nil
}

func newLogFileName() string {
	return time.Now().Format("2006-01-02_15") + ".log"
}
