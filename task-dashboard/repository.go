package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RepositoryInfo contains detailed information about a repository
type RepositoryInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	AddedAt      time.Time `json:"addedAt"`
	IsValid      bool      `json:"isValid"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	TaskCount    int       `json:"taskCount"`
	HasPlanFile  bool      `json:"hasPlanFile"`
}

// ValidateRepository performs comprehensive validation of a repository path
func ValidateRepository(path string) (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		Path:    path,
		Name:    GetRepositoryName(path),
		IsValid: true,
		AddedAt: time.Now(),
	}
	
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		info.IsValid = false
		info.ErrorMessage = "Path does not exist"
		return info, nil
	}
	
	// Check for plan directory
	planDir := filepath.Join(path, "plan")
	if _, err := os.Stat(planDir); err != nil {
		info.IsValid = false
		info.ErrorMessage = "Missing plan directory"
		return info, nil
	}
	
	// Check for task.json
	taskFile := filepath.Join(planDir, "task.json")
	if _, err := os.Stat(taskFile); err != nil {
		info.IsValid = false
		info.ErrorMessage = "Missing plan/task.json file"
		return info, nil
	}
	
	// Check for plan.md
	planFile := filepath.Join(planDir, "plan.md")
	if _, err := os.Stat(planFile); err == nil {
		info.HasPlanFile = true
	}
	
	// Try to count tasks
	tasks, err := loadTasksFromPath(taskFile)
	if err != nil {
		info.IsValid = false
		info.ErrorMessage = fmt.Sprintf("Invalid task.json: %v", err)
		return info, nil
	}
	info.TaskCount = len(tasks)
	
	return info, nil
}

// loadTasksFromPath loads tasks from a specific path
func loadTasksFromPath(taskFile string) ([]Task, error) {
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, err
	}
	
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// GetRepositoryName attempts to determine a good display name for a repository
func GetRepositoryName(path string) string {
	// First, try to use the directory name
	name := filepath.Base(path)
	
	// If it's a generic name, try to get parent/child combination
	genericNames := []string{".", "..", "plan", "task-dashboard", "cc_task_dash"}
	for _, generic := range genericNames {
		if strings.EqualFold(name, generic) {
			parent := filepath.Base(filepath.Dir(path))
			if parent != "" && parent != "." && parent != ".." {
				name = parent
			}
			break
		}
	}
	
	// Clean up the name
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Title(strings.ToLower(name))
	
	return name
}

// FindRepositoriesInDirectory searches for task dashboard repositories in a directory
func FindRepositoriesInDirectory(searchPath string) ([]Repository, error) {
	var repositories []Repository
	
	// Walk the directory tree, but not too deep
	maxDepth := 3
	err := walkDirectoryWithDepth(searchPath, maxDepth, func(path string) error {
		// Check if this is a repository
		taskFile := filepath.Join(path, "plan", "task.json")
		if _, err := os.Stat(taskFile); err == nil {
			repo := Repository{
				ID:   generateID(),
				Name: GetRepositoryName(path),
				Path: path,
			}
			repositories = append(repositories, repo)
		}
		return nil
	})
	
	return repositories, err
}

// walkDirectoryWithDepth walks a directory tree with a maximum depth
func walkDirectoryWithDepth(root string, maxDepth int, fn func(string) error) error {
	return walkDirectoryWithDepthHelper(root, 0, maxDepth, fn)
}

func walkDirectoryWithDepthHelper(path string, currentDepth, maxDepth int, fn func(string) error) error {
	if currentDepth > maxDepth {
		return nil
	}
	
	// Call the function for current directory
	if err := fn(path); err != nil {
		return err
	}
	
	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil // Skip directories we can't read
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories and common non-project directories
			name := entry.Name()
			if strings.HasPrefix(name, ".") || 
			   name == "node_modules" || 
			   name == "vendor" || 
			   name == "target" || 
			   name == "dist" || 
			   name == "build" {
				continue
			}
			
			subPath := filepath.Join(path, name)
			if err := walkDirectoryWithDepthHelper(subPath, currentDepth+1, maxDepth, fn); err != nil {
				return err
			}
		}
	}
	
	return nil
}