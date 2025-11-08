package resolver_test

import (
	"pb-cli/internal/resolver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCommandResolver ensures that the constructor initializes the resolver correctly.
func TestNewCommandResolver(t *testing.T) {
	r := resolver.NewCommandResolver()

	// The resolver and its internal map should not be nil.
	require.NotNil(t, r)

	// Check if a known command category was initialized.
	contextCommands := r.GetCommands("context")
	assert.NotEmpty(t, contextCommands, "Expected 'context' commands to be initialized")
	assert.Contains(t, contextCommands, "create")
}

// TestResolveCommand covers various scenarios for resolving partial command inputs.
func TestResolveCommand(t *testing.T) {
	r := resolver.NewCommandResolver()

	// Define test cases in a table-driven format for clarity and maintainability.
	testCases := []struct {
		name        string
		category    string
		partial     string
		expectedCmd string
		expectErr   bool
		errContains string
	}{
		{
			name:        "Success - Unambiguous Prefix",
			category:    "context",
			partial:     "cre",
			expectedCmd: "create",
			expectErr:   false,
		},
		{
			name:        "Success - Full Command",
			category:    "collections",
			partial:     "delete",
			expectedCmd: "delete",
			expectErr:   false,
		},
		{
			name:        "Success - Case Insensitive",
			category:    "context",
			partial:     "SELECT",
			expectedCmd: "select",
			expectErr:   false,
		},
		{
			name:        "Failure - Ambiguous Prefix",
			category:    "context",
			partial:     "c",
			expectErr:   true,
			errContains: "ambiguous command 'c'. Possible matches: collections, create",
		},
		{
			name:        "Failure - No Match",
			category:    "collections",
			partial:     "find",
			expectErr:   true,
			errContains: "unknown command 'find'",
		},
		{
			name:        "Failure - Unknown Category",
			category:    "nonexistent",
			partial:     "list",
			expectErr:   true,
			errContains: "unknown command category: nonexistent",
		},
		{
			name:        "Failure - Empty Input",
			category:    "context",
			partial:     "",
			expectErr:   true,
			errContains: "empty command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedCmd, err := r.ResolveCommand(tc.category, tc.partial)

			if tc.expectErr {
				require.Error(t, err, "Expected an error but got none")
				assert.Contains(t, err.Error(), tc.errContains, "Error message did not contain expected text")
			} else {
				require.NoError(t, err, "Expected no error but got one")
				assert.Equal(t, tc.expectedCmd, resolvedCmd, "Resolved command does not match expected value")
			}
		})
	}
}

// TestValidateCollection tests the logic for validating a collection against a list of available collections.
func TestValidateCollection(t *testing.T) {
	r := resolver.NewCommandResolver()
	availableCollections := []string{"posts", "users", "user_profiles"}

	testCases := []struct {
		name                 string
		collectionToValidate string
		availableCollections []string
		expectErr            bool
		errContains          string
	}{
		{
			name:                 "Success - Collection Exists",
			collectionToValidate: "posts",
			availableCollections: availableCollections,
			expectErr:            false,
		},
		{
			name:                 "Failure - Collection Does Not Exist",
			collectionToValidate: "comments",
			availableCollections: availableCollections,
			expectErr:            true,
			errContains:          "collection 'comments' not configured in context. Available collections: posts, users, user_profiles",
		},
		{
			name:                 "Failure - No Collections Configured",
			collectionToValidate: "posts",
			availableCollections: []string{},
			expectErr:            true,
			errContains:          "No collections configured in context",
		},
		{
			name:                 "Failure - Empty Collection Name",
			collectionToValidate: "",
			availableCollections: availableCollections,
			expectErr:            true,
			errContains:          "collection name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := r.ValidateCollection(tc.collectionToValidate, tc.availableCollections)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetCommands verifies that the correct list of commands is returned for a category.
func TestGetCommands(t *testing.T) {
	r := resolver.NewCommandResolver()

	// Test a known category
	collectionsCmds := r.GetCommands("collections")
	expectedCmds := []string{"list", "get", "create", "update", "delete"}
	assert.ElementsMatch(t, expectedCmds, collectionsCmds, "Should return the correct commands for 'collections'")

	// Test an unknown category
	unknownCmds := r.GetCommands("nonexistent")
	assert.Nil(t, unknownCmds, "Should return nil for an unknown category")
}

// TestGetMinimumPrefix checks the logic for finding the shortest unambiguous command prefix.
func TestGetMinimumPrefix(t *testing.T) {
	r := resolver.NewCommandResolver()

	// In `collections`: list, get, create, update, delete
	// "d" is unique for "delete"
	prefix, err := r.GetMinimumPrefix("collections", "delete")
	require.NoError(t, err)
	assert.Equal(t, "d", prefix)

	// In `context`: create, list, select, show, delete, collections
	// "s" is ambiguous (select, show), so "se" for "select" and "sh" for "show"
	prefix, err = r.GetMinimumPrefix("context", "select")
	require.NoError(t, err)
	assert.Equal(t, "se", prefix)

	prefix, err = r.GetMinimumPrefix("context", "show")
	require.NoError(t, err)
	assert.Equal(t, "sh", prefix)

	// "c" is ambiguous (create, collections), "cr" for "create", "co" for "collections"
	prefix, err = r.GetMinimumPrefix("context", "create")
	require.NoError(t, err)
	assert.Equal(t, "cr", prefix) // "cr" is unique

	// Test for a command that doesn't exist
	_, err = r.GetMinimumPrefix("context", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command 'nonexistent' not found")
}
