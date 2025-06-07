package scip

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const tmpLogDir = "/tmp/logs"

// newFileLogWriter initializes the indexFileLogWriter
func newFileLogWriter(logDir string, fileNamePrefix string) (io.WriteCloser, error) {
	if logDir == "" {
		logDir = tmpLogDir
	}
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s, err: %w", logDir, err)
	}

	// Open log file
	logFile, err := os.OpenFile(filepath.Join(logDir, logFileName(fileNamePrefix)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return logFile, nil
}

func logFileName(filenamePrefix string) string {
	return filenamePrefix + "-" + "index.log"
}

func indexLogInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		fmt.Printf(format, args...)
		return
	}
	format = fmt.Sprintf("[%s] - %s\n", time.Now().Format("2006-01-02 15:04:05"), format)
	_, _ = fmt.Fprintf(w, format, args...)
}
