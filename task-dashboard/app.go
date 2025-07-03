package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Status   string `json:"status"`   // "todo", "doing", "done"
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
	// Get the path to the task.json file
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
		workDir = "."
	}
	
	// Look for plan/task.json in parent directories
	taskFile := filepath.Join(workDir, "..", "plan", "task.json")
	if _, err := os.Stat(taskFile); os.IsNotExist(err) {
		// Try current directory
		taskFile = filepath.Join(workDir, "plan", "task.json")
		if _, err := os.Stat(taskFile); os.IsNotExist(err) {
			// Default to relative path
			taskFile = "../plan/task.json"
		}
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

// LoadTasks reads tasks from the plan/task.json file
func (a *App) LoadTasks() ([]Task, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if err := a.loadTasks(); err != nil {
		return nil, err
	}
	
	return a.tasks, nil
}

// SaveTasks writes tasks to the plan/task.json file with atomic operation
func (a *App) SaveTasks(tasks []Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Validate tasks
	for _, task := range tasks {
		if task.Title == "" {
			return fmt.Errorf("task with ID %d has empty title", task.ID)
		}
		if task.Status != "todo" && task.Status != "doing" && task.Status != "done" {
			return fmt.Errorf("task with ID %d has invalid status: %s", task.ID, task.Status)
		}
		if task.Priority != "high" && task.Priority != "medium" && task.Priority != "low" {
			return fmt.Errorf("task with ID %d has invalid priority: %s", task.ID, task.Priority)
		}
	}
	
	// Create backup
	if err := a.createBackup(); err != nil {
		a.logError("Failed to create backup", err)
		// Continue anyway
	}
	
	// Atomic write
	if err := a.atomicWriteTasks(tasks); err != nil {
		return err
	}
	
	a.tasks = tasks
	a.logInfo("Tasks saved successfully")
	return nil
}

// UpdateTask updates a specific task
func (a *App) UpdateTask(task Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if err := a.loadTasks(); err != nil {
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
	if err := a.atomicWriteTasks(a.tasks); err != nil {
		return err
	}
	
	a.logInfo(fmt.Sprintf("Task %d updated successfully", task.ID))
	return nil
}

// MoveTask moves a task to a different status column
func (a *App) MoveTask(taskID int, newStatus string) error {
	if newStatus != "todo" && newStatus != "doing" && newStatus != "done" {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if err := a.loadTasks(); err != nil {
		return err
	}
	
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
	if err := a.atomicWriteTasks(a.tasks); err != nil {
		return err
	}
	
	a.logInfo(fmt.Sprintf("Task %d moved to %s", taskID, newStatus))
	return nil
}

// GetTasksByStatus returns tasks filtered by status
func (a *App) GetTasksByStatus(status string) ([]Task, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if err := a.loadTasks(); err != nil {
		return nil, err
	}
	
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
	data, err := ioutil.ReadFile(a.taskFile)
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
	if err := ioutil.WriteFile(tmpFile, data, 0644); err != nil {
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
	
	data, err := ioutil.ReadFile(a.taskFile)
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(backupFile, data, 0644)
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
	
	// Construct log file path
	logDir := filepath.Join(filepath.Dir(a.taskFile), "..", "logs")
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
