package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Task represents a single task in the kanban board
type Task struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`   // "backlog", "todo", "doing", "done"
	Priority string `json:"priority"` // "high", "medium", "low"
	Deps     []int  `json:"deps"`     // array of task IDs this task depends on
	Parent   *int   `json:"parent"`   // parent task ID, null if top-level
}

// App struct
type App struct {
	ctx      context.Context
	taskFile string
	mu       sync.RWMutex
	tasks    []Task
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Get the user's home directory for a more reliable path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v", err)
		homeDir = "."
	}
	
	// Try multiple possible locations for the task.json file
	possiblePaths := []string{
		// Try the project directory in repos
		filepath.Join(homeDir, "repos", "cc_task_dash", "plan", "task.json"),
		// Try current working directory
		filepath.Join(".", "plan", "task.json"),
		// Try parent directory
		filepath.Join("..", "plan", "task.json"),
		// Try relative to executable
		filepath.Join("../../plan", "task.json"),
		// Fallback: create in user documents
		filepath.Join(homeDir, "Documents", "TaskDashboard", "task.json"),
	}
	
	var taskFile string
	found := false
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			taskFile = path
			found = true
			break
		}
	}
	
	// If no existing file found, use the Documents fallback
	if !found {
		taskFile = filepath.Join(homeDir, "Documents", "TaskDashboard", "task.json")
	}

	app := &App{
		taskFile: taskFile,
		tasks:    []Task{},
	}
	
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Load tasks on startup
	if err := a.loadTasks(); err != nil {
		a.logError("Failed to load tasks on startup", err)
	} else {
		a.logInfo("Tasks loaded successfully on startup")
	}
}

// LoadTasks reloads tasks from disk and returns them
func (a *App) LoadTasks() ([]Task, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Reload from disk to pick up external changes
	data, err := os.ReadFile(a.taskFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty task file
			a.tasks = []Task{}
			if writeErr := a.atomicWriteTasks(a.tasks); writeErr != nil {
				a.logError("Failed to create empty task file", writeErr)
				return a.tasks, writeErr
			}
		} else {
			a.logError("Failed to read task file", err)
			return a.tasks, fmt.Errorf("failed to read task file: %v", err)
		}
	} else {
		if err := json.Unmarshal(data, &a.tasks); err != nil {
			a.logError("Failed to parse task file", err)
			return a.tasks, fmt.Errorf("failed to parse task file: %v", err)
		}
	}
	
	a.logInfo("Tasks reloaded successfully from disk")
	return a.tasks, nil
}

// SaveTasks writes tasks to the plan/task.json file with atomic operation
func (a *App) SaveTasks(tasks []Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Validate tasks
	if err := a.validateTasks(tasks); err != nil {
		return err
	}
	
	// Update in-memory tasks
	a.tasks = tasks
	
	// Save to disk
	if err := a.saveTasks(); err != nil {
		return err
	}
	
	return nil
}

// UpdateTask updates a specific task
func (a *App) UpdateTask(task Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Validate single task
	if err := a.validateTasks([]Task{task}); err != nil {
		return err
	}
	
	// Find and update the task
	found := false
	for i, t := range a.tasks {
		if t.ID == task.ID {
			a.tasks[i] = task
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", task.ID)
	}
	
	// Save updated tasks
	if err := a.saveTasks(); err != nil {
		return err
	}
	
	a.logInfo(fmt.Sprintf("Task %d updated successfully", task.ID))
	return nil
}

// MoveTask moves a task to a different status column
func (a *App) MoveTask(taskID int, newStatus string) error {
	if newStatus != "backlog" && newStatus != "todo" && newStatus != "doing" && newStatus != "done" {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Find and update the task status
	found := false
	for i, task := range a.tasks {
		if task.ID == taskID {
			a.tasks[i].Status = newStatus
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	// Save updated tasks
	if err := a.saveTasks(); err != nil {
		return err
	}
	
	a.logInfo(fmt.Sprintf("Task %d moved to %s", taskID, newStatus))
	return nil
}

// GetTasksByStatus returns tasks filtered by status
func (a *App) GetTasksByStatus(status string) ([]Task, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var filtered []Task
	for _, task := range a.tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	
	return filtered, nil
}

// Private helper methods

func (a *App) loadTasks() error {
	data, err := os.ReadFile(a.taskFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty task file
			a.tasks = []Task{}
			return a.atomicWriteTasks(a.tasks)
		}
		return fmt.Errorf("failed to read task file: %v", err)
	}
	
	if err := json.Unmarshal(data, &a.tasks); err != nil {
		return fmt.Errorf("failed to parse task file: %v", err)
	}
	
	return nil
}

// validateTasks validates a slice of tasks
func (a *App) validateTasks(tasks []Task) error {
	for _, task := range tasks {
		if task.Title == "" {
			return fmt.Errorf("task with ID %d has empty title", task.ID)
		}
		if task.Status != "backlog" && task.Status != "todo" && task.Status != "doing" && task.Status != "done" {
			return fmt.Errorf("task with ID %d has invalid status: %s", task.ID, task.Status)
		}
		if task.Priority != "high" && task.Priority != "medium" && task.Priority != "low" {
			return fmt.Errorf("task with ID %d has invalid priority: %s", task.ID, task.Priority)
		}
	}
	return nil
}

// saveTasks persists the current in-memory tasks to disk
func (a *App) saveTasks() error {
	// Create backup before saving
	if err := a.createBackup(); err != nil {
		a.logError("Failed to create backup", err)
		return fmt.Errorf("failed to create backup: %v", err)
	}
	
	// Atomic write
	if err := a.atomicWriteTasks(a.tasks); err != nil {
		return err
	}
	
	a.logInfo("Tasks saved successfully")
	return nil
}

func (a *App) atomicWriteTasks(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(a.taskFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Write to temporary file first
	tmpFile := a.taskFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %v", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpFile, a.taskFile); err != nil {
		os.Remove(tmpFile) // Clean up
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}
	
	return nil
}

func (a *App) createBackup() error {
	if _, err := os.Stat(a.taskFile); os.IsNotExist(err) {
		return nil // No file to backup
	}
	
	timestamp := time.Now().Format("20060102_150405")
	backupFile := a.taskFile + ".backup." + timestamp
	
	data, err := os.ReadFile(a.taskFile)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupFile, data, 0644)
}

func (a *App) logInfo(message string) {
	a.logToFile("INFO", message)
}

func (a *App) logError(message string, err error) {
	fullMessage := fmt.Sprintf("%s: %v", message, err)
	a.logToFile("ERROR", fullMessage)
}

func (a *App) logToFile(level, message string) {
	// Get current date for log file
	now := time.Now()
	logDate := now.Format("2006-01-02")
	
	// Try to find the logs directory in the same structure as task file
	var logDir string
	taskDir := filepath.Dir(a.taskFile)
	
	// If task file is in a "plan" directory, logs should be sibling to plan
	if filepath.Base(taskDir) == "plan" {
		logDir = filepath.Join(filepath.Dir(taskDir), "logs")
	} else {
		// Otherwise, create logs next to task file
		logDir = filepath.Join(taskDir, "logs")
	}
	
	logFile := filepath.Join(logDir, "universal_logs-"+logDate+".log")
	
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return
	}
	
	// Format log entry
	timestamp := now.Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s task-dashboard: %s\n", timestamp, level, message)
	
	// Append to log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer f.Close()
	
	if _, err := f.WriteString(logEntry); err != nil {
		log.Printf("Failed to write to log file: %v", err)
	}
}
