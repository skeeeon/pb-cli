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
	pageFlag   int
	limitFlag  int
	allFlag    bool
	filterFlag string
	sortFlag   string
	fieldsFlag []string
	expandFlag []string
)

var listCmd = &cobra.Command{
	Use:   "list <collection>",
	Short: "List records from a collection",
	Long: `List records from a collection with filtering, sorting, and pagination.

By default a single page is returned (--page / --limit). Use --all to fetch every
matching record across all pages; --all cannot be combined with --page or --limit.

Examples:
  pb collections list posts
  pb collections list posts --filter 'published=true' --sort '-created'
  pb collections list users --limit 10 --page 2
  pb collections list posts --all --filter 'published=true'
  pb collections list posts --fields title,content,created --expand author
  pb c list posts --output table`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := args[0]

		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		client := createPocketBaseClient(ctx)

		options := &pocketbase.ListOptions{
			Page:    pageFlag,
			PerPage: limitFlag,
			Filter:  filterFlag,
			Sort:    sortFlag,
			Fields:  fieldsFlag,
			Expand:  expandFlag,
		}

		var result *pocketbase.RecordsList
		if allFlag {
			utils.PrintDebug(fmt.Sprintf("Listing all records from collection '%s' (filter='%s', sort='%s')",
				collection, options.Filter, options.Sort))
			result, err = client.ListAllRecords(collection, options)
		} else {
			if err := validatePaginationOptions(options); err != nil {
				return fmt.Errorf("invalid pagination options: %w", err)
			}
			utils.PrintDebug(fmt.Sprintf("Listing records from collection '%s' with options: page=%d, perPage=%d, filter='%s', sort='%s', fields=%v, expand=%v",
				collection, options.Page, options.PerPage, options.Filter, options.Sort, options.Fields, options.Expand))
			result, err = client.ListRecords(collection, options)
		}
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to list records")
			}
			return fmt.Errorf("failed to list records: %w", err)
		}

		outputFormat := getOutputFormat()

		switch outputFormat {
		case config.OutputFormatJSON:
			return utils.OutputData(result, config.OutputFormatJSON)
		case config.OutputFormatYAML:
			return utils.OutputData(result, config.OutputFormatYAML)
		case config.OutputFormatTable:
			return displayListTable(result, collection)
		default:
			return fmt.Errorf("unsupported output format: %s", outputFormat)
		}
	},
}

func init() {
	listCmd.Flags().IntVar(&pageFlag, "page", 1, "Page number for pagination")
	listCmd.Flags().IntVar(&limitFlag, "limit", 30, "Maximum number of records to return")
	listCmd.Flags().BoolVar(&allFlag, "all", false, "Fetch all records across all pages (cannot be used with --page/--limit)")
	listCmd.Flags().StringVar(&filterFlag, "filter", "", "PocketBase filter expression (e.g., 'published=true && title~\"test\"')")
	listCmd.Flags().StringVar(&sortFlag, "sort", "", "Sort expression (e.g., 'title', '-created', 'title,-updated')")
	listCmd.Flags().StringSliceVar(&fieldsFlag, "fields", nil, "Specific fields to return (comma-separated)")
	listCmd.Flags().StringSliceVar(&expandFlag, "expand", nil, "Relations to expand (comma-separated)")

	// --all supersedes manual pagination; make the conflict explicit rather than silent.
	listCmd.MarkFlagsMutuallyExclusive("all", "page")
	listCmd.MarkFlagsMutuallyExclusive("all", "limit")
}

// validatePaginationOptions validates pagination parameters
func validatePaginationOptions(options *pocketbase.ListOptions) error {
	if options.PerPage < 1 {
		return fmt.Errorf("limit must be at least 1")
	}
	if options.PerPage > 500 {
		return fmt.Errorf("limit cannot exceed 500 records")
	}
	if options.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	return nil
}
