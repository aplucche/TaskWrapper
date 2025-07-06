package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	AllowedOrigins   []string
	AllowedPaths     []string
	RestrictedPaths  []string
	MaxPathDepth     int
	EnablePathChecks bool
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	homeDir, _ := os.UserHomeDir()
	
	return &SecurityConfig{
		AllowedOrigins: []string{
			"http://localhost:5173",     // Vite dev server
			"http://localhost:34115",    // Wails dev server
			"wails://wails",            // Wails internal
			"file://",                  // Local file protocol
		},
		AllowedPaths: []string{
			homeDir,
		},
		RestrictedPaths: []string{
			"/etc",
			"/usr",
			"/bin",
			"/sbin",
			"/System",
			"/Windows",
			"/Program Files",
		},
		MaxPathDepth:     10,
		EnablePathChecks: true,
	}
}

// PathValidator provides secure path validation
type PathValidator struct {
	config *SecurityConfig
	logger Logger
}

// NewPathValidator creates a new path validator
func NewPathValidator(config *SecurityConfig, logger Logger) *PathValidator {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	return &PathValidator{
		config: config,
		logger: logger,
	}
}

// ValidatePath checks if a path is safe to access
func (pv *PathValidator) ValidatePath(inputPath string) (string, error) {
	if !pv.config.EnablePathChecks {
		return inputPath, nil
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(inputPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check for path traversal attempts
	if strings.Contains(inputPath, "..") {
		pv.logger.ErrorWithFields("Path traversal attempt detected", nil, map[string]interface{}{
			"input_path": inputPath,
			"resolved":   absPath,
		})
		return "", fmt.Errorf("path traversal not allowed")
	}

	// Check path depth
	depth := len(strings.Split(absPath, string(os.PathSeparator)))
	if depth > pv.config.MaxPathDepth {
		return "", fmt.Errorf("path too deep: %d levels (max: %d)", depth, pv.config.MaxPathDepth)
	}

	// Check against restricted paths
	for _, restricted := range pv.config.RestrictedPaths {
		if strings.HasPrefix(absPath, restricted) {
			pv.logger.ErrorWithFields("Access to restricted path attempted", nil, map[string]interface{}{
				"path":       absPath,
				"restricted": restricted,
			})
			return "", fmt.Errorf("access to restricted path denied: %s", restricted)
		}
	}

	// Check if path is within allowed paths
	allowed := false
	for _, allowedPath := range pv.config.AllowedPaths {
		allowedAbs, _ := filepath.Abs(allowedPath)
		if strings.HasPrefix(absPath, allowedAbs) {
			allowed = true
			break
		}
	}

	if !allowed && len(pv.config.AllowedPaths) > 0 {
		return "", fmt.Errorf("path not in allowed directories")
	}

	pv.logger.InfoWithFields("Path validated successfully", map[string]interface{}{
		"input":    inputPath,
		"resolved": absPath,
	})

	return absPath, nil
}

// ValidateExecutable checks if a path points to a safe executable
func (pv *PathValidator) ValidateExecutable(execPath string) (string, error) {
	// First validate the path
	validPath, err := pv.ValidatePath(execPath)
	if err != nil {
		return "", err
	}

	// Check if file exists
	info, err := os.Stat(validPath)
	if err != nil {
		return "", fmt.Errorf("executable not found: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", validPath)
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("file is not executable: %s", validPath)
	}

	// Additional checks for known script types
	if strings.HasSuffix(validPath, ".sh") || strings.HasSuffix(validPath, ".bash") {
		// Read first line to check shebang
		file, err := os.Open(validPath)
		if err == nil {
			defer file.Close()
			buf := make([]byte, 2)
			if n, _ := file.Read(buf); n >= 2 && string(buf) != "#!" {
				return "", fmt.Errorf("shell script missing shebang: %s", validPath)
			}
		}
	}

	return validPath, nil
}

// SanitizeFilename removes potentially dangerous characters from a filename
func (pv *PathValidator) SanitizeFilename(filename string) string {
	// Remove path separators and other dangerous characters
	dangerous := []string{"/", "\\", "..", "~", "$", "`", "|", "&", ";", "(", ")", "[", "]", "{", "}", "<", ">", "!", "?", "*", "'", "\"", "\n", "\r", "\t"}
	
	sanitized := filename
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Limit length
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// OriginValidator provides origin validation for WebSocket connections
type OriginValidator struct {
	allowedOrigins map[string]bool
	logger         Logger
}

// NewOriginValidator creates a new origin validator
func NewOriginValidator(origins []string, logger Logger) *OriginValidator {
	allowed := make(map[string]bool)
	for _, origin := range origins {
		allowed[origin] = true
	}
	
	return &OriginValidator{
		allowedOrigins: allowed,
		logger:         logger,
	}
}

// ValidateOrigin checks if an origin is allowed
func (ov *OriginValidator) ValidateOrigin(origin string) bool {
	// If no origins specified, allow all (development mode)
	if len(ov.allowedOrigins) == 0 {
		ov.logger.Info("No origin restrictions configured (development mode)")
		return true
	}

	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		ov.logger.ErrorWithFields("Invalid origin URL", err, map[string]interface{}{
			"origin": origin,
		})
		return false
	}

	// Check exact match
	if ov.allowedOrigins[origin] {
		return true
	}

	// Check with normalized URL (scheme://host:port)
	normalized := fmt.Sprintf("%s://%s", originURL.Scheme, originURL.Host)
	if ov.allowedOrigins[normalized] {
		return true
	}

	// Check wildcard for local development
	if originURL.Hostname() == "localhost" || originURL.Hostname() == "127.0.0.1" {
		for allowed := range ov.allowedOrigins {
			if strings.Contains(allowed, "localhost") || strings.Contains(allowed, "127.0.0.1") {
				return true
			}
		}
	}

	ov.logger.ErrorWithFields("Origin not allowed", nil, map[string]interface{}{
		"origin":  origin,
		"allowed": ov.allowedOrigins,
	})

	return false
}