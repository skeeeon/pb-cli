package context

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var selectCmd = &cobra.Command{
	Use:   "select <n>",
	Short: "Set the active PocketBase context",
	Long: `Set the active context for PocketBase operations.

The active context determines which PocketBase instance and collection settings
are used for all pb commands.

Examples:
  pb context select production
  pb context select development
  pb con sel prod  # Using partial matching`,
	Aliases: []string{"use", "switch"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		contextName := args[0]

		// Verify the context exists
		ctx, err := configManager.LoadContext(contextName)
		if err != nil {
			// Try to provide helpful suggestions
			contexts, listErr := configManager.ListContexts()
			if listErr == nil && len(contexts) > 0 {
				return fmt.Errorf("context '%s' not found. Available contexts: %v", 
					contextName, contexts)
			}
			return fmt.Errorf("context '%s' not found", contextName)
		}

		// Set as active context
		if err := configManager.SetActiveContext(contextName); err != nil {
			return fmt.Errorf("failed to set active context: %w", err)
		}

		// Success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		
		fmt.Printf("%s Context switched to '%s'\n", 
			green("✓"), cyan(contextName))

		// Show context details
		contextDir := configManager.GetContextDir(contextName)
		fmt.Printf("\nContext Details:\n")
		fmt.Printf("  Directory: %s\n", contextDir)
		fmt.Printf("  PocketBase URL: %s\n", ctx.PocketBase.URL)
		fmt.Printf("  Auth Collection: %s\n", ctx.PocketBase.AuthCollection)
		
		if len(ctx.PocketBase.AvailableCollections) > 0 {
			fmt.Printf("  Collections: %v\n", ctx.PocketBase.AvailableCollections)
		} else {
			fmt.Printf("  Collections: %s\n", 
				color.New(color.FgYellow).Sprint("None configured"))
		}

		// Authentication status
		if ctx.PocketBase.AuthToken != "" {
			if ctx.PocketBase.AuthExpires != nil {
				fmt.Printf("  Authentication: %s (expires %s)\n", 
					green("Valid"), 
					ctx.PocketBase.AuthExpires.Format("2006-01-02 15:04"))
			} else {
				fmt.Printf("  Authentication: %s\n", green("Valid"))
			}
		} else {
			fmt.Printf("  Authentication: %s\n", 
				color.New(color.FgYellow).Sprint("Required"))
			
			// Suggest authentication
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  Authenticate with PocketBase: %s\n", 
				color.New(color.FgCyan).Sprint("pb auth"))
		}

		// Show collection management suggestions if no collections
		if len(ctx.PocketBase.AvailableCollections) == 0 {
			fmt.Printf("  Configure collections: %s\n", 
				color.New(color.FgCyan).Sprint("pb context collections add <collection_names>"))
		}

		return nil
	},
}
