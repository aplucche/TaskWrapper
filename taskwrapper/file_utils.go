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

// AtomicWriteJSON writes JSON data atomically with .tracked file backup
func (fu *FileUtils) AtomicWriteJSON(filePath string, data interface{}) error {
	// Update .tracked file first
	trackedPath := filePath + ".tracked"
	if err := fu.UpdateTrackedFile(filePath, trackedPath); err != nil {
		fu.logger.Error("Failed to update tracked file before write", err)
		// Continue anyway - tracked file is nice to have but not critical
	}

	// Marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write atomically
	if err := fu.AtomicWrite(filePath, jsonData); err != nil {
		// Attempt rollback from .tracked file
		fu.logger.Info("Attempting rollback after write failure")
		if rollbackErr := fu.RestoreFromTracked(filePath, trackedPath); rollbackErr != nil {
			fu.logger.Error("Rollback from tracked file failed", rollbackErr)
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

// UpdateTrackedFile copies the current file to its .tracked version
func (fu *FileUtils) UpdateTrackedFile(filePath, trackedPath string) error {
	// Check if source file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to track
	}

	// Copy file to .tracked version
	if err := fu.CopyFile(filePath, trackedPath); err != nil {
		return fmt.Errorf("failed to update tracked file: %w", err)
	}

	fu.logger.InfoWithFields("Tracked file updated", map[string]interface{}{
		"original": filePath,
		"tracked":  trackedPath,
	})

	return nil
}

// RestoreFromTracked restores a file from its .tracked version
func (fu *FileUtils) RestoreFromTracked(filePath, trackedPath string) error {
	if trackedPath == "" || filePath == "" {
		return fmt.Errorf("invalid file paths for restore")
	}

	// Check if tracked file exists
	if _, err := os.Stat(trackedPath); os.IsNotExist(err) {
		return fmt.Errorf("tracked file does not exist: %s", trackedPath)
	}

	// Copy tracked file to original location
	if err := fu.CopyFile(trackedPath, filePath); err != nil {
		return fmt.Errorf("failed to restore from tracked file: %w", err)
	}

	fu.logger.InfoWithFields("Restored from tracked file", map[string]interface{}{
		"file":    filePath,
		"tracked": trackedPath,
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

