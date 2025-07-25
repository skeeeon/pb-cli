package context

import (
	"fmt"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
)

// ContextCmd represents the context command
var ContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage PocketBase environment contexts",
	Long: `Context management allows you to work with multiple PocketBase environments.
Each context contains PocketBase configuration and collection settings.

Each context is stored in its own directory within the pb configuration directory,
containing the context configuration file.

Context Directory Structure:
  ~/.config/pb/
  ├── config.yaml           # Global configuration
  ├── production/           # Production context directory
  │   └── context.yaml     # Context configuration
  ├── development/          # Development context directory
  │   └── context.yaml
  └── staging/              # Staging context directory
      └── context.yaml

Examples:
  pb context create production --url https://api.example.com --collections posts,users
  pb context select production
  pb context collections add comments categories
  pb context list
  pb context show production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show usage instead of full help when no subcommand provided
		return fmt.Errorf("missing subcommand. See 'pb context --help' for available commands")
	},
}

var configManager *config.Manager

func init() {
	// Add subcommands
	ContextCmd.AddCommand(createCmd)
	ContextCmd.AddCommand(listCmd)
	ContextCmd.AddCommand(selectCmd)
	ContextCmd.AddCommand(showCmd)
	ContextCmd.AddCommand(deleteCmd)
	ContextCmd.AddCommand(collectionsCmd)
}

// SetConfigManager sets the configuration manager for the context commands
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// validateConfigManager ensures the config manager is available
func validateConfigManager() error {
	if configManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return nil
}
