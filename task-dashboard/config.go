package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the application configuration
type Config struct {
	Version          string       `json:"version"`
	ActiveRepository string       `json:"activeRepository"`
	Repositories     []Repository `json:"repositories"`
}

// Repository represents a single repository configuration
type Repository struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	AddedAt time.Time `json:"addedAt"`
}

// ConfigManager handles loading and saving configuration
type ConfigManager struct {
	configPath string
	config     *Config
	repoUtils  *RepositoryUtils
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %v", err)
	}
	
	configPath := filepath.Join(configDir, "config.json")
	
	cm := &ConfigManager{
		configPath: configPath,
		repoUtils:  &RepositoryUtils{},
	}
	
	if err := cm.Load(); err != nil {
		// If config doesn't exist, create default
		if os.IsNotExist(err) {
			cm.config = cm.createDefaultConfig()
			if err := cm.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %v", err)
		}
	}
	
	return cm, nil
}

// getConfigDir returns the application configuration directory
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	configDir := filepath.Join(homeDir, ".config", "task-dashboard")
	
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	
	return configDir, nil
}

// Load reads the configuration from disk
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}
	
	cm.config = &config
	return nil
}

// Save writes the configuration to disk
func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	// Write to temp file first for atomic operation
	tmpFile := cm.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpFile, cm.configPath); err != nil {
		os.Remove(tmpFile) // Clean up
		return fmt.Errorf("failed to save config: %v", err)
	}
	
	return nil
}

// createDefaultConfig creates a default configuration
func (cm *ConfigManager) createDefaultConfig() *Config {
	// Try to detect current repository
	currentRepo := cm.detectCurrentRepository()
	
	config := &Config{
		Version:          "1.0.0",
		ActiveRepository: currentRepo.Path,
		Repositories:     []Repository{currentRepo},
	}
	
	return config
}

// detectCurrentRepository intelligently detects the repository based on launch location
func (cm *ConfigManager) detectCurrentRepository() Repository {
	// Get current working directory where app was launched
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	
	// Strategy 1: Walk up from current directory to find the nearest repository
	// This is the most important strategy - find the repo we're actually inside of
	repoPath := cm.findRepositoryRootFromPath(cwd)
	if repoPath != "" {
		return Repository{
			ID:      generateID(),
			Name:    GetRepositoryName(repoPath),
			Path:    repoPath,
			AddedAt: time.Now(),
		}
	}
	
	// Strategy 2: No valid repository found - return empty repository
	// This indicates the app should default to settings mode
	homeDir, _ := os.UserHomeDir()
	fallbackPath := filepath.Join(homeDir, "Documents", "TaskDashboard")
	return Repository{
		ID:      generateID(),
		Name:    "No Repository",
		Path:    fallbackPath,
		AddedAt: time.Now(),
	}
}

// findRepositoryRootFromPath finds the root of a repository that contains task dashboard files
// by walking up the directory tree from the given path
func (cm *ConfigManager) findRepositoryRootFromPath(startPath string) string {
	path := startPath
	
	for {
		// Check if this is a valid repository (has plan/task.json)
		if cm.repoUtils.IsValidRepository(path) {
			return path
		}
		
		// Move up one directory
		parent := filepath.Dir(path)
		if parent == path {
			// Reached filesystem root
			break
		}
		path = parent
	}
	return ""
}


// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// GetActiveRepository returns the active repository
func (cm *ConfigManager) GetActiveRepository() (*Repository, error) {
	for _, repo := range cm.config.Repositories {
		if repo.Path == cm.config.ActiveRepository {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("active repository not found")
}

// AddRepository adds a new repository to the configuration
func (cm *ConfigManager) AddRepository(name, path string) (*Repository, error) {
	// Validate path
	if err := validateRepositoryPath(path); err != nil {
		return nil, err
	}
	
	// Check if already exists
	for _, repo := range cm.config.Repositories {
		if repo.Path == path {
			return nil, fmt.Errorf("repository already exists")
		}
	}
	
	// If name is empty, use directory name
	if name == "" {
		name = filepath.Base(path)
	}
	
	repo := Repository{
		ID:      generateID(),
		Name:    name,
		Path:    path,
		AddedAt: time.Now(),
	}
	
	cm.config.Repositories = append(cm.config.Repositories, repo)
	
	// If this is the first repository, make it active
	if len(cm.config.Repositories) == 1 {
		cm.config.ActiveRepository = path
	}
	
	return &repo, cm.Save()
}

// RemoveRepository removes a repository from the configuration
func (cm *ConfigManager) RemoveRepository(id string) error {
	// Find and remove repository
	var newRepos []Repository
	removed := false
	
	for _, repo := range cm.config.Repositories {
		if repo.ID != id {
			newRepos = append(newRepos, repo)
		} else {
			removed = true
			// If removing active repository, need to select a new one
			if repo.Path == cm.config.ActiveRepository && len(cm.config.Repositories) > 1 {
				// Select first remaining repository
				for _, r := range cm.config.Repositories {
					if r.ID != id {
						cm.config.ActiveRepository = r.Path
						break
					}
				}
			}
		}
	}
	
	if !removed {
		return fmt.Errorf("repository not found")
	}
	
	cm.config.Repositories = newRepos
	
	// If no repositories left, clear active repository
	if len(cm.config.Repositories) == 0 {
		cm.config.ActiveRepository = ""
	}
	
	return cm.Save()
}

// SetActiveRepository sets the active repository
func (cm *ConfigManager) SetActiveRepository(id string) error {
	// Find repository
	found := false
	for _, repo := range cm.config.Repositories {
		if repo.ID == id {
			cm.config.ActiveRepository = repo.Path
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("repository not found")
	}
	
	return cm.Save()
}

// validateRepositoryPath validates that a path contains a valid task dashboard repository
func validateRepositoryPath(path string) error {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path does not exist: %v", err)
	}
	
	// Check if plan/task.json exists
	taskFile := filepath.Join(path, "plan", "task.json")
	if _, err := os.Stat(taskFile); err != nil {
		return fmt.Errorf("not a valid task dashboard repository (missing plan/task.json)")
	}
	
	return nil
}

// generateID generates a unique ID for a repository
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

