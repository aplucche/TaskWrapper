package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// FileLogger implements Logger interface with file-based logging
type FileLogger struct {
	logDir string
}

// NewFileLogger creates a new file-based logger
func NewFileLogger(logDir string) *FileLogger {
	return &FileLogger{
		logDir: logDir,
	}
}

// Info logs an info message
func (fl *FileLogger) Info(message string) {
	fl.logToFile("INFO", message)
}

// Error logs an error message
func (fl *FileLogger) Error(message string, err error) {
	fullMessage := fmt.Sprintf("%s: %v", message, err)
	fl.logToFile("ERROR", fullMessage)
}

// InfoWithFields logs an info message with structured fields
func (fl *FileLogger) InfoWithFields(message string, fields map[string]interface{}) {
	fieldsStr := fl.formatFields(fields)
	fullMessage := fmt.Sprintf("%s %s", message, fieldsStr)
	fl.logToFile("INFO", fullMessage)
}

// ErrorWithFields logs an error message with structured fields
func (fl *FileLogger) ErrorWithFields(message string, err error, fields map[string]interface{}) {
	fieldsStr := fl.formatFields(fields)
	fullMessage := fmt.Sprintf("%s: %v %s", message, err, fieldsStr)
	fl.logToFile("ERROR", fullMessage)
}

// formatFields formats structured fields for logging
func (fl *FileLogger) formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	
	result := "["
	first := true
	for key, value := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", key, value)
		first = false
	}
	result += "]"
	return result
}

// logToFile writes log entries to the universal log file
func (fl *FileLogger) logToFile(level, message string) {
	// Get current date for log file
	now := time.Now()
	logDate := now.Format("2006-01-02")
	
	logFile := filepath.Join(fl.logDir, "universal_logs-"+logDate+".log")
	
	// Ensure log directory exists
	if err := os.MkdirAll(fl.logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return
	}
	
	// Format log entry
	timestamp := now.Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s taskwrapper: %s\n", timestamp, level, message)
	
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

// ConsoleLogger implements Logger interface with console output
type ConsoleLogger struct{}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

// Info logs an info message to console
func (cl *ConsoleLogger) Info(message string) {
	log.Printf("[INFO] %s", message)
}

// Error logs an error message to console
func (cl *ConsoleLogger) Error(message string, err error) {
	log.Printf("[ERROR] %s: %v", message, err)
}

// InfoWithFields logs an info message with fields to console
func (cl *ConsoleLogger) InfoWithFields(message string, fields map[string]interface{}) {
	log.Printf("[INFO] %s %s", message, cl.formatFields(fields))
}

// ErrorWithFields logs an error message with fields to console
func (cl *ConsoleLogger) ErrorWithFields(message string, err error, fields map[string]interface{}) {
	log.Printf("[ERROR] %s: %v %s", message, err, cl.formatFields(fields))
}

// formatFields formats structured fields for console output
func (cl *ConsoleLogger) formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	
	result := "["
	first := true
	for key, value := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", key, value)
		first = false
	}
	result += "]"
	return result
}