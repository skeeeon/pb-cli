package context

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"pb-cli/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available contexts",
	Long: `List all configured PocketBase contexts with their status and configuration details.

The currently active context is highlighted with an asterisk (*).

Each context is stored in its own directory within the pb configuration directory,
containing the context configuration file.

Examples:
  pb context list
  pb context ls`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateConfigManager(); err != nil {
			return err
		}

		// Get all contexts
		contexts, err := configManager.ListContexts()
		if err != nil {
			return fmt.Errorf("failed to list contexts: %w", err)
		}

		if len(contexts) == 0 {
			fmt.Printf("No contexts configured in %s.\n", configManager.GetConfigDir())
			fmt.Printf("\nCreate your first context:\n  %s\n", 
				color.New(color.FgCyan).Sprint("pb context create <n> --url <url> --collections <collections>"))
			return nil
		}

		// Get active context
		globalConfig, err := configManager.LoadGlobalConfig()
		if err != nil {
			return fmt.Errorf("failed to load global config: %w", err)
		}

		// Process contexts and display
		displayContextsTable(contexts, globalConfig.ActiveContext)

		// Show active context summary
		if globalConfig.ActiveContext != "" {
			fmt.Printf("\nActive context: %s\n", 
				color.New(color.FgCyan).Sprint(globalConfig.ActiveContext))
		} else {
			fmt.Printf("\nNo active context set. Use %s to select one.\n", 
				color.New(color.FgCyan).Sprint("pb context select <n>"))
		}

		return nil
	},
}

// ContextDisplayInfo holds processed context information for display
type ContextDisplayInfo struct {
	Name          string
	Status        string
	PocketBaseURL string
	Collections   string
	AuthCollection string
	LastAuth      string
	IsActive      bool
	HasError      bool
}

// displayContextsTable processes contexts and displays them in a properly formatted table
func displayContextsTable(contextNames []string, activeContext string) {
	// Process all contexts first
	var contexts []ContextDisplayInfo
	for _, name := range contextNames {
		ctx := processContextForDisplay(name, activeContext)
		contexts = append(contexts, ctx)
	}

	// Create and configure table
	table := createContextTable()
	
	// Add rows to table
	for _, ctx := range contexts {
		table.Append([]string{
			ctx.Name,
			ctx.Status,
			ctx.PocketBaseURL,
			ctx.AuthCollection,
			ctx.Collections,
			ctx.LastAuth,
		})
	}

	fmt.Printf("PocketBase Contexts (stored in %s):\n", configManager.GetConfigDir())
	table.Render()
}

// processContextForDisplay loads and processes a single context for display
func processContextForDisplay(contextName, activeContext string) ContextDisplayInfo {
	ctx, err := configManager.LoadContext(contextName)
	if err != nil {
		return ContextDisplayInfo{
			Name:          contextName,
			Status:        color.New(color.FgRed).Sprint("ERROR"),
			PocketBaseURL: "N/A",
			Collections:   "N/A",
			AuthCollection: "N/A", 
			LastAuth:      "N/A",
			HasError:      true,
		}
	}

	isActive := activeContext == contextName
	
	return ContextDisplayInfo{
		Name:          formatContextName(contextName, isActive),
		Status:        formatContextStatus(ctx, isActive),
		PocketBaseURL: formatPocketBaseURL(ctx.PocketBase.URL),
		AuthCollection: formatAuthCollection(ctx.PocketBase.AuthCollection),
		Collections:   formatCollections(ctx.PocketBase.AvailableCollections),
		LastAuth:      formatLastAuth(ctx),
		IsActive:      isActive,
		HasError:      false,
	}
}

// createContextTable creates and configures the table with proper column settings
func createContextTable() *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	
	// Set headers
	table.SetHeader([]string{"NAME", "STATUS", "POCKETBASE URL", "AUTH COLLECTION", "COLLECTIONS", "LAST AUTH"})
	
	// Configure table appearance - no borders for clean look
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowSeparator("")
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	
	// Prevent text wrapping and set wider table width
	table.SetColWidth(150)
	table.SetAutoWrapText(false) // Critical: disable auto-wrapping
	
	// Set minimum column widths for better formatting
	table.SetColMinWidth(0, 12)  // NAME column
	table.SetColMinWidth(1, 18)  // STATUS column
	table.SetColMinWidth(2, 25)  // POCKETBASE URL column
	table.SetColMinWidth(3, 15)  // AUTH COLLECTION column
	table.SetColMinWidth(4, 20)  // COLLECTIONS column
	table.SetColMinWidth(5, 12)  // LAST AUTH column
	
	return table
}

// formatContextName formats the context name with active indicator
func formatContextName(name string, isActive bool) string {
	if isActive {
		return color.New(color.FgCyan).Sprint("* " + name)
	}
	return name
}

// formatContextStatus formats the authentication status
func formatContextStatus(ctx *config.Context, isActive bool) string {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	
	hasAuth := ctx.PocketBase.AuthToken != ""
	
	if isActive && hasAuth {
		return green("Active & Authenticated")
	} else if isActive && !hasAuth {
		return yellow("Active (Not Authenticated)")
	} else if !isActive && hasAuth {
		return green("Authenticated")
	} else {
		return yellow("Not Authenticated")
	}
}

// formatPocketBaseURL formats the PocketBase URL for display
func formatPocketBaseURL(url string) string {
	// Truncation for better readability
	if len(url) > 35 {
		return url[:32] + "..."
	}
	return url
}

// formatAuthCollection formats auth collection for display
func formatAuthCollection(authCollection string) string {
	if authCollection == "" {
		return color.New(color.FgYellow).Sprint("users")
	}
	return authCollection
}

// formatCollections formats collections information for display
func formatCollections(collections []string) string {
	if len(collections) == 0 {
		return color.New(color.FgHiBlack).Sprint("none")
	}

	if len(collections) == 1 {
		return collections[0]
	}

	if len(collections) <= 3 {
		return strings.Join(collections, ", ")
	}

	// Show first 2 collections with count indicator
	return fmt.Sprintf("%s, %s (+%d more)", 
		collections[0], collections[1], len(collections)-2)
}

// formatLastAuth formats the last authentication time
func formatLastAuth(ctx *config.Context) string {
	if ctx.PocketBase.AuthExpires == nil {
		return color.New(color.FgHiBlack).Sprint("Never")
	}
	return ctx.PocketBase.AuthExpires.Format("01-02 15:04")
}
