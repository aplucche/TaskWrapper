package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Domain Types

// Task represents a single task in the kanban board
type Task struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`   // "backlog", "todo", "doing", "pending_review", "done"
	Priority string `json:"priority"` // "high", "medium", "low"
	Deps     []int  `json:"deps"`     // array of task IDs this task depends on
	Parent   *int   `json:"parent"`   // parent task ID, null if top-level
}

// Terminal represents a running terminal session
type Terminal struct {
	ID      string
	Cmd     *exec.Cmd
	Pty     *os.File
	Conn    *websocket.Conn
	Done    chan bool
	Buffer  *TerminalBuffer
}

// TerminalBuffer stores recent terminal output for reconnection
type TerminalBuffer struct {
	Lines    []string
	MaxLines int
	MaxBytes int
	mu       sync.Mutex
}

// TerminalMessage represents messages sent between frontend and backend
type TerminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// AgentWorktree represents a single subagent worktree
type AgentWorktree struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	TaskID    string `json:"taskId,omitempty"`
	TaskTitle string `json:"taskTitle,omitempty"`
	PID       string `json:"pid,omitempty"`
	Started   string `json:"started,omitempty"`
}

// AgentStatusInfo represents the overall agent status
type AgentStatusInfo struct {
	Worktrees     []AgentWorktree `json:"worktrees"`
	TotalWorktrees int            `json:"totalWorktrees"`
	IdleCount     int            `json:"idleCount"`
	BusyCount     int            `json:"busyCount"`
	MaxSubagents  int            `json:"maxSubagents"`
}

// Logger interface for structured logging
type Logger interface {
	Info(message string)
	Error(message string, err error)
	InfoWithFields(message string, fields map[string]interface{})
	ErrorWithFields(message string, err error, fields map[string]interface{})
}

// Service Interfaces

// TaskServiceInterface defines the task service contract
type TaskServiceInterface interface {
	LoadTasks() ([]Task, error)
	SaveTasks(tasks []Task) error
	UpdateTask(task Task) error
	MoveTask(taskID int, newStatus string) error
	GetTasksByStatus(status string) ([]Task, error)
	GetTasks() []Task
	SetTaskFile(path string)
}

// TerminalServiceInterface defines the terminal service contract
type TerminalServiceInterface interface {
	StartTerminalSession() string
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	CleanupTerminal(terminalID string)
	GetTerminal(terminalID string) (*Terminal, bool)
	SetContext(ctx context.Context)
}

// AgentServiceInterface defines the agent service contract
type AgentServiceInterface interface {
	LaunchClaudeAgent(task Task) error
	ApproveTask(taskID int, taskTitle string) error
	RejectTask(taskID int, taskTitle string) error
	GetAgentStatus() (AgentStatusInfo, error)
	SetProjectRoot(root string)
	SetContext(ctx context.Context)
}

// ConfigServiceInterface defines the config service contract
type ConfigServiceInterface interface {
	GetConfig() (*Config, error)
	GetRepositories() ([]Repository, error)
	GetActiveRepository() (*Repository, error)
	AddRepository(name, path string) (*Repository, error)
	RemoveRepository(id string) error
	SetActiveRepository(id string) error
	ValidateRepositoryPath(path string) (*RepositoryInfo, error)
	FindRepositories(searchPath string) ([]Repository, error)
	GetActiveRepositoryPath() (string, error)
}

// Helper methods for TerminalBuffer

// NewTerminalBuffer creates a new terminal buffer
func NewTerminalBuffer() *TerminalBuffer {
	return &TerminalBuffer{
		Lines:    make([]string, 0, 100), // Pre-allocate capacity
		MaxLines: 100,                    // Store last 100 lines
		MaxBytes: 50000,                  // Limit to ~50KB of buffer data
	}
}

// AddLine adds a line to the terminal buffer
func (tb *TerminalBuffer) AddLine(line string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.Lines = append(tb.Lines, line)
	
	// Keep only the last MaxLines and respect MaxBytes limit
	for len(tb.Lines) > tb.MaxLines || tb.getTotalBytes() > tb.MaxBytes {
		if len(tb.Lines) > 0 {
			tb.Lines = tb.Lines[1:]
		} else {
			break
		}
	}
}

// getTotalBytes calculates total bytes in buffer (must be called with lock held)
func (tb *TerminalBuffer) getTotalBytes() int {
	total := 0
	for _, line := range tb.Lines {
		total += len(line)
	}
	return total
}

// GetHistory returns all stored lines
func (tb *TerminalBuffer) GetHistory() []string {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	// Return a copy to avoid race conditions
	history := make([]string, len(tb.Lines))
	copy(history, tb.Lines)
	return history
}

// App struct with dependency injection
type App struct {
	ctx context.Context
	mu  sync.RWMutex

	// Services
	taskService     TaskServiceInterface
	terminalService TerminalServiceInterface
	agentService    AgentServiceInterface
	configService   ConfigServiceInterface
	logger          Logger
}

// NewApp creates a new App application struct with dependency injection
func NewApp() *App {
	// Create logger first
	logger := NewFileLogger("") // Will be updated with correct path after config is loaded
	
	// Initialize configuration service
	configService, err := NewConfigService(logger)
	if err != nil {
		logger.Error("Error initializing config service", err)
		// Fall back to old behavior
		return newAppWithoutConfig(logger)
	}
	
	// Get active repository from config
	activeRepo, err := configService.GetActiveRepository()
	if err != nil {
		logger.Error("Error getting active repository", err)
		// Fall back to old behavior
		return newAppWithoutConfig(logger)
	}
	
	// Update logger with correct log directory
	logDir := getLogDirectory(activeRepo.Path)
	logger = NewFileLogger(logDir)
	
	// Initialize services
	taskFile := filepath.Join(activeRepo.Path, "plan", "task.json")
	taskService := NewTaskService(taskFile, logger)
	
	terminalService := NewTerminalService(logger, []string{}) // Allow all origins for development
	
	agentService := NewAgentService(activeRepo.Path, logger)
	
	app := &App{
		taskService:     taskService,
		terminalService: terminalService,
		agentService:    agentService,
		configService:   configService,
		logger:          logger,
	}
	
	return app
}

// newAppWithoutConfig creates an app without configuration (fallback)
func newAppWithoutConfig(logger Logger) *App {
	// Create a temporary config manager to reuse detection logic
	tempConfigMgr := &ConfigManager{}
	repo := tempConfigMgr.detectCurrentRepository()
	
	// Update logger with correct log directory
	logDir := getLogDirectory(repo.Path)
	logger = NewFileLogger(logDir)
	
	// Initialize services with fallback repository
	taskFile := filepath.Join(repo.Path, "plan", "task.json")
	taskService := NewTaskService(taskFile, logger)
	
	terminalService := NewTerminalService(logger, []string{}) // Allow all origins for development
	
	agentService := NewAgentService(repo.Path, logger)
	
	app := &App{
		taskService:     taskService,
		terminalService: terminalService,
		agentService:    agentService,
		configService:   nil, // No config service in fallback mode
		logger:          logger,
	}
	
	return app
}

// getLogDirectory determines the correct log directory based on repository path
func getLogDirectory(repoPath string) string {
	return filepath.Join(repoPath, "logs")
}

// startup is called when the app starts. The context is saved so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Set context on services that need it
	a.terminalService.SetContext(ctx)
	a.agentService.SetContext(ctx)
	
	// Load tasks on startup
	if _, err := a.taskService.LoadTasks(); err != nil {
		a.logger.Error("Failed to load tasks on startup", err)
	} else {
		a.logger.Info("Tasks loaded successfully on startup")
	}
}

// Task-related API methods

// LoadTasks reloads tasks from disk and returns them
func (a *App) LoadTasks() ([]Task, error) {
	return a.taskService.LoadTasks()
}

// SaveTasks writes tasks to the plan/task.json file with atomic operation
func (a *App) SaveTasks(tasks []Task) error {
	return a.taskService.SaveTasks(tasks)
}

// UpdateTask updates a specific task
func (a *App) UpdateTask(task Task) error {
	return a.taskService.UpdateTask(task)
}

// MoveTask moves a task to a different status column
func (a *App) MoveTask(taskID int, newStatus string) error {
	// Get the task to check the old status
	tasks := a.taskService.GetTasks()
	var oldStatus string
	var updatedTask Task
	found := false
	
	for _, task := range tasks {
		if task.ID == taskID {
			oldStatus = task.Status
			updatedTask = task
			updatedTask.Status = newStatus
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	// Move the task
	if err := a.taskService.MoveTask(taskID, newStatus); err != nil {
		return err
	}
	
	// Only launch Claude agent if moving from "todo" to "doing"
	if oldStatus == "todo" && newStatus == "doing" {
		go func() {
			if err := a.agentService.LaunchClaudeAgent(updatedTask); err != nil {
				a.logger.Error("Failed to launch Claude agent", err)
			}
		}()
	}
	
	return nil
}

// GetTasksByStatus returns tasks filtered by status
func (a *App) GetTasksByStatus(status string) ([]Task, error) {
	return a.taskService.GetTasksByStatus(status)
}

// ApproveTask merges the task branch and marks task as done
func (a *App) ApproveTask(taskID int) error {
	// Get task info
	tasks := a.taskService.GetTasks()
	var task Task
	found := false
	
	for _, t := range tasks {
		if t.ID == taskID {
			if t.Status != "pending_review" {
				return fmt.Errorf("task %d is not in pending_review status", taskID)
			}
			task = t
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	// Approve through agent service
	if err := a.agentService.ApproveTask(taskID, task.Title); err != nil {
		return err
	}
	
	// Update task status to done
	task.Status = "done"
	if err := a.taskService.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task status after approval: %v", err)
	}
	
	return nil
}

// RejectTask deletes the task branch and marks task as done with NOT MERGED prefix
func (a *App) RejectTask(taskID int) error {
	// Get task info
	tasks := a.taskService.GetTasks()
	var task Task
	found := false
	
	for _, t := range tasks {
		if t.ID == taskID {
			if t.Status != "pending_review" {
				return fmt.Errorf("task %d is not in pending_review status", taskID)
			}
			task = t
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	// Reject through agent service
	if err := a.agentService.RejectTask(taskID, task.Title); err != nil {
		return err
	}
	
	// Update task with NOT MERGED prefix and done status
	if task.Title != "" && !strings.HasPrefix(task.Title, "NOT MERGED: ") {
		task.Title = "NOT MERGED: " + task.Title
	}
	task.Status = "done"
	
	if err := a.taskService.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task status after rejection: %v", err)
	}
	
	return nil
}

// Plan-related API methods

// LoadPlan loads the plan.md file and returns its content
func (a *App) LoadPlan() (string, error) {
	activeRepoPath, err := a.getActiveRepositoryPath()
	if err != nil {
		return "", err
	}
	
	planFile := filepath.Join(activeRepoPath, "plan", "plan.md")
	a.logger.InfoWithFields("Loading plan", map[string]interface{}{
		"plan_file": planFile,
	})

	content, err := readFileContent(planFile)
	if err != nil {
		a.logger.Error("Failed to load plan.md", err)
		return "", fmt.Errorf("failed to read plan.md: %w", err)
	}

	a.logger.Info("Plan loaded successfully")
	return content, nil
}

// SavePlan saves content to the plan.md file
func (a *App) SavePlan(content string) error {
	activeRepoPath, err := a.getActiveRepositoryPath()
	if err != nil {
		return err
	}
	
	planFile := filepath.Join(activeRepoPath, "plan", "plan.md")
	a.logger.InfoWithFields("Saving plan", map[string]interface{}{
		"plan_file": planFile,
	})

	// Create backup of plan.md
	if err := createFileBackup(planFile); err != nil {
		a.logger.Error("Failed to create backup of plan.md", err)
		// Continue with save even if backup fails
	}

	// Write the new content
	if err := writeFileContent(planFile, content); err != nil {
		a.logger.Error("Failed to save plan.md", err)
		return fmt.Errorf("failed to write plan.md: %w", err)
	}

	a.logger.Info("Plan saved successfully")
	return nil
}

// Terminal-related API methods

// StartTerminalSession creates a new terminal session and returns its ID
func (a *App) StartTerminalSession() string {
	return a.terminalService.StartTerminalSession()
}

// Agent-related API methods

// GetAgentStatus returns the current status of all subagents
func (a *App) GetAgentStatus() (AgentStatusInfo, error) {
	return a.agentService.GetAgentStatus()
}

// Configuration API methods

// GetConfig returns the current configuration
func (a *App) GetConfig() (*Config, error) {
	if a.configService == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return a.configService.GetConfig()
}

// GetRepositories returns all configured repositories
func (a *App) GetRepositories() ([]Repository, error) {
	if a.configService == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return a.configService.GetRepositories()
}

// AddRepository adds a new repository to the configuration
func (a *App) AddRepository(name, path string) (*Repository, error) {
	if a.configService == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return a.configService.AddRepository(name, path)
}

// RemoveRepository removes a repository from the configuration
func (a *App) RemoveRepository(id string) error {
	if a.configService == nil {
		return fmt.Errorf("configuration not initialized")
	}
	return a.configService.RemoveRepository(id)
}

// SetActiveRepository switches the active repository
func (a *App) SetActiveRepository(id string) error {
	if a.configService == nil {
		return fmt.Errorf("configuration not initialized")
	}
	
	err := a.configService.SetActiveRepository(id)
	if err != nil {
		return err
	}
	
	// Update services with new repository
	activeRepo, err := a.configService.GetActiveRepository()
	if err != nil {
		return err
	}
	
	// Update task service with new task file path
	taskFile := filepath.Join(activeRepo.Path, "plan", "task.json")
	a.taskService.SetTaskFile(taskFile)
	
	// Update agent service with new project root
	a.agentService.SetProjectRoot(activeRepo.Path)
	
	// Reload tasks from new repository
	if _, err := a.taskService.LoadTasks(); err != nil {
		a.logger.Error("Failed to load tasks from new repository", err)
		return fmt.Errorf("failed to load tasks from new repository: %v", err)
	}
	
	a.logger.InfoWithFields("Switched to repository", map[string]interface{}{
		"name": activeRepo.Name,
		"path": activeRepo.Path,
	})
	
	return nil
}

// ValidateRepositoryPath validates a repository path
func (a *App) ValidateRepositoryPath(path string) (*RepositoryInfo, error) {
	if a.configService == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return a.configService.ValidateRepositoryPath(path)
}

// FindRepositories searches for repositories in a directory
func (a *App) FindRepositories(searchPath string) ([]Repository, error) {
	if a.configService == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return a.configService.FindRepositories(searchPath)
}

// OpenDirectoryDialog opens a directory selection dialog
func (a *App) OpenDirectoryDialog() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not available")
	}
	
	// Use Wails runtime to open directory dialog
	selectedPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Repository Directory",
	})
	
	if err != nil {
		a.logger.Error("Failed to open directory dialog", err)
		return "", err
	}
	
	a.logger.InfoWithFields("Selected directory", map[string]interface{}{
		"path": selectedPath,
	})
	
	return selectedPath, nil
}

// WebSocket handling (delegated to terminal service)
func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	a.terminalService.HandleWebSocket(w, r)
}

// Private helper methods

func (a *App) getActiveRepositoryPath() (string, error) {
	if a.configService == nil {
		return "", fmt.Errorf("configuration not initialized")
	}
	return a.configService.GetActiveRepositoryPath()
}