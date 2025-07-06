package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskService handles task-related operations
type TaskService struct {
	taskFile string
	mu       sync.RWMutex
	tasks    []Task
	logger   Logger
}

// NewTaskService creates a new task service
func NewTaskService(taskFile string, logger Logger) *TaskService {
	return &TaskService{
		taskFile: taskFile,
		tasks:    []Task{},
		logger:   logger,
	}
}

// LoadTasks reloads tasks from disk and returns them
func (ts *TaskService) LoadTasks() ([]Task, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Reload from disk to pick up external changes
	data, err := os.ReadFile(ts.taskFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty task file
			ts.tasks = []Task{}
			if writeErr := ts.atomicWriteTasks(ts.tasks); writeErr != nil {
				ts.logger.Error("Failed to create empty task file", writeErr)
				return ts.tasks, writeErr
			}
		} else {
			ts.logger.Error("Failed to read task file", err)
			return ts.tasks, fmt.Errorf("failed to read task file: %v", err)
		}
	} else {
		if err := json.Unmarshal(data, &ts.tasks); err != nil {
			ts.logger.Error("Failed to parse task file", err)
			return ts.tasks, fmt.Errorf("failed to parse task file: %v", err)
		}
	}
	
	ts.logger.Info("Tasks reloaded successfully from disk")
	return ts.tasks, nil
}

// SaveTasks writes tasks to the plan/task.json file with atomic operation
func (ts *TaskService) SaveTasks(tasks []Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Validate tasks
	if err := ts.validateTasks(tasks); err != nil {
		return err
	}
	
	// Update in-memory tasks
	ts.tasks = tasks
	
	// Save to disk
	if err := ts.saveTasks(); err != nil {
		return err
	}
	
	return nil
}

// UpdateTask updates a specific task
func (ts *TaskService) UpdateTask(task Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Validate single task
	if err := ts.validateTasks([]Task{task}); err != nil {
		return err
	}
	
	// Find and update the task
	found := false
	for i, t := range ts.tasks {
		if t.ID == task.ID {
			ts.tasks[i] = task
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", task.ID)
	}
	
	// Save updated tasks
	if err := ts.saveTasks(); err != nil {
		return err
	}
	
	ts.logger.Info(fmt.Sprintf("Task %d updated successfully", task.ID))
	return nil
}

// MoveTask moves a task to a different status column
func (ts *TaskService) MoveTask(taskID int, newStatus string) error {
	if newStatus != "backlog" && newStatus != "todo" && newStatus != "doing" && newStatus != "pending_review" && newStatus != "done" {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Find and update the task status
	found := false
	var oldStatus string
	for i, task := range ts.tasks {
		if task.ID == taskID {
			oldStatus = task.Status
			ts.tasks[i].Status = newStatus
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	// Save updated tasks
	if err := ts.saveTasks(); err != nil {
		return err
	}
	
	ts.logger.Info(fmt.Sprintf("Task %d moved from %s to %s", taskID, oldStatus, newStatus))
	return nil
}

// GetTasksByStatus returns tasks filtered by status
func (ts *TaskService) GetTasksByStatus(status string) ([]Task, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	var filtered []Task
	for _, task := range ts.tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	
	return filtered, nil
}

// GetTasks returns all tasks (thread-safe)
func (ts *TaskService) GetTasks() []Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	tasksCopy := make([]Task, len(ts.tasks))
	copy(tasksCopy, ts.tasks)
	return tasksCopy
}

// SetTaskFile changes the task file path
func (ts *TaskService) SetTaskFile(path string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.taskFile = path
}

// Private helper methods

// validateTasks validates a slice of tasks
func (ts *TaskService) validateTasks(tasks []Task) error {
	for _, task := range tasks {
		if task.Title == "" {
			return fmt.Errorf("task with ID %d has empty title", task.ID)
		}
		if task.Status != "backlog" && task.Status != "todo" && task.Status != "doing" && task.Status != "pending_review" && task.Status != "done" {
			return fmt.Errorf("task with ID %d has invalid status: %s", task.ID, task.Status)
		}
		if task.Priority != "high" && task.Priority != "medium" && task.Priority != "low" {
			return fmt.Errorf("task with ID %d has invalid priority: %s", task.ID, task.Priority)
		}
	}
	return nil
}

// saveTasks persists the current in-memory tasks to disk
func (ts *TaskService) saveTasks() error {
	// Create backup before saving
	if err := ts.createBackup(); err != nil {
		ts.logger.Error("Failed to create backup", err)
		return fmt.Errorf("failed to create backup: %v", err)
	}
	
	// Atomic write
	if err := ts.atomicWriteTasks(ts.tasks); err != nil {
		return err
	}
	
	ts.logger.Info("Tasks saved successfully")
	return nil
}

// atomicWriteTasks writes tasks to file atomically
func (ts *TaskService) atomicWriteTasks(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(ts.taskFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Write to temporary file first
	tmpFile := ts.taskFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %v", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpFile, ts.taskFile); err != nil {
		os.Remove(tmpFile) // Clean up
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}
	
	return nil
}

// createBackup creates a backup of the task file
func (ts *TaskService) createBackup() error {
	if _, err := os.Stat(ts.taskFile); os.IsNotExist(err) {
		return nil // No file to backup
	}
	
	timestamp := time.Now().Format("20060102_150405")
	backupFile := ts.taskFile + ".backup." + timestamp
	
	data, err := os.ReadFile(ts.taskFile)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupFile, data, 0644)
}