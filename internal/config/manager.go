package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// Manager handles configuration and context management
type Manager struct {
	configDir string
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	// Create XDG-compliant config directory
	configDir := filepath.Join(xdg.ConfigHome, "pb")

	// Ensure main config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Manager{
		configDir: configDir,
	}, nil
}

// GetConfigDir returns the main configuration directory path
func (m *Manager) GetConfigDir() string {
	return m.configDir
}

// GetGlobalConfigPath returns the path to the global config file
func (m *Manager) GetGlobalConfigPath() string {
	return filepath.Join(m.configDir, "config.yaml")
}

// GetContextDir returns the directory path for a specific context
func (m *Manager) GetContextDir(name string) string {
	return filepath.Join(m.configDir, name)
}

// GetContextPath returns the path to a specific context configuration file
func (m *Manager) GetContextPath(name string) string {
	return filepath.Join(m.GetContextDir(name), "context.yaml")
}

// LoadGlobalConfig loads the global configuration
func (m *Manager) LoadGlobalConfig() (*GlobalConfig, error) {
	configPath := m.GetGlobalConfigPath()
	
	// Create default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &GlobalConfig{
			ActiveContext:  "",
			OutputFormat:   "json",
			ColorsEnabled:  true,
			PaginationSize: 30,
		}
		
		if err := m.SaveGlobalConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		
		return defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveGlobalConfig saves the global configuration
func (m *Manager) SaveGlobalConfig(config *GlobalConfig) error {
	configPath := m.GetGlobalConfigPath()
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadContext loads a specific context configuration
func (m *Manager) LoadContext(name string) (*Context, error) {
	if name == "" {
		return nil, fmt.Errorf("context name cannot be empty")
	}

	contextPath := m.GetContextPath(name)
	
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("context '%s' not found", name)
	}

	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	var context Context
	if err := yaml.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("failed to parse context file: %w", err)
	}

	return &context, nil
}

// SaveContext saves a context configuration
func (m *Manager) SaveContext(context *Context) error {
	if context.Name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	// Create context directory if it doesn't exist
	contextDir := m.GetContextDir(context.Name)
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}

	// Save context configuration
	contextPath := m.GetContextPath(context.Name)
	
	data, err := yaml.Marshal(context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	return nil
}

// ListContexts returns all available context names
func (m *Manager) ListContexts() ([]string, error) {
	// Read all directories in the config directory
	entries, err := os.ReadDir(m.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var contexts []string
	for _, entry := range entries {
		// Skip files and check only directories
		if !entry.IsDir() {
			continue
		}

		// Check if the directory contains a context.yaml file
		contextPath := filepath.Join(m.configDir, entry.Name(), "context.yaml")
		if _, err := os.Stat(contextPath); err == nil {
			contexts = append(contexts, entry.Name())
		}
	}

	return contexts, nil
}

// DeleteContext removes a context configuration and its directory
func (m *Manager) DeleteContext(name string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	contextDir := m.GetContextDir(name)
	
	if _, err := os.Stat(contextDir); os.IsNotExist(err) {
		return fmt.Errorf("context '%s' not found", name)
	}

	// Remove the entire context directory
	if err := os.RemoveAll(contextDir); err != nil {
		return fmt.Errorf("failed to delete context directory: %w", err)
	}

	return nil
}

// GetActiveContext returns the currently active context
func (m *Manager) GetActiveContext() (*Context, error) {
	globalConfig, err := m.LoadGlobalConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	if globalConfig.ActiveContext == "" {
		return nil, fmt.Errorf("no active context set")
	}

	return m.LoadContext(globalConfig.ActiveContext)
}

// SetActiveContext sets the active context
func (m *Manager) SetActiveContext(name string) error {
	// Verify context exists
	if _, err := m.LoadContext(name); err != nil {
		return err
	}

	globalConfig, err := m.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	globalConfig.ActiveContext = name

	return m.SaveGlobalConfig(globalConfig)
}

// ContextExists checks if a context exists
func (m *Manager) ContextExists(name string) bool {
	contextPath := m.GetContextPath(name)
	_, err := os.Stat(contextPath)
	return err == nil
}

// ValidateContextName validates a context name format
func (m *Manager) ValidateContextName(name string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '-' || char == '_') {
			return fmt.Errorf("context name can only contain letters, numbers, hyphens, and underscores")
		}
	}

	// Check length
	if len(name) > 50 {
		return fmt.Errorf("context name must be 50 characters or less")
	}

	return nil
}

// GetContextCollections returns the collections configured for a context
func (m *Manager) GetContextCollections(name string) ([]string, error) {
	ctx, err := m.LoadContext(name)
	if err != nil {
		return nil, err
	}
	
	return ctx.PocketBase.AvailableCollections, nil
}

// AddContextCollection adds a collection to a context
func (m *Manager) AddContextCollection(contextName, collection string) error {
	ctx, err := m.LoadContext(contextName)
	if err != nil {
		return err
	}
	
	if err := ctx.AddCollection(collection); err != nil {
		return err
	}
	
	return m.SaveContext(ctx)
}

// RemoveContextCollection removes a collection from a context
func (m *Manager) RemoveContextCollection(contextName, collection string) error {
	ctx, err := m.LoadContext(contextName)
	if err != nil {
		return err
	}
	
	if err := ctx.RemoveCollection(collection); err != nil {
		return err
	}
	
	return m.SaveContext(ctx)
}

// ClearContextCollections clears all collections from a context
func (m *Manager) ClearContextCollections(contextName string) error {
	ctx, err := m.LoadContext(contextName)
	if err != nil {
		return err
	}
	
	ctx.PocketBase.AvailableCollections = []string{}
	
	return m.SaveContext(ctx)
}
