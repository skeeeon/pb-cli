package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"pb-cli/cmd/auth"
	"pb-cli/cmd/backup"
	"pb-cli/cmd/collections"
	"pb-cli/cmd/context"
	"pb-cli/internal/config"
)

var (
	configManager *config.Manager
	
	// Global flags
	globalOutputFormat string
	globalColorsEnabled bool
	globalDebug bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pb",
	Short: "Generic CLI client for PocketBase",
	Long: `pb is a command-line interface for managing PocketBase instances.
It provides comprehensive tools for managing contexts, authenticating with PocketBase,
and performing CRUD operations on collections.

Features:
- Multi-environment context management
- PocketBase authentication with multiple collection support
- Generic CRUD operations on any collection
- Backup management (requires admin authentication)`,
	Version: "0.1.0",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show usage instead of full help when no subcommand provided
		return fmt.Errorf("missing subcommand. See 'pb --help' for available commands")
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration manager
		var err error
		configManager, err = config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize configuration: %w", err)
		}

		// Load global configuration from file
		globalConfig, err := configManager.LoadGlobalConfig()
		if err != nil {
			// If we can't load global config, use defaults but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to load global config, using defaults: %v\n", err)
			globalConfig = &config.GlobalConfig{
				OutputFormat:   "json",
				ColorsEnabled:  true,
				PaginationSize: 30,
				Debug:          false,
			}
		}

		// Apply global config to config.Global, but allow command-line flags to override
		if !cmd.Flags().Changed("output") {
			config.Global.OutputFormat = globalConfig.OutputFormat
		} else {
			config.Global.OutputFormat = globalOutputFormat
		}

		if !cmd.Flags().Changed("colors") {
			config.Global.ColorsEnabled = globalConfig.ColorsEnabled
		} else {
			config.Global.ColorsEnabled = globalColorsEnabled
		}

		if !cmd.Flags().Changed("debug") {
			config.Global.Debug = globalConfig.Debug
		} else {
			config.Global.Debug = globalDebug
		}

		// Apply pagination size (no command line flag for this)
		config.Global.PaginationSize = globalConfig.PaginationSize

		// Pass config manager to command groups
		context.SetConfigManager(configManager)
		auth.SetConfigManager(configManager)
		backup.SetConfigManager(configManager)
		collections.SetConfigManager(configManager)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags with proper variable binding
	rootCmd.PersistentFlags().StringVarP(&globalOutputFormat, "output", "o", "json", "Output format (json|yaml|table)")
	rootCmd.PersistentFlags().BoolVar(&globalColorsEnabled, "colors", true, "Enable colored output")
	rootCmd.PersistentFlags().BoolVar(&globalDebug, "debug", false, "Enable debug output")

	// Bind flags to viper for config file support
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("colors", rootCmd.PersistentFlags().Lookup("colors"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Add command groups
	addCommands()
}

// addCommands adds all command groups to the root command
func addCommands() {
	// Context management commands
	rootCmd.AddCommand(context.ContextCmd)
	
	// Authentication commands
	rootCmd.AddCommand(auth.AuthCmd)
	
	// Backup management commands
	rootCmd.AddCommand(backup.BackupCmd)
	
	// Collections CRUD commands
	rootCmd.AddCommand(collections.CollectionsCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set config file type
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	
	// Look for config in XDG config directory
	if configManager != nil {
		viper.AddConfigPath(configManager.GetConfigDir())
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("PB")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil && config.Global.Debug {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// GetConfigManager returns the global configuration manager instance
func GetConfigManager() *config.Manager {
	return configManager
}
