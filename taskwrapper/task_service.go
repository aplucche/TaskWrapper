package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TaskService handles task-related operations
type TaskService struct {
	taskFile  string
	mu        sync.RWMutex
	tasks     []Task
	logger    Logger
	fileUtils *FileUtils
}

// NewTaskService creates a new task service
func NewTaskService(taskFile string, logger Logger) *TaskService {
	return &TaskService{
		taskFile:  taskFile,
		tasks:     []Task{},
		logger:    logger,
		fileUtils: NewFileUtils(logger),
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
			if writeErr := ts.fileUtils.AtomicWriteJSON(ts.taskFile, ts.tasks); writeErr != nil {
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
	// Parse and validate the new status
	status, err := ParseTaskStatus(newStatus)
	if err != nil {
		return err
	}
	
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Find and update the task status
	found := false
	var oldStatus TaskStatus
	for i, task := range ts.tasks {
		if task.ID == taskID {
			oldStatus = task.Status
			ts.tasks[i].Status = status
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
	
	// Parse the status string
	taskStatus, err := ParseTaskStatus(status)
	if err != nil {
		return nil, err
	}
	
	var filtered []Task
	for _, task := range ts.tasks {
		if task.Status == taskStatus {
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
		if !task.Status.Valid() {
			return fmt.Errorf("task with ID %d has invalid status: %s", task.ID, task.Status)
		}
		if !task.Priority.Valid() {
			return fmt.Errorf("task with ID %d has invalid priority: %s", task.ID, task.Priority)
		}
	}
	return nil
}

// saveTasks persists the current in-memory tasks to disk
func (ts *TaskService) saveTasks() error {
	// Use FileUtils for atomic write with automatic backup
	if err := ts.fileUtils.AtomicWriteJSON(ts.taskFile, ts.tasks); err != nil {
		ts.logger.Error("Failed to save tasks", err)
		return fmt.Errorf("failed to save tasks: %v", err)
	}
	
	ts.logger.Info("Tasks saved successfully")
	
	// Clean up old backups (older than 7 days)
	go func() {
		pattern := ts.taskFile + ".backup.*"
		if err := ts.fileUtils.CleanupOldBackups(pattern, 7*24*time.Hour); err != nil {
			ts.logger.Error("Failed to cleanup old backups", err)
		}
	}()
	
	return nil
}

