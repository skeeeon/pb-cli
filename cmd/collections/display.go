package collections

import (
	"fmt"
	"strings"

	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

// displayListTable displays the results in a user-friendly table format
func displayListTable(result *pocketbase.RecordsList, collection string) error {
	if result == nil || len(result.Items) == 0 {
		fmt.Printf("No %s found.\n", collection)
		return nil
	}

	// Show pagination info
	fmt.Printf("%s (%d-%d of %d total)\n\n", 
		utils.TitleCase(collection),
		((result.Page-1)*result.PerPage)+1,
		min(result.Page*result.PerPage, result.TotalItems),
		result.TotalItems)

	// Display table
	if err := utils.OutputData(result.Items, config.OutputFormatTable); err != nil {
		return fmt.Errorf("failed to display table: %w", err)
	}

	// Show pagination navigation hints
	if result.TotalPages > 1 {
		fmt.Printf("\nPagination:\n")
		if result.Page > 1 {
			prevOffset := (result.Page-2) * result.PerPage
			fmt.Printf("  Previous: --offset %d\n", prevOffset)
		}
		if result.Page < result.TotalPages {
			nextOffset := result.Page * result.PerPage
			fmt.Printf("  Next: --offset %d\n", nextOffset)
		}
		fmt.Printf("  Page %d of %d (use --offset to navigate)\n", 
			result.Page, result.TotalPages)
	}

	return nil
}

// displayGetTable displays a single record in table format
func displayGetTable(record map[string]interface{}, collection, recordID string) error {
	if record == nil {
		return fmt.Errorf("no record data received")
	}

	// Show header
	fmt.Printf("%s Record: %s\n", utils.TitleCase(collection), recordID)
	fmt.Println(strings.Repeat("=", 50))

	// Display record details in an organized way
	if err := displayRecordDetails(record, collection); err != nil {
		// Fallback to generic table if specific display fails
		return utils.OutputData(record, config.OutputFormatTable)
	}

	return nil
}

// displayRecordDetails displays record details with intelligent field ordering
func displayRecordDetails(record map[string]interface{}, collection string) error {
	// Common important fields that should be displayed first
	priorityFields := []string{"id", "name", "title", "email", "username"}
	
	// Descriptive fields that should come next
	descriptiveFields := []string{"description", "content", "bio", "summary"}
	
	// Status and type fields
	statusFields := []string{"active", "published", "status", "type", "category"}
	
	// Time fields that should be displayed last
	timeFields := []string{"created", "updated", "deleted"}

	// Display priority fields first
	for _, field := range priorityFields {
		if value, exists := record[field]; exists && value != nil {
			fmt.Printf("  %s: %v\n", utils.TitleCase(field), value)
		}
	}

	// Display descriptive fields
	for _, field := range descriptiveFields {
		if value, exists := record[field]; exists && value != nil {
			displayValue := formatFieldValue(value)
			fmt.Printf("  %s: %s\n", utils.TitleCase(field), displayValue)
		}
	}

	// Display status fields
	for _, field := range statusFields {
		if value, exists := record[field]; exists && value != nil {
			fmt.Printf("  %s: %v\n", utils.TitleCase(field), value)
		}
	}

	// Display other fields (excluding ones we already handled)
	skipFields := make(map[string]bool)
	allKnownFields := append(append(append(priorityFields, descriptiveFields...), statusFields...), timeFields...)
	for _, field := range allKnownFields {
		skipFields[field] = true
	}
	
	// Skip expand field as it will be handled separately
	skipFields["expand"] = true

	for key, value := range record {
		if !skipFields[key] && value != nil {
			displayValue := formatFieldValue(value)
			fmt.Printf("  %s: %s\n", utils.TitleCase(key), displayValue)
		}
	}

	// Display time fields last
	for _, field := range timeFields {
		if value, exists := record[field]; exists && value != nil {
			fmt.Printf("  %s: %v\n", utils.TitleCase(field), value)
		}
	}

	// Display expanded relations
	if expand, exists := record["expand"]; exists && expand != nil {
		fmt.Printf("\nExpanded Relations:\n")
		if err := displayExpandedRelations(expand); err != nil {
			fmt.Printf("  %v\n", expand)
		}
	}

	return nil
}

// formatFieldValue formats field values for display
func formatFieldValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Truncate very long strings
		if len(v) > 100 {
			return v[:97] + "..."
		}
		return v
	case []interface{}:
		// Handle arrays
		if len(v) == 0 {
			return "[]"
		}
		if len(v) == 1 {
			return fmt.Sprintf("[%v]", v[0])
		}
		return fmt.Sprintf("[%v, ... (%d items)]", v[0], len(v))
	case map[string]interface{}:
		// Handle objects
		if len(v) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{...} (%d fields)", len(v))
	default:
		return fmt.Sprintf("%v", value)
	}
}

// displayExpandedRelations displays expanded relation data
func displayExpandedRelations(expand interface{}) error {
	switch expandData := expand.(type) {
	case map[string]interface{}:
		for relationName, relationData := range expandData {
			fmt.Printf("  %s:\n", utils.TitleCase(relationName))
			
			switch relData := relationData.(type) {
			case []interface{}:
				// Multiple related records
				for i, item := range relData {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if name := getRecordDisplayName(itemMap); name != "" {
							fmt.Printf("    %d. %s\n", i+1, name)
						} else {
							fmt.Printf("    %d. %v\n", i+1, item)
						}
					}
				}
			case map[string]interface{}:
				// Single related record
				if name := getRecordDisplayName(relData); name != "" {
					fmt.Printf("    %s\n", name)
				} else {
					fmt.Printf("    %v\n", relData)
				}
			default:
				fmt.Printf("    %v\n", relData)
			}
		}
	default:
		return fmt.Errorf("unexpected expand data format")
	}
	
	return nil
}

// getRecordDisplayName attempts to get a display name for a record
func getRecordDisplayName(record map[string]interface{}) string {
	// Try common name fields
	nameFields := []string{"name", "title", "display_name", "full_name"}
	for _, field := range nameFields {
		if name, ok := record[field].(string); ok && name != "" {
			return name
		}
	}
	
	// Try email or username
	if email, ok := record["email"].(string); ok && email != "" {
		return email
	}
	if username, ok := record["username"].(string); ok && username != "" {
		return username
	}
	
	// Fallback to ID
	if id, ok := record["id"].(string); ok {
		return fmt.Sprintf("ID: %s", id)
	}
	
	return ""
}
