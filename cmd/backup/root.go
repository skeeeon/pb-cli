package backup

import (
	"fmt"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
)

var (
	outputFlag string
	forceFlag  bool
	nameFlag   string
)

// BackupCmd represents the backup command
var BackupCmd = &cobra.Command{
	Use:     "backup",
	Aliases: []string{"b"},
	Short:   "Manage PocketBase backups",
	Long: `Manage PocketBase database backups.

This command provides comprehensive backup management including creating,
downloading, uploading, listing, and restoring PocketBase backups.

Backup files are stored in your context directory by default:
  ~/.config/pb/<context>/backups/

Note: Backup operations typically require admin authentication.

Examples:
  pb backup list                        # List all available backups
  pb backup create                      # Create a new backup
  pb backup create --name "pre-update"  # Create backup with custom name
  pb backup download backup_2024_01_15  # Download to context folder
  pb backup restore backup_2024_01_15   # Restore from backup
  pb backup delete old_backup           # Delete backup (with confirmation)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("missing subcommand. Available: list, create, download, upload, delete, restore")
	},
}

var configManager *config.Manager

func init() {
	// Add subcommands
	BackupCmd.AddCommand(listCmd)
	BackupCmd.AddCommand(createCmd)
	BackupCmd.AddCommand(downloadCmd)
	BackupCmd.AddCommand(uploadCmd)
	BackupCmd.AddCommand(deleteCmd)
	BackupCmd.AddCommand(restoreCmd)

	// Global flags. Output defaults to empty so it falls back to the global
	// (or root --output) format; pass -o table for the human-readable view.
	BackupCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format (json|yaml|table)")
	BackupCmd.PersistentFlags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompts")
}

// SetConfigManager sets the configuration manager for the backup commands
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// getOutputFormat returns the effective output format for backup commands.
func getOutputFormat() string {
	if outputFlag != "" {
		return outputFlag
	}
	return config.Global.OutputFormat
}

// validateConfigManager ensures the config manager is available
func validateConfigManager() error {
	if configManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return nil
}

// validateActiveContext ensures there's an active context with authentication
func validateActiveContext() (*config.Context, error) {
	if err := validateConfigManager(); err != nil {
		return nil, err
	}

	ctx, err := configManager.GetActiveContext()
	if err != nil {
		return nil, fmt.Errorf("no active context set. Use 'pb context select <name>' to set one")
	}

	// Check authentication
	if ctx.PocketBase.AuthToken == "" {
		return nil, fmt.Errorf("authentication required. Run 'pb auth' to authenticate")
	}

	if err := pocketbase.EnsureFreshAuth(ctx, configManager); err != nil {
		return nil, err
	}

	if !pocketbase.IsAuthValid(ctx) {
		return nil, fmt.Errorf("authentication has expired. Run 'pb auth' to re-authenticate")
	}

	return ctx, nil
}

// getBackupDir returns the backup directory for the current context
func getBackupDir(ctx *config.Context) string {
	return configManager.GetBackupDir(ctx.Name)
}
