package collections

import (
	"fmt"

	"pb-cli/internal/utils"
)

// validateCreateData validates the JSON data for creating a record
func validateCreateData(data map[string]interface{}, collection string) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("record data cannot be empty")
	}

	// Check for fields that should not be manually set
	restrictedFields := []string{"id", "created", "updated"}
	
	for _, field := range restrictedFields {
		if _, exists := data[field]; exists {
			return fmt.Errorf("field '%s' is automatically managed and should not be included", field)
		}
	}

	// Basic validation - PocketBase will handle detailed schema validation
	return validateBasicDataTypes(data)
}

// validateUpdateData validates the JSON data for updating a record
func validateUpdateData(data map[string]interface{}, collection string) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("update data cannot be empty")
	}

	// Check for fields that should not be manually updated
	restrictedFields := []string{"id", "created", "updated"}
	
	for _, field := range restrictedFields {
		if _, exists := data[field]; exists {
			return fmt.Errorf("field '%s' is automatically managed and cannot be updated", field)
		}
	}

	// Basic validation - PocketBase will handle detailed schema validation
	return validateBasicDataTypes(data)
}

// validateBasicDataTypes performs basic validation on data types
func validateBasicDataTypes(data map[string]interface{}) error {
	for key, value := range data {
		if err := validateFieldValue(key, value); err != nil {
			return err
		}
	}
	return nil
}

// validateFieldValue validates a single field value
func validateFieldValue(fieldName string, value interface{}) error {
	if value == nil {
		return nil // Null values are generally acceptable
	}

	switch v := value.(type) {
	case string:
		// Basic string validation
		if len(v) > 10000 { // Reasonable upper limit
			return fmt.Errorf("field '%s' exceeds maximum length (10000 characters)", fieldName)
		}
		
		// Validate email format if field name suggests it's an email
		if fieldName == "email" {
			if err := utils.ValidateEmail(v); err != nil {
				return fmt.Errorf("field '%s' has invalid email format: %w", fieldName, err)
			}
		}
		
	case float64:
		// Numbers are generally fine, but check for reasonable ranges
		if v < -1e15 || v > 1e15 {
			return fmt.Errorf("field '%s' has unreasonable numeric value", fieldName)
		}
		
	case bool:
		// Booleans are always valid
		
	case []interface{}:
		// Arrays - validate each element
		for i, item := range v {
			if err := validateFieldValue(fmt.Sprintf("%s[%d]", fieldName, i), item); err != nil {
				return err
			}
		}
		
	case map[string]interface{}:
		// Objects - validate each field
		for subKey, subValue := range v {
			if err := validateFieldValue(fmt.Sprintf("%s.%s", fieldName, subKey), subValue); err != nil {
				return err
			}
		}
		
	default:
		// Unknown type - let PocketBase handle it
		utils.PrintDebug(fmt.Sprintf("Unknown data type for field '%s': %T", fieldName, value))
	}
	
	return nil
}

// validateRecordID validates a record ID format
func validateRecordID(recordID string) error {
	if recordID == "" {
		return fmt.Errorf("record ID cannot be empty")
	}
	
	// Basic validation - PocketBase will handle detailed ID validation
	if len(recordID) < 1 || len(recordID) > 255 {
		return fmt.Errorf("record ID must be between 1 and 255 characters")
	}
	
	return nil
}

// validateCollectionName validates a collection name format
func validateCollectionName(collection string) error {
	if collection == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	
	// Basic validation - PocketBase will handle detailed collection validation
	if len(collection) < 1 || len(collection) > 50 {
		return fmt.Errorf("collection name must be between 1 and 50 characters")
	}
	
	return nil
}

// provideSuggestions provides helpful suggestions based on common errors
func provideSuggestions(collection string, action string, err error) string {
	errMsg := err.Error()
	
	// Common suggestions based on error patterns
	if contains(errMsg, "required") {
		return fmt.Sprintf("Check the required fields for the '%s' collection schema", collection)
	}
	
	if contains(errMsg, "unique") {
		return fmt.Sprintf("Ensure unique fields in '%s' collection have unique values", collection)
	}
	
	if contains(errMsg, "not found") {
		return fmt.Sprintf("Verify the record exists in the '%s' collection", collection)
	}
	
	if contains(errMsg, "permission") || contains(errMsg, "access") {
		return fmt.Sprintf("Check your permissions for the '%s' collection", collection)
	}
	
	if action == "create" {
		return fmt.Sprintf("Review the schema for the '%s' collection and ensure all required fields are provided", collection)
	}
	
	if action == "update" {
		return fmt.Sprintf("Verify the record ID exists and you have permission to update the '%s' collection", collection)
	}
	
	return "Check your data format and try again"
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(s) > len(substr) && (
			s[:len(substr)] == substr || 
			s[len(s)-len(substr):] == substr ||
			strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}

// Additional helper imports needed
import "strings"
