package tracer

import (
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// CmdLogger manages log files and handles rotation and cleanup
type CmdLogger struct {
	logDir        string
	retentionDays int
	mu            sync.RWMutex // Read-write lock to protect file operations
	taskRunning   bool
	stopCh        chan bool
	activeFiles   map[string]*fileState // Track active log files
}

// fileState tracks file state and reference count
type fileState struct {
	refCount int
	lastUsed time.Time
}

// NewCmdLogger creates a new logger manager
func NewCmdLogger(logDir string, retentionDays int) (*CmdLogger, error) {
	if retentionDays <= 0 || logDir == types.EmptyString {
		return nil, fmt.Errorf("init cmd logger err: invalid retentionDays %d or logDir %s", retentionDays, logDir)
	}
	logx.Infof("cmd_logger: Initializing logger manager: logDir=%s, retentionDays=%d", logDir, retentionDays)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logx.Errorf("cmd_logger: Failed to create log directory: %v", err)
		return nil, err
	}

	logx.Infof("cmd_logger: Log directory %s created successfully", logDir)
	return &CmdLogger{
		logDir:        logDir,
		retentionDays: retentionDays,
		stopCh:        make(chan bool),
		activeFiles:   make(map[string]*fileState),
	}, nil
}

// GetWriter gets a log writer for the specified serviceName
func (l *CmdLogger) GetWriter(serviceName string) (io.WriteCloser, error) {
	logx.Debugf("cmd_logger: Getting log writer: serviceName=%s", serviceName)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Increase file reference count
	state, exists := l.activeFiles[serviceName]
	if !exists {
		logx.Debugf("cmd_logger: Creating new file state: serviceName=%s", serviceName)
		state = &fileState{refCount: 1, lastUsed: time.Now()}
		l.activeFiles[serviceName] = state
	} else {
		logx.Debugf("cmd_logger: Increasing file reference count: serviceName=%s, current count=%d", serviceName, state.refCount+1)
		state.refCount++
		state.lastUsed = time.Now()
	}

	// Open or create log file
	filePath := filepath.Join(l.logDir, serviceName+".log")
	logx.Debugf("cmd_logger: Opening log file: path=%s", filePath)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logx.Errorf("cmd_logger: Failed to open log file: path=%s, err=%v", filePath, err)

		// Decrease reference count on error
		state.refCount--
		if state.refCount <= 0 {
			delete(l.activeFiles, serviceName)
			logx.Debugf("cmd_logger: Deleting file state: serviceName=%s", serviceName)
		}

		return nil, err
	}

	logx.Debugf("cmd_logger: Successfully obtained log writer: serviceName=%s", serviceName)
	return &prefixWriter{
		writer:      file,
		logger:      l,
		serviceName: serviceName,
		openTime:    time.Now(),
	}, nil
}

// prefixWriter adds timestamp prefix to log entries
type prefixWriter struct {
	writer      io.WriteCloser
	logger      *CmdLogger
	serviceName string
	openTime    time.Time // File open time for detecting need to reopen
}

func (w *prefixWriter) Write(p []byte) (n int, err error) {
	logx.Debugf("cmd_logger: Writing log, bytes=%d", len(p))
	// Acquire write lock to ensure thread-safety during file reopening
	w.logger.mu.Lock()
	defer w.logger.mu.Unlock()
	// Check if file needs to be reopened (after midnight)
	if w.needReopen() {
		logx.Infof("cmd_logger: File reopen required, serviceName=%s, openTime=%s",
			w.serviceName, w.openTime.Format("2006-01-02 15:04:05"))
		if err := w.reopenFile(); err != nil {
			logx.Errorf("cmd_logger: Failed to reopen file, error=%v", err)
			return 0, err
		}
	}

	// Generate current logItems for each write
	logItems := []byte("[" + time.Now().Format("2006-01-02 15:04:05") + "]" + "  ")

	// Process each line of the log
	lines := strings.Split(string(p), "\n")
	for i, line := range lines {
		if line == "" && i == len(lines)-1 {
			continue // Skip last empty line
		}

		// Retry mechanism with up to 3 attempts
		var attempt int
		var writeErr error

		for attempt < 3 {
			// Combine logItems and log line
			fullLine := append(logItems, []byte(line+"\n")...)

			// Perform the write operation
			_, writeErr = w.writer.Write(fullLine)
			if writeErr != nil {
				attempt++
				logx.Errorf("cmd_logger: Write failed, retrying (attempt %d), error=%v", attempt, writeErr)

				if attempt >= 3 {
					logx.Errorf("cmd_logger: Max retries reached, write failed, error=%v", writeErr)
					return 0, writeErr
				}

				// Release lock during retry delay to prevent blocking
				w.logger.mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				w.logger.mu.Lock() // Re-acquire lock before retrying
			} else {
				break // Successful write, exit retry loop
			}
		}
	}

	logx.Debugf("cmd_logger: Successfully wrote log, bytes=%d", len(p))
	return len(p), nil
}

func (w *prefixWriter) needReopen() bool {
	// Check if file needs to be reopened if opened on a different day
	now := time.Now()
	result := now.Year() != w.openTime.Year() ||
		now.Month() != w.openTime.Month() ||
		now.Day() != w.openTime.Day()

	logx.Debugf("cmd_logger: Checking file reopen needed:,result=%t", result)
	return result
}

func (w *prefixWriter) reopenFile() error {
	logx.Infof("cmd_logger: Reopening file:%s", w.serviceName)

	// Acquire write lock to ensure no other operations during reopen
	w.logger.mu.Lock()
	defer w.logger.mu.Unlock()

	// Close current file
	logx.Debugf("cmd_logger: Closing old file: serviceName=%s", w.serviceName)
	if err := w.writer.Close(); err != nil {
		logx.Errorf("cmd_logger: Failed to close old file: serviceName=%s, err=%v", w.serviceName, err)
		return err
	}

	// Open new file
	filePath := filepath.Join(w.logger.logDir, w.serviceName+".log")
	logx.Debugf("cmd_logger: Opening new file: path=%s", filePath)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logx.Errorf("cmd_logger: Failed to open new file: path=%s, err=%v", filePath, err)
		return err
	}

	// Update writer and open time
	w.writer = file
	w.openTime = time.Now()

	logx.Info("cmd_logger: Successfully reopened file:", w.serviceName)
	return nil
}

func (w *prefixWriter) Close() error {
	logx.Debugf("cmd_logger: Closing log writer: serviceName=%s", w.serviceName)

	w.logger.mu.Lock()
	defer w.logger.mu.Unlock()

	// Decrease file reference count
	if state, exists := w.logger.activeFiles[w.serviceName]; exists {
		logx.Debugf("cmd_logger: Decreasing file reference count: serviceName=%s, current count=%d", w.serviceName, state.refCount-1)
		state.refCount--
		if state.refCount <= 0 {
			delete(w.logger.activeFiles, w.serviceName)
			logx.Debugf("cmd_logger: Deleting file state: serviceName=%s", w.serviceName)
		}
	}

	err := w.writer.Close()
	if err != nil {
		logx.Errorf("cmd_logger: Failed to close file: serviceName=%s, err=%v", w.serviceName, err)
	} else {
		logx.Debugf("cmd_logger: Successfully closed file: serviceName=%s", w.serviceName)
	}

	return err
}

// StartRotateBackground starts the daily rotation and cleanup task
func (l *CmdLogger) StartRotateBackground() {
	logx.Info("cmd_logger: Starting daily log rotation task")

	l.mu.Lock()
	defer l.mu.Unlock()

	// Prevent duplicate starts
	if l.taskRunning {
		logx.Infof("cmd_logger: Log rotation task is already running")
		return
	}
	l.taskRunning = true

	// Calculate midnight time
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// If midnight has passed today, start from tomorrow
	if midnight.Before(now) {
		midnight = midnight.Add(24 * time.Hour)
	}

	logx.Infof("cmd_logger: Task will execute first at %s", midnight.Format("2006-01-02 15:04:05"))

	// Goroutine to wait until midnight for first execution
	go func() {
		select {
		case <-time.After(midnight.Sub(now)):
			logx.Info("cmd_logger: Executing initial log rotation task")
			l.rotateAndClean()

			// Set up daily ticker
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					logx.Info("cmd_logger: Executing daily log rotation task")
					l.rotateAndClean()
				case <-l.stopCh:
					logx.Info("cmd_logger: Stopping log rotation task")
					return
				}
			}
		case <-l.stopCh:
			logx.Info("cmd_logger: Stopping log rotation task")
			return
		}
	}()
}

// Close stops the daily rotation task
func (l *CmdLogger) Close() {
	logx.Info("cmd_logger: Closing logger manager")

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.taskRunning {
		logx.Info("cmd_logger: Sending stop signal to rotation task")
		l.stopCh <- true
		l.taskRunning = false
	}
}

// rotateAndClean performs log rotation and cleanup
func (l *CmdLogger) rotateAndClean() {
	logx.Info("cmd_logger: Starting log rotation and cleanup")

	l.mu.Lock()
	defer l.mu.Unlock()

	// Log rotation
	logx.Info("cmd_logger: Starting log rotation")
	if err := l.rotateFiles(); err != nil {
		logx.Errorf("Log rotation failed: %v", err)
	} else {
		logx.Info("cmd_logger: Log rotation completed")
	}

	// Log cleanup
	logx.Info("cmd_logger: Starting log cleanup")
	if err := l.cleanOldLogs(); err != nil {
		logx.Errorf("Log cleanup failed: %v", err)
	} else {
		logx.Info("cmd_logger: Log cleanup completed")
	}
}

// rotateFiles rotates all log files
func (l *CmdLogger) rotateFiles() error {
	logx.Info("cmd_logger: Processing log file rotation")

	date := time.Now().Format("2006-01-02")
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		logx.Errorf("cmd_logger: Failed to read log directory: %v", err)
		return err
	}

	rotatedCount := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".log") {
			continue
		}

		// Extract serviceName
		serviceName := strings.TrimSuffix(f.Name(), ".log")

		// Skip active files being written to
		if state, ok := l.activeFiles[serviceName]; ok {
			// Check if file hasn't been used for a long time
			if time.Since(state.lastUsed) > 30*time.Minute {
				logx.Infof("cmd_logger: File marked active but unused for long time: serviceName=%s, lastUsed=%s",
					serviceName, state.lastUsed.Format("2006-01-02 15:04:05"))
				continue
			}

			logx.Debugf("cmd_logger: Skipping active file: serviceName=%s, refCount=%d", serviceName, state.refCount)
			continue
		}

		// Skip already rotated files
		if strings.Contains(f.Name(), "-") {
			logx.Debugf("cmd_logger: Skipping already rotated file: name=%s", f.Name())
			continue
		}

		oldPath := filepath.Join(l.logDir, f.Name())
		newPath := filepath.Join(l.logDir, serviceName+"-"+date+".log")

		logx.Infof("cmd_logger: Rotating log file: from=%s, to=%s", oldPath, newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			logx.Errorf("cmd_logger: Failed to rename file: from=%s, to=%s, err=%v", oldPath, newPath, err)
			return err
		}

		rotatedCount++
	}

	logx.Info("cmd_logger: Log rotation completed: processed", rotatedCount, "files")
	return nil
}

// cleanOldLogs cleans up expired log files
func (l *CmdLogger) cleanOldLogs() error {
	logx.Info("cmd_logger: Starting expired log cleanup")

	cutoff := time.Now().Add(-24 * time.Hour * time.Duration(l.retentionDays))
	logx.Info("cmd_logger: Log retention cutoff date:", cutoff.Format("2006-01-02"))

	files, err := os.ReadDir(l.logDir)
	if err != nil {
		logx.Errorf("cmd_logger: Failed to read log directory: %v", err)
		return err
	}

	deletedCount := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		filePath := filepath.Join(l.logDir, f.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			logx.Errorf("cmd_logger: Failed to get file info: path=%s, err=%v", filePath, err)
			continue
		}

		// Skip active files being written to
		podID := extractPodIDFromFileName(f.Name())
		if _, ok := l.activeFiles[podID]; ok {
			logx.Debugf("cmd_logger: Skipping active file: serviceName=%s", podID)
			continue
		}

		// Check file modification time
		if fileInfo.ModTime().Before(cutoff) {
			// Double-check with date in filename
			fileDate := extractDateFromFileName(f.Name())
			if !fileDate.IsZero() && fileDate.Before(cutoff) {
				logx.Infof("cmd_logger: Deleting expired log file: path=%s, modTime=%s, date=%s",
					filePath, fileInfo.ModTime().Format("2006-01-02"),
					fileDate.Format("2006-01-02"))

				if err := os.Remove(filePath); err != nil {
					logx.Errorf("cmd_logger: Failed to delete file: path=%s, err=%v", filePath, err)
					return err
				}

				deletedCount++
			} else {
				logx.Debugf("cmd_logger: File modTime expired but date not expired: path=%s, modTime=%s",
					filePath, fileInfo.ModTime().Format("2006-01-02"))
			}
		} else {
			logx.Debugf("cmd_logger: File not expired: path=%s, modTime=%s",
				filePath, fileInfo.ModTime().Format("2006-01-02"))
		}
	}

	logx.Info("cmd_logger: Log cleanup completed: deleted", deletedCount, "files")
	return nil
}

// extractPodIDFromFileName extracts serviceName from filename
func extractPodIDFromFileName(name string) string {
	// Handle formats: serviceName.log or serviceName-2025-06-19.log
	if !strings.Contains(name, "-") {
		return strings.TrimSuffix(name, ".log")
	}

	// For files with date suffix, extract the prefix
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return ""
	}

	// Check if last part is a date
	if _, err := time.Parse("2006-01-02", parts[len(parts)-1]); err == nil {
		return strings.Join(parts[:len(parts)-1], "-")
	}

	return strings.TrimSuffix(name, ".log")
}

// extractDateFromFileName extracts date from filename
func extractDateFromFileName(name string) time.Time {
	logx.Debugf("cmd_logger: Attempting to extract date from filename: name=%s", name)

	// Support format: serviceName-YYYY-MM-DD.log
	if !strings.Contains(name, "-") || !strings.HasSuffix(name, ".log") {
		logx.Debugf("cmd_logger: Filename format mismatch, cannot extract date: name=%s", name)
		return time.Time{}
	}

	// Extract part after last hyphen
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		logx.Debugf("cmd_logger: Filename format mismatch, cannot extract date: name=%s", name)
		return time.Time{}
	}

	datePart := parts[len(parts)-1]
	datePart = strings.TrimSuffix(datePart, ".log")

	// Strict date format check
	t, err := time.Parse("2006-01-02", datePart)
	if err != nil {
		logx.Debugf("cmd_logger: Failed to parse date: name=%s, datePart=%s, err=%v", name, datePart, err)
		return time.Time{}
	}

	logx.Debugf("cmd_logger: Successfully extracted date from filename: name=%s, date=%s", name, t.Format("2006-01-02"))
	return t
}
