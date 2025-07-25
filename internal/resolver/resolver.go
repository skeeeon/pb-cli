package resolver

import (
	"fmt"
	"sort"
	"strings"
)

// CommandResolver handles partial command matching for Cisco-style CLI
type CommandResolver struct {
	commands map[string][]string // category -> list of commands
}

// NewCommandResolver creates a new command resolver
func NewCommandResolver() *CommandResolver {
	resolver := &CommandResolver{
		commands: make(map[string][]string),
	}

	// Initialize with generic PocketBase commands
	resolver.initializeCommands()
	
	return resolver
}

// initializeCommands sets up the available commands for each category
func (r *CommandResolver) initializeCommands() {
	// Root level commands
	r.commands["root"] = []string{
		"context",
		"collections", 
		"auth",
		"version",
		"help",
	}

	// Context subcommands
	r.commands["context"] = []string{
		"create",
		"list",
		"select",
		"show",
		"delete",
		"collections", // For managing collections in context
	}

	// Collections subcommands (actions - collection names are validated separately)
	r.commands["collections"] = []string{
		"list",
		"get",
		"create",
		"update",
		"delete",
	}

	// Auth subcommands
	r.commands["auth"] = []string{
		"pb",    // PocketBase auth (main command)
	}

	// Context collections subcommands
	r.commands["context_collections"] = []string{
		"add",
		"remove",
		"list",
		"clear",
	}
}

// ResolveCommand resolves a partial command to its full form
// Returns the resolved command or an error if ambiguous or not found
func (r *CommandResolver) ResolveCommand(category, partial string) (string, error) {
	commands, exists := r.commands[category]
	if !exists {
		return "", fmt.Errorf("unknown command category: %s", category)
	}

	if partial == "" {
		return "", fmt.Errorf("empty command")
	}

	// Find all matching commands
	var matches []string
	partial = strings.ToLower(partial)

	for _, cmd := range commands {
		if strings.HasPrefix(strings.ToLower(cmd), partial) {
			matches = append(matches, cmd)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("unknown command '%s'. Available commands: %s", 
			partial, strings.Join(commands, ", "))
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("ambiguous command '%s'. Possible matches: %s", 
			partial, strings.Join(matches, ", "))
	}
}

// GetCommands returns all available commands for a category
func (r *CommandResolver) GetCommands(category string) []string {
	commands, exists := r.commands[category]
	if !exists {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]string, len(commands))
	copy(result, commands)
	return result
}

// AddCommand adds a new command to a category
func (r *CommandResolver) AddCommand(category, command string) {
	if r.commands[category] == nil {
		r.commands[category] = make([]string, 0)
	}
	
	// Avoid duplicates
	for _, existing := range r.commands[category] {
		if existing == command {
			return
		}
	}
	
	r.commands[category] = append(r.commands[category], command)
}

// ValidateCommand checks if a command exists in the category (exact match)
func (r *CommandResolver) ValidateCommand(category, command string) bool {
	commands, exists := r.commands[category]
	if !exists {
		return false
	}

	command = strings.ToLower(command)
	for _, cmd := range commands {
		if strings.ToLower(cmd) == command {
			return true
		}
	}

	return false
}

// ValidateCollection checks if a collection name is valid (exact match required)
func (r *CommandResolver) ValidateCollection(collection string, availableCollections []string) error {
	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	// Check against available collections from context
	for _, available := range availableCollections {
		if collection == available {
			return nil
		}
	}

	// Collection not found - provide helpful error with suggestions
	if len(availableCollections) > 0 {
		return fmt.Errorf("collection '%s' not configured in context. Available collections: %s. Add with 'pb context collections add %s'", 
			collection, strings.Join(availableCollections, ", "), collection)
	}

	return fmt.Errorf("collection '%s' not found. No collections configured in context. Add with 'pb context collections add %s'", collection, collection)
}

// GetMinimumPrefix returns the minimum unambiguous prefix for a command
func (r *CommandResolver) GetMinimumPrefix(category, command string) (string, error) {
	commands, exists := r.commands[category]
	if !exists {
		return "", fmt.Errorf("unknown command category: %s", category)
	}

	command = strings.ToLower(command)
	var targetCmd string
	
	// Find the target command
	for _, cmd := range commands {
		if strings.ToLower(cmd) == command {
			targetCmd = cmd
			break
		}
	}
	
	if targetCmd == "" {
		return "", fmt.Errorf("command '%s' not found in category '%s'", command, category)
	}

	// Find minimum prefix that's unambiguous
	for i := 1; i <= len(targetCmd); i++ {
		prefix := strings.ToLower(targetCmd[:i])
		matchCount := 0
		
		for _, cmd := range commands {
			if strings.HasPrefix(strings.ToLower(cmd), prefix) {
				matchCount++
			}
		}
		
		if matchCount == 1 {
			return targetCmd[:i], nil
		}
	}

	// If we get here, return the full command (shouldn't happen)
	return targetCmd, nil
}

// SuggestCommands provides command suggestions based on partial input
func (r *CommandResolver) SuggestCommands(category, partial string) []string {
	commands, exists := r.commands[category]
	if !exists {
		return nil
	}

	if partial == "" {
		return commands
	}

	var suggestions []string
	partial = strings.ToLower(partial)

	for _, cmd := range commands {
		if strings.HasPrefix(strings.ToLower(cmd), partial) {
			suggestions = append(suggestions, cmd)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}
