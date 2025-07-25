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

var downloadCmd = &cobra.Command{
	Use:   "download <backup_name> [output_path]",
	Short: "Download a backup file",
	Long: `Download a backup file from PocketBase.

If no output path is specified, the backup will be downloaded to:
  ~/.config/pb/<context>/backups/<backup_name>

If only a directory is specified, the backup will be saved with
its original name in that directory.

Examples:
  pb backup download backup_2024_01_15                    # Download to context folder
  pb backup download backup_2024_01_15 ./my-backups/     # Download to specific directory
  pb backup download backup_2024_01_15 ./backup.zip      # Download with specific filename`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupName := args[0]
		var outputPath string

		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Determine output path
		if len(args) > 1 {
			outputPath = args[1]
		} else {
			// Default to context backup directory
			backupDir := getBackupDir(ctx)
			outputPath = filepath.Join(backupDir, backupName)
		}

		// If outputPath is a directory, append the backup name
		if stat, err := os.Stat(outputPath); err == nil && stat.IsDir() {
			outputPath = filepath.Join(outputPath, backupName)
		}

		// Create PocketBase client
		client := pocketbase.NewClientFromContext(ctx)

		// Get backup info first to validate it exists and show details
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

		// Display download info
		fmt.Printf("\nDownload Details:\n")
		fmt.Printf("  Backup: %s\n", backup.Key)
		fmt.Printf("  Size: %s\n", backup.GetHumanSize())
		fmt.Printf("  Created: %s\n", backup.GetFormattedDate())
		fmt.Printf("  Output: %s\n", outputPath)

		// Check if output file already exists
		if _, err := os.Stat(outputPath); err == nil {
			if !forceFlag {
				return fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath)
			}
			utils.PrintWarning(fmt.Sprintf("Overwriting existing file: %s", outputPath))
		}

		// Download with progress
		utils.PrintInfo("Downloading backup...")

		var lastProgress int
		progressCallback := func(downloaded, total int64) {
			if total > 0 {
				progress := int((downloaded * 100) / total)
				if progress != lastProgress && progress%10 == 0 { // Show every 10%
					fmt.Printf("  Progress: %d%% (%s / %s)\n", 
						progress, 
						utils.FormatBytes(downloaded), 
						utils.FormatBytes(total))
					lastProgress = progress
				}
			}
		}

		err = client.DownloadBackupWithProgress(backupName, outputPath, progressCallback)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to download backup")
			}
			return fmt.Errorf("failed to download backup: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()
		
		fmt.Printf("\n%s Backup downloaded successfully!\n", green("✓"))
		fmt.Printf("  Downloaded: %s\n", backup.GetHumanSize())
		fmt.Printf("  Location: %s\n", outputPath)
		fmt.Printf("  Context: %s\n", cyan(ctx.Name))

		// Show next steps
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  Restore from backup: %s\n", 
			cyan(fmt.Sprintf("pb backup restore %s", backupName)))
		fmt.Printf("  Upload to another instance: %s\n", 
			cyan(fmt.Sprintf("pb backup upload %s", outputPath)))

		return nil
	},
}
