package backup

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Long: `List all available backups in the PocketBase instance.

This displays information about each backup including name, size,
and creation date.

Examples:
  pb backup list
  pb backup list --output json
  pb backup list --output table`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Create PocketBase client
		client := pocketbase.NewClientFromContext(ctx)

		utils.PrintInfo("Fetching backups from PocketBase...")

		// List backups from PocketBase
		backups, err := client.ListBackups()
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to list backups")
			}
			return fmt.Errorf("failed to list backups: %w", err)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found.")
			fmt.Printf("\nCreate your first backup with: %s\n", 
				color.New(color.FgCyan).Sprint("pb backup create"))
			return nil
		}

		// Display results based on output format
		switch outputFlag {
		case config.OutputFormatJSON:
			return utils.OutputData(backups, config.OutputFormatJSON)
		case config.OutputFormatYAML:
			return utils.OutputData(backups, config.OutputFormatYAML)
		case config.OutputFormatTable, "":
			return displayBackupsTable(backups, ctx)
		default:
			return fmt.Errorf("unsupported output format: %s", outputFlag)
		}
	},
}

// displayBackupsTable displays backups in a table format
func displayBackupsTable(backups pocketbase.BackupsList, ctx *config.Context) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "SIZE", "CREATED", "AGE"})
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowSeparator("")
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, backup := range backups {
		age := utils.FormatTimeAgo(backup.Modified.Time)
		
		table.Append([]string{
			backup.Key,
			backup.GetHumanSize(),
			backup.GetFormattedDate(),
			age,
		})
	}

	fmt.Printf("Backups for context '%s' (%d total):\n", ctx.Name, len(backups))
	table.Render()

	// Show helpful commands
	fmt.Printf("\nUseful commands:\n")
	if len(backups) > 0 {
		firstBackup := backups[0].Key
		fmt.Printf("  Download backup: %s\n", 
			color.New(color.FgCyan).Sprintf("pb backup download %s", firstBackup))
		fmt.Printf("  Restore from backup: %s\n", 
			color.New(color.FgCyan).Sprintf("pb backup restore %s", firstBackup))
		fmt.Printf("  Delete backup: %s\n", 
			color.New(color.FgCyan).Sprintf("pb backup delete %s", firstBackup))
	}
	fmt.Printf("  Create new backup: %s\n", 
		color.New(color.FgCyan).Sprint("pb backup create"))

	return nil
}
