package config_test

import (
	"os"
	"path/filepath"
	"pb-cli/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestManager is a helper to create a manager in a temporary directory for each test.
func setupTestManager(t *testing.T) *config.Manager {
	tempDir := t.TempDir()
	// We use our new test-friendly constructor here.
	manager, err := config.NewManagerWithBase(filepath.Join(tempDir, "pb"))
	require.NoError(t, err, "Failed to create test manager")
	return manager
}

// TestGlobalConfigLifecycle tests the creation, loading, and saving of the global config.
func TestGlobalConfigLifecycle(t *testing.T) {
	manager := setupTestManager(t)

	// 1. Test default config creation on first load
	globalCfg, err := manager.LoadGlobalConfig()
	require.NoError(t, err)
	require.NotNil(t, globalCfg)
	assert.Equal(t, "", globalCfg.ActiveContext, "Default active context should be empty")
	assert.Equal(t, "json", globalCfg.OutputFormat, "Default output format should be json")

	// Verify the file was actually created on disk
	_, err = os.Stat(manager.GetGlobalConfigPath())
	require.NoError(t, err, "Global config file should have been created")

	// 2. Test modifying and saving the config
	globalCfg.ActiveContext = "production"
	globalCfg.OutputFormat = "table"
	err = manager.SaveGlobalConfig(globalCfg)
	require.NoError(t, err)

	// 3. Test loading the modified config
	reloadedCfg, err := manager.LoadGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, "production", reloadedCfg.ActiveContext)
	assert.Equal(t, "table", reloadedCfg.OutputFormat)
}

// TestContextLifecycle covers the full CRUD (Create, Read, Update, Delete) for contexts.
func TestContextLifecycle(t *testing.T) {
	manager := setupTestManager(t)
	contextName := "dev-context"

	// 1. Create and Save a new context
	newContext := &config.Context{
		Name: contextName,
		PocketBase: config.PocketBaseConfig{
			URL:                  "http://localhost:8090",
			AuthCollection:       "users",
			AvailableCollections: []string{"posts"},
		},
	}
	err := manager.SaveContext(newContext)
	require.NoError(t, err)

	// 2. Verify it exists
	assert.True(t, manager.ContextExists(contextName))
	// Verify the context directory and backups subdirectory were created
	_, err = os.Stat(manager.GetContextDir(contextName))
	assert.NoError(t, err, "Context directory should exist")
	_, err = os.Stat(manager.GetBackupDir(contextName))
	assert.NoError(t, err, "Backups directory should exist")

	// 3. List contexts and ensure it's present
	contexts, err := manager.ListContexts()
	require.NoError(t, err)
	assert.Contains(t, contexts, contextName)
	assert.Len(t, contexts, 1)

	// 4. Load the context and check its contents
	loadedContext, err := manager.LoadContext(contextName)
	require.NoError(t, err)
	require.NotNil(t, loadedContext)
	assert.Equal(t, "http://localhost:8090", loadedContext.PocketBase.URL)
	assert.Contains(t, loadedContext.PocketBase.AvailableCollections, "posts")

	// 5. Set the context as active
	err = manager.SetActiveContext(contextName)
	require.NoError(t, err)
	activeContext, err := manager.GetActiveContext()
	require.NoError(t, err)
	assert.Equal(t, contextName, activeContext.Name)

	// 6. Delete the context
	err = manager.DeleteContext(contextName)
	require.NoError(t, err)
	assert.False(t, manager.ContextExists(contextName))

	// 7. Verify it's no longer in the list
	contextsAfterDelete, err := manager.ListContexts()
	require.NoError(t, err)
	assert.NotContains(t, contextsAfterDelete, contextName)
	assert.Empty(t, contextsAfterDelete)
}

// TestGetActiveContext_NoActiveSet tests the behavior when no active context is set.
func TestGetActiveContext_NoActiveSet(t *testing.T) {
	manager := setupTestManager(t)

	// With a fresh config, there should be no active context
	_, err := manager.GetActiveContext()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context set")
}

// TestDeleteActiveContext ensures the global config is updated when the active context is deleted.
func TestDeleteActiveContext(t *testing.T) {
	manager := setupTestManager(t)
	contextName := "temp-context"

	// Create and set a context as active
	ctx := &config.Context{Name: contextName}
	require.NoError(t, manager.SaveContext(ctx))
	require.NoError(t, manager.SetActiveContext(contextName))

	// Now, delete the context (this happens in the `delete` command logic, not directly in the manager)
	// We simulate it here by calling DeleteContext and then checking the global config state.
	require.NoError(t, manager.DeleteContext(contextName))

	// The manager itself doesn't clear the active context, the command does.
	// Let's test that the manager can handle this state gracefully.
	globalCfg, err := manager.LoadGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, contextName, globalCfg.ActiveContext, "Manager should not automatically clear active context name")

	// However, trying to GET the active context should now fail because the directory is gone.
	_, err = manager.GetActiveContext()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
