package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test fixtures - minimal test data
var testTasks = []Task{
	{ID: 1, Title: "Test Task 1", Status: "todo", Priority: "high", Deps: []int{}, Parent: nil},
	{ID: 2, Title: "Test Task 2", Status: "doing", Priority: "medium", Deps: []int{1}, Parent: nil},
	{ID: 3, Title: "Test Task 3", Status: "done", Priority: "low", Deps: []int{}, Parent: nil},
}

func setupTestApp(t *testing.T) (*App, func()) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	app := &App{
		taskFile: filepath.Join(tmpDir, "task.json"),
		tasks:    []Task{},
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
		{"Valid task", Task{ID: 1, Title: "Valid", Status: "todo", Priority: "high"}, false},
		{"Empty title", Task{ID: 1, Title: "", Status: "todo", Priority: "high"}, true},
		{"Invalid status", Task{ID: 1, Title: "Test", Status: "invalid", Priority: "high"}, true},
		{"Invalid priority", Task{ID: 1, Title: "Test", Status: "todo", Priority: "invalid"}, true},
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
		if task.ID == 1 && task.Status == "doing" {
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
	updatedTask := Task{ID: 1, Title: "Updated Task", Status: "doing", Priority: "low", Deps: []int{}, Parent: nil}
	if err := app.UpdateTask(updatedTask); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify update
	tasks, _ := app.LoadTasks()
	found := false
	for _, task := range tasks {
		if task.ID == 1 && task.Title == "Updated Task" && task.Priority == "low" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Task was not updated correctly")
	}

	// Test updating non-existent task
	nonExistentTask := Task{ID: 999, Title: "Ghost", Status: "todo", Priority: "high"}
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

	// Verify main file exists
	if _, err := os.Stat(app.taskFile); os.IsNotExist(err) {
		t.Error("Task file was not created")
	}

	// Save again to trigger backup
	modifiedTasks := append(testTasks, Task{ID: 4, Title: "New Task", Status: "todo", Priority: "medium"})
	if err := app.SaveTasks(modifiedTasks); err != nil {
		t.Fatalf("Second SaveTasks failed: %v", err)
	}

	// Check that backup was created (backup files have .backup.timestamp format)
	taskDir := filepath.Dir(app.taskFile)
	files, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatalf("Failed to read task directory: %v", err)
	}

	backupFound := false
	for _, file := range files {
		if filepath.Base(file.Name()) != filepath.Base(app.taskFile) && 
		   filepath.HasPrefix(file.Name(), filepath.Base(app.taskFile)+".backup.") {
			backupFound = true
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}
}

// Test 6: Error Handling - File system errors
func TestErrorHandling(t *testing.T) {
	// Test with invalid directory path
	app := &App{
		taskFile: "/root/impossible/path/task.json", // Should fail on most systems
		tasks:    []Task{},
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
		{ID: 10, Title: "Fresh Todo", Status: "todo", Priority: "high", Deps: []int{}, Parent: nil},
		{ID: 11, Title: "Fresh Doing", Status: "doing", Priority: "medium", Deps: []int{}, Parent: nil},
		{ID: 12, Title: "Fresh Done", Status: "done", Priority: "low", Deps: []int{}, Parent: nil},
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
	if len(todoTasks) > 0 && todoTasks[0].Status != "todo" {
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

	// Simulate external file modification by directly writing to task file
	externalTasks := []Task{
		{ID: 99, Title: "External Task", Status: "todo", Priority: "high", Deps: []int{}, Parent: nil},
	}
	
	data, err := json.MarshalIndent(externalTasks, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal external tasks: %v", err)
	}
	
	if err := os.WriteFile(app.taskFile, data, 0644); err != nil {
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

// Test 10: Claude Agent Prompt Generation
func TestGenerateTaskPrompt(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	tests := []struct {
		name     string
		task     Task
		expected string
	}{
		{
			name: "Simple task",
			task: Task{ID: 1, Title: "Simple Task", Status: "todo", Priority: "medium"},
			expected: "Review plan.md and task.json. Begin task #1: Simple Task. Update task.json status to 'pending_review' when done, commit to branch task_1.",
		},
		{
			name: "High priority task",
			task: Task{ID: 2, Title: "Urgent Task", Status: "todo", Priority: "high"},
			expected: "Review plan.md and task.json. Begin task #2: Urgent Task. Update task.json status to 'pending_review' when done, commit to branch task_2.",
		},
		{
			name: "Task with parent",
			task: Task{ID: 3, Title: "Subtask", Status: "todo", Priority: "low", Parent: &[]int{10}[0]},
			expected: "Review plan.md and task.json. Begin task #3: Subtask. Update task.json status to 'pending_review' when done, commit to branch task_3.",
		},
		{
			name: "Task with dependencies",
			task: Task{ID: 4, Title: "Dependent Task", Status: "todo", Priority: "medium", Deps: []int{1, 2}},
			expected: "Review plan.md and task.json. Begin task #4: Dependent Task. Update task.json status to 'pending_review' when done, commit to branch task_4.",
		},
		{
			name: "Complex task",
			task: Task{ID: 5, Title: "Complex Task", Status: "todo", Priority: "high", Parent: &[]int{20}[0], Deps: []int{3, 4}},
			expected: "Review plan.md and task.json. Begin task #5: Complex Task. Update task.json status to 'pending_review' when done, commit to branch task_5.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.generateTaskPrompt(tt.task)
			if result != tt.expected {
				t.Errorf("generateTaskPrompt() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// Test 11: Todo to Doing Transition (Claude agent trigger condition)
func TestTodoToDoingTransition(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Setup initial tasks with specific status
	testTasksWithStatus := []Task{
		{ID: 1, Title: "Test Task 1", Status: "todo", Priority: "high", Deps: []int{}, Parent: nil},
		{ID: 2, Title: "Test Task 2", Status: "backlog", Priority: "medium", Deps: []int{}, Parent: nil},
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
		if task.Status == "doing" {
			doingCount++
		}
	}

	if doingCount != 2 {
		t.Errorf("Expected 2 tasks in 'doing' status, got %d", doingCount)
	}

	// Note: The actual Claude agent launch happens in a goroutine and can't be
	// easily tested in unit tests. The condition (oldStatus == "todo" && newStatus == "doing")
	// is the key logic that determines when agents are launched.
}