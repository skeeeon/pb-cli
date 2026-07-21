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

var restoreCmd = &cobra.Command{
	Use:   "restore <backup_name>",
	Short: "Restore from a backup",
	Long: `Restore the PocketBase database from a backup.

This operation will replace the current database with the backup data.
All current data will be lost and cannot be recovered unless you have
another backup.

Note: Restoring from backups requires admin authentication and will
restart the PocketBase instance.

Examples:
  pb backup restore backup_2024_01_15      # Restore with confirmation
  pb backup restore backup_2024_01_15 --force  # Restore without confirmation`,
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

		// Recommend creating a current backup before restore
		fmt.Printf("\n%s Consider creating a backup of the current state before proceeding:\n",
			color.New(color.FgYellow).Sprint("Recommendation:"))
		fmt.Printf("  %s\n", color.New(color.FgCyan).Sprintf("pb backup create --name \"pre-restore-$(date +%%Y%%m%%d-%%H%%M)\""))

		// Show confirmation prompt (unless --force is used)
		if !forceFlag {
			confirmed, err := confirmRestore(backup, ctx)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(os.Stderr, "Restore cancelled.")
				return nil
			}
		}

		// Restore from the backup
		utils.PrintInfo(fmt.Sprintf("Restoring from backup '%s'...", backupName))
		fmt.Printf("%s This may take several minutes and will restart PocketBase.\n",
			color.New(color.FgYellow).Sprint("Note:"))

		err = client.RestoreBackup(backupName)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to restore backup")
			}
			return fmt.Errorf("failed to restore backup: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()

		fmt.Printf("\n%s Backup restore initiated successfully!\n", green("✓"))
		fmt.Printf("\nRestore Details:\n")
		fmt.Printf("  Backup: %s\n", backup.Key)
		fmt.Printf("  Size: %s\n", backup.GetHumanSize())
		fmt.Printf("  Backup date: %s\n", backup.GetFormattedDate())
		fmt.Printf("  Context: %s\n", cyan(ctx.Name))

		// Important post-restore information
		fmt.Printf("\n%s Important Notes:\n", yellow("⚠"))
		fmt.Printf("  • PocketBase may be restarting - wait a moment before making requests\n")
		fmt.Printf("  • Your authentication token may be invalidated\n")
		fmt.Printf("  • All data has been replaced with the backup data\n")

		// Show next steps
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  Re-authenticate if needed: %s\n",
			cyan("pb auth"))
		fmt.Printf("  Verify restoration: %s\n",
			cyan("pb collections list <collection>"))
		fmt.Printf("  Create new backup: %s\n",
			cyan("pb backup create"))

		return nil
	},
}

// confirmRestore shows restore details and requires the user to type "restore"
// to confirm. It returns true only when the user types the exact word.
func confirmRestore(backup *pocketbase.Backup, ctx *config.Context) (bool, error) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Fprintf(os.Stderr, "%s DATABASE RESTORE OPERATION\n", red("⚠"))
	fmt.Fprintf(os.Stderr, "\nBackup to restore:\n")
	fmt.Fprintf(os.Stderr, "  Name: %s\n", bold(backup.Key))
	fmt.Fprintf(os.Stderr, "  Size: %s\n", backup.GetHumanSize())
	fmt.Fprintf(os.Stderr, "  Created: %s\n", backup.GetFormattedDate())
	fmt.Fprintf(os.Stderr, "  Age: %s\n", utils.FormatTimeAgo(backup.Modified.Time))
	fmt.Fprintf(os.Stderr, "  Context: %s\n", ctx.Name)

	fmt.Fprintf(os.Stderr, "\n%s CRITICAL WARNING:\n", red("🚨"))
	fmt.Fprintf(os.Stderr, "  • This will REPLACE ALL current database data\n")
	fmt.Fprintf(os.Stderr, "  • All current records, users, and settings will be LOST\n")
	fmt.Fprintf(os.Stderr, "  • PocketBase will restart during the restoration process\n")
	fmt.Fprintf(os.Stderr, "  • Your current authentication session will be invalidated\n")
	fmt.Fprintf(os.Stderr, "  • This action CANNOT BE UNDONE\n")

	fmt.Fprintf(os.Stderr, "\n%s Make sure you have a current backup before proceeding!\n", yellow("Recommendation:"))
	return utils.ConfirmWord("\nType 'restore' to confirm this dangerous operation: ", "restore")
}
