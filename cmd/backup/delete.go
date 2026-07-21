package backup

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <backup_name>",
	Short: "Delete a backup",
	Long: `Delete a backup from PocketBase.

This permanently removes the backup file from the PocketBase instance.
This action cannot be undone.

Note: Deleting backups requires admin authentication.

Examples:
  pb backup delete backup_2024_01_15      # Delete with confirmation
  pb backup delete old_backup --force     # Delete without confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupName := args[0]

		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Create PocketBase client
		client := pocketbase.NewClientFromContext(ctx)

		// Get backup info first to show details and validate it exists
		utils.PrintInfo(fmt.Sprintf("Checking backup '%s'...", backupName))
		backup, err := client.GetBackup(backupName)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to get backup info")
			}
			return fmt.Errorf("failed to get backup info: %w", err)
		}

		// Show confirmation prompt (unless --force is used)
		if !forceFlag {
			confirmed, err := confirmDeletion(backup, ctx)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(os.Stderr, "Backup deletion cancelled.")
				return nil
			}
		}

		// Delete the backup
		utils.PrintInfo(fmt.Sprintf("Deleting backup '%s'...", backupName))

		err = client.DeleteBackup(backupName)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to delete backup")
			}
			return fmt.Errorf("failed to delete backup: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Printf("%s Backup deleted successfully!\n", green("✓"))
		fmt.Printf("  Backup: %s\n", backup.Key)
		fmt.Printf("  Size freed: %s\n", backup.GetHumanSize())
		fmt.Printf("  Context: %s\n", cyan(ctx.Name))

		// Show next steps
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  List remaining backups: %s\n",
			cyan("pb backup list"))
		fmt.Printf("  Create new backup: %s\n",
			cyan("pb backup create"))

		return nil
	},
}

// confirmDeletion shows backup details and prompts the user to confirm deletion.
// It returns true only when the user explicitly confirms.
func confirmDeletion(backup *pocketbase.Backup, ctx *config.Context) (bool, error) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Fprintf(os.Stderr, "%s Backup to be deleted:\n", red("⚠"))
	fmt.Fprintf(os.Stderr, "  Name: %s\n", bold(backup.Key))
	fmt.Fprintf(os.Stderr, "  Size: %s\n", backup.GetHumanSize())
	fmt.Fprintf(os.Stderr, "  Created: %s\n", backup.GetFormattedDate())
	fmt.Fprintf(os.Stderr, "  Age: %s\n", utils.FormatTimeAgo(backup.Modified.Time))
	fmt.Fprintf(os.Stderr, "  Context: %s\n", ctx.Name)

	fmt.Fprintf(os.Stderr, "\n%s This action cannot be undone.\n", yellow("Warning:"))
	return utils.Confirm("Are you sure you want to delete this backup? (y/N): ")
}
