package collections

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var (
	getFieldsFlag []string
	getExpandFlag []string
)

var getCmd = &cobra.Command{
	Use:   "get <collection> <id>",
	Short: "Get a single record by ID",
	Long: `Get a single record from a collection by its ID.

Examples:
  pb collections get posts post_123
  pb collections get users user_abc --expand profile
  pb collections get posts post_123 --fields title,content --output yaml
  pb c get posts post_123`,
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

		utils.PrintDebug(fmt.Sprintf("Getting record '%s' from collection '%s' with expand=%v, fields=%v",
			recordID, collection, getExpandFlag, getFieldsFlag))

		record, err := client.GetRecord(collection, recordID, getExpandFlag, getFieldsFlag)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to get record")
			}
			return fmt.Errorf("failed to get record: %w", err)
		}

		outputFormat := getOutputFormat()

		switch outputFormat {
		case config.OutputFormatJSON:
			return utils.OutputData(record, config.OutputFormatJSON)
		case config.OutputFormatYAML:
			return utils.OutputData(record, config.OutputFormatYAML)
		case config.OutputFormatTable:
			return displayGetTable(record, collection, recordID)
		default:
			return fmt.Errorf("unsupported output format: %s", outputFormat)
		}
	},
}

func init() {
	getCmd.Flags().StringSliceVar(&getFieldsFlag, "fields", nil, "Specific fields to return (comma-separated)")
	getCmd.Flags().StringSliceVar(&getExpandFlag, "expand", nil, "Relations to expand (comma-separated)")
}
