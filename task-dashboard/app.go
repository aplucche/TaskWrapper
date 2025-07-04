package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

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
	Conn    *websocket.Conn
	Done    chan bool
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

// App struct
type App struct {
	ctx        context.Context
	taskFile   string
	mu         sync.RWMutex
	tasks      []Task
	upgrader   websocket.Upgrader
	terminals  map[string]*Terminal
	terminalMu sync.RWMutex
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
		taskFile:  taskFile,
		tasks:     []Task{},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		terminals: make(map[string]*Terminal),
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
	if newStatus != "backlog" && newStatus != "todo" && newStatus != "doing" && newStatus != "pending_review" && newStatus != "done" {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Find and update the task status
	found := false
	var updatedTask Task
	var oldStatus string
	for i, task := range a.tasks {
		if task.ID == taskID {
			oldStatus = task.Status
			a.tasks[i].Status = newStatus
			updatedTask = a.tasks[i]
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
	
	a.logInfo(fmt.Sprintf("Task %d moved from %s to %s", taskID, oldStatus, newStatus))
	
	// Only launch Claude agent if moving from "todo" to "doing"
	if oldStatus == "todo" && newStatus == "doing" {
		go a.launchClaudeAgent(updatedTask) // Non-blocking
	}
	
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

// launchClaudeAgent starts a Claude Code agent for the given task
func (a *App) launchClaudeAgent(task Task) {
	// Determine project root directory (go up from plan/ to project root)
	projectRoot := filepath.Dir(filepath.Dir(a.taskFile))
	
	// Use the new agent_spawn.sh script
	scriptPath := filepath.Join(projectRoot, "plan", "helpers_and_tools", "agent_spawn.sh")
	
	// Create the command with task ID and title as arguments
	cmd := exec.Command(scriptPath, strconv.Itoa(task.ID), task.Title)
	cmd.Dir = projectRoot
	
	// Log the launch
	a.logInfo(fmt.Sprintf("Launching Claude agent for task #%d: %s", task.ID, task.Title))
	a.logInfo(fmt.Sprintf("Using agent spawner: %s", scriptPath))
	a.logInfo(fmt.Sprintf("Working directory: %s", projectRoot))
	
	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logError(fmt.Sprintf("Failed to launch Claude agent for task #%d: %s", task.ID, string(output)), err)
		return
	}
	
	a.logInfo(fmt.Sprintf("Agent spawner output: %s", string(output)))
}

// generateTaskPrompt creates a minimal prompt for the Claude agent
func (a *App) generateTaskPrompt(task Task) string {
	return fmt.Sprintf("Review plan.md and task.json. Begin task #%d: %s. Update task.json status to 'pending_review' when done, commit to branch task_%d.", task.ID, task.Title, task.ID)
}

// ApproveTask merges the task branch and marks task as done
func (a *App) ApproveTask(taskID int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Find the task
	taskIndex := -1
	for i, task := range a.tasks {
		if task.ID == taskID {
			if task.Status != "pending_review" {
				return fmt.Errorf("task %d is not in pending_review status", taskID)
			}
			taskIndex = i
			break
		}
	}
	
	if taskIndex == -1 {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	task := a.tasks[taskIndex]
	projectRoot := filepath.Dir(filepath.Dir(a.taskFile))
	branchName := fmt.Sprintf("task_%d", task.ID)
	
	a.logInfo(fmt.Sprintf("Approving task #%d: merging branch %s", task.ID, branchName))
	
	// Check if branch exists
	checkCmd := exec.Command("git", "branch", "--list", branchName)
	checkCmd.Dir = projectRoot
	checkOutput, err := checkCmd.CombinedOutput()
	if err != nil || len(strings.TrimSpace(string(checkOutput))) == 0 {
		a.logError(fmt.Sprintf("Branch %s not found for task #%d", branchName, task.ID), fmt.Errorf("branch not found"))
		return fmt.Errorf("branch %s not found", branchName)
	}
	
	// Merge the branch
	mergeCmd := exec.Command("git", "merge", branchName, "--no-ff", "-m", 
		fmt.Sprintf("Merge task #%d: %s", task.ID, task.Title))
	mergeCmd.Dir = projectRoot
	mergeOutput, err := mergeCmd.CombinedOutput()
	if err != nil {
		a.logError(fmt.Sprintf("Failed to merge branch %s: %s", branchName, string(mergeOutput)), err)
		return fmt.Errorf("merge failed: %v - %s", err, string(mergeOutput))
	}
	
	// Delete the branch after successful merge
	deleteCmd := exec.Command("git", "branch", "-d", branchName)
	deleteCmd.Dir = projectRoot
	deleteOutput, err := deleteCmd.CombinedOutput()
	if err != nil {
		a.logInfo(fmt.Sprintf("Warning: Failed to delete branch %s: %s", branchName, string(deleteOutput)))
	}
	
	// Update task status to done
	a.tasks[taskIndex].Status = "done"
	
	// Save tasks
	if err := a.saveTasks(); err != nil {
		return fmt.Errorf("failed to save tasks after approval: %v", err)
	}
	
	a.logInfo(fmt.Sprintf("Task #%d approved and merged successfully", task.ID))
	return nil
}

// RejectTask deletes the task branch and marks task as done with NOT MERGED prefix
func (a *App) RejectTask(taskID int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Find the task
	taskIndex := -1
	for i, task := range a.tasks {
		if task.ID == taskID {
			if task.Status != "pending_review" {
				return fmt.Errorf("task %d is not in pending_review status", taskID)
			}
			taskIndex = i
			break
		}
	}
	
	if taskIndex == -1 {
		return fmt.Errorf("task with ID %d not found", taskID)
	}
	
	task := a.tasks[taskIndex]
	projectRoot := filepath.Dir(filepath.Dir(a.taskFile))
	branchName := fmt.Sprintf("task_%d", task.ID)
	
	a.logInfo(fmt.Sprintf("Rejecting task #%d: deleting branch %s", task.ID, branchName))
	
	// Delete the branch (force delete to ensure it's removed even if not merged)
	deleteCmd := exec.Command("git", "branch", "-D", branchName)
	deleteCmd.Dir = projectRoot
	deleteOutput, err := deleteCmd.CombinedOutput()
	if err != nil {
		a.logInfo(fmt.Sprintf("Warning: Failed to delete branch %s: %s", branchName, string(deleteOutput)))
	}
	
	// Update task with NOT MERGED prefix and done status
	if !strings.HasPrefix(task.Title, "NOT MERGED: ") {
		a.tasks[taskIndex].Title = "NOT MERGED: " + task.Title
	}
	a.tasks[taskIndex].Status = "done"
	
	// Save tasks
	if err := a.saveTasks(); err != nil {
		return fmt.Errorf("failed to save tasks after rejection: %v", err)
	}
	
	a.logInfo(fmt.Sprintf("Task #%d rejected and branch deleted", task.ID))
	return nil
}

// LoadPlan loads the plan.md file and returns its content
func (a *App) LoadPlan() (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	planFile := filepath.Join(filepath.Dir(a.taskFile), "plan.md")
	a.logInfo(fmt.Sprintf("Loading plan from: %s", planFile))

	content, err := os.ReadFile(planFile)
	if err != nil {
		a.logError("Failed to load plan.md", err)
		return "", fmt.Errorf("failed to read plan.md: %w", err)
	}

	a.logInfo("Plan loaded successfully")
	return string(content), nil
}

// SavePlan saves content to the plan.md file
func (a *App) SavePlan(content string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	planFile := filepath.Join(filepath.Dir(a.taskFile), "plan.md")
	a.logInfo(fmt.Sprintf("Saving plan to: %s", planFile))

	// Create backup of plan.md
	if _, err := os.Stat(planFile); err == nil {
		timestamp := time.Now().Format("20060102_150405")
		backupFile := planFile + ".backup." + timestamp
		
		data, err := os.ReadFile(planFile)
		if err != nil {
			a.logError("Failed to read plan.md for backup", err)
			// Continue with save even if backup fails
		} else if err := os.WriteFile(backupFile, data, 0644); err != nil {
			a.logError("Failed to create backup of plan.md", err)
			// Continue with save even if backup fails
		}
	}

	// Write the new content
	if err := os.WriteFile(planFile, []byte(content), 0644); err != nil {
		a.logError("Failed to save plan.md", err)
		return fmt.Errorf("failed to write plan.md: %w", err)
	}

	a.logInfo("Plan saved successfully")
	return nil
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

// StartTerminalSession creates a new terminal session and returns its ID
func (a *App) StartTerminalSession() string {
	terminalID := uuid.New().String()
	a.logInfo(fmt.Sprintf("Creating terminal session: %s", terminalID))
	
	// Start WebSocket server if not already running
	go a.startWebSocketServer()
	
	return terminalID
}

// GetAgentStatus returns the current status of all subagents
func (a *App) GetAgentStatus() (AgentStatusInfo, error) {
	projectRoot := filepath.Dir(filepath.Dir(a.taskFile))
	scriptPath := filepath.Join(projectRoot, "plan", "helpers_and_tools", "agent_status.sh")
	
	cmd := exec.Command(scriptPath)
	cmd.Dir = projectRoot
	
	output, err := cmd.Output()
	if err != nil {
		a.logError("Failed to get agent status", err)
		return AgentStatusInfo{}, fmt.Errorf("failed to run agent_status.sh: %v", err)
	}
	
	return a.parseAgentStatus(string(output)), nil
}

// parseAgentStatus parses the output from agent_status.sh script
func (a *App) parseAgentStatus(output string) AgentStatusInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	info := AgentStatusInfo{
		Worktrees:    []AgentWorktree{},
		MaxSubagents: 5, // Default from agent_status.sh
	}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Parse different status types
		if strings.Contains(line, "Total Worktrees:") {
			if parts := strings.Split(line, ":"); len(parts) > 1 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					info.TotalWorktrees = count
				}
			}
		} else if strings.Contains(line, "Idle:") {
			if parts := strings.Split(line, ":"); len(parts) > 1 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					info.IdleCount = count
				}
			}
		} else if strings.Contains(line, "Busy:") {
			if parts := strings.Split(line, ":"); len(parts) > 1 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					info.BusyCount = count
				}
			}
		} else if strings.Contains(line, "Max Subagents:") {
			if parts := strings.Split(line, ":"); len(parts) > 1 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					info.MaxSubagents = count
				}
			}
		} else if strings.Contains(line, " - ") {
			// Parse individual worktree entries
			parts := strings.Split(line, " - ")
			if len(parts) >= 2 {
				worktree := AgentWorktree{
					Name:   strings.TrimSpace(parts[0]),
					Status: "idle", // default
				}
				
				statusPart := strings.TrimSpace(parts[1])
				if strings.Contains(statusPart, "IDLE") {
					worktree.Status = "idle"
				} else if strings.Contains(statusPart, "BUSY") {
					worktree.Status = "busy"
					// Extract task information from busy status
					if strings.Contains(statusPart, "Task #") {
						if taskStart := strings.Index(statusPart, "Task #"); taskStart != -1 {
							taskInfo := statusPart[taskStart:]
							if colonIdx := strings.Index(taskInfo, ":"); colonIdx != -1 {
								taskIDStr := strings.TrimSpace(taskInfo[5:colonIdx])
								taskTitle := strings.TrimSpace(taskInfo[colonIdx+1:])
								worktree.TaskID = taskIDStr
								worktree.TaskTitle = taskTitle
							}
						}
					}
				} else if strings.Contains(statusPart, "STALE") {
					worktree.Status = "stale"
				}
				
				info.Worktrees = append(info.Worktrees, worktree)
			}
		}
	}
	
	return info
}

// startWebSocketServer starts the WebSocket server for terminal sessions
func (a *App) startWebSocketServer() {
	http.HandleFunc("/ws/terminal/", a.handleWebSocket)
	
	a.logInfo("Starting WebSocket server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		a.logError("WebSocket server failed", err)
	}
}

// handleWebSocket handles WebSocket connections for terminal sessions
func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract terminal ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid terminal ID", http.StatusBadRequest)
		return
	}
	terminalID := pathParts[3]
	
	a.logInfo(fmt.Sprintf("WebSocket connection for terminal: %s", terminalID))
	
	// Upgrade connection to WebSocket
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.logError("Failed to upgrade WebSocket connection", err)
		return
	}
	defer conn.Close()
	
	// Create and start terminal session
	terminal, err := a.createTerminal(terminalID, conn)
	if err != nil {
		a.logError("Failed to create terminal", err)
		return
	}
	
	// Store terminal session
	a.terminalMu.Lock()
	a.terminals[terminalID] = terminal
	a.terminalMu.Unlock()
	
	// Clean up when done
	defer func() {
		a.terminalMu.Lock()
		delete(a.terminals, terminalID)
		a.terminalMu.Unlock()
		
		if terminal.Cmd != nil && terminal.Cmd.Process != nil {
			terminal.Cmd.Process.Kill()
		}
	}()
	
	// Handle messages
	a.handleTerminalMessages(terminal)
}

// createTerminal creates a new terminal process
func (a *App) createTerminal(terminalID string, conn *websocket.Conn) (*Terminal, error) {
	// Create a new shell process
	var cmd *exec.Cmd
	
	// Use bash on Unix-like systems
	cmd = exec.Command("/bin/bash")
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"PS1=\\u@\\h:\\w$ ",
	)
	
	terminal := &Terminal{
		ID:   terminalID,
		Cmd:  cmd,
		Conn: conn,
		Done: make(chan bool),
	}
	
	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start terminal process: %v", err)
	}
	
	a.logInfo(fmt.Sprintf("Terminal process started for session %s (PID: %d)", terminalID, cmd.Process.Pid))
	return terminal, nil
}

// handleTerminalMessages handles the message loop for a terminal session
func (a *App) handleTerminalMessages(terminal *Terminal) {
	// Handle WebSocket messages
	for {
		var message TerminalMessage
		err := terminal.Conn.ReadJSON(&message)
		if err != nil {
			a.logError("Failed to read WebSocket message", err)
			break
		}
		
		if message.Type == "input" {
			// Send input to terminal process (this is a simplified version)
			// In a real implementation, you'd need proper PTY handling
			a.logInfo(fmt.Sprintf("Received input for terminal %s: %s", terminal.ID, message.Data))
			
			// Echo back the input for demo purposes
			response := TerminalMessage{
				Type: "output",
				Data: message.Data,
			}
			
			if err := terminal.Conn.WriteJSON(response); err != nil {
				a.logError("Failed to send terminal output", err)
				break
			}
		}
	}
}
