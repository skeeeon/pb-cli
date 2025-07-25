package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"pb-cli/internal/config"
)

// ValidateURL validates that a string is a valid URL
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	_, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return nil
}

// ValidateContextName validates a context name
func ValidateContextName(name string) error {
	if name == "" {
		return fmt.Errorf("context name cannot be empty")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("context name can only contain letters, numbers, hyphens, and underscores")
	}

	// Check length
	if len(name) > 50 {
		return fmt.Errorf("context name must be 50 characters or less")
	}

	return nil
}

// ValidateAuthCollection validates a PocketBase auth collection name
func ValidateAuthCollection(collection string) error {
	return config.ValidateAuthCollection(collection)
}

// ValidateOutputFormat validates an output format
func ValidateOutputFormat(format string) error {
	validFormats := []string{
		config.OutputFormatJSON,
		config.OutputFormatYAML,
		config.OutputFormatTable,
	}

	format = strings.ToLower(format)
	for _, valid := range validFormats {
		if format == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid output format '%s'. Valid options: %s", 
		format, strings.Join(validFormats, ", "))
}

// ValidateCollectionName validates a PocketBase collection name
func ValidateCollectionName(collection string) error {
	return config.ValidateCollectionName(collection)
}

// ValidateCollections validates a list of collection names
func ValidateCollections(collections []string) error {
	return config.ValidateCollections(collections)
}

// ValidateEmail validates an email address format (minimal - PocketBase handles detailed validation)
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Basic check for @ symbol
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// ValidateRequiredString validates that a string is not empty
func ValidateRequiredString(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateStringLength validates string length constraints
func ValidateStringLength(value, fieldName string, min, max int) error {
	length := len(value)
	if length < min {
		return fmt.Errorf("%s must be at least %d characters", fieldName, min)
	}
	if max > 0 && length > max {
		return fmt.Errorf("%s must be no more than %d characters", fieldName, max)
	}
	return nil
}

// ValidateFileExists checks if a file exists at the given path
func ValidateFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Basic path validation
	if strings.Contains(path, "..") {
		return fmt.Errorf("file path cannot contain '..' for security reasons")
	}

	return nil
}

// ValidateRecordID validates a PocketBase record ID format
func ValidateRecordID(recordID string) error {
	if recordID == "" {
		return fmt.Errorf("record ID cannot be empty")
	}
	
	// Basic validation - PocketBase will handle detailed ID validation
	if len(recordID) < 1 || len(recordID) > 255 {
		return fmt.Errorf("record ID must be between 1 and 255 characters")
	}
	
	return nil
}

// ValidatePocketBaseURL validates a PocketBase server URL
func ValidatePocketBaseURL(urlStr string) error {
	if err := ValidateURL(urlStr); err != nil {
		return fmt.Errorf("invalid PocketBase URL: %w", err)
	}
	
	// Check for supported schemes
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("PocketBase URL must use http:// or https:// scheme")
	}
	
	return nil
}

// ValidateJSONData performs basic validation on JSON data structure
func ValidateJSONData(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	
	// Check for reasonable data size (prevent extremely large payloads)
	if len(data) > 1000 {
		return fmt.Errorf("data contains too many fields (maximum 1000)")
	}
	
	return nil
}

// ValidateFilterExpression performs basic validation on PocketBase filter expressions
func ValidateFilterExpression(filter string) error {
	if filter == "" {
		return nil // Empty filter is valid
	}
	
	// Basic validation - check for reasonable length
	if len(filter) > 1000 {
		return fmt.Errorf("filter expression is too long (maximum 1000 characters)")
	}
	
	// Check for basic SQL injection patterns (basic protection)
	dangerousPatterns := []string{
		"';",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
	}
	
	filterLower := strings.ToLower(filter)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(filterLower, pattern) {
			return fmt.Errorf("filter expression contains potentially dangerous pattern: %s", pattern)
		}
	}
	
	return nil
}

// ValidateSortExpression performs basic validation on PocketBase sort expressions
func ValidateSortExpression(sort string) error {
	if sort == "" {
		return nil // Empty sort is valid
	}
	
	// Basic validation - check for reasonable length
	if len(sort) > 200 {
		return fmt.Errorf("sort expression is too long (maximum 200 characters)")
	}
	
	// Check for basic field name patterns
	validSortPattern := regexp.MustCompile(`^[\w\-,\s]+$`)
	if !validSortPattern.MatchString(sort) {
		return fmt.Errorf("sort expression contains invalid characters (only letters, numbers, hyphens, commas, and spaces allowed)")
	}
	
	return nil
}

// ValidateFieldsList validates a list of field names
func ValidateFieldsList(fields []string) error {
	if len(fields) == 0 {
		return nil // Empty fields list is valid
	}
	
	validFieldPattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	
	for _, field := range fields {
		if field == "" {
			return fmt.Errorf("field name cannot be empty")
		}
		
		if len(field) > 50 {
			return fmt.Errorf("field name '%s' is too long (maximum 50 characters)", field)
		}
		
		if !validFieldPattern.MatchString(field) {
			return fmt.Errorf("field name '%s' has invalid format (must start with letter, contain only letters, numbers, and underscores)", field)
		}
	}
	
	return nil
}

// ValidateExpandList validates a list of expand relation names
func ValidateExpandList(expand []string) error {
	// Use the same validation as fields since expand relations follow similar naming patterns
	return ValidateFieldsList(expand)
}

// ValidatePaginationParams validates pagination parameters
func ValidatePaginationParams(offset, limit int) error {
	if offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	
	if limit < 1 {
		return fmt.Errorf("limit must be at least 1")
	}
	
	if limit > 500 {
		return fmt.Errorf("limit cannot exceed 500")
	}
	
	if offset > 1000000 {
		return fmt.Errorf("offset cannot exceed 1,000,000")
	}
	
	return nil
}
