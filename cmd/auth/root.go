package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
)

// AuthCmd represents the auth command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with PocketBase",
	Long: `Authentication commands for PocketBase instances.

PocketBase supports authentication against different collections depending on your
application's setup. Common collections include:
- users: Regular user accounts (default)
- admins: Administrative accounts  
- clients: API client accounts
- <custom>: Any custom authentication collection

The authentication will validate your credentials and store a session token
in your active context for subsequent operations.

Examples:
  # Authenticate with default users collection
  pb auth

  # Authenticate with specific collection
  pb auth --collection admins

  # Authenticate with custom collection
  pb auth --collection customers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to pb subcommand for backward compatibility
		return pbCmd.RunE(cmd, args)
	},
}

var configManager *config.Manager

func init() {
	// Add main authentication command (PocketBase only now)
	AuthCmd.AddCommand(pbCmd)
	
	// Add flags from pb command to the root auth command for convenience
	AuthCmd.Flags().StringVarP(&pbEmail, "email", "e", "", "Email address for authentication")
	AuthCmd.Flags().StringVarP(&pbPassword, "password", "p", "", "Password for authentication (will prompt if not provided)")
	AuthCmd.Flags().StringVarP(&pbCollection, "collection", "c", "users", "Authentication collection")
}

// SetConfigManager sets the configuration manager for the auth commands
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

// validateActiveContext ensures there's an active context
func validateActiveContext() (*config.Context, error) {
	if err := validateConfigManager(); err != nil {
		return nil, err
	}

	ctx, err := configManager.GetActiveContext()
	if err != nil {
		return nil, fmt.Errorf("no active context set. Use 'pb context select <name>' to set one")
	}

	return ctx, nil
}
