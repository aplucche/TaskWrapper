package main

import "fmt"

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusBacklog       TaskStatus = "backlog"
	StatusTodo          TaskStatus = "todo"
	StatusDoing         TaskStatus = "doing"
	StatusPendingReview TaskStatus = "pending_review"
	StatusDone          TaskStatus = "done"
)

// Valid returns true if the status is valid
func (s TaskStatus) Valid() bool {
	switch s {
	case StatusBacklog, StatusTodo, StatusDoing, StatusPendingReview, StatusDone:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status
func (s TaskStatus) String() string {
	return string(s)
}

// TaskPriority represents the priority of a task
type TaskPriority string

const (
	PriorityHigh   TaskPriority = "high"
	PriorityMedium TaskPriority = "medium"
	PriorityLow    TaskPriority = "low"
)

// Valid returns true if the priority is valid
func (p TaskPriority) Valid() bool {
	switch p {
	case PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}

// String returns the string representation of the priority
func (p TaskPriority) String() string {
	return string(p)
}

// ParseTaskStatus converts a string to TaskStatus
func ParseTaskStatus(s string) (TaskStatus, error) {
	status := TaskStatus(s)
	if !status.Valid() {
		return "", fmt.Errorf("invalid task status: %s", s)
	}
	return status, nil
}

// ParseTaskPriority converts a string to TaskPriority
func ParseTaskPriority(p string) (TaskPriority, error) {
	priority := TaskPriority(p)
	if !priority.Valid() {
		return "", fmt.Errorf("invalid task priority: %s", p)
	}
	return priority, nil
}

// AllStatuses returns all valid task statuses
func AllStatuses() []TaskStatus {
	return []TaskStatus{
		StatusBacklog,
		StatusTodo,
		StatusDoing,
		StatusPendingReview,
		StatusDone,
	}
}

// AllPriorities returns all valid task priorities
func AllPriorities() []TaskPriority {
	return []TaskPriority{
		PriorityHigh,
		PriorityMedium,
		PriorityLow,
	}
}