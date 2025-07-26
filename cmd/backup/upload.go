package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <file_path> [--name <backup_name>]",
	Short: "Upload a backup file",
	Long: `Upload a backup file to PocketBase.

The backup file will be uploaded and made available for restore operations.
The backup name will be determined based on the filename unless specified otherwise.

Note: Uploading backups requires admin authentication.

Examples:
  pb backup upload ./backup.zip                      # Upload with filename as backup name
  pb backup upload ./backup.zip --name "production"  # Upload with custom name
  pb backup upload ~/Downloads/backup_2024_01_15     # Upload from different location`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Check if file exists
		fileInfo, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("backup file does not exist: %s", filePath)
		}
		if err != nil {
			return fmt.Errorf("failed to access backup file: %w", err)
		}

		// Determine backup name (for display purposes - PocketBase will determine actual name)
		displayName := nameFlag
		if displayName == "" {
			displayName = filepath.Base(filePath)
		}

		// Display upload info
		fmt.Printf("Upload Details:\n")
		fmt.Printf("  File: %s\n", filePath)
		fmt.Printf("  Size: %s\n", utils.FormatBytes(fileInfo.Size()))
		fmt.Printf("  Expected name: %s\n", displayName)
		fmt.Printf("  Context: %s\n", ctx.Name)

		// Create PocketBase client
		client := pocketbase.NewClientFromContext(ctx)

		// Upload the backup
		utils.PrintInfo("Uploading backup...")

		var lastProgress int
		progressCallback := func(uploaded, total int64) {
			if total > 0 {
				progress := int((uploaded * 100) / total)
				if progress != lastProgress && progress%10 == 0 { // Show every 10%
					fmt.Printf("  Progress: %d%% (%s / %s)\n", 
						progress, 
						utils.FormatBytes(uploaded), 
						utils.FormatBytes(total))
					lastProgress = progress
				}
			}
		}

		backup, err := client.UploadBackup(filePath, nameFlag, progressCallback)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to upload backup")
			}
			return fmt.Errorf("failed to upload backup: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		
		fmt.Printf("\n%s Backup uploaded successfully!\n", green("✓"))
		
		if backup != nil {
			fmt.Printf("\nUpload Results:\n")
			fmt.Printf("  Backup name: %s\n", backup.Key)
			fmt.Printf("  Size: %s\n", backup.GetHumanSize())
			if backup.Modified.Time.IsZero() {
				fmt.Printf("  Status: Uploaded\n")
			} else {
				fmt.Printf("  Created: %s\n", backup.GetFormattedDate())
			}
			fmt.Printf("  Context: %s\n", cyan(ctx.Name))

			// Show next steps
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  List all backups: %s\n", 
				cyan("pb backup list"))
			fmt.Printf("  Restore from backup: %s\n", 
				cyan(fmt.Sprintf("pb backup restore %s", backup.Key)))
		} else {
			fmt.Printf("  File: %s\n", filePath)
			fmt.Printf("  Context: %s\n", cyan(ctx.Name))
			
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  List all backups: %s\n", 
				cyan("pb backup list"))
		}

		return nil
	},
}

func init() {
	uploadCmd.Flags().StringVar(&nameFlag, "name", "", "Custom backup name (PocketBase will use filename if not specified)")
}
