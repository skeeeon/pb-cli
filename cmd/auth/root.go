package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"pb-cli/internal/config"
	"pb-cli/internal/pocketbase"
	"pb-cli/internal/utils"
)

var (
	pbEmail         string
	pbPassword      string
	pbCollection    string
	pbPasswordStdin bool
)

// AuthCmd represents the auth command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with PocketBase",
	Long: `Authenticate with a PocketBase instance.

PocketBase supports authentication against different collections depending on your
application's setup. Common collections include:
  users        Regular user accounts (default)
  _superusers  Superuser (admin) accounts — required for backups and 'pb schema'
  <custom>     Any custom authentication collection

The authentication will:
  1. Validate your credentials with PocketBase
  2. Store the session token securely in your context
  3. Enable access to collections and operations

Credentials are resolved in this order:
  email:    --email flag  > PB_EMAIL env    > interactive prompt
  password: --password    > --password-stdin > PB_PASSWORD env > interactive prompt

Examples:
  # Interactive authentication (prompts for credentials)
  pb auth

  # Authenticate with specific credentials
  pb auth --email user@example.com --password mypassword

  # Non-interactive / CI (no password in argv or shell history)
  PB_EMAIL=ci@example.com PB_PASSWORD=secret pb auth
  echo "$PB_PASSWORD" | pb auth --email ci@example.com --password-stdin

  # Authenticate as a superuser (needed for backups and 'pb schema')
  pb auth --collection _superusers --email admin@example.com

  # Check status or clear the stored token
  pb auth status
  pb auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		// Use collection from context if not specified, default to users
		if pbCollection == "" {
			if ctx.PocketBase.AuthCollection != "" {
				pbCollection = ctx.PocketBase.AuthCollection
			} else {
				pbCollection = config.AuthCollectionUsers
			}
		}

		// Validate collection
		if err := config.ValidateAuthCollection(pbCollection); err != nil {
			return err
		}

		// Resolve email: --email flag > PB_EMAIL env > interactive prompt.
		if pbEmail == "" {
			pbEmail = os.Getenv("PB_EMAIL")
		}
		if pbEmail == "" {
			pbEmail, err = promptForEmail()
			if err != nil {
				return fmt.Errorf("failed to get email: %w", err)
			}
		}

		// Resolve password: --password flag > --password-stdin > PB_PASSWORD env >
		// interactive prompt. This lets CI authenticate without a TTY and without
		// leaking the password into argv/shell history.
		if pbPassword == "" && pbPasswordStdin {
			pbPassword, err = readPasswordStdin()
			if err != nil {
				return fmt.Errorf("failed to read password from stdin: %w", err)
			}
		}
		if pbPassword == "" {
			pbPassword = os.Getenv("PB_PASSWORD")
		}
		if pbPassword == "" {
			pbPassword, err = promptForPassword()
			if err != nil {
				return fmt.Errorf("failed to get password: %w", err)
			}
		}

		// Basic email validation
		if pbEmail == "" || !strings.Contains(pbEmail, "@") {
			return fmt.Errorf("invalid email format")
		}

		// Create PocketBase client
		client := pocketbase.NewClient(ctx.PocketBase.URL)

		// Test connection first
		utils.PrintInfo("Testing connection to PocketBase...")
		if err := client.GetHealth(); err != nil {
			return fmt.Errorf("failed to connect to PocketBase at %s: %w", ctx.PocketBase.URL, err)
		}

		// Perform authentication
		utils.PrintInfo(fmt.Sprintf("Authenticating with collection '%s'...", pbCollection))

		authResp, err := client.Authenticate(pbCollection, pbEmail, pbPassword)
		if err != nil {
			if pbErr, ok := err.(*pocketbase.PocketBaseError); ok {
				utils.PrintError(fmt.Errorf("%s", pbErr.GetFriendlyMessage()))
				if suggestion := pbErr.GetSuggestion(); suggestion != "" {
					fmt.Printf("\nSuggestion: %s\n", suggestion)
				}
				return fmt.Errorf("authentication failed")
			}
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Update context with authentication data
		if err := pocketbase.UpdateAuthContextFromResponse(ctx, authResp); err != nil {
			return fmt.Errorf("failed to update context: %w", err)
		}

		// Update auth collection in context
		ctx.PocketBase.AuthCollection = pbCollection

		// Save updated context
		if err := configManager.SaveContext(ctx); err != nil {
			return fmt.Errorf("failed to save authentication: %w", err)
		}

		// Display success message
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Printf("\n%s Authentication successful!\n", green("✓"))

		fmt.Printf("\nAuthentication Details:\n")
		fmt.Printf("  Collection: %s\n", pocketbase.GetCollectionDisplayName(pbCollection))
		fmt.Printf("  Identity:   %s\n", pbEmail)
		if ctx.PocketBase.AuthExpires != nil {
			expiresAtFormatted := ctx.PocketBase.AuthExpires.Format("2006-01-02 15:04:05 MST")
			fmt.Printf("  Expires:    %s\n", expiresAtFormatted)
		}
		fmt.Printf("  Context:    %s\n", cyan(ctx.Name))

		if authResp.Record != nil {
			if name := getRecordDisplayName(authResp.Record, pbCollection); name != "" {
				fmt.Printf("  Name:       %s\n", name)
			}
		}

		// Show available next steps
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  List collections: %s\n", cyan("pb schema"))
		fmt.Printf("  List records:     %s\n", cyan("pb collections list <collection>"))

		return nil
	},
}

var configManager *config.Manager

func init() {
	AuthCmd.Flags().StringVarP(&pbEmail, "email", "e", "", "Email address (or set PB_EMAIL; prompts if unset)")
	AuthCmd.Flags().StringVarP(&pbPassword, "password", "p", "", "Password (insecure in shell history; prefer --password-stdin or PB_PASSWORD)")
	AuthCmd.Flags().BoolVar(&pbPasswordStdin, "password-stdin", false, "Read the password from stdin (for non-interactive/CI use)")
	AuthCmd.Flags().StringVarP(&pbCollection, "collection", "c", "", "Authentication collection (defaults to context setting or 'users')")

	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(statusCmd)
}

// SetConfigManager sets the configuration manager for the auth commands
func SetConfigManager(cm *config.Manager) {
	configManager = cm
}

// validateConfigManager ensures the config manager is available
func validateConfigManager() error {
	if configManager == nil {
		return fmt.Errorf("configuration manager not initialized")
	}
	return nil
}

// validateActiveContext ensures there's an active context
func validateActiveContext() (*config.Context, error) {
	if err := validateConfigManager(); err != nil {
		return nil, err
	}

	ctx, err := configManager.GetActiveContext()
	if err != nil {
		return nil, fmt.Errorf("no active context set. Use 'pb context select <name>' to set one")
	}

	return ctx, nil
}

// promptForEmail prompts the user for their email address
func promptForEmail() (string, error) {
	fmt.Print("Email: ")
	reader := bufio.NewReader(os.Stdin)
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(email), nil
}

// promptForPassword prompts the user for their password (hidden input)
func promptForPassword() (string, error) {
	fmt.Print("Password: ")

	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	fmt.Println()
	return string(passwordBytes), nil
}

// readPasswordStdin reads a single line (the password) from stdin.
func readPasswordStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no password provided on stdin")
}

// logoutCmd clears the stored auth token for the active context.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the stored auth token for the active context",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		if ctx.PocketBase.AuthToken == "" {
			utils.PrintInfo(fmt.Sprintf("Context '%s' is already logged out.", ctx.Name))
			return nil
		}

		ctx.PocketBase.AuthToken = ""
		ctx.PocketBase.AuthExpires = nil
		ctx.PocketBase.AuthRecord = nil

		if err := configManager.SaveContext(ctx); err != nil {
			return fmt.Errorf("failed to clear stored auth: %w", err)
		}

		utils.PrintSuccess(fmt.Sprintf("Logged out of context '%s'.", ctx.Name))
		return nil
	},
}

// statusCmd reports the authentication state for the active context.
var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"whoami"},
	Short:   "Show authentication status for the active context",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := validateActiveContext()
		if err != nil {
			return err
		}

		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("Context:    %s\n", cyan(ctx.Name))
		fmt.Printf("URL:        %s\n", ctx.PocketBase.URL)

		if ctx.PocketBase.AuthToken == "" {
			fmt.Printf("Status:     not authenticated (run 'pb auth')\n")
			return nil
		}

		collection := ctx.PocketBase.AuthCollection
		if collection == "" {
			collection = config.AuthCollectionUsers
		}
		fmt.Printf("Collection: %s\n", pocketbase.GetCollectionDisplayName(collection))

		if identity := getRecordDisplayName(ctx.PocketBase.AuthRecord, collection); identity != "" {
			fmt.Printf("Identity:   %s\n", identity)
		}

		if pocketbase.IsAuthValid(ctx) {
			fmt.Printf("Status:     %s\n", color.New(color.FgGreen).Sprint("valid"))
		} else {
			fmt.Printf("Status:     %s\n", color.New(color.FgYellow).Sprint("expired (run 'pb auth')"))
		}
		if ctx.PocketBase.AuthExpires != nil {
			fmt.Printf("Expires:    %s\n", ctx.PocketBase.AuthExpires.Format("2006-01-02 15:04:05 MST"))
		}

		return nil
	},
}

// getRecordDisplayName returns a human-readable display name for a record
func getRecordDisplayName(record map[string]interface{}, collection string) string {
	nameFields := []string{"name", "full_name", "display_name", "title"}
	for _, field := range nameFields {
		if name, ok := record[field].(string); ok && name != "" {
			return name
		}
	}

	if firstName, ok := record["first_name"].(string); ok {
		if lastName, ok := record["last_name"].(string); ok {
			if firstName != "" && lastName != "" {
				return firstName + " " + lastName
			} else if firstName != "" {
				return firstName
			} else if lastName != "" {
				return lastName
			}
		}
	}

	if username, ok := record["username"].(string); ok && username != "" {
		return username
	}

	if email, ok := record["email"].(string); ok && email != "" {
		return email
	}
	if id, ok := record["id"].(string); ok {
		return id
	}

	return ""
}
