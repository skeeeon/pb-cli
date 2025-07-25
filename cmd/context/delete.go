package context

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var forceDelete bool

var deleteCmd = &cobra.Command{
	Use:   "delete <n>",
	Short: "Delete a PocketBase context",
	Long: `Delete a context configuration from the system.

This action is irreversible and will remove the entire context directory
including the context configuration and any context-specific files.

If the context being deleted is currently active, you will need to
select a different context or create a new one.

Examples:
  pb context delete development
  pb context delete old-prod --force  # Skip confirmation prompt
  pb con del dev`,
	Aliases: []string{"remove", "rm"},
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

		// Check if it's the active context
		globalConfig, err := configManager.LoadGlobalConfig()
		if err != nil {
			return fmt.Errorf("failed to load global config: %w", err)
		}

		isActive := globalConfig.ActiveContext == contextName

		// Show context details before deletion
		red := color.New(color.FgRed).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()

		contextDir := configManager.GetContextDir(contextName)

		fmt.Printf("%s Context to be deleted: %s\n", 
			red("⚠"), bold(contextName))
		fmt.Printf("  Directory: %s\n", contextDir)
		fmt.Printf("  PocketBase URL: %s\n", ctx.PocketBase.URL)
		fmt.Printf("  Auth Collection: %s\n", ctx.PocketBase.AuthCollection)
		
		if len(ctx.PocketBase.AvailableCollections) > 0 {
			fmt.Printf("  Collections: %v\n", ctx.PocketBase.AvailableCollections)
		} else {
			fmt.Printf("  Collections: None configured\n")
		}

		if isActive {
			fmt.Printf("  Status: %s\n", red("CURRENTLY ACTIVE"))
		}

		// Confirmation prompt (unless --force is used)
		if !forceDelete {
			fmt.Printf("\n%s This will permanently delete the entire context directory and all its contents.\n", 
				yellow("Warning:"))
			fmt.Print("Are you sure you want to delete this context? (y/N): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Context deletion cancelled.")
				return nil
			}
		}

		// Delete the context (removes entire directory)
		if err := configManager.DeleteContext(contextName); err != nil {
			return fmt.Errorf("failed to delete context: %w", err)
		}

		// If this was the active context, clear the active context
		if isActive {
			globalConfig.ActiveContext = ""
			if err := configManager.SaveGlobalConfig(globalConfig); err != nil {
				fmt.Printf("%s Context deleted but failed to clear active context: %v\n", 
					yellow("Warning:"), err)
			}
		}

		// Success message
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Context '%s' and its directory deleted successfully\n", 
			green("✓"), contextName)

		// Show next steps if needed
		if isActive {
			contexts, err := configManager.ListContexts()
			if err == nil && len(contexts) > 0 {
				fmt.Printf("\nSelect a new active context:\n")
				for _, ctx := range contexts {
					fmt.Printf("  %s\n", 
						color.New(color.FgCyan).Sprintf("pb context select %s", ctx))
				}
			} else {
				fmt.Printf("\nCreate a new context:\n")
				fmt.Printf("  %s\n", 
					color.New(color.FgCyan).Sprint("pb context create <n> --url <url> --collections <collections>"))
			}
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, 
		"Skip confirmation prompt")
}
