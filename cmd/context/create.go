package context

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/utils"
)

var (
	pbURL           string
	pbAuthCollection string
	availableCollections []string
	pbAutoRefresh           bool
	pbAutoRefreshThreshold  string
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new PocketBase context",
	Long: `Create a new context configuration for a PocketBase environment.

A context contains all the connection information needed to work with a specific
PocketBase deployment including the server URL, authentication settings, and
available collections.

The context will be created as a directory containing:
- context.yaml: Main context configuration

Examples:
  pb context create production \\
    --url https://api.example.com \\
    --collections posts,comments,users

  pb context create development \\
    --url http://localhost:8090 \\
    --collections posts,users`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		contextName := args[0]
		if contextName == "" {
			return fmt.Errorf("context name cannot be empty")
		}

		// Validate required flags
		if pbURL == "" {
			return fmt.Errorf("--url is required")
		}
		if err := utils.ValidatePocketBaseURL(pbURL); err != nil {
			return fmt.Errorf("invalid --url: %w", err)
		}

		// Validate auth collection
		if pbAuthCollection != "" {
			if err := config.ValidateAuthCollection(pbAuthCollection); err != nil {
				return fmt.Errorf("invalid auth collection: %w", err)
			}
		} else {
			pbAuthCollection = config.AuthCollectionUsers // Default to users
		}

		// Validate collections if provided
		if len(availableCollections) > 0 {
			if err := config.ValidateCollections(availableCollections); err != nil {
				return fmt.Errorf("invalid collections: %w", err)
			}
		}

		// Validate auto-refresh threshold if provided
		if pbAutoRefreshThreshold != "" {
			d, err := time.ParseDuration(pbAutoRefreshThreshold)
			if err != nil {
				return fmt.Errorf("invalid --auto-refresh-threshold %q: %w (use e.g. '15m', '1h')",
					pbAutoRefreshThreshold, err)
			}
			if d <= 0 {
				return fmt.Errorf("--auto-refresh-threshold must be positive")
			}
		}

		// Check if context already exists
		if configManager.ContextExists(contextName) {
			return fmt.Errorf("context '%s' already exists", contextName)
		}

		// Create new context configuration
		newContext := &config.Context{
			Name: contextName,
			PocketBase: config.PocketBaseConfig{
				URL:                  pbURL,
				AuthCollection:       pbAuthCollection,
				AvailableCollections: availableCollections,
				AutoRefresh:          pbAutoRefresh,
				AutoRefreshThreshold: pbAutoRefreshThreshold,
			},
		}

		// Save the context (this will create the directory structure)
		if err := configManager.SaveContext(newContext); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		// Print success message
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Context '%s' created successfully\n", 
			green("✓"), contextName)

		// Show context directory information
		contextDir := configManager.GetContextDir(contextName)
		fmt.Printf("\nContext directory: %s\n", contextDir)

		// Show configuration summary
		fmt.Printf("\nContext Configuration:\n")
		fmt.Printf("  Name: %s\n", contextName)
		fmt.Printf("  PocketBase URL: %s\n", pbURL)
		fmt.Printf("  Auth Collection: %s\n", pbAuthCollection)
		if pbAutoRefresh {
			thresholdDisplay := pbAutoRefreshThreshold
			if thresholdDisplay == "" {
				thresholdDisplay = config.DefaultAutoRefreshThreshold.String() + " (default)"
			}
			fmt.Printf("  Auto-refresh: enabled (threshold: %s)\n", thresholdDisplay)
		}

		if len(availableCollections) > 0 {
			fmt.Printf("  Collections: %s\n", strings.Join(availableCollections, ", "))
		} else {
			fmt.Printf("  Collections: %s\n", color.New(color.FgYellow).Sprint("None configured"))
		}

		// Suggest next steps
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Select this context: %s\n", 
			color.New(color.FgCyan).Sprintf("pb context select %s", contextName))
		fmt.Printf("  2. Authenticate with PocketBase: %s\n", 
			color.New(color.FgCyan).Sprint("pb auth"))
		
		if len(availableCollections) == 0 {
			fmt.Printf("  3. Add collections: %s\n", 
				color.New(color.FgCyan).Sprint("pb context collections add <collection_names>"))
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&pbURL, "url", "", "PocketBase server URL (required)")
	createCmd.Flags().StringVar(&pbAuthCollection, "auth-collection", config.AuthCollectionUsers, 
		"PocketBase auth collection (users|admins|clients|custom)")
	createCmd.Flags().StringSliceVar(&availableCollections, "collections", nil,
		"Available collections (comma-separated)")
	createCmd.Flags().BoolVar(&pbAutoRefresh, "auto-refresh", false,
		"Automatically refresh the auth token when it's near expiry")
	createCmd.Flags().StringVar(&pbAutoRefreshThreshold, "auto-refresh-threshold", "",
		"Refresh when remaining lifetime falls below this duration (e.g. '15m', '1h'). Defaults to 15m")

	// Mark required flags
	createCmd.MarkFlagRequired("url")
}
