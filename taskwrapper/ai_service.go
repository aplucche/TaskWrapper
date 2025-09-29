package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// AIService handles AI-powered task suggestions
type AIService struct {
	logger Logger
}

// NewAIService creates a new AI service
func NewAIService(logger Logger) *AIService {
	return &AIService{
		logger: logger,
	}
}

// SuggestedTask represents a task suggestion from AI
type SuggestedTask struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Reason   string `json:"reason"`
}

// SuggestTasks uses AI to suggest new tasks based on plan and existing tasks
func (s *AIService) SuggestTasks(planContent string, existingTasks []Task, guidance string) ([]SuggestedTask, error) {
	if guidance != "" {
		s.logger.InfoWithFields("Generating task suggestions using Claude AI", map[string]interface{}{
			"guidance": guidance,
		})
	} else {
		s.logger.Info("Generating task suggestions using Claude AI")
	}

	// Build existing tasks summary
	tasksSummary := s.buildTasksSummary(existingTasks)

	// Build prompt for AI
	prompt := s.buildPrompt(planContent, tasksSummary, guidance)

	// Call Claude CLI
	output, err := s.callClaude(prompt)
	if err != nil {
		s.logger.Error("Failed to call Claude CLI", err)
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	// Parse suggestions from output
	suggestions, err := s.parseSuggestions(output)
	if err != nil {
		s.logger.Error("Failed to parse AI suggestions", err)
		return nil, fmt.Errorf("failed to parse suggestions: %w", err)
	}

	s.logger.InfoWithFields("Generated task suggestions", map[string]interface{}{
		"count": len(suggestions),
	})

	return suggestions, nil
}

// buildTasksSummary creates a summary of existing tasks
func (s *AIService) buildTasksSummary(tasks []Task) string {
	var summary strings.Builder

	todoCount := 0
	doingCount := 0
	doneCount := 0

	summary.WriteString("Existing tasks:\n")
	for _, task := range tasks {
		summary.WriteString(fmt.Sprintf("- [%s] %s (Priority: %s)\n", task.Status, task.Title, task.Priority))

		switch task.Status {
		case StatusTodo:
			todoCount++
		case StatusDoing:
			doingCount++
		case StatusDone, StatusPendingReview:
			doneCount++
		}
	}

	summary.WriteString(fmt.Sprintf("\nSummary: %d todo, %d doing, %d done\n", todoCount, doingCount, doneCount))

	return summary.String()
}

// buildPrompt creates the AI prompt
func (s *AIService) buildPrompt(planContent, tasksSummary, guidance string) string {
	basePrompt := fmt.Sprintf(`You are a task planning assistant. Analyze the project plan and existing tasks, then suggest 1-3 NEW tasks that would help implement the plan.

PROJECT PLAN:
%s

%s`, planContent, tasksSummary)

	var guidanceSection string
	if guidance != "" {
		guidanceSection = fmt.Sprintf(`

USER GUIDANCE:
The user has requested tasks related to: "%s"
Focus your suggestions on this area while considering the overall plan.`, guidance)
	}

	instructions := `

INSTRUCTIONS:
1. Suggest 1-3 new tasks that are NOT already covered by existing tasks
2. Focus on high-impact tasks that move the project forward
3. Consider dependencies and logical next steps
4. Return ONLY valid JSON in this exact format (no markdown, no extra text):
[
  {
    "title": "Task title",
    "priority": "high|medium|low",
    "reason": "Brief explanation why this task is important"
  }
]

Return the JSON array now:`

	return basePrompt + guidanceSection + instructions
}

// callClaude executes the claude CLI command
func (s *AIService) callClaude(prompt string) (string, error) {
	// Use claude CLI with prompt
	cmd := exec.Command("claude", "-p", prompt)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// parseSuggestions parses JSON suggestions from AI output
func (s *AIService) parseSuggestions(output string) ([]SuggestedTask, error) {
	// Try to find JSON array in output (in case AI added extra text)
	output = strings.TrimSpace(output)

	// Find first [ and last ]
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON array found in output")
	}

	jsonStr := output[start : end+1]

	var suggestions []SuggestedTask
	err := json.Unmarshal([]byte(jsonStr), &suggestions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate suggestions
	for i, suggestion := range suggestions {
		if suggestion.Title == "" {
			return nil, fmt.Errorf("suggestion %d has empty title", i)
		}
		if suggestion.Priority == "" {
			suggestions[i].Priority = "medium" // Default priority
		}
	}

	return suggestions, nil
}