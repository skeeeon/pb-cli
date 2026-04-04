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

var createFileFlag string

var createCmd = &cobra.Command{
	Use:   "create <collection> [json_data]",
	Short: "Create a new record in a collection",
	Long: `Create a new record in a collection from JSON data.

Data can be provided as:
  1. A JSON string argument
  2. A file via --file flag
  3. Piped from stdin

Examples:
  pb collections create posts '{"title":"My Post","content":"Hello world"}'
  pb collections create posts --file post.json
  cat post.json | pb collections create posts
  pb c create posts '{"title":"New"}'`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := args[0]
		var jsonData string
		if len(args) > 1 {
			jsonData = args[1]
		}

		ctx, err := validateCollection(collection)
		if err != nil {
			return err
		}

		data, err := parseJSONInput(jsonData, createFileFlag)
		if err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}

		if err := validateCreateData(data, collection); err != nil {
			return fmt.Errorf("invalid create data: %w", err)
		}

		client := createPocketBaseClient(ctx)

		utils.PrintDebug(fmt.Sprintf("Creating record in collection '%s' with data: %+v", collection, data))

		record, err := client.CreateRecord(collection, data)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
				}
				if additionalSuggestion := provideSuggestions(collection, "create", err); additionalSuggestion != "" {
					fmt.Fprintf(os.Stderr, "Additional tip: %s\n", additionalSuggestion)
				}
				return fmt.Errorf("failed to create record")
			}
			return fmt.Errorf("failed to create record: %w", err)
		}

		recordID := ""
		if id, ok := record["id"].(string); ok {
			recordID = id
		}

		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(os.Stderr, "%s Record created successfully!\n", green("✓"))

		if recordID != "" {
			fmt.Fprintf(os.Stderr, "  Record ID: %s\n", recordID)
			fmt.Fprintf(os.Stderr, "  Collection: %s\n", collection)

			if name := getRecordDisplayName(record); name != "" {
				fmt.Fprintf(os.Stderr, "  Display: %s\n", name)
			}
		}

		outputFormat := getOutputFormat()

		fmt.Fprintf(os.Stderr, "\nCreated Record:\n")
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
	createCmd.Flags().StringVar(&createFileFlag, "file", "", "Path to JSON file containing record data")
}
