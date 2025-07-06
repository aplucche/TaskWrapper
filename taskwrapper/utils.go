package main

import (
	"fmt"
	"os"
	"time"
)

// readFileContent reads content from a file
func readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// writeFileContent writes content to a file
func writeFileContent(filePath string, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// createFileBackup creates a timestamped backup of a file
func createFileBackup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to backup
	}
	
	timestamp := time.Now().Format("20060102_150405")
	backupFile := fmt.Sprintf("%s.backup.%s", filePath, timestamp)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupFile, data, 0644)
}