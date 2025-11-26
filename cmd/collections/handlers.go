package collections

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

// handleListAction handles the list action for a collection
func handleListAction(ctx *config.Context, collection string, args []string) error {
	// Use global flags instead of manual parsing
	// Note: args are any remaining positional arguments (none expected for list)
	if len(args) > 0 {
		return fmt.Errorf("list action does not accept positional arguments, use flags instead")
	}

	// Create PocketBase client
	client := createPocketBaseClient(ctx)

	// Build list options from global flags
	options := &pocketbase.ListOptions{
		Page:    calculatePage(offsetFlag, limitFlag),
		PerPage: limitFlag,
		Filter:  filterFlag,
		Sort:    sortFlag,
		Fields:  fieldsFlag,
		Expand:  expandFlag,
	}

	// Validate pagination parameters
	if err := validatePaginationOptions(options); err != nil {
		return fmt.Errorf("invalid pagination options: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Listing records from collection '%s' with options: page=%d, perPage=%d, filter='%s', sort='%s', fields=%v, expand=%v", 
		collection, options.Page, options.PerPage, options.Filter, options.Sort, options.Fields, options.Expand))

	// List records from PocketBase
	result, err := client.ListRecords(collection, options)
	if err != nil {
		if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
			utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
			if suggestion := pbErr.GetSuggestion(); suggestion != "" {
				fmt.Printf("\nSuggestion: %s\n", suggestion)
			}
			return fmt.Errorf("failed to list records")
		}
		return fmt.Errorf("failed to list records: %w", err)
	}

	// Display results based on output format
	outputFormat := outputFlag
	if outputFormat == "" {
		outputFormat = config.Global.OutputFormat
	}

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
}

// handleGetAction handles the get action for a collection
func handleGetAction(ctx *config.Context, collection string, args []string) error {
	// Expect exactly one positional argument (record ID)
	if len(args) != 1 {
		return fmt.Errorf("get requires exactly one record ID argument")
	}
	
	recordID := args[0]

	// Basic validation
	if err := validateRecordID(recordID); err != nil {
		return fmt.Errorf("invalid record ID: %w", err)
	}

	// Create PocketBase client
	client := createPocketBaseClient(ctx)

	utils.PrintDebug(fmt.Sprintf("Getting record '%s' from collection '%s' with expand=%v, fields=%v", 
		recordID, collection, expandFlag, fieldsFlag))

	// Get record from PocketBase - now passing fieldsFlag
	record, err := client.GetRecord(collection, recordID, expandFlag, fieldsFlag)
	if err != nil {
		if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
			utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
			if suggestion := pbErr.GetSuggestion(); suggestion != "" {
				fmt.Printf("\nSuggestion: %s\n", suggestion)
			}
			return fmt.Errorf("failed to get record")
		}
		return fmt.Errorf("failed to get record: %w", err)
	}

	// Display result based on output format
	outputFormat := outputFlag
	if outputFormat == "" {
		outputFormat = config.Global.OutputFormat
	}

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
}

// handleCreateAction handles the create action for a collection
func handleCreateAction(ctx *config.Context, collection string, args []string) error {
	// Get JSON data from positional argument or file flag
	var jsonData string
	if len(args) > 0 {
		jsonData = args[0]
	}

	// Parse JSON input from string or file
	data, err := parseJSONInput(jsonData, fileFlag)
	if err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate that we don't have restricted fields
	if err := validateCreateData(data, collection); err != nil {
		return fmt.Errorf("invalid create data: %w", err)
	}

	// Create PocketBase client
	client := createPocketBaseClient(ctx)

	utils.PrintDebug(fmt.Sprintf("Creating record in collection '%s' with data: %+v", collection, data))

	// Create record in PocketBase
	record, err := client.CreateRecord(collection, data)
	if err != nil {
		if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
			utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
			if suggestion := pbErr.GetSuggestion(); suggestion != "" {
				fmt.Printf("\nSuggestion: %s\n", suggestion)
			}
			// Provide additional suggestions based on the error
			if additionalSuggestion := provideSuggestions(collection, "create", err); additionalSuggestion != "" {
				fmt.Printf("Additional tip: %s\n", additionalSuggestion)
			}
			return fmt.Errorf("failed to create record")
		}
		return fmt.Errorf("failed to create record: %w", err)
	}

	// Display success message
	recordID := ""
	if id, ok := record["id"].(string); ok {
		recordID = id
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Record created successfully!\n", green("✓"))
	
	if recordID != "" {
		fmt.Printf("  Record ID: %s\n", recordID)
		fmt.Printf("  Collection: %s\n", collection)
		
		// Show record name if available
		if name := getRecordDisplayName(record); name != "" {
			fmt.Printf("  Display: %s\n", name)
		}
	}

	// Display result based on output format
	outputFormat := outputFlag
	if outputFormat == "" {
		outputFormat = config.Global.OutputFormat
	}

	fmt.Printf("\nCreated Record:\n")
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
}

// handleUpdateAction handles the update action for a collection
func handleUpdateAction(ctx *config.Context, collection string, args []string) error {
	// Expect at least record ID, optionally JSON data
	if len(args) < 1 {
		return fmt.Errorf("update requires a record ID argument")
	}
	
	recordID := args[0]
	var jsonData string
	if len(args) > 1 {
		jsonData = args[1]
	}

	// Basic validation
	if err := validateRecordID(recordID); err != nil {
		return fmt.Errorf("invalid record ID: %w", err)
	}

	// Parse JSON input from string or file
	data, err := parseJSONInput(jsonData, fileFlag)
	if err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate that we don't have restricted fields
	if err := validateUpdateData(data, collection); err != nil {
		return fmt.Errorf("invalid update data: %w", err)
	}

	// Create PocketBase client
	client := createPocketBaseClient(ctx)

	utils.PrintDebug(fmt.Sprintf("Updating record '%s' in collection '%s' with data: %+v", recordID, collection, data))

	// Update record in PocketBase
	record, err := client.UpdateRecord(collection, recordID, data)
	if err != nil {
		if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
			utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
			if suggestion := pbErr.GetSuggestion(); suggestion != "" {
				fmt.Printf("\nSuggestion: %s\n", suggestion)
			}
			// Provide additional suggestions based on the error
			if additionalSuggestion := provideSuggestions(collection, "update", err); additionalSuggestion != "" {
				fmt.Printf("Additional tip: %s\n", additionalSuggestion)
			}
			return fmt.Errorf("failed to update record")
		}
		return fmt.Errorf("failed to update record: %w", err)
	}

	// Display success message
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Record updated successfully!\n", green("✓"))
	
	fmt.Printf("  Record ID: %s\n", recordID)
	fmt.Printf("  Collection: %s\n", collection)
	
	// Show record name if available
	if name := getRecordDisplayName(record); name != "" {
		fmt.Printf("  Display: %s\n", name)
	}

	// Show which fields were updated
	fieldCount := len(data)
	if fieldCount > 0 {
		fmt.Printf("  Updated %d field(s)\n", fieldCount)
	}

	// Display result based on output format
	outputFormat := outputFlag
	if outputFormat == "" {
		outputFormat = config.Global.OutputFormat
	}

	fmt.Printf("\nUpdated Record:\n")
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
}

// handleDeleteAction handles the delete action for a collection
func handleDeleteAction(ctx *config.Context, collection string, args []string) error {
	// Expect exactly one positional argument (record ID)
	if len(args) != 1 {
		return fmt.Errorf("delete requires exactly one record ID argument")
	}
	
	recordID := args[0]

	// Basic validation
	if err := validateRecordID(recordID); err != nil {
		return fmt.Errorf("invalid record ID: %w", err)
	}

	// Create PocketBase client
	client := createPocketBaseClient(ctx)

	// Get record details for confirmation (unless forced)
	var record map[string]interface{}
	if !forceFlag {
		utils.PrintDebug(fmt.Sprintf("Fetching record details for confirmation: %s", recordID))
		
		var err error
		// Pass nil for both expand and fields since we just need basic info
		record, err = client.GetRecord(collection, recordID, nil, nil)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("failed to retrieve record for confirmation")
			}
			return fmt.Errorf("failed to retrieve record: %w", err)
		}

		// Show confirmation prompt with record details
		if err := confirmDeletion(collection, recordID, record); err != nil {
			return err
		}
	}

	utils.PrintDebug(fmt.Sprintf("Deleting record '%s' from collection '%s'", recordID, collection))

	// Delete record from PocketBase
	err := client.DeleteRecord(collection, recordID)
	if err != nil {
		if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
			utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
			if suggestion := pbErr.GetSuggestion(); suggestion != "" {
				fmt.Printf("\nSuggestion: %s\n", suggestion)
			}
			return fmt.Errorf("failed to delete record")
		}
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Display success message (unless quiet mode)
	if !quietFlag {
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s Record deleted successfully!\n", green("✓"))
		fmt.Printf("  Record ID: %s\n", recordID)
		fmt.Printf("  Collection: %s\n", collection)
		
		// Show record name if we have it
		if record != nil {
			if name := getRecordDisplayName(record); name != "" {
				fmt.Printf("  Display: %s\n", name)
			}
		}
	}

	return nil
}

// Helper functions

// calculatePage calculates the page number from offset and limit
func calculatePage(offset, limit int) int {
	if offset <= 0 || limit <= 0 {
		return 1
	}
	return (offset / limit) + 1
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

// confirmDeletion prompts the user to confirm deletion and shows record details
func confirmDeletion(collection, recordID string, record map[string]interface{}) error {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("%s Record to be deleted:\n", red("⚠"))
	fmt.Printf("  Collection: %s\n", bold(collection))
	fmt.Printf("  Record ID: %s\n", recordID)

	// Show key record details
	if record != nil {
		if name := getRecordDisplayName(record); name != "" {
			fmt.Printf("  Display: %s\n", name)
		}
		
		// Show a few key fields if available
		keyFields := []string{"email", "title", "description", "content"}
		for _, field := range keyFields {
			if value, ok := record[field].(string); ok && value != "" {
				displayValue := value
				if len(displayValue) > 50 {
					displayValue = displayValue[:47] + "..."
				}
				fmt.Printf("  %s: %s\n", utils.TitleCase(field), displayValue)
				break // Only show one descriptive field
			}
		}
	}

	fmt.Printf("\n%s This action cannot be undone.\n", yellow("Warning:"))
	fmt.Print("Are you sure you want to delete this record? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Deletion cancelled.")
		return fmt.Errorf("deletion cancelled by user")
	}

	return nil
}
