package utils_test

import (
	"bytes"
	"io"
	"os"
	"pb-cli/internal/config"
	"pb-cli/internal/utils"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput is a helper function to capture what's written to stdout.
func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestOutputData tests the main output dispatcher.
func TestOutputData(t *testing.T) {
	sampleData := []map[string]interface{}{
		{"id": "1", "name": "First Post", "published": true},
		{"id": "2", "name": "Second Post", "published": false},
	}

	t.Run("JSON Output", func(t *testing.T) {
		output := captureOutput(func() {
			err := utils.OutputData(sampleData, "json")
			require.NoError(t, err)
		})
		assert.Contains(t, output, `"id": "1"`)
		assert.Contains(t, output, `"name": "Second Post"`)
		assert.True(t, strings.HasPrefix(output, "[\n  {\n")) // Check for indentation
	})

	t.Run("YAML Output", func(t *testing.T) {
		output := captureOutput(func() {
			err := utils.OutputData(sampleData, "yaml")
			require.NoError(t, err)
		})
		assert.Contains(t, output, "- id: \"1\"")
		assert.Contains(t, output, "name: Second Post")
	})

	t.Run("Table Output for Slice", func(t *testing.T) {
		output := captureOutput(func() {
			err := utils.OutputData(sampleData, "table")
			require.NoError(t, err)
		})
		assert.Contains(t, output, "ID")
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "PUBLISHED")
		assert.Contains(t, output, "First Post")
		assert.Contains(t, output, "✓") // true
		assert.Contains(t, output, "✗") // false
	})

	t.Run("Table Output for Single Map", func(t *testing.T) {
		singleMap := map[string]interface{}{"id": "1", "name": "First Post", "published": true}
		output := captureOutput(func() {
			err := utils.OutputData(singleMap, "table")
			require.NoError(t, err)
		})
		assert.Contains(t, output, "Id") // TitleCase
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "First Post")
	})

	t.Run("Unsupported Format", func(t *testing.T) {
		err := utils.OutputData(sampleData, "xml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported output format: xml")
	})
}

// TestPrintHelpers checks the colored/uncolored output of info/error messages.
func TestPrintHelpers(t *testing.T) {
	// Disable colors for predictable output testing
	originalColorsEnabled := config.Global.ColorsEnabled
	config.Global.ColorsEnabled = false
	defer func() { config.Global.ColorsEnabled = originalColorsEnabled }()

	t.Run("PrintError", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		utils.PrintError(assert.AnError)

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "Error: assert.AnError general error for testing")
	})

	t.Run("PrintInfo", func(t *testing.T) {
		output := captureOutput(func() {
			utils.PrintInfo("This is an info message")
		})
		assert.Contains(t, output, "Info: This is an info message")
	})
}

// TestFormatBytes checks the byte formatting logic.
func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		input    int64
		expected string
	}{
		{"Bytes", 512, "512 B"},
		{"Kilobytes", 1024, "1.0 KB"},
		{"Kilobytes decimal", 1536, "1.5 KB"},
		{"Megabytes", 1024 * 1024 * 5, "5.0 MB"},
		// CORRECTED LINE: Use integer arithmetic to represent 2.3 GB
		{"Gigabytes", (23 * 1024 * 1024 * 1024) / 10, "2.3 GB"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, utils.FormatBytes(tc.input))
		})
	}
}

// TestFormatTimeAgo checks the relative time formatting.
func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"Just now", now.Add(-10 * time.Second), "just now"},
		{"Minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"One hour ago", now.Add(-65 * time.Minute), "1 hour ago"},
		{"Hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"One day ago", now.Add(-25 * time.Hour), "1 day ago"},
		{"Days ago", now.Add(-10 * 24 * time.Hour), "10 days ago"},
		{"Months ago", now.Add(-70 * 24 * time.Hour), "2 months ago"},
		{"Years ago", now.Add(-2 * 366 * 24 * time.Hour), "2 years ago"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, utils.FormatTimeAgo(tc.input))
		})
	}
}
