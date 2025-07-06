package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileUtils provides atomic file operations with backup and rollback
type FileUtils struct {
	logger Logger
}

// NewFileUtils creates a new file utilities instance
func NewFileUtils(logger Logger) *FileUtils {
	return &FileUtils{
		logger: logger,
	}
}

// AtomicWriteJSON writes JSON data atomically with automatic backup
func (fu *FileUtils) AtomicWriteJSON(filePath string, data interface{}) error {
	// Create backup first
	backupPath, err := fu.CreateBackup(filePath)
	if err != nil {
		fu.logger.Error("Failed to create backup before write", err)
		// Continue anyway - backup is nice to have but not critical
	}

	// Marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write atomically
	if err := fu.AtomicWrite(filePath, jsonData); err != nil {
		// Attempt rollback if we have a backup
		if backupPath != "" {
			fu.logger.Info("Attempting rollback after write failure")
			if rollbackErr := fu.Rollback(filePath, backupPath); rollbackErr != nil {
				fu.logger.Error("Rollback failed", rollbackErr)
			}
		}
		return err
	}

	return nil
}

// AtomicWrite performs an atomic file write operation
func (fu *FileUtils) AtomicWrite(filePath string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tmpFile := fmt.Sprintf("%s.tmp.%d", filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Ensure data is flushed to disk
	if file, err := os.OpenFile(tmpFile, os.O_RDWR, 0644); err == nil {
		file.Sync()
		file.Close()
	}

	// Atomic rename
	if err := os.Rename(tmpFile, filePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	fu.logger.InfoWithFields("Atomic write completed", map[string]interface{}{
		"file": filePath,
		"size": len(data),
	})

	return nil
}

// CreateBackup creates a timestamped backup of a file
func (fu *FileUtils) CreateBackup(filePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil // No file to backup
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", filePath, timestamp)

	// Copy file
	if err := fu.CopyFile(filePath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	fu.logger.InfoWithFields("Backup created", map[string]interface{}{
		"original": filePath,
		"backup":   backupPath,
	})

	return backupPath, nil
}

// Rollback restores a file from backup
func (fu *FileUtils) Rollback(filePath, backupPath string) error {
	if backupPath == "" || filePath == "" {
		return fmt.Errorf("invalid file paths for rollback")
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Copy backup to original location
	if err := fu.CopyFile(backupPath, filePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	fu.logger.InfoWithFields("Rollback completed", map[string]interface{}{
		"file":   filePath,
		"backup": backupPath,
	})

	return nil
}

// CopyFile copies a file from src to dst
func (fu *FileUtils) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy data
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// CleanupOldBackups removes backup files older than the specified duration
func (fu *FileUtils) CleanupOldBackups(pattern string, maxAge time.Duration) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				fu.logger.Error("Failed to remove old backup", err)
			} else {
				removed++
			}
		}
	}

	if removed > 0 {
		fu.logger.InfoWithFields("Cleaned up old backups", map[string]interface{}{
			"pattern": pattern,
			"removed": removed,
		})
	}

	return nil
}