package context

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
)

var showOutputFormat string

var showCmd = &cobra.Command{
	Use:   "show [n]",
	Short: "Show detailed context configuration",
	Long: `Display detailed configuration for a specific context or the active context.

If no context name is provided, shows the currently active context.
The output format can be controlled with the --output flag.

The context information includes the directory location, configuration details,
and authentication status.

Examples:
  pb context show                    # Show active context
  pb context show production         # Show specific context
  pb context show prod --output yaml # Show in YAML format`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		var contextName string
		var ctx *config.Context
		var err error

		// Determine which context to show
		if len(args) == 0 {
			// Show active context
			ctx, err = configManager.GetActiveContext()
			if err != nil {
				return fmt.Errorf("no active context set. Use 'pb context select <n>' to set one")
			}
			contextName = ctx.Name
		} else {
			// Show specified context
			contextName = args[0]
			ctx, err = configManager.LoadContext(contextName)
			if err != nil {
				// Try to provide helpful suggestions
				contexts, listErr := configManager.ListContexts()
				if listErr == nil && len(contexts) > 0 {
					return fmt.Errorf("context '%s' not found. Available contexts: %v",
						contextName, contexts)
				}
				return fmt.Errorf("context '%s' not found", contextName)
			}
		}

		// Check if it's the active context
		globalConfig, err := configManager.LoadGlobalConfig()
		if err != nil {
			return fmt.Errorf("failed to load global config: %w", err)
		}

		isActive := globalConfig.ActiveContext == contextName

		// Create a display version of the context (hide sensitive data)
		displayCtx := *ctx
		if displayCtx.PocketBase.AuthToken != "" {
			displayCtx.PocketBase.AuthToken = "***HIDDEN***"
		}

		// Output based on format
		switch strings.ToLower(showOutputFormat) {
		case "json":
			output, err := json.MarshalIndent(displayCtx, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal context to JSON: %w", err)
			}
			fmt.Println(string(output))

		case "yaml":
			output, err := yaml.Marshal(displayCtx)
			if err != nil {
				return fmt.Errorf("failed to marshal context to YAML: %w", err)
			}
			fmt.Print(string(output))

		case "table", "":
			// Default table format
			showContextTable(ctx, isActive, configManager)

		default:
			return fmt.Errorf("invalid output format '%s'. Valid formats: json, yaml, table",
				showOutputFormat)
		}

		return nil
	},
}

func showContextTable(ctx *config.Context, isActive bool, configManager *config.Manager) {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	// Header
	fmt.Printf("%s Context: %s", bold("PocketBase"), cyan(ctx.Name))
	if isActive {
		fmt.Printf(" %s", green("(ACTIVE)"))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))

	// Show context directory
	contextDir := configManager.GetContextDir(ctx.Name)
	fmt.Printf("Context Directory: %s\n\n", contextDir)

	// PocketBase Configuration
	fmt.Printf("%s\n", bold("PocketBase Configuration:"))
	fmt.Printf("  URL:                %s\n", ctx.PocketBase.URL)
	fmt.Printf("  Auth Collection:    %s\n", ctx.PocketBase.AuthCollection)

	// --- START: CORRECTED AUTHENTICATION STATUS LOGIC ---
	// Authentication status
	if ctx.PocketBase.AuthToken != "" {
		// Use the actual validation logic here
		if pocketbase.IsAuthValid(ctx) {
			expirationInfo := ""
			if ctx.PocketBase.AuthExpires != nil {
				expirationInfo = fmt.Sprintf(" (expires %s)", ctx.PocketBase.AuthExpires.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  Authentication:     %s%s\n", green("Valid"), expirationInfo)
		} else {
			expirationInfo := ""
			if ctx.PocketBase.AuthExpires != nil {
				// Use a more descriptive "expired on" for clarity
				expirationInfo = fmt.Sprintf(" (expired on %s)", ctx.PocketBase.AuthExpires.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  Authentication:     %s%s\n", red("Expired"), expirationInfo)
		}
	} else {
		fmt.Printf("  Authentication:     %s\n", yellow("Not Authenticated"))
	}
	// --- END: CORRECTED AUTHENTICATION STATUS LOGIC ---

	// Available collections
	fmt.Printf("  Available Collections: %d\n", len(ctx.PocketBase.AvailableCollections))
	if len(ctx.PocketBase.AvailableCollections) > 0 {
		fmt.Printf("    %s\n", strings.Join(ctx.PocketBase.AvailableCollections, ", "))
	} else {
		fmt.Printf("    %s\n", yellow("None configured"))
	}

	fmt.Println()

	// Show helpful commands
	if !isActive {
		fmt.Printf("%s\n", bold("Commands:"))
		fmt.Printf("  Select this context: %s\n",
			cyan(fmt.Sprintf("pb context select %s", ctx.Name)))
	} else if ctx.PocketBase.AuthToken == "" || !pocketbase.IsAuthValid(ctx) { // Prompt for auth if not authenticated OR expired
		fmt.Printf("%s\n", bold("Next Steps:"))
		fmt.Printf("  Authenticate: %s\n", cyan("pb auth"))
		if len(ctx.PocketBase.AvailableCollections) == 0 {
			fmt.Printf("  Add collections: %s\n", cyan("pb context collections add <collection_names>"))
		}
	} else if len(ctx.PocketBase.AvailableCollections) == 0 {
		fmt.Printf("%s\n", bold("Next Steps:"))
		fmt.Printf("  Add collections: %s\n", cyan("pb context collections add <collection_names>"))
	} else {
		fmt.Printf("%s\n", bold("Available Operations:"))
		// Show a few example collection operations
		for i, collection := range ctx.PocketBase.AvailableCollections {
			if i >= 3 { // Limit to first 3 collections
				break
			}
			fmt.Printf("  %s\n", cyan(fmt.Sprintf("pb collections %s list", collection)))
		}
		if len(ctx.PocketBase.AvailableCollections) > 3 {
			fmt.Printf("  ... and %d more collections\n", len(ctx.PocketBase.AvailableCollections)-3)
		}
	}
}

func init() {
	showCmd.Flags().StringVarP(&showOutputFormat, "output", "o", "table",
		"Output format (table|json|yaml)")
}
