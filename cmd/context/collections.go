package context

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
)

var collectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "Manage collections in the active context",
	Long: `Manage the list of available collections for the active context.

Collections define which PocketBase collections can be accessed via the CLI.
You can add, remove, list, or clear collections for your current context.

Examples:
  pb context collections add posts comments users
  pb context collections remove comments  
  pb context collections list
  pb context collections clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show usage when no subcommand provided
		return fmt.Errorf("missing subcommand. Available: add, remove, list, clear")
	},
}

var addCollectionsCmd = &cobra.Command{
	Use:   "add <collections...>",
	Short: "Add collections to the active context",
	Long: `Add one or more collections to the active context's available collections list.

This allows you to perform CRUD operations on these collections using the
'pb collections' commands.

Examples:
  pb context collections add posts
  pb context collections add comments users categories
  pb context collections add blog_posts user_profiles`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		// Get active context
		ctx, err := configManager.GetActiveContext()
		if err != nil {
			return fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
		}

		// Validate collection names
		for _, collection := range args {
			if err := config.ValidateCollectionName(collection); err != nil {
				return fmt.Errorf("invalid collection name '%s': %w", collection, err)
			}
		}

		// Add collections to context
		var added []string
		var skipped []string

		for _, collection := range args {
			if ctx.IsCollectionAvailable(collection) {
				skipped = append(skipped, collection)
			} else {
				if err := ctx.AddCollection(collection); err != nil {
					return fmt.Errorf("failed to add collection '%s': %w", collection, err)
				}
				added = append(added, collection)
			}
		}

		// Save updated context
		if len(added) > 0 {
			if err := configManager.SaveContext(ctx); err != nil {
				return fmt.Errorf("failed to save context: %w", err)
			}
		}

		// Report results
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()

		if len(added) > 0 {
			fmt.Printf("%s Added collections: %s\n", 
				green("✓"), strings.Join(added, ", "))
		}

		if len(skipped) > 0 {
			fmt.Printf("%s Already configured: %s\n", 
				yellow("ℹ"), strings.Join(skipped, ", "))
		}

		if len(added) == 0 && len(skipped) > 0 {
			fmt.Printf("\nAll specified collections were already configured.\n")
		}

		if len(added) > 0 {
			fmt.Printf("\nYou can now use these collections with:\n")
			for _, collection := range added {
				fmt.Printf("  %s\n", 
					color.New(color.FgCyan).Sprintf("pb collections %s list", collection))
			}
		}

		return nil
	},
}

var removeCollectionsCmd = &cobra.Command{
	Use:   "remove <collection>",
	Short: "Remove a collection from the active context",
	Long: `Remove a collection from the active context's available collections list.

This will prevent CRUD operations on this collection until it's added back.

Examples:
  pb context collections remove comments
  pb context collections remove old_collection`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		collection := args[0]

		// Get active context
		ctx, err := configManager.GetActiveContext()
		if err != nil {
			return fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
		}

		// Remove collection from context
		if err := ctx.RemoveCollection(collection); err != nil {
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("%s Collection '%s' was not configured in this context\n", 
				yellow("ℹ"), collection)
			return nil
		}

		// Save updated context
		if err := configManager.SaveContext(ctx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		// Report success
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Removed collection: %s\n", green("✓"), collection)

		return nil
	},
}

var listCollectionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List collections in the active context",
	Long: `List all collections configured for the active context.

These are the collections you can perform CRUD operations on using the
'pb collections' commands.

Examples:
  pb context collections list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		// Get active context
		ctx, err := configManager.GetActiveContext()
		if err != nil {
			return fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
		}

		collections := ctx.PocketBase.AvailableCollections

		if len(collections) == 0 {
			fmt.Printf("No collections configured for context '%s'.\n", ctx.Name)
			fmt.Printf("\nAdd collections with:\n  %s\n", 
				color.New(color.FgCyan).Sprint("pb context collections add <collection_names>"))
			return nil
		}

		fmt.Printf("Collections configured for context '%s':\n", ctx.Name)
		for i, collection := range collections {
			fmt.Printf("  %d. %s\n", i+1, collection)
		}

		fmt.Printf("\nTotal: %d collection(s)\n", len(collections))

		fmt.Printf("\nUsage examples:\n")
		for _, collection := range collections[:min(3, len(collections))] {
			fmt.Printf("  %s\n", 
				color.New(color.FgCyan).Sprintf("pb collections %s list", collection))
		}

		return nil
	},
}

var clearCollectionsCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all collections from the active context",
	Long: `Remove all collections from the active context's available collections list.

This will prevent CRUD operations on all collections until they're added back.
Use with caution.

Examples:
  pb context collections clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		// Get active context
		ctx, err := configManager.GetActiveContext()
		if err != nil {
			return fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
		}

		if len(ctx.PocketBase.AvailableCollections) == 0 {
			fmt.Printf("No collections configured for context '%s'.\n", ctx.Name)
			return nil
		}

		// Clear collections
		ctx.PocketBase.AvailableCollections = []string{}

		// Save updated context
		if err := configManager.SaveContext(ctx); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		// Report success
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Cleared all collections from context '%s'\n", 
			green("✓"), ctx.Name)

		fmt.Printf("\nAdd collections with:\n  %s\n", 
			color.New(color.FgCyan).Sprint("pb context collections add <collection_names>"))

		return nil
	},
}

func init() {
	// Add subcommands to collections command
	collectionsCmd.AddCommand(addCollectionsCmd)
	collectionsCmd.AddCommand(removeCollectionsCmd)
	collectionsCmd.AddCommand(listCollectionsCmd)
	collectionsCmd.AddCommand(clearCollectionsCmd)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
