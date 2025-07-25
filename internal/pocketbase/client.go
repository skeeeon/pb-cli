package pocketbase

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"pb-cli/internal/config"
	"pb-cli/internal/utils"
)

// Client represents a PocketBase HTTP client
type Client struct {
	httpClient *resty.Client
	baseURL    string
	authToken  string
	authRecord map[string]interface{}
}

// NewClient creates a new PocketBase client
func NewClient(baseURL string) *Client {
	client := resty.New()
	
	// Set common headers
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", "pb-cli/0.1.0")
	
	// Set timeout
	client.SetTimeout(30 * time.Second)
	
	// Enable debug mode if configured
	if config.Global.Debug {
		client.SetDebug(true)
	}

	return &Client{
		httpClient: client,
		baseURL:    baseURL,
	}
}

// NewClientFromContext creates a PocketBase client from a context configuration
func NewClientFromContext(ctx *config.Context) *Client {
	client := NewClient(ctx.PocketBase.URL)
	
	// Set authentication if available
	if ctx.PocketBase.AuthToken != "" {
		client.SetAuthToken(ctx.PocketBase.AuthToken)
		client.authRecord = ctx.PocketBase.AuthRecord
	}
	
	return client
}

// SetAuthToken sets the authentication token for requests
func (c *Client) SetAuthToken(token string) {
	c.authToken = token
	c.httpClient.SetAuthToken(token)
}

// GetAuthToken returns the current authentication token
func (c *Client) GetAuthToken() string {
	return c.authToken
}

// GetAuthRecord returns the current authentication record
func (c *Client) GetAuthRecord() map[string]interface{} {
	return c.authRecord
}

// IsAuthenticated checks if the client has a valid authentication token
func (c *Client) IsAuthenticated() bool {
	return c.authToken != ""
}

// makeRequest performs an HTTP request with error handling
func (c *Client) makeRequest(method, endpoint string, body interface{}) (*resty.Response, error) {
	url := fmt.Sprintf("%s/api/%s", c.baseURL, endpoint)
	
	utils.PrintDebug(fmt.Sprintf("Making %s request to %s", method, url))
	
	var resp *resty.Response
	var err error
	
	switch method {
	case "GET":
		resp, err = c.httpClient.R().Get(url)
	case "POST":
		resp, err = c.httpClient.R().SetBody(body).Post(url)
	case "PATCH":
		resp, err = c.httpClient.R().SetBody(body).Patch(url)
	case "DELETE":
		resp, err = c.httpClient.R().Delete(url)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	
	utils.PrintDebug(fmt.Sprintf("Response status: %d", resp.StatusCode()))
	
	// Handle HTTP errors
	if resp.StatusCode() >= 400 {
		return resp, NewPocketBaseError(resp)
	}
	
	return resp, nil
}

// GetHealth checks the PocketBase server health
func (c *Client) GetHealth() error {
	resp, err := c.makeRequest("GET", "health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode())
	}
	
	return nil
}

// GetCollections returns available collections from PocketBase
func (c *Client) GetCollections() ([]Collection, error) {
	resp, err := c.makeRequest("GET", "collections", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get collections: %w", err)
	}
	
	var result struct {
		Items []Collection `json:"items"`
	}
	
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse collections response: %w", err)
	}
	
	return result.Items, nil
}

// ValidateCollection checks if a collection exists and is accessible
func (c *Client) ValidateCollection(collection string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}

	// Try to get collection info (this will fail if collection doesn't exist or isn't accessible)
	endpoint := fmt.Sprintf("collections/%s", collection)
	_, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		if pbErr, ok := err.(*PocketBaseError); ok {
			if pbErr.StatusCode == 404 {
				return fmt.Errorf("collection '%s' not found", collection)
			}
			if pbErr.StatusCode == 403 {
				return fmt.Errorf("access denied to collection '%s'", collection)
			}
		}
		return fmt.Errorf("failed to validate collection '%s': %w", collection, err)
	}
	
	return nil
}

// ListRecords retrieves records from a collection with pagination and filtering
func (c *Client) ListRecords(collection string, options *ListOptions) (*RecordsList, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s/records", collection)
	
	// Add query parameters
	req := c.httpClient.R()
	if options != nil {
		if options.Page > 0 {
			req.SetQueryParam("page", fmt.Sprintf("%d", options.Page))
		}
		if options.PerPage > 0 {
			req.SetQueryParam("perPage", fmt.Sprintf("%d", options.PerPage))
		}
		if options.Filter != "" {
			req.SetQueryParam("filter", options.Filter)
		}
		if options.Sort != "" {
			req.SetQueryParam("sort", options.Sort)
		}
		if len(options.Fields) > 0 {
			req.SetQueryParam("fields", strings.Join(options.Fields, ","))
		}
		if len(options.Expand) > 0 {
			req.SetQueryParam("expand", strings.Join(options.Expand, ","))
		}
	}
	
	url := fmt.Sprintf("%s/api/%s", c.baseURL, endpoint)
	resp, err := req.Get(url)
	
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	
	if resp.StatusCode() >= 400 {
		return nil, NewPocketBaseError(resp)
	}
	
	var result RecordsList
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}
	
	return &result, nil
}

// GetRecord retrieves a single record by ID
func (c *Client) GetRecord(collection, id string, expand []string) (map[string]interface{}, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s/records/%s", collection, id)
	
	req := c.httpClient.R()
	if len(expand) > 0 {
		req.SetQueryParam("expand", strings.Join(expand, ","))
	}
	
	url := fmt.Sprintf("%s/api/%s", c.baseURL, endpoint)
	resp, err := req.Get(url)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	
	if resp.StatusCode() >= 400 {
		return nil, NewPocketBaseError(resp)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse record response: %w", err)
	}
	
	return result, nil
}

// CreateRecord creates a new record in a collection
func (c *Client) CreateRecord(collection string, data map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s/records", collection)
	
	resp, err := c.makeRequest("POST", endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse create response: %w", err)
	}
	
	return result, nil
}

// UpdateRecord updates an existing record
func (c *Client) UpdateRecord(collection, id string, data map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s/records/%s", collection, id)
	
	resp, err := c.makeRequest("PATCH", endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse update response: %w", err)
	}
	
	return result, nil
}

// DeleteRecord deletes a record by ID
func (c *Client) DeleteRecord(collection, id string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s/records/%s", collection, id)
	
	_, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	
	return nil
}

// GetCollectionSchema returns the schema for a collection
func (c *Client) GetCollectionSchema(collection string) (*Collection, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}
	
	endpoint := fmt.Sprintf("collections/%s", collection)
	
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection schema: %w", err)
	}
	
	var result Collection
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse collection schema response: %w", err)
	}
	
	return &result, nil
}

// CollectionExists checks if a collection exists (without requiring auth for the actual collection)
func (c *Client) CollectionExists(collection string) bool {
	if !c.IsAuthenticated() {
		return false
	}
	
	err := c.ValidateCollection(collection)
	return err == nil
}
