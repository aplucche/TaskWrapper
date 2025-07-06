package main

import (
	"os"
	"path/filepath"
	"strings"
)

// RepositoryUtils provides shared repository validation and search utilities
type RepositoryUtils struct{}

// IsValidRepository checks if a directory contains a valid task dashboard repository
func (ru *RepositoryUtils) IsValidRepository(path string) bool {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return false
	}
	
	// Check if plan/task.json exists
	taskFile := filepath.Join(path, "plan", "task.json")
	if _, err := os.Stat(taskFile); err != nil {
		return false
	}
	
	return true
}

// FindFirstRepositoryInDirectory finds the first valid repository in a directory
func (ru *RepositoryUtils) FindFirstRepositoryInDirectory(searchDir string) string {
	if _, err := os.Stat(searchDir); err != nil {
		return ""
	}
	
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories and common non-project directories
			name := entry.Name()
			if ru.shouldSkipDirectory(name) {
				continue
			}
			
			candidatePath := filepath.Join(searchDir, name)
			if ru.IsValidRepository(candidatePath) {
				return candidatePath
			}
		}
	}
	
	return ""
}

// shouldSkipDirectory returns true if directory should be skipped during search
func (ru *RepositoryUtils) shouldSkipDirectory(name string) bool {
	skipList := []string{"node_modules", "vendor", "target", "dist", "build", ".git", ".vscode", ".idea"}
	
	// Skip hidden directories
	if strings.HasPrefix(name, ".") {
		for _, allowed := range []string{".github", ".vscode"} {
			if name == allowed {
				return false
			}
		}
		return true
	}
	
	// Skip common build/dependency directories
	for _, skip := range skipList {
		if name == skip {
			return true
		}
	}
	
	return false
}

// GetCommonSearchDirectories returns common directories to search for repositories
func (ru *RepositoryUtils) GetCommonSearchDirectories(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "repos"),
		filepath.Join(homeDir, "projects"),
		filepath.Join(homeDir, "code"),
		filepath.Join(homeDir, "workspace"),
		filepath.Join(homeDir, "dev"),
		filepath.Join(homeDir, "Documents"),
	}
}