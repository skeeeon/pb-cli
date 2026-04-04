package collections

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var updateFileFlag string

var updateCmd = &cobra.Command{
	Use:   "update <collection> <id> [json_data]",
	Short: "Update an existing record",
	Long: `Update an existing record in a collection with JSON data.

Data can be provided as:
  1. A JSON string argument
  2. A file via --file flag
  3. Piped from stdin

Examples:
  pb collections update posts post_123 '{"published":true}'
  pb collections update posts post_123 --file updates.json
  pb c update posts post_123 '{"title":"Updated"}'`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := args[0]
		recordID := args[1]
		var jsonData string
		if len(args) > 2 {
			jsonData = args[2]
		}

		ctx, err := validateCollection(collection)
		if err != nil {
			return err
		}

		if err := validateRecordID(recordID); err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		data, err := parseJSONInput(jsonData, updateFileFlag)
		if err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}

		if err := validateUpdateData(data, collection); err != nil {
			return fmt.Errorf("invalid update data: %w", err)
		}

		client := createPocketBaseClient(ctx)

		utils.PrintDebug(fmt.Sprintf("Updating record '%s' in collection '%s' with data: %+v", recordID, collection, data))

		record, err := client.UpdateRecord(collection, recordID, data)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
				}
				if additionalSuggestion := provideSuggestions(collection, "update", err); additionalSuggestion != "" {
					fmt.Fprintf(os.Stderr, "Additional tip: %s\n", additionalSuggestion)
				}
				return fmt.Errorf("failed to update record")
			}
			return fmt.Errorf("failed to update record: %w", err)
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(os.Stderr, "%s Record updated successfully!\n", green("✓"))

		fmt.Fprintf(os.Stderr, "  Record ID: %s\n", recordID)
		fmt.Fprintf(os.Stderr, "  Collection: %s\n", collection)

		if name := getRecordDisplayName(record); name != "" {
			fmt.Fprintf(os.Stderr, "  Display: %s\n", name)
		}

		fieldCount := len(data)
		if fieldCount > 0 {
			fmt.Fprintf(os.Stderr, "  Updated %d field(s)\n", fieldCount)
		}

		outputFormat := getOutputFormat()

		fmt.Fprintf(os.Stderr, "\nUpdated Record:\n")
		switch outputFormat {
		case config.OutputFormatJSON:
			return utils.OutputData(record, config.OutputFormatJSON)
		case config.OutputFormatYAML:
			return utils.OutputData(record, config.OutputFormatYAML)
		case config.OutputFormatTable:
			return utils.OutputData(record, config.OutputFormatTable)
		default:
			return fmt.Errorf("unsupported output format: %s", outputFormat)
		}
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateFileFlag, "file", "", "Path to JSON file containing record data")
}
