package collections

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/resolver"
)

// Global flag variables
var (
	// List flags
	offsetFlag int
	limitFlag  int
	filterFlag string
	sortFlag   string
	fieldsFlag []string
	expandFlag []string

	// Create/Update flags
	fileFlag string

	// Delete flags
	forceFlag bool
	quietFlag bool

	// Common flags
	outputFlag string
)

// CollectionsCmd represents the collections command
var CollectionsCmd = &cobra.Command{
	Use:   "collections <collection> <action> [args]",
	Short: "Manage PocketBase collections",
	Long: `Perform CRUD operations on PocketBase collections.

Collections are the data entities in your PocketBase instance. This command provides
full CRUD (Create, Read, Update, Delete) operations for any collection that you have
configured in your active context.

The available collections are managed through your context configuration. Use
'pb context collections' commands to configure which collections are available.

Usage Pattern:
  pb collections <collection> <action> [args] [flags]

Data for 'create' and 'update' actions can be provided in one of three ways:
1. As a JSON string argument:
   pb collections posts create '{"title":"My Post"}'

2. From a file using the --file flag:
   pb collections posts create --file post.json

3. Piped from stdin (standard input):
   cat post.json | pb collections posts create
   pb collections posts get <id> | jq '.title="New"' | pb collections posts update <id>

Examples:
  # List all posts
  pb collections posts list

  # List with filtering and custom fields
  pb collections posts list --filter 'published=true' --fields title,content,created

  # Get a specific user by ID with expanded relations
  pb collections users get user_abc123 --expand profile

  # Create a new post from JSON
  pb collections posts create '{"title":"My Post","content":"Hello world","published":true}'

  # Update a post
  pb collections posts update post_123 '{"published":true}' --output table

  # Delete a user with confirmation skip
  pb collections users delete user_456 --force

Available Actions:
  list     List records from a collection with filtering and pagination
  get      Get a single record by ID with optional expansion
  create   Create a new record from JSON data or file
  update   Update an existing record with JSON data or file
  delete   Delete a record with confirmation

Note: All operations require authentication with PocketBase and appropriate
permissions for the target collection. Collections must be configured in
your active context before use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Need at least collection and action
		if len(args) < 2 {
			return fmt.Errorf("missing required arguments: <collection> <action>")
		}

		collection := args[0]
		action := args[1]
		actionArgs := args[2:] // Remaining args for the action

		// Validate collection against context
		ctx, err := validateCollection(collection)
		if err != nil {
			return err
		}

		// Resolve partial action matching
		resolvedAction, err := resolveAction(action)
		if err != nil {
			return err
		}

		// Route to appropriate action handler
		return routeToAction(ctx, collection, resolvedAction, actionArgs)
	},
}

var (
	configManager *config.Manager
	cmdResolver   *resolver.CommandResolver
)

func init() {
	// Register all possible flags for collections commands
	// List-specific flags
	CollectionsCmd.Flags().IntVar(&offsetFlag, "offset", 0, "Number of records to skip (for pagination)")
	CollectionsCmd.Flags().IntVar(&limitFlag, "limit", 30, "Maximum number of records to return")
	CollectionsCmd.Flags().StringVar(&filterFlag, "filter", "", "PocketBase filter expression (e.g., 'published=true && title~\"test\"')")
	CollectionsCmd.Flags().StringVar(&sortFlag, "sort", "", "Sort expression (e.g., 'title', '-created', 'title,-updated')")
	CollectionsCmd.Flags().StringSliceVar(&fieldsFlag, "fields", nil, "Specific fields to return (comma-separated)")
	CollectionsCmd.Flags().StringSliceVar(&expandFlag, "expand", nil, "Relations to expand (comma-separated)")

	// Create/Update flags
	CollectionsCmd.Flags().StringVar(&fileFlag, "file", "", "Path to JSON file containing record data")

	// Delete flags
	CollectionsCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Skip confirmation prompt")
	CollectionsCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress success messages")

	// Common flags
	CollectionsCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output format (json|yaml|table)")
}

// SetConfigManager sets the configuration manager for the collections commands
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// SetCommandResolver sets the command resolver for partial matching
func SetCommandResolver(cr *resolver.CommandResolver) {
	cmdResolver = cr
}

// validateConfigManager ensures the config manager is available
func validateConfigManager() error {
	if configManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return nil
}

// validateCommandResolver ensures the command resolver is available
func validateCommandResolver() error {
	if cmdResolver == nil {
		return fmt.Errorf("command resolver not initialized")
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
		return nil, fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
	}

	// Check authentication
	if ctx.PocketBase.AuthToken == "" {
		return nil, fmt.Errorf("authentication required. Run 'pb auth' to authenticate")
	}

	// Check if authentication is still valid
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

	if err := validateCommandResolver(); err != nil {
		return nil, err
	}

	// Validate collection against context's available collections
	if err := cmdResolver.ValidateCollection(collection, ctx.PocketBase.AvailableCollections); err != nil {
		return nil, err
	}

	return ctx, nil
}

// resolveAction resolves a partial action command to its full form
func resolveAction(partialAction string) (string, error) {
	if err := validateCommandResolver(); err != nil {
		return "", err
	}

	return cmdResolver.ResolveCommand("collections", partialAction)
}

// createPocketBaseClient creates an authenticated PocketBase client from context
func createPocketBaseClient(ctx *config.Context) *pocketbase.Client {
	return pocketbase.NewClientFromContext(ctx)
}

// routeToAction routes the command to the appropriate action handler
func routeToAction(ctx *config.Context, collection, action string, args []string) error {
	switch action {
	case "list":
		return handleListAction(ctx, collection, args)
	case "get":
		return handleGetAction(ctx, collection, args)
	case "create":
		return handleCreateAction(ctx, collection, args)
	case "update":
		return handleUpdateAction(ctx, collection, args)
	case "delete":
		return handleDeleteAction(ctx, collection, args)
	default:
		return fmt.Errorf("unknown action '%s'. Available actions: list, get, create, update, delete", action)
	}
}

// --- START: MODIFIED FUNCTION ---
// parseJSONInput parses JSON input from a file, string argument, or stdin.
// Precedence: file > argument > stdin
func parseJSONInput(jsonStr, filePath string) (map[string]interface{}, error) {
	var jsonData []byte
	var err error

	if filePath != "" {
		// 1. Read from file (highest precedence)
		jsonData, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
	} else if jsonStr != "" {
		// 2. Use string argument
		jsonData = []byte(jsonStr)
	} else {
		// 3. Try to read from stdin (lowest precedence)
		stat, _ := os.Stdin.Stat()
		// Check if data is being piped or redirected
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

	// Validate and parse JSON
	return validateAndParseJSON(string(jsonData))
}
// --- END: MODIFIED FUNCTION ---

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

// readJSONFile is no longer needed as its logic is merged into parseJSONInput
// func readJSONFile(filePath string) (string, error) { ... }
