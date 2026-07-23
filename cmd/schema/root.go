package schema

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var outputFlag string

// SchemaCmd represents the schema command
var SchemaCmd = &cobra.Command{
	Use:   "schema [collection]",
	Short: "Inspect PocketBase collections and their fields",
	Long: `Inspect the collections defined on the active PocketBase instance.

With no argument, lists every collection. With a collection name, shows that
collection's fields and access rules.

Reading collection definitions is a superuser-only operation in PocketBase, so the
active context must be authenticated as a superuser:
  pb auth --collection _superusers

Examples:
  pb schema                 # List all collections
  pb schema posts           # Show the schema for the 'posts' collection
  pb schema posts -o json   # Same, as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		client := pocketbase.NewClientFromContext(ctx)

		if len(args) == 0 {
			return listCollections(client)
		}
		return showCollection(client, args[0])
	},
}

var configManager *config.Manager

func init() {
	SchemaCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output format (json|yaml|table)")
}

// SetConfigManager sets the configuration manager for the schema command
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// getOutputFormat returns the effective output format (flag, else global default).
func getOutputFormat() string {
	if outputFlag != "" {
		return outputFlag
	}
	return config.Global.OutputFormat
}

// validateActiveContext ensures there's an active, authenticated context.
func validateActiveContext() (*config.Context, error) {
	if configManager == nil {
		return nil, fmt.Errorf("configuration manager not initialized")
	}

	ctx, err := configManager.GetActiveContext()
	if err != nil {
		return nil, fmt.Errorf("no active context set. Use 'pb context select <name>' to set one")
	}

	if ctx.PocketBase.AuthToken == "" {
		return nil, fmt.Errorf("authentication required. Run 'pb auth' to authenticate")
	}

	if err := pocketbase.EnsureFreshAuth(ctx, configManager); err != nil {
		return nil, err
	}

	if !pocketbase.IsAuthValid(ctx) {
		return nil, fmt.Errorf("authentication has expired. Run 'pb auth' to re-authenticate")
	}

	return ctx, nil
}

// listCollections prints every collection on the instance.
func listCollections(client *pocketbase.Client) error {
	collections, err := client.GetCollections()
	if err != nil {
		return superuserError(err, "read collections")
	}

	switch getOutputFormat() {
	case config.OutputFormatJSON, config.OutputFormatYAML:
		return utils.OutputData(collections, getOutputFormat())
	}

	if len(collections) == 0 {
		fmt.Println("No collections found.")
		return nil
	}

	table := newTable("NAME", "TYPE", "FIELDS")
	for _, c := range collections {
		table.Append([]string{c.Name, c.Type, fmt.Sprintf("%d", len(c.Fields))})
	}
	fmt.Printf("Collections (%d):\n", len(collections))
	table.Render()
	return nil
}

// showCollection prints the fields and rules for a single collection.
func showCollection(client *pocketbase.Client, name string) error {
	collection, err := client.GetCollectionSchema(name)
	if err != nil {
		return superuserError(err, "read collection schema")
	}

	switch getOutputFormat() {
	case config.OutputFormatJSON, config.OutputFormatYAML:
		return utils.OutputData(collection, getOutputFormat())
	}

	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Collection: %s (%s)\n", bold("PocketBase"), collection.Name, collection.Type)

	table := newTable("FIELD", "TYPE", "REQUIRED")
	for _, f := range collection.Fields {
		table.Append([]string{f.Name, f.Type, yesNo(f.Required)})
	}
	table.Render()

	printRules(collection)
	return nil
}

// printRules shows the access rules. A nil rule means "superusers only"; an empty
// (non-nil) rule means "public".
func printRules(c *pocketbase.Collection) {
	fmt.Printf("\nAccess rules:\n")
	rules := []struct {
		label string
		rule  *string
	}{
		{"list", c.ListRule},
		{"view", c.ViewRule},
		{"create", c.CreateRule},
		{"update", c.UpdateRule},
		{"delete", c.DeleteRule},
	}
	for _, r := range rules {
		fmt.Printf("  %-7s %s\n", r.label+":", formatRule(r.rule))
	}
}

func formatRule(rule *string) string {
	if rule == nil {
		return "(superusers only)"
	}
	if *rule == "" {
		return "(public)"
	}
	return *rule
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// superuserError adds a superuser hint for the 401/403 that collection endpoints
// return to non-superusers, and otherwise surfaces the friendly PocketBase message.
func superuserError(err error, action string) error {
	if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
		if pbErr.StatusCode == 401 || pbErr.StatusCode == 403 {
			utils.PrintError(fmt.Errorf("reading collection definitions requires superuser access"))
			fmt.Fprintln(os.Stderr, "\nSuggestion: authenticate as a superuser with 'pb auth --collection _superusers'")
			return fmt.Errorf("failed to %s", action)
		}
		utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
		if suggestion := pbErr.GetSuggestion(); suggestion != "" {
			fmt.Fprintf(os.Stderr, "\nSuggestion: %s\n", suggestion)
		}
		return fmt.Errorf("failed to %s", action)
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}

// newTable builds a borderless table matching the style used elsewhere in the CLI.
func newTable(headers ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowSeparator("")
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	return table
}
