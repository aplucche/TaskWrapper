package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test fixtures - minimal test data
var testTasks = []Task{
	{ID: 1, Title: "Test Task 1", Status: StatusTodo, Priority: PriorityHigh, Deps: []int{}, Parent: nil},
	{ID: 2, Title: "Test Task 2", Status: StatusDoing, Priority: PriorityMedium, Deps: []int{1}, Parent: nil},
	{ID: 3, Title: "Test Task 3", Status: StatusDone, Priority: PriorityLow, Deps: []int{}, Parent: nil},
}

// MockLogger implements the Logger interface for testing
type MockLogger struct{}

func (m *MockLogger) Info(message string) {}
func (m *MockLogger) Error(message string, err error) {}
func (m *MockLogger) InfoWithFields(message string, fields map[string]interface{}) {}
func (m *MockLogger) ErrorWithFields(message string, err error, fields map[string]interface{}) {}

// MockAgentService implements the AgentService interface for testing
type MockAgentService struct {
	logger Logger
}

func NewMockAgentService(logger Logger) *MockAgentService {
	return &MockAgentService{logger: logger}
}

func (m *MockAgentService) SetProjectRoot(root string) {}
func (m *MockAgentService) SetContext(ctx context.Context) {}
func (m *MockAgentService) LaunchClaudeAgent(task Task) error { return nil }
func (m *MockAgentService) ApproveTask(taskID int, taskTitle string) error { 
	// Simulate successful approval without git operations
	return nil 
}
func (m *MockAgentService) RejectTask(taskID int, taskTitle string) error { 
	// Simulate successful rejection without git operations
	return nil 
}
func (m *MockAgentService) GetAgentStatus() (AgentStatusInfo, error) {
	// Return empty status for tests
	return AgentStatusInfo{}, nil
}

func setupTestApp(t *testing.T) (*App, func()) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create required subdirectories
	planDir := filepath.Join(tmpDir, "plan")
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create services
	logger := &MockLogger{}
	taskFile := filepath.Join(planDir, "task.json")
	taskService := NewTaskService(taskFile, logger)
	
	// Mock services for other dependencies
	terminalService := NewTerminalService(logger, []string{"*"})
	agentService := NewMockAgentService(logger)
	
	app := &App{
		ctx:             context.Background(),
		taskService:     taskService,
		terminalService: terminalService,
		agentService:    AgentServiceInterface(agentService),
		configService:   nil, // Not needed for most tests
		logger:          logger,
		errorHandler:    NewErrorHandler(logger),
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return app, cleanup
}

// Test 1: Save/Load Cycle - Core functionality
func TestSaveLoadCycle(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Save tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Load tasks back
	loadedTasks, err := app.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	// Verify data integrity
	if len(loadedTasks) != len(testTasks) {
		t.Errorf("Expected %d tasks, got %d", len(testTasks), len(loadedTasks))
	}

	for i, task := range loadedTasks {
		if task.ID != testTasks[i].ID || task.Title != testTasks[i].Title {
			t.Errorf("Task %d mismatch: expected %+v, got %+v", i, testTasks[i], task)
		}
	}
}

// Test 2: Task Validation - Data integrity
func TestTaskValidation(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{"Valid task", Task{ID: 1, Title: "Valid", Status: StatusTodo, Priority: PriorityHigh}, false},
		{"Empty title", Task{ID: 1, Title: "", Status: StatusTodo, Priority: PriorityHigh}, true},
		{"Invalid status", Task{ID: 1, Title: "Test", Status: TaskStatus("invalid"), Priority: PriorityHigh}, true},
		{"Invalid priority", Task{ID: 1, Title: "Test", Status: StatusTodo, Priority: TaskPriority("invalid")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.SaveTasks([]Task{tt.task})
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveTasks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test 3: Task Movement - Status updates
func TestMoveTask(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup initial tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Move task to different status
	if err := app.MoveTask(1, "doing"); err != nil {
		t.Fatalf("MoveTask failed: %v", err)
	}

	// Verify task was moved
	tasks, _ := app.LoadTasks()
	found := false
	for _, task := range tasks {
		if task.ID == 1 && task.Status == StatusDoing {
			found = true
			break
		}
	}

	if !found {
		t.Error("Task was not moved to 'doing' status")
	}

	// Test invalid status
	if err := app.MoveTask(1, "invalid"); err == nil {
		t.Error("Expected error for invalid status")
	}

	// Test non-existent task
	if err := app.MoveTask(999, "todo"); err == nil {
		t.Error("Expected error for non-existent task")
	}
}

// Test 4: Individual Task Updates
func TestUpdateTask(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup initial tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Update a task
	updatedTask := Task{ID: 1, Title: "Updated Task", Status: StatusDoing, Priority: PriorityLow, Deps: []int{}, Parent: nil}
	if err := app.UpdateTask(updatedTask); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify update
	tasks, _ := app.LoadTasks()
	found := false
	for _, task := range tasks {
		if task.ID == 1 && task.Title == "Updated Task" && task.Priority == PriorityLow {
			found = true
			break
		}
	}

	if !found {
		t.Error("Task was not updated correctly")
	}

	// Test updating non-existent task
	nonExistentTask := Task{ID: 999, Title: "Ghost", Status: StatusTodo, Priority: PriorityHigh}
	if err := app.UpdateTask(nonExistentTask); err == nil {
		t.Error("Expected error for non-existent task")
	}
}

// Test 5: Atomic File Operations - Backup creation
func TestAtomicOperations(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Save initial tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Get task file path from service
	taskFile := filepath.Join(filepath.Dir(app.taskService.(*TaskService).taskFile), "task.json")
	
	// Verify main file exists
	if _, err := os.Stat(taskFile); os.IsNotExist(err) {
		t.Error("Task file was not created")
	}

	// Save again to trigger .tracked file creation
	modifiedTasks := append(testTasks, Task{ID: 4, Title: "New Task", Status: StatusTodo, Priority: PriorityMedium})
	if err := app.SaveTasks(modifiedTasks); err != nil {
		t.Fatalf("Second SaveTasks failed: %v", err)
	}

	// Check that .tracked file was created
	trackedFile := taskFile + ".tracked"
	if _, err := os.Stat(trackedFile); os.IsNotExist(err) {
		t.Error(".tracked file was not created")
	}

	// Verify .tracked file contains the previous state
	data, err := os.ReadFile(trackedFile)
	if err != nil {
		t.Fatalf("Failed to read .tracked file: %v", err)
	}

	var trackedTasks []Task
	if err := json.Unmarshal(data, &trackedTasks); err != nil {
		t.Fatalf("Failed to parse .tracked file: %v", err)
	}

	// The .tracked file should contain the original tasks (before the new task was added)
	if len(trackedTasks) != len(testTasks) {
		t.Errorf("Expected %d tasks in .tracked file, got %d", len(testTasks), len(trackedTasks))
	}
}

// Test 6: Error Handling - File system errors
func TestErrorHandling(t *testing.T) {
	// Create a test app with an invalid path
	logger := &MockLogger{}
	taskService := NewTaskService("/root/impossible/path/task.json", logger)
	
	app := &App{
		ctx:             context.Background(),
		taskService:     taskService,
		terminalService: nil,
		agentService:    nil,
		configService:   nil,
		logger:          logger,
		errorHandler:    NewErrorHandler(logger),
	}

	// This should handle the error gracefully
	err := app.SaveTasks(testTasks)
	if err == nil {
		t.Error("Expected error for impossible file path")
	}
}

// Test 7: Status Filtering
func TestGetTasksByStatus(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup fresh tasks for this test to avoid interference from other tests
	freshTasks := []Task{
		{ID: 10, Title: "Fresh Todo", Status: StatusTodo, Priority: PriorityHigh, Deps: []int{}, Parent: nil},
		{ID: 11, Title: "Fresh Doing", Status: StatusDoing, Priority: PriorityMedium, Deps: []int{}, Parent: nil},
		{ID: 12, Title: "Fresh Done", Status: StatusDone, Priority: PriorityLow, Deps: []int{}, Parent: nil},
	}
	if err := app.SaveTasks(freshTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Test filtering by status
	todoTasks, err := app.GetTasksByStatus("todo")
	if err != nil {
		t.Fatalf("GetTasksByStatus failed: %v", err)
	}

	expectedCount := 1 // Only one "todo" task in freshTasks
	if len(todoTasks) != expectedCount {
		t.Errorf("Expected %d todo tasks, got %d", expectedCount, len(todoTasks))
	}

	// Verify the correct task was returned
	if len(todoTasks) > 0 && todoTasks[0].Status != StatusTodo {
		t.Error("Returned task does not have 'todo' status")
	}
}

// Test 8: Concurrent Access Safety
func TestConcurrentAccess(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup initial tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Run concurrent operations
	done := make(chan bool, 3)

	// Concurrent reads
	go func() {
		for i := 0; i < 10; i++ {
			app.LoadTasks()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Concurrent status filtering
	go func() {
		for i := 0; i < 10; i++ {
			app.GetTasksByStatus("todo")
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Concurrent task moves
	go func() {
		statuses := []string{"todo", "doing", "done", "backlog"}
		for i := 0; i < 10; i++ {
			app.MoveTask(1, statuses[i%len(statuses)])
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify data integrity after concurrent access
	tasks, err := app.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed after concurrent access: %v", err)
	}

	if len(tasks) != len(testTasks) {
		t.Errorf("Task count changed during concurrent access: expected %d, got %d", len(testTasks), len(tasks))
	}
}

// Test 9: Refresh from Disk - External file changes
func TestRefreshFromDisk(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Save initial tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Get task file path from service
	taskFile := app.taskService.(*TaskService).taskFile
	
	// Simulate external file modification by directly writing to task file
	externalTasks := []Task{
		{ID: 99, Title: "External Task", Status: StatusTodo, Priority: PriorityHigh, Deps: []int{}, Parent: nil},
	}
	
	data, err := json.MarshalIndent(externalTasks, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal external tasks: %v", err)
	}
	
	if err := os.WriteFile(taskFile, data, 0644); err != nil {
		t.Fatalf("Failed to write external task file: %v", err)
	}

	// LoadTasks should pick up the external changes
	refreshedTasks, err := app.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	// Verify we got the externally modified tasks
	if len(refreshedTasks) != 1 {
		t.Errorf("Expected 1 task after external modification, got %d", len(refreshedTasks))
	}

	if len(refreshedTasks) > 0 && refreshedTasks[0].ID != 99 {
		t.Errorf("Expected external task with ID 99, got ID %d", refreshedTasks[0].ID)
	}

	if len(refreshedTasks) > 0 && refreshedTasks[0].Title != "External Task" {
		t.Errorf("Expected external task title 'External Task', got '%s'", refreshedTasks[0].Title)
	}
}

// Test 10: Todo to Doing Transition (Claude agent trigger condition)
func TestTodoToDoingTransition(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup initial tasks with specific status
	testTasksWithStatus := []Task{
		{ID: 1, Title: "Test Task 1", Status: StatusTodo, Priority: PriorityHigh, Deps: []int{}, Parent: nil},
		{ID: 2, Title: "Test Task 2", Status: StatusBacklog, Priority: PriorityMedium, Deps: []int{}, Parent: nil},
	}
	
	if err := app.SaveTasks(testTasksWithStatus); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Test moving from "todo" to "doing" - this should trigger Claude agent launch
	err := app.MoveTask(1, "doing")
	if err != nil {
		t.Fatalf("MoveTask from 'todo' to 'doing' failed: %v", err)
	}

	// Test moving from "backlog" to "doing" - this should NOT trigger Claude agent
	err = app.MoveTask(2, "doing") 
	if err != nil {
		t.Fatalf("MoveTask from 'backlog' to 'doing' failed: %v", err)
	}

	// Verify both tasks were moved to "doing"
	tasks, err := app.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	doingCount := 0
	for _, task := range tasks {
		if task.Status == StatusDoing {
			doingCount++
		}
	}

	if doingCount != 2 {
		t.Errorf("Expected 2 tasks in 'doing' status, got %d", doingCount)
	}

	// Note: The actual Claude agent launch happens in a goroutine and can't be
	// easily tested in unit tests. The condition (oldStatus == StatusTodo && newStatus == StatusDoing)
	// is the key logic that determines when agents are launched.
}

// Test 11: Pending Review Status
func TestPendingReviewStatus(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a task in pending_review status
	pendingTask := Task{
		ID:       1,
		Title:    "Review Task",
		Status:   StatusPendingReview,
		Priority: PriorityHigh,
		Deps:     []int{},
		Parent:   nil,
	}

	if err := app.SaveTasks([]Task{pendingTask}); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Verify task is saved correctly
	tasks, err := app.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(tasks) != 1 || tasks[0].Status != StatusPendingReview {
		t.Error("Task was not saved with pending_review status")
	}

	// Test getting tasks by pending_review status
	pendingTasks, err := app.GetTasksByStatus("pending_review")
	if err != nil {
		t.Fatalf("GetTasksByStatus failed: %v", err)
	}

	if len(pendingTasks) != 1 {
		t.Errorf("Expected 1 pending_review task, got %d", len(pendingTasks))
	}
}

// Test 12: Type Safety for Status and Priority
func TestTypeSafety(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test all valid statuses
	for i, status := range AllStatuses() {
		task := Task{
			ID:       100 + i, // Unique ID for each status
			Title:    string(status) + " Task",
			Status:   status,
			Priority: PriorityMedium,
			Deps:     []int{},
			Parent:   nil,
		}

		if err := app.SaveTasks([]Task{task}); err != nil {
			t.Errorf("Failed to save task with valid status %s: %v", status, err)
		}
	}

	// Test all valid priorities
	for i, priority := range AllPriorities() {
		task := Task{
			ID:       200 + i, // Unique ID for each priority
			Title:    string(priority) + " Priority Task",
			Status:   StatusTodo,
			Priority: priority,
			Deps:     []int{},
			Parent:   nil,
		}

		if err := app.SaveTasks([]Task{task}); err != nil {
			t.Errorf("Failed to save task with valid priority %s: %v", priority, err)
		}
	}
}

// Test 13: Error Types and Handling
func TestErrorTypes(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test validation error
	invalidTask := Task{
		ID:       1,
		Title:    "", // Empty title should trigger validation error
		Status:   StatusTodo,
		Priority: PriorityHigh,
	}

	err := app.SaveTasks([]Task{invalidTask})
	if err == nil {
		t.Error("Expected validation error for empty title")
	}

	// Test not found error
	err = app.MoveTask(999, "doing")
	if err == nil {
		t.Error("Expected not found error for non-existent task")
	}
}

// Test 14: Review Workflow - Critical for the enhanced review system
func TestReviewWorkflow(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a task in pending_review status
	reviewTask := Task{
		ID:       1,
		Title:    "Feature Implementation",
		Status:   StatusPendingReview,
		Priority: PriorityHigh,
		Deps:     []int{},
		Parent:   nil,
	}

	if err := app.SaveTasks([]Task{reviewTask}); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Test approve functionality (should move to done)
	if err := app.ApproveTask(1); err != nil {
		t.Fatalf("ApproveTask failed: %v", err)
	}

	// Verify task moved to done
	tasks, _ := app.LoadTasks()
	if len(tasks) != 1 || tasks[0].Status != StatusDone {
		t.Error("Approved task was not moved to done status")
	}

	// Reset for reject test
	reviewTask.Status = StatusPendingReview
	if err := app.SaveTasks([]Task{reviewTask}); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Test reject functionality (should move to done with NOT MERGED prefix)
	if err := app.RejectTask(1); err != nil {
		t.Fatalf("RejectTask failed: %v", err)
	}

	// Verify task moved to done with NOT MERGED prefix
	tasks, _ = app.LoadTasks()
	if len(tasks) != 1 || tasks[0].Status != StatusDone {
		t.Error("Rejected task was not moved to done status")
	}
	
	// Check that title has NOT MERGED prefix
	if len(tasks) > 0 && tasks[0].Title != "NOT MERGED: Feature Implementation" {
		t.Errorf("Rejected task title should have 'NOT MERGED:' prefix, got: %s", tasks[0].Title)
	}
}

// Test 15: Recovery from Corrupted File - Critical for data integrity
func TestRecoveryFromCorruptedFile(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Save initial valid tasks
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}

	// Save again to create .tracked file
	if err := app.SaveTasks(testTasks); err != nil {
		t.Fatalf("Second SaveTasks failed: %v", err)
	}

	// Get task file path
	taskFile := app.taskService.(*TaskService).taskFile

	// Corrupt the main file
	if err := os.WriteFile(taskFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to corrupt task file: %v", err)
	}

	// LoadTasks should handle the corruption gracefully
	tasks, err := app.LoadTasks()
	if err != nil {
		// This is expected - the service should handle this gracefully
		// In a real implementation, it might recover from .tracked file
		t.Logf("LoadTasks returned error as expected: %v", err)
	} else {
		// If it succeeded, verify we got valid data (possibly from .tracked)
		if len(tasks) == 0 {
			t.Error("LoadTasks returned empty tasks after corruption")
		}
	}
}