package config

import (
	"fmt"
	"time"
)

// GlobalConfig represents the global CLI configuration
type GlobalConfig struct {
	ActiveContext  string `yaml:"active_context"`
	OutputFormat   string `yaml:"output_format"` // json|yaml|table
	ColorsEnabled  bool   `yaml:"colors_enabled"`
	PaginationSize int    `yaml:"pagination_size"`
	Debug          bool   `yaml:"debug"`
}

// Context represents a single environment context configuration
type Context struct {
	Name       string           `yaml:"name"`
	PocketBase PocketBaseConfig `yaml:"pocketbase"`
}

// PocketBaseConfig contains PocketBase-specific configuration
type PocketBaseConfig struct {
	URL                  string                 `yaml:"url"`
	AuthCollection       string                 `yaml:"auth_collection"`        // e.g. users|_superusers|custom
	AuthToken            string                 `yaml:"auth_token"`             // Session token
	AuthExpires          *time.Time             `yaml:"auth_expires"`           // Token expiration
	AuthRecord           map[string]interface{} `yaml:"auth_record"`            // Cached auth record
	AutoRefresh          bool                   `yaml:"auto_refresh"`           // Refresh token proactively when nearing expiry
	AutoRefreshThreshold string                 `yaml:"auto_refresh_threshold"` // Duration string (e.g. "15m"); empty => default
}

// DefaultAutoRefreshThreshold is used when AutoRefresh is enabled but no threshold is set.
const DefaultAutoRefreshThreshold = 15 * time.Minute

// GetAutoRefreshThreshold returns the parsed auto-refresh threshold, falling back to
// DefaultAutoRefreshThreshold on empty or invalid values.
func (p *PocketBaseConfig) GetAutoRefreshThreshold() time.Duration {
	if p.AutoRefreshThreshold == "" {
		return DefaultAutoRefreshThreshold
	}
	d, err := time.ParseDuration(p.AutoRefreshThreshold)
	if err != nil || d <= 0 {
		return DefaultAutoRefreshThreshold
	}
	return d
}

// Output format constants
const (
	OutputFormatJSON  = "json"
	OutputFormatYAML  = "yaml"
	OutputFormatTable = "table"
)

// PocketBase auth collection constants. Any collection name is allowed; these are
// just the common ones for v0.23+ (superuser auth lives in the _superusers collection).
const (
	AuthCollectionUsers      = "users"
	AuthCollectionSuperusers = "_superusers"
)

// ValidateAuthCollection validates a PocketBase auth collection name
// Note: This is permissive to allow any collection name since PocketBase supports custom auth collections
func ValidateAuthCollection(collection string) error {
	if collection == "" {
		return fmt.Errorf("auth collection cannot be empty")
	}

	// Basic validation - PocketBase will handle the actual validation
	if len(collection) < 1 || len(collection) > 50 {
		return fmt.Errorf("auth collection name must be between 1 and 50 characters")
	}

	return nil
}

// Global configuration instance (will be populated by root command)
var Global = &GlobalConfig{
	OutputFormat:   OutputFormatJSON,
	ColorsEnabled:  true,
	PaginationSize: 30,
	Debug:          false,
}
