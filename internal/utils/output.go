package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"pb-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// OutputData formats and prints data according to the specified format
func OutputData(data interface{}, format string) error {
	switch strings.ToLower(format) {
	case config.OutputFormatJSON, "":
		return outputJSON(data)
	case config.OutputFormatYAML:
		return outputYAML(data)
	case config.OutputFormatTable:
		return outputTable(data)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// outputJSON prints data in JSON format
func outputJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

// outputYAML prints data in YAML format
func outputYAML(data interface{}) error {
	output, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(output))
	return nil
}

// outputTable prints data in table format
func outputTable(data interface{}) error {
	switch v := data.(type) {
	case []map[string]interface{}:
		return outputMapSliceTable(v)
	case map[string]interface{}:
		return outputMapTable(v)
	default:
		// Fallback to JSON for complex types
		return outputJSON(data)
	}
}

// outputMapSliceTable outputs a slice of maps as a table
func outputMapSliceTable(data []map[string]interface{}) error {
	if len(data) == 0 {
		fmt.Println("No data found.")
		return nil
	}

	// Extract headers from first item, prioritizing common fields
	var headers []string
	commonFields := []string{"id", "name", "title", "email", "created", "updated"}
	
	// Add common fields first if they exist
	firstItem := data[0]
	for _, field := range commonFields {
		if _, exists := firstItem[field]; exists {
			headers = append(headers, field)
		}
	}
	
	// Add remaining fields
	for key := range firstItem {
		found := false
		for _, existing := range headers {
			if existing == key {
				found = true
				break
			}
		}
		if !found {
			headers = append(headers, key)
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowSeparator("")
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	// Add rows
	for _, item := range data {
		var row []string
		for _, header := range headers {
			value := formatTableValue(item[header])
			row = append(row, value)
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

// outputMapTable outputs a single map as a vertical table
func outputMapTable(data map[string]interface{}) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Field", "Value"})
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowSeparator("")
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	// Sort fields for consistent output
	priorityFields := []string{"id", "name", "title", "email", "description", "type", "created", "updated"}
	var orderedKeys []string
	
	// Add priority fields first
	for _, field := range priorityFields {
		if _, exists := data[field]; exists {
			orderedKeys = append(orderedKeys, field)
		}
	}
	
	// Add remaining fields
	for key := range data {
		found := false
		for _, existing := range orderedKeys {
			if existing == key {
				found = true
				break
			}
		}
		if !found {
			orderedKeys = append(orderedKeys, key)
		}
	}

	for _, key := range orderedKeys {
		value := formatTableValue(data[key])
		table.Append([]string{TitleCase(key), value})
	}

	table.Render()
	return nil
}

// formatTableValue formats a value for table display
func formatTableValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		// Truncate very long strings for table display
		if len(v) > 50 {
			return v[:47] + "..."
		}
		return v
	case bool:
		if v {
			return "✓"
		}
		return "✗"
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		if len(v) == 1 {
			return fmt.Sprintf("[%s]", formatTableValue(v[0]))
		}
		return fmt.Sprintf("[%s, ... (%d items)]", formatTableValue(v[0]), len(v))
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{...} (%d fields)", len(v))
	default:
		str := fmt.Sprintf("%v", value)
		if len(str) > 50 {
			return str[:47] + "..."
		}
		return str
	}
}

// PrintError prints an error message with consistent formatting
func PrintError(err error) {
	if !config.Global.ColorsEnabled {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	red := color.New(color.FgRed).SprintFunc()
	fmt.Fprintf(os.Stderr, "%s %v\n", red("Error:"), err)
}

// PrintWarning prints a warning message with consistent formatting
func PrintWarning(message string) {
	if !config.Global.ColorsEnabled {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", message)
		return
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Fprintf(os.Stderr, "%s %s\n", yellow("Warning:"), message)
}

// PrintSuccess prints a success message with consistent formatting
func PrintSuccess(message string) {
	if !config.Global.ColorsEnabled {
		fmt.Printf("Success: %s\n", message)
		return
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("✓"), message)
}

// PrintInfo prints an info message with consistent formatting
func PrintInfo(message string) {
	if !config.Global.ColorsEnabled {
		fmt.Printf("Info: %s\n", message)
		return
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s %s\n", cyan("ℹ"), message)
}

// PrintDebug prints a debug message if debug mode is enabled
func PrintDebug(message string) {
	if !config.Global.Debug {
		return
	}

	if !config.Global.ColorsEnabled {
		fmt.Fprintf(os.Stderr, "Debug: %s\n", message)
		return
	}

	gray := color.New(color.FgHiBlack).SprintFunc()
	fmt.Fprintf(os.Stderr, "%s %s\n", gray("Debug:"), message)
}

// TruncateString truncates a string to the specified length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatDuration formats a duration string in a human-readable way
func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// TitleCase converts a string to title case (first letter uppercase)
func TitleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ToJSON converts data to JSON bytes
func ToJSON(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// FormatJSON formats JSON bytes for pretty printing
func FormatJSON(data []byte) (string, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", err
	}
	
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(formatted), nil
}
