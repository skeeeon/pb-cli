package utils_test

import (
	"pb-cli/internal/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateURL checks various URL formats with the new stricter logic.
func TestValidateURL(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid HTTPS", "https://example.com", false},
		{"Valid HTTP with port", "http://localhost:8090", false},
		{"Valid with path", "https://example.com/api/health", false},
		{"Invalid - missing scheme", "example.com", true},
		{"Invalid - missing host", "http://", true},
		// CORRECTED TEST CASE: Now expects an error due to stricter validation.
		{"Invalid format", "not a url", true},
		{"Empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidateURL(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePocketBaseURL checks for stricter http/https schemes.
func TestValidatePocketBaseURL(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid HTTPS", "https://pb.example.com", false},
		{"Valid HTTP", "http://127.0.0.1:8090", false},
		{"Invalid scheme", "ftp://pb.example.com", true},
		{"Missing scheme", "pb.example.com", true},
		{"Empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidatePocketBaseURL(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateContextName checks for valid context names.
func TestValidateContextName(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid name", "production", false},
		{"Valid with numbers", "prod-1", false},
		{"Valid with underscore", "dev_local", false},
		{"Invalid with space", "my context", true},
		{"Invalid with special char", "prod!", true},
		{"Too long", "a-very-long-context-name-that-is-definitely-over-fifty-characters", true},
		{"Empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidateContextName(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateOutputFormat checks for valid output format strings.
func TestValidateOutputFormat(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid json", "json", false},
		{"Valid yaml", "yaml", false},
		{"Valid table", "table", false},
		{"Valid case-insensitive", "JSON", false},
		{"Invalid format", "xml", true},
		// CORRECTED TEST CASE: An empty string is invalid and should produce an error.
		{"Failure - Empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidateOutputFormat(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFilterExpression checks for potentially unsafe filter strings.
func TestValidateFilterExpression(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid simple filter", "published=true", false},
		{"Valid complex filter", "name~'test' && (total > 10 || author.id='abc')", false},
		{"Empty filter is valid", "", false},
		{"Invalid with semicolon", "id='123';", true},
		{"Invalid with comment", "id='123' --", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidateFilterExpression(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePaginationParams checks pagination boundaries.
func TestValidatePaginationParams(t *testing.T) {
	testCases := []struct {
		name      string
		offset    int
		limit     int
		expectErr bool
	}{
		{"Valid params", 0, 30, false},
		{"Valid high offset", 1000, 100, false},
		{"Invalid negative offset", -1, 30, true},
		{"Invalid zero limit", 0, 0, true},
		{"Invalid limit too high", 0, 501, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.ValidatePaginationParams(tc.offset, tc.limit)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
