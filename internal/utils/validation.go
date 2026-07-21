package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURL validates that a string is a valid and useful URL with a scheme and host.
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// url.Parse is lenient; require both a scheme and a host to be useful.
	if parsedURL.Scheme == "" {
		return fmt.Errorf("invalid URL: missing scheme (e.g., http:// or https://)")
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("invalid URL: missing host (e.g., example.com)")
	}

	return nil
}

// ValidatePocketBaseURL validates a PocketBase server URL.
func ValidatePocketBaseURL(urlStr string) error {
	if err := ValidateURL(urlStr); err != nil {
		return err
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("PocketBase URL must use http:// or https:// scheme")
	}

	return nil
}

// ValidateEmail validates an email address format (minimal - PocketBase handles detailed validation)
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}
