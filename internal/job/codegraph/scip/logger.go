package scip

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	indexLogger logx.Logger
	once        sync.Once
)

// InitIndexLogger initializes the index logger
func InitIndexLogger(logPath string) error {
	var err error
	once.Do(func() {
		// Create log directory if not exists
		if err = os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return
		}

		// Create index logger with file writer
		writer, err := logx.NewFileWriter(logPath)
		if err != nil {
			return
		}

		indexLogger = logx.NewLogger(writer, logx.WithCallerSkip(1))
	})
	return err
}

// GetIndexLogger returns the index logger
func GetIndexLogger() logx.Logger {
	if indexLogger == nil {
		// Fallback to default logger if not initialized
		return logx.WithCallerSkip(1)
	}
	return indexLogger
}

// LogIndexInfo logs info level message for indexing
func LogIndexInfo(format string, args ...interface{}) {
	GetIndexLogger().Infof(format, args...)
}

// LogIndexError logs error level message for indexing
func LogIndexError(format string, args ...interface{}) {
	GetIndexLogger().Errorf(format, args...)
}

// LogIndexDebug logs debug level message for indexing
func LogIndexDebug(format string, args ...interface{}) {
	GetIndexLogger().Debugf(format, args...)
} 