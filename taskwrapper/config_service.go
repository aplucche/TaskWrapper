package main

import (
	"fmt"
	"sync"
)

// ConfigService handles configuration operations in a thread-safe manner
type ConfigService struct {
	configManager *ConfigManager
	mu            sync.RWMutex
	logger        Logger
}

// NewConfigService creates a new config service
func NewConfigService(logger Logger) (*ConfigService, error) {
	configManager, err := NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %v", err)
	}

	return &ConfigService{
		configManager: configManager,
		logger:        logger,
	}, nil
}

// GetConfig returns the current configuration
func (cs *ConfigService) GetConfig() (*Config, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	if cs.configManager == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	
	return cs.configManager.GetConfig(), nil
}

// GetRepositories returns all configured repositories
func (cs *ConfigService) GetRepositories() ([]Repository, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	if cs.configManager == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	
	config := cs.configManager.GetConfig()
	if config == nil {
		return []Repository{}, nil
	}
	
	return config.Repositories, nil
}

// GetActiveRepository returns the active repository
func (cs *ConfigService) GetActiveRepository() (*Repository, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	if cs.configManager == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	
	repo, err := cs.configManager.GetActiveRepository()
	if err != nil {
		cs.logger.Error("Failed to get active repository", err)
		return nil, err
	}
	
	return repo, nil
}

// GetActiveRepositoryPath returns the path of the active repository
func (cs *ConfigService) GetActiveRepositoryPath() (string, error) {
	activeRepo, err := cs.GetActiveRepository()
	if err != nil {
		return "", err
	}
	return activeRepo.Path, nil
}

// AddRepository adds a new repository to the configuration
func (cs *ConfigService) AddRepository(name, path string) (*Repository, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.configManager == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	
	cs.logger.InfoWithFields("Adding repository", map[string]interface{}{
		"name": name,
		"path": path,
	})
	
	repo, err := cs.configManager.AddRepository(name, path)
	if err != nil {
		cs.logger.ErrorWithFields("Failed to add repository", err, map[string]interface{}{
			"name": name,
			"path": path,
		})
		return nil, err
	}
	
	cs.logger.InfoWithFields("Repository added successfully", map[string]interface{}{
		"name": repo.Name,
		"path": repo.Path,
		"id":   repo.ID,
	})
	
	return repo, nil
}

// RemoveRepository removes a repository from the configuration
func (cs *ConfigService) RemoveRepository(id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.configManager == nil {
		return fmt.Errorf("configuration not initialized")
	}
	
	cs.logger.InfoWithFields("Removing repository", map[string]interface{}{
		"id": id,
	})
	
	err := cs.configManager.RemoveRepository(id)
	if err != nil {
		cs.logger.ErrorWithFields("Failed to remove repository", err, map[string]interface{}{
			"id": id,
		})
		return err
	}
	
	cs.logger.InfoWithFields("Repository removed successfully", map[string]interface{}{
		"id": id,
	})
	
	return nil
}

// SetActiveRepository switches the active repository
func (cs *ConfigService) SetActiveRepository(id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.configManager == nil {
		return fmt.Errorf("configuration not initialized")
	}
	
	cs.logger.InfoWithFields("Setting active repository", map[string]interface{}{
		"id": id,
	})
	
	err := cs.configManager.SetActiveRepository(id)
	if err != nil {
		cs.logger.ErrorWithFields("Failed to set active repository", err, map[string]interface{}{
			"id": id,
		})
		return err
	}
	
	// Get repository info for logging
	activeRepo, err := cs.configManager.GetActiveRepository()
	if err == nil {
		cs.logger.InfoWithFields("Active repository set successfully", map[string]interface{}{
			"id":   id,
			"name": activeRepo.Name,
			"path": activeRepo.Path,
		})
	}
	
	return nil
}

// ValidateRepositoryPath validates a repository path
func (cs *ConfigService) ValidateRepositoryPath(path string) (*RepositoryInfo, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	cs.logger.InfoWithFields("Validating repository path", map[string]interface{}{
		"path": path,
	})
	
	info, err := ValidateRepository(path)
	if err != nil {
		cs.logger.ErrorWithFields("Repository validation failed", err, map[string]interface{}{
			"path": path,
		})
		return nil, err
	}
	
	cs.logger.InfoWithFields("Repository validation successful", map[string]interface{}{
		"path":          path,
		"name":          info.Name,
		"is_valid":      info.IsValid,
		"has_plan_file": info.HasPlanFile,
		"task_count":    info.TaskCount,
	})
	
	return info, nil
}

// FindRepositories searches for repositories in a directory
func (cs *ConfigService) FindRepositories(searchPath string) ([]Repository, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	cs.logger.InfoWithFields("Searching for repositories", map[string]interface{}{
		"search_path": searchPath,
	})
	
	repos, err := FindRepositoriesInDirectory(searchPath)
	if err != nil {
		cs.logger.ErrorWithFields("Failed to find repositories", err, map[string]interface{}{
			"search_path": searchPath,
		})
		return nil, err
	}
	
	cs.logger.InfoWithFields("Repository search completed", map[string]interface{}{
		"search_path": searchPath,
		"found_count": len(repos),
	})
	
	return repos, nil
}

// ReloadConfig reloads the configuration from disk
func (cs *ConfigService) ReloadConfig() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.configManager == nil {
		return fmt.Errorf("configuration not initialized")
	}
	
	err := cs.configManager.Load()
	if err != nil {
		cs.logger.Error("Failed to reload configuration", err)
		return err
	}
	
	cs.logger.Info("Configuration reloaded successfully")
	return nil
}

// GetConfigPath returns the path to the configuration file
func (cs *ConfigService) GetConfigPath() (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	if cs.configManager == nil {
		return "", fmt.Errorf("configuration not initialized")
	}
	
	return cs.configManager.configPath, nil
}