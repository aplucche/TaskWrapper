package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AgentService handles Claude agent operations and Git branch management
type AgentService struct {
	projectRoot   string
	logger        Logger
	mu            sync.RWMutex
	ctx           context.Context
	pathValidator *PathValidator
}

// NewAgentService creates a new agent service
func NewAgentService(projectRoot string, logger Logger) *AgentService {
	securityConfig := DefaultSecurityConfig()
	return &AgentService{
		projectRoot:   projectRoot,
		logger:        logger,
		pathValidator: NewPathValidator(securityConfig, logger),
	}
}

// SetProjectRoot sets the project root directory
func (as *AgentService) SetProjectRoot(root string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.projectRoot = root
}

// SetContext sets the application context
func (as *AgentService) SetContext(ctx context.Context) {
	as.ctx = ctx
}

// LaunchClaudeAgent starts a Claude Code agent for the given task
func (as *AgentService) LaunchClaudeAgent(task Task) error {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	// Validate project root path
	validRoot, err := as.pathValidator.ValidatePath(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid project root: %w", err)
	}

	// Use the agent_spawn.sh script
	scriptPath := filepath.Join(validRoot, "plan", "helpers_and_tools", "agent_spawn.sh")
	
	// Validate script path
	validScript, err := as.pathValidator.ValidateExecutable(scriptPath)
	if err != nil {
		return fmt.Errorf("invalid script path: %w", err)
	}
	
	// Sanitize task title to prevent command injection
	sanitizedTitle := as.pathValidator.SanitizeFilename(task.Title)
	
	// Create command with timeout context
	ctx := as.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	
	// Set a reasonable timeout for agent spawning (30 seconds)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Create the command with validated inputs
	cmd := exec.CommandContext(ctx, validScript, strconv.Itoa(task.ID), sanitizedTitle)
	cmd.Dir = validRoot
	
	// Set restricted environment
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TASK_ID=" + strconv.Itoa(task.ID),
		"TASK_TITLE=" + sanitizedTitle,
	}
	
	// Log the launch
	as.logger.InfoWithFields("Launching Claude agent for task", map[string]interface{}{
		"task_id":    task.ID,
		"task_title": task.Title,
		"script":     scriptPath,
		"work_dir":   projectRoot,
	})
	
	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		as.logger.ErrorWithFields("Failed to launch Claude agent", err, map[string]interface{}{
			"task_id": task.ID,
			"output":  string(output),
		})
		return fmt.Errorf("failed to launch agent for task #%d: %v - %s", task.ID, err, string(output))
	}
	
	as.logger.InfoWithFields("Agent spawner completed", map[string]interface{}{
		"task_id": task.ID,
		"output":  string(output),
	})
	
	return nil
}

// ApproveTask merges the task branch and marks task as approved
func (as *AgentService) ApproveTask(taskID int, taskTitle string) error {
	branchName := fmt.Sprintf("task_%d", taskID)
	
	as.logger.InfoWithFields("Approving task", map[string]interface{}{
		"task_id": taskID,
		"branch":  branchName,
	})
	
	// Check if branch exists
	if err := as.checkBranchExists(branchName); err != nil {
		return fmt.Errorf("branch validation failed: %v", err)
	}
	
	// Merge the branch
	if err := as.mergeBranch(branchName, taskID, taskTitle); err != nil {
		return fmt.Errorf("merge failed: %v", err)
	}
	
	// Delete the branch after successful merge
	if err := as.deleteBranch(branchName); err != nil {
		as.logger.InfoWithFields("Warning: Failed to delete branch", map[string]interface{}{
			"branch": branchName,
			"error":  err.Error(),
		})
	}
	
	as.logger.InfoWithFields("Task approved and merged successfully", map[string]interface{}{
		"task_id": taskID,
		"branch":  branchName,
	})
	
	return nil
}

// RejectTask deletes the task branch and marks task as rejected
func (as *AgentService) RejectTask(taskID int, taskTitle string) error {
	branchName := fmt.Sprintf("task_%d", taskID)
	
	as.logger.InfoWithFields("Rejecting task", map[string]interface{}{
		"task_id": taskID,
		"branch":  branchName,
	})
	
	// Force delete the branch
	if err := as.forceDeleteBranch(branchName); err != nil {
		as.logger.InfoWithFields("Warning: Failed to delete branch", map[string]interface{}{
			"branch": branchName,
			"error":  err.Error(),
		})
	}
	
	as.logger.InfoWithFields("Task rejected and branch deleted", map[string]interface{}{
		"task_id": taskID,
		"branch":  branchName,
	})
	
	return nil
}

// GetAgentStatus returns the current status of all subagents
func (as *AgentService) GetAgentStatus() (AgentStatusInfo, error) {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	scriptPath := filepath.Join(projectRoot, "plan", "helpers_and_tools", "agent_status.sh")
	
	cmd := exec.Command(scriptPath)
	cmd.Dir = projectRoot
	
	// Add context cancellation if available
	if as.ctx != nil {
		ctx, cancel := context.WithTimeout(as.ctx, 10*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, scriptPath)
		cmd.Dir = projectRoot
	}
	
	output, err := cmd.Output()
	if err != nil {
		as.logger.Error("Failed to get agent status", err)
		return AgentStatusInfo{}, fmt.Errorf("failed to run agent_status.sh: %v", err)
	}
	
	return as.parseAgentStatus(string(output)), nil
}

// Private helper methods

func (as *AgentService) checkBranchExists(branchName string) error {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	cmd := exec.Command("git", "branch", "--list", branchName)
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch check failed: %v", err)
	}
	
	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("branch %s not found", branchName)
	}
	
	return nil
}

func (as *AgentService) mergeBranch(branchName string, taskID int, taskTitle string) error {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	mergeCmd := exec.Command("git", "merge", branchName, "--no-ff", "-m", 
		fmt.Sprintf("Merge task #%d: %s", taskID, taskTitle))
	mergeCmd.Dir = projectRoot
	
	// Add context cancellation if available
	if as.ctx != nil {
		ctx, cancel := context.WithTimeout(as.ctx, 30*time.Second)
		defer cancel()
		mergeCmd = exec.CommandContext(ctx, "git", "merge", branchName, "--no-ff", "-m", 
			fmt.Sprintf("Merge task #%d: %s", taskID, taskTitle))
		mergeCmd.Dir = projectRoot
	}
	
	output, err := mergeCmd.CombinedOutput()
	if err != nil {
		as.logger.ErrorWithFields("Git merge failed", err, map[string]interface{}{
			"branch": branchName,
			"output": string(output),
		})
		return fmt.Errorf("git merge failed: %v - %s", err, string(output))
	}
	
	return nil
}

func (as *AgentService) deleteBranch(branchName string) error {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	cmd := exec.Command("git", "branch", "-d", branchName)
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch delete failed: %v - %s", err, string(output))
	}
	
	return nil
}

func (as *AgentService) forceDeleteBranch(branchName string) error {
	as.mu.RLock()
	projectRoot := as.projectRoot
	as.mu.RUnlock()

	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch force delete failed: %v - %s", err, string(output))
	}
	
	return nil
}

// parseAgentStatus parses the output from agent_status.sh script
func (as *AgentService) parseAgentStatus(output string) AgentStatusInfo {
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