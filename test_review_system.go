package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Simple integration test for the review system
func main() {
	// Setup test app
	projectRoot, _ := os.Getwd()
	taskFile := filepath.Join(projectRoot, "plan", "task.json")
	
	app := &App{
		taskFile: taskFile,
		tasks:    []Task{},
	}
	
	// Load current tasks
	err := app.loadTasks()
	if err != nil {
		log.Fatalf("Failed to load tasks: %v", err)
	}
	
	// Find our test task
	var testTask *Task
	for i, task := range app.tasks {
		if task.ID == 999 {
			testTask = &app.tasks[i]
			break
		}
	}
	
	if testTask == nil {
		log.Fatalf("Test task #999 not found")
	}
	
	fmt.Printf("Found test task: %s (status: %s)\n", testTask.Title, testTask.Status)
	
	if testTask.Status != "pending_review" {
		log.Fatalf("Expected task to be in pending_review status, got: %s", testTask.Status)
	}
	
	fmt.Println("✅ Test task setup is correct")
	fmt.Println("✅ Integration test passed - review system is ready for UI testing")
}