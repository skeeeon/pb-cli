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
	pbEmail      string
	pbPassword   string
	pbCollection string
)

var pbCmd = &cobra.Command{
	Use:   "pb",
	Short: "Authenticate with PocketBase",
	Long: `Authenticate with a PocketBase instance.

PocketBase handles all database operations and API access. You can authenticate
using different collections depending on your application setup:

Collections:
  users        Regular user accounts (default)
  admins       Administrative accounts
  clients      API client accounts
  <custom>     Any custom authentication collection

The authentication will:
1. Validate your credentials with PocketBase
2. Store the session token securely in your context
3. Enable access to collections and operations

Examples:
  # Interactive authentication (prompts for credentials)
  pb auth

  # Authenticate with specific credentials
  pb auth --email user@example.com --password mypassword

  # Authenticate against admin collection
  pb auth --collection admins --email admin@example.com

  # Authenticate against custom collection
  pb auth --collection customers --email customer@example.com`,
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

		// Get credentials if not provided
		if pbEmail == "" {
			pbEmail, err = promptForEmail()
			if err != nil {
				return fmt.Errorf("failed to get email: %w", err)
			}
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

		// --- START: MODIFIED AUTHENTICATION DETAILS BLOCK ---
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
		// --- END: MODIFIED AUTHENTICATION DETAILS BLOCK ---


		// Show available next steps
		fmt.Printf("\nNext steps:\n")

		// Show collections management if no collections configured
		if len(ctx.PocketBase.AvailableCollections) == 0 {
			fmt.Printf("  Configure collections: %s\n",
				cyan("pb context collections add <collection_names>"))
		} else {
			fmt.Printf("  List available collections: %s\n",
				cyan("pb context collections list"))

			// Show example collection operation
			if len(ctx.PocketBase.AvailableCollections) > 0 {
				firstCollection := ctx.PocketBase.AvailableCollections[0]
				fmt.Printf("  Example operation: %s\n",
					cyan(fmt.Sprintf("pb collections %s list", firstCollection)))
			}
		}

		return nil
	},
}

func init() {
	pbCmd.Flags().StringVarP(&pbEmail, "email", "e", "", "Email address for authentication")
	pbCmd.Flags().StringVarP(&pbPassword, "password", "p", "", "Password for authentication (will prompt if not provided)")
	pbCmd.Flags().StringVarP(&pbCollection, "collection", "c", "", "Authentication collection (defaults to context setting or 'users')")
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

	// Hide password input
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	fmt.Println() // New line after hidden input
	return string(passwordBytes), nil
}

// getRecordDisplayName returns a human-readable display name for a record
func getRecordDisplayName(record map[string]interface{}, collection string) string {
	// Try common name fields
	nameFields := []string{"name", "full_name", "display_name", "title"}
	for _, field := range nameFields {
		if name, ok := record[field].(string); ok && name != "" {
			return name
		}
	}

	// Try combining first/last name
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

	// Try username
	if username, ok := record["username"].(string); ok && username != "" {
		return username
	}

	// Fallback to email or ID
	if email, ok := record["email"].(string); ok && email != "" {
		return email
	}
	if id, ok := record["id"].(string); ok {
		return id
	}

	return ""
}
