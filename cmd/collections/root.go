package collections

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
)

var outputFlag string

// CollectionsCmd represents the collections command
var CollectionsCmd = &cobra.Command{
	Use:     "collections",
	Aliases: []string{"c"},
	Short:   "Manage PocketBase collections",
	Long: `Perform CRUD operations on PocketBase collections.

Usage Pattern:
  pb collections <action> <collection> [args] [flags]

Actions:
  list     List records from a collection with filtering and pagination
  get      Get a single record by ID
  create   Create a new record from JSON data or file
  update   Update an existing record with JSON data or file
  delete   Delete a record with confirmation

Data for 'create' and 'update' actions can be provided in one of three ways:
  1. As a JSON string argument
  2. From a file using the --file flag
  3. Piped from stdin

Examples:
  pb collections list posts
  pb collections list posts --filter 'published=true' --sort '-created'
  pb collections get users user_abc123 --expand profile
  pb collections create posts '{"title":"My Post","content":"Hello world"}'
  pb collections update posts post_123 '{"published":true}'
  pb collections delete users user_456 --force

  # Short alias
  pb c list posts
  pb c get posts post_123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("missing subcommand. Available: list, get, create, update, delete")
	},
}

var configManager *config.Manager

func init() {
	CollectionsCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format (json|yaml|table)")

	CollectionsCmd.AddCommand(listCmd)
	CollectionsCmd.AddCommand(getCmd)
	CollectionsCmd.AddCommand(createCmd)
	CollectionsCmd.AddCommand(updateCmd)
	CollectionsCmd.AddCommand(deleteCmd)
}

// SetConfigManager sets the configuration manager for the collections commands
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// getOutputFormat returns the effective output format
func getOutputFormat() string {
	if outputFlag != "" {
		return outputFlag
	}
	return config.Global.OutputFormat
}

// validateConfigManager ensures the config manager is available
func validateConfigManager() error {
	if configManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return nil
}

// validateActiveContext ensures there's an active context with authentication
func validateActiveContext() (*config.Context, error) {
	if err := validateConfigManager(); err != nil {
		return nil, err
	}

	ctx, err := configManager.GetActiveContext()
	if err != nil {
		return nil, fmt.Errorf("no active context set. Use 'pb context select <name>' to set one")
	}

	if ctx.PocketBase.AuthToken == "" {
		return nil, fmt.Errorf("authentication required. Run 'pb auth' to authenticate")
	}

	if !pocketbase.IsAuthValid(ctx) {
		return nil, fmt.Errorf("authentication has expired. Run 'pb auth' to re-authenticate")
	}

	return ctx, nil
}

// validateCollection validates that the collection is available in the current context
func validateCollection(collection string) (*config.Context, error) {
	ctx, err := validateActiveContext()
	if err != nil {
		return nil, err
	}

	if err := validateCollectionInContext(collection, ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

// createPocketBaseClient creates an authenticated PocketBase client from context
func createPocketBaseClient(ctx *config.Context) *pocketbase.Client {
	return pocketbase.NewClientFromContext(ctx)
}

// parseJSONInput parses JSON input from a file, string argument, or stdin.
// Precedence: file > argument > stdin
func parseJSONInput(jsonStr, filePath string) (map[string]interface{}, error) {
	var jsonData []byte
	var err error

	if filePath != "" {
		jsonData, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
	} else if jsonStr != "" {
		jsonData = []byte(jsonStr)
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			jsonData, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("failed to read from stdin: %w", err)
			}
		}
	}

	if len(jsonData) == 0 {
		return nil, fmt.Errorf("JSON data is required either from an argument, the --file flag, or piped from stdin")
	}

	return validateAndParseJSON(string(jsonData))
}

// validateAndParseJSON validates JSON format and parses to map
func validateAndParseJSON(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, fmt.Errorf("JSON data cannot be empty")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	return data, nil
}
