package pocketbase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"pb-cli/internal/config"
	"pb-cli/internal/utils"
)

// AuthResponse represents a PocketBase authentication response
type AuthResponse struct {
	Token  string                 `json:"token"`
	Record map[string]interface{} `json:"record"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

// AuthRequest represents authentication request data
type AuthRequest struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

// Authenticate performs authentication against a specific collection
func (c *Client) Authenticate(collection, identity, password string) (*AuthResponse, error) {
	// Validate collection
	if err := config.ValidateAuthCollection(collection); err != nil {
		return nil, fmt.Errorf("invalid auth collection: %w", err)
	}

	// Basic validation
	if identity == "" {
		return nil, fmt.Errorf("identity (email/username) is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Prepare authentication request
	authData := AuthRequest{
		Identity: identity,
		Password: password,
	}

	// Make authentication request
	endpoint := fmt.Sprintf("collections/%s/auth-with-password", collection)

	utils.PrintDebug(fmt.Sprintf("Authenticating with collection: %s", collection))

	resp, err := c.makeRequest("POST", endpoint, authData)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Parse response
	var authResp AuthResponse
	if err := json.Unmarshal(resp.Body(), &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse authentication response: %w", err)
	}

	// Set authentication token
	c.SetAuthToken(authResp.Token)
	c.authRecord = authResp.Record

	utils.PrintDebug("Authentication successful")

	return &authResp, nil
}

// RefreshAuth refreshes the current authentication token
func (c *Client) RefreshAuth(collection string) (*AuthResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	endpoint := fmt.Sprintf("collections/%s/auth-refresh", collection)

	utils.PrintDebug("Refreshing authentication token")

	resp, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh authentication: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(resp.Body(), &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Update authentication
	c.SetAuthToken(authResp.Token)
	c.authRecord = authResp.Record

	utils.PrintDebug("Authentication refreshed successfully")

	return &authResp, nil
}

// ValidateAuth checks if the current authentication is valid
func (c *Client) ValidateAuth(collection string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("not authenticated")
	}

	endpoint := fmt.Sprintf("collections/%s/auth-refresh", collection)

	// Try to refresh - if it fails, auth is invalid
	_, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("authentication is invalid or expired: %w", err)
	}

	return nil
}

// GetAuthenticatedUser returns the currently authenticated user record
func (c *Client) GetAuthenticatedUser() (map[string]interface{}, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	if c.authRecord == nil {
		return nil, fmt.Errorf("no authentication record available")
	}

	return c.authRecord, nil
}

// UpdateAuthContextFromResponse updates a context with authentication data
func UpdateAuthContextFromResponse(ctx *config.Context, authResp *AuthResponse) error {
	if authResp == nil {
		return fmt.Errorf("authentication response is nil")
	}

	// Update context with auth data
	ctx.PocketBase.AuthToken = authResp.Token
	ctx.PocketBase.AuthRecord = authResp.Record

	// Define a simple claims struct to extract the 'exp' field
	type Claims struct {
		jwt.RegisteredClaims
	}

	// Parse the token without verifying the signature. This is safe because
	// we just received it from the PocketBase server over a secure connection.
	// We only need to read the claims.
	token, _, err := new(jwt.Parser).ParseUnverified(authResp.Token, &Claims{})
	if err != nil {
		// If parsing fails, fall back to the old 7-day logic as a safety measure
		// but warn the user.
		utils.PrintWarning("Could not parse JWT to determine expiration, defaulting to 7 days.")
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		ctx.PocketBase.AuthExpires = &expiresAt
		return nil
	}

	if claims, ok := token.Claims.(*Claims); ok && claims.ExpiresAt != nil {
		// The 'exp' claim is a Unix timestamp. Convert it to time.Time.
		expiresAt := claims.ExpiresAt.Time
		ctx.PocketBase.AuthExpires = &expiresAt
		utils.PrintDebug(fmt.Sprintf("JWT expiration successfully parsed: %s", expiresAt.Format(time.RFC3339)))
	} else {
		// If token has no expiration claim, fall back
		utils.PrintWarning("JWT has no expiration claim, defaulting to 7 days.")
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		ctx.PocketBase.AuthExpires = &expiresAt
	}

	return nil
}

// IsAuthValid checks if the authentication in a context is still valid
func IsAuthValid(ctx *config.Context) bool {
	if ctx.PocketBase.AuthToken == "" {
		return false
	}

	if ctx.PocketBase.AuthExpires == nil {
		// No expiration set, assume valid for backward compatibility
		return true
	}

	// --- START: CORRECTED LOGIC ---
	// Check if the current time is before the token's expiration time.
	// The buffer has been removed as it caused issues with short-lived tokens.
	return time.Now().Before(*ctx.PocketBase.AuthExpires)
	// --- END: CORRECTED LOGIC ---
}

// GetCollectionDisplayName returns a human-readable name for auth collections
func GetCollectionDisplayName(collection string) string {
	switch collection {
	case config.AuthCollectionUsers:
		return "Users"
	case config.AuthCollectionAdmins:
		return "Administrators"
	case config.AuthCollectionClients:
		return "API Clients"
	default:
		// For custom collections, return a formatted version
		return fmt.Sprintf("Collection: %s", collection)
	}
}

// ValidateAuthCollection ensures the collection supports authentication
func ValidateAuthCollection(collection string) error {
	// Since we're making this generic, we'll allow any collection name
	// PocketBase will validate if the collection actually supports auth
	return config.ValidateAuthCollection(collection)
}
