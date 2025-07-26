package backup

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var createCmd = &cobra.Command{
	Use:   "create [--name <name>]",
	Short: "Create a new backup",
	Long: `Create a new backup of the PocketBase database.

If no name is specified, PocketBase will generate one automatically
based on the current timestamp.

Note: Creating backups requires admin authentication.

Examples:
  pb backup create                        # Auto-generated name
  pb backup create --name "pre-update"    # Custom name
  pb backup create --name "backup-$(date +%Y%m%d)"  # With shell substitution`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Create PocketBase client
		client := pocketbase.NewClientFromContext(ctx)

		// Display what we're about to do
		if nameFlag != "" {
			utils.PrintInfo(fmt.Sprintf("Creating backup with name: %s", nameFlag))
		} else {
			utils.PrintInfo("Creating backup with auto-generated name...")
		}

		// Create the backup
		backup, err := client.CreateBackup(nameFlag)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to create backup")
			}
			return fmt.Errorf("failed to create backup: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		
		fmt.Printf("%s Backup created successfully!\n", green("✓"))
		
		// Show backup details if available
		if backup != nil {
			fmt.Printf("\nBackup Details:\n")
			fmt.Printf("  Name: %s\n", backup.Key)
			fmt.Printf("  Size: %s\n", backup.GetHumanSize())
			fmt.Printf("  Created: %s\n", backup.GetFormattedDate())
			fmt.Printf("  Context: %s\n", cyan(ctx.Name))

			// Show next steps
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  Download backup: %s\n", 
				cyan(fmt.Sprintf("pb backup download %s", backup.Key)))
			fmt.Printf("  List all backups: %s\n", 
				cyan("pb backup list"))
		} else {
			// Fallback message when we can't get backup details
			fmt.Printf("  Context: %s\n", cyan(ctx.Name))
			fmt.Printf("\nNote: Backup was created successfully, but details are not immediately available.\n")
			fmt.Printf("You can view all backups with: %s\n", 
				cyan("pb backup list"))
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&nameFlag, "name", "", "Custom backup name (optional)")
}
