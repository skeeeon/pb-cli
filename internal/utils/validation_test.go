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
