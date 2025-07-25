package config

import (
	"fmt"
	"strings"
	"time"
)

// GlobalConfig represents the global CLI configuration
type GlobalConfig struct {
	ActiveContext       string `yaml:"active_context"`
	OutputFormat        string `yaml:"output_format"`        // json|yaml|table
	ColorsEnabled       bool   `yaml:"colors_enabled"`
	PaginationSize      int    `yaml:"pagination_size"`
	Debug               bool   `yaml:"debug"`
}

// Context represents a single environment context configuration
type Context struct {
	Name       string             `yaml:"name"`
	PocketBase PocketBaseConfig   `yaml:"pocketbase"`
}

// PocketBaseConfig contains PocketBase-specific configuration
type PocketBaseConfig struct {
	URL                    string                 `yaml:"url"`
	AuthCollection         string                 `yaml:"auth_collection"`         // users|admins|clients|custom
	AvailableCollections   []string               `yaml:"available_collections"`   // manually configured
	AuthToken              string                 `yaml:"auth_token"`              // Session token
	AuthExpires            *time.Time             `yaml:"auth_expires"`            // Token expiration
	AuthRecord             map[string]interface{} `yaml:"auth_record"`             // Cached auth record
}

// Output format constants
const (
	OutputFormatJSON  = "json"
	OutputFormatYAML  = "yaml"
	OutputFormatTable = "table"
)

// PocketBase auth collection constants (common ones, but allows custom)
const (
	AuthCollectionUsers   = "users"
	AuthCollectionAdmins  = "admins"
	AuthCollectionClients = "clients"
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

// ValidateCollectionName validates a collection name format
func ValidateCollectionName(collection string) error {
	if collection == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	
	// Basic validation - PocketBase will handle the actual validation
	if len(collection) < 1 || len(collection) > 50 {
		return fmt.Errorf("collection name must be between 1 and 50 characters")
	}
	
	return nil
}

// ValidateCollections validates a list of collection names
func ValidateCollections(collections []string) error {
	if len(collections) == 0 {
		return fmt.Errorf("at least one collection must be specified")
	}
	
	for _, collection := range collections {
		if err := ValidateCollectionName(collection); err != nil {
			return fmt.Errorf("invalid collection '%s': %w", collection, err)
		}
	}
	
	return nil
}

// IsCollectionAvailable checks if a collection is available in the context
func (c *Context) IsCollectionAvailable(collection string) bool {
	for _, available := range c.PocketBase.AvailableCollections {
		if available == collection {
			return true
		}
	}
	return false
}

// AddCollection adds a collection to the context's available collections
func (c *Context) AddCollection(collection string) error {
	if err := ValidateCollectionName(collection); err != nil {
		return err
	}
	
	// Check if already exists
	if c.IsCollectionAvailable(collection) {
		return fmt.Errorf("collection '%s' already exists in context", collection)
	}
	
	c.PocketBase.AvailableCollections = append(c.PocketBase.AvailableCollections, collection)
	return nil
}

// RemoveCollection removes a collection from the context's available collections
func (c *Context) RemoveCollection(collection string) error {
	for i, available := range c.PocketBase.AvailableCollections {
		if available == collection {
			// Remove from slice
			c.PocketBase.AvailableCollections = append(
				c.PocketBase.AvailableCollections[:i],
				c.PocketBase.AvailableCollections[i+1:]...,
			)
			return nil
		}
	}
	return fmt.Errorf("collection '%s' not found in context", collection)
}

// GetCollectionValidationError returns a helpful error message for invalid collections
func (c *Context) GetCollectionValidationError(collection string) error {
	available := c.PocketBase.AvailableCollections
	if len(available) == 0 {
		return fmt.Errorf("collection '%s' not configured in context. No collections available. Add collections with 'pb context collections add %s'", 
			collection, collection)
	}
	
	return fmt.Errorf("collection '%s' not configured in context. Available collections: %s. Add with 'pb context collections add %s'", 
		collection, strings.Join(available, ", "), collection)
}

// Global configuration instance (will be populated by root command)
var Global = &GlobalConfig{
	OutputFormat:   OutputFormatJSON,
	ColorsEnabled:  true,
	PaginationSize: 30,
	Debug:          false,
}
