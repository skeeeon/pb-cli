package collections

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var (
	forceFlag bool
	quietFlag bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <collection> <id>",
	Short: "Delete a record from a collection",
	Long: `Delete a record from a collection by its ID.

By default, prompts for confirmation before deleting.

Examples:
  pb collections delete posts post_123
  pb collections delete users user_456 --force
  pb c delete posts post_123 -f -q`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := args[0]
		recordID := args[1]

		ctx, err := validateCollection(collection)
		if err != nil {
			return err
		}

		if err := validateRecordID(recordID); err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		client := createPocketBaseClient(ctx)

		var record map[string]interface{}
		if !forceFlag {
			utils.PrintDebug(fmt.Sprintf("Fetching record details for confirmation: %s", recordID))

			record, err = client.GetRecord(collection, recordID, nil, nil)
			if err != nil {
				if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
					utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
					if suggestion := pbErr.GetSuggestion(); suggestion != "" {
						fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
					}
					return fmt.Errorf("failed to retrieve record for confirmation")
				}
				return fmt.Errorf("failed to retrieve record: %w", err)
			}

			if err := confirmDeletion(collection, recordID, record); err != nil {
				return err
			}
		}

		utils.PrintDebug(fmt.Sprintf("Deleting record '%s' from collection '%s'", recordID, collection))

		if err := client.DeleteRecord(collection, recordID); err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to delete record")
			}
			return fmt.Errorf("failed to delete record: %w", err)
		}

		if !quietFlag {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Fprintf(os.Stderr, "%s Record deleted successfully!\n", green("✓"))
			fmt.Fprintf(os.Stderr, "  Record ID: %s\n", recordID)
			fmt.Fprintf(os.Stderr, "  Collection: %s\n", collection)

			if record != nil {
				if name := getRecordDisplayName(record); name != "" {
					fmt.Fprintf(os.Stderr, "  Display: %s\n", name)
				}
			}
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress success messages")
}

// confirmDeletion prompts the user to confirm deletion and shows record details
func confirmDeletion(collection, recordID string, record map[string]interface{}) error {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Fprintf(os.Stderr, "%s Record to be deleted:\n", red("⚠"))
	fmt.Fprintf(os.Stderr, "  Collection: %s\n", bold(collection))
	fmt.Fprintf(os.Stderr, "  Record ID: %s\n", recordID)

	if record != nil {
		if name := getRecordDisplayName(record); name != "" {
			fmt.Fprintf(os.Stderr, "  Display: %s\n", name)
		}

		keyFields := []string{"email", "title", "description", "content"}
		for _, field := range keyFields {
			if value, ok := record[field].(string); ok && value != "" {
				displayValue := value
				if len(displayValue) > 50 {
					displayValue = displayValue[:47] + "..."
				}
				fmt.Fprintf(os.Stderr, "  %s: %s\n", utils.TitleCase(field), displayValue)
				break
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n%s This action cannot be undone.\n", yellow("Warning:"))
	fmt.Fprint(os.Stderr, "Are you sure you want to delete this record? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Fprintln(os.Stderr, "Deletion cancelled.")
		return fmt.Errorf("deletion cancelled by user")
	}

	return nil
}
