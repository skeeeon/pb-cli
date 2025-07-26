package pocketbase

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// FileTokenResponse represents the response from /api/files/token
type FileTokenResponse struct {
	Token string `json:"token"`
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

// GetFileToken requests a file access token for protected file downloads
func (c *Client) GetFileToken() (string, error) {
	if !c.IsAuthenticated() {
		return "", fmt.Errorf("authentication required")
	}

	utils.PrintDebug("Requesting file token for protected file access")

	resp, err := c.makeRequest("POST", "files/token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get file token: %w", err)
	}

	var tokenResp FileTokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse file token response: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Received file token: %s...", tokenResp.Token[:10]))
	
	return tokenResp.Token, nil
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

// Backup Management Methods

// ListBackups retrieves all available backups
func (c *Client) ListBackups() (BackupsList, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}

	utils.PrintDebug("Listing backups from PocketBase")

	resp, err := c.makeRequest("GET", "backups", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	var backups BackupsList
	if err := json.Unmarshal(resp.Body(), &backups); err != nil {
		return nil, fmt.Errorf("failed to parse backups response: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Found %d backups", len(backups)))
	return backups, nil
}

// CreateBackup creates a new backup
func (c *Client) CreateBackup(name string) (*Backup, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}

	utils.PrintDebug(fmt.Sprintf("Creating backup with name: %s", name))

	var requestData map[string]interface{}
	if name != "" {
		requestData = map[string]interface{}{
			"name": name,
		}
	}

	resp, err := c.makeRequest("POST", "backups", requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Handle 204 No Content response (successful backup creation with no body)
	if resp.StatusCode() == 204 {
		utils.PrintDebug("Backup created successfully (204 No Content)")
		
		// Since we don't get backup details in the response, we need to fetch it
		// by listing backups and finding the most recent one
		backups, err := c.ListBackups()
		if err != nil {
			return nil, fmt.Errorf("backup created but failed to retrieve details: %w", err)
		}
		
		if len(backups) == 0 {
			return nil, fmt.Errorf("backup created but no backups found")
		}
		
		// Find the most recent backup (assuming it's the one we just created)
		var mostRecent *Backup
		for i := range backups {
			if mostRecent == nil || backups[i].Modified.Time.After(mostRecent.Modified.Time) {
				mostRecent = &backups[i]
			}
		}
		
		if mostRecent == nil {
			return nil, fmt.Errorf("backup created but could not identify the new backup")
		}
		
		utils.PrintDebug(fmt.Sprintf("Found most recent backup: %s", mostRecent.Key))
		return mostRecent, nil
	}

	// Handle response with backup data (status 200/201)
	var backup Backup
	if err := json.Unmarshal(resp.Body(), &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup response: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Created backup: %s", backup.Key))
	return &backup, nil
}

// GetBackup gets information about a specific backup
func (c *Client) GetBackup(backupKey string) (*Backup, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}

	backups, err := c.ListBackups()
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		if backup.Key == backupKey {
			return &backup, nil
		}
	}

	return nil, fmt.Errorf("backup '%s' not found", backupKey)
}

// DownloadBackup downloads a backup file to the specified path using file token authentication
func (c *Client) DownloadBackup(backupKey, outputPath string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}

	utils.PrintDebug(fmt.Sprintf("Downloading backup %s to %s", backupKey, outputPath))

	// Step 1: Get file access token
	utils.PrintDebug("Requesting file access token...")
	fileToken, err := c.GetFileToken()
	if err != nil {
		return fmt.Errorf("failed to get file access token: %w", err)
	}

	// Step 2: Create the output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 3: Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Step 4: Download using file token
	url := fmt.Sprintf("%s/api/backups/%s", c.baseURL, backupKey)
	
	utils.PrintDebug(fmt.Sprintf("Downloading from URL: %s", url))
	
	// Create a fresh client without auth headers but with file token as query param
	downloadClient := resty.New()
	downloadClient.SetTimeout(30 * time.Second)
	downloadClient.SetHeader("User-Agent", "pb-cli/0.1.0")
	
	resp, err := downloadClient.R().
		SetQueryParam("token", fileToken).
		SetDoNotParseResponse(true).
		Get(url)
	
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	utils.PrintDebug(fmt.Sprintf("Download response status: %d", resp.StatusCode()))
	utils.PrintDebug(fmt.Sprintf("Content-Length: %s", resp.Header().Get("Content-Length")))

	// Step 5: Copy response body to file
	written, err := io.Copy(outFile, resp.RawBody())
	if err != nil {
		return fmt.Errorf("failed to save backup file: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Downloaded %d bytes to: %s", written, outputPath))
	
	if written == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

// DownloadBackupWithProgress downloads a backup with progress reporting using file token authentication
func (c *Client) DownloadBackupWithProgress(backupKey, outputPath string, progressCallback func(downloaded, total int64)) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}

	// Get backup info for size
	backup, err := c.GetBackup(backupKey)
	if err != nil {
		return fmt.Errorf("failed to get backup info: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Downloading backup %s (%s) to %s", backupKey, backup.GetHumanSize(), outputPath))

	// Step 1: Get file access token
	utils.PrintDebug("Requesting file access token...")
	fileToken, err := c.GetFileToken()
	if err != nil {
		return fmt.Errorf("failed to get file access token: %w", err)
	}

	// Step 2: Create the output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Step 3: Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Step 4: Download using file token
	url := fmt.Sprintf("%s/api/backups/%s", c.baseURL, backupKey)
	
	utils.PrintDebug(fmt.Sprintf("Downloading from URL: %s", url))
	
	// Create a fresh client without auth headers but with file token as query param
	downloadClient := resty.New()
	downloadClient.SetTimeout(30 * time.Second)
	downloadClient.SetHeader("User-Agent", "pb-cli/0.1.0")
	
	resp, err := downloadClient.R().
		SetQueryParam("token", fileToken).
		SetDoNotParseResponse(true).
		Get(url)
	
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	// Step 5: Copy with progress
	var written int64
	if progressCallback != nil {
		written, err = io.Copy(outFile, &progressReader{
			reader:   resp.RawBody(),
			total:    backup.Size,
			callback: progressCallback,
		})
	} else {
		written, err = io.Copy(outFile, resp.RawBody())
	}

	if err != nil {
		return fmt.Errorf("failed to save backup file: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Downloaded %d bytes to: %s", written, outputPath))
	
	if written == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

// UploadBackup uploads a backup file using the correct PocketBase upload API
func (c *Client) UploadBackup(filePath, backupName string, progressCallback func(uploaded, total int64)) (*Backup, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("authentication required")
	}

	utils.PrintDebug(fmt.Sprintf("Uploading backup from %s", filePath))

	// Check if file exists and get file info
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("backup file does not exist: %s", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access backup file: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Uploading %d bytes from file: %s", fileInfo.Size(), filePath))

	// Use the correct PocketBase upload endpoint with proper authentication
	url := fmt.Sprintf("%s/api/backups/upload", c.baseURL)
	
	utils.PrintDebug(fmt.Sprintf("Upload URL: %s", url))
	
	// Upload using authenticated client with POST to /api/backups/upload
	resp, err := c.httpClient.R().
		SetFile("file", filePath).  // Use "file" field name as per API docs
		Post(url)                   // Use POST method as per API docs

	if err != nil {
		return nil, fmt.Errorf("failed to upload backup: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Upload response status: %d", resp.StatusCode()))

	// Handle HTTP errors with proper PocketBase error parsing
	if resp.StatusCode() >= 400 {
		return nil, NewPocketBaseError(resp)
	}

	// Handle different response patterns
	if resp.StatusCode() == 204 {
		utils.PrintDebug("Backup uploaded successfully (204 No Content)")
		
		// For 204 responses, we need to determine the backup name
		// If no custom name provided, use the original filename
		uploadedName := backupName
		if uploadedName == "" {
			uploadedName = filepath.Base(filePath)
		}
		
		// Try to fetch the uploaded backup info
		backup, err := c.GetBackup(uploadedName)
		if err != nil {
			// If we can't get the specific backup, that's OK for 204 responses
			// Return a basic backup info
			utils.PrintDebug("Could not fetch uploaded backup details, returning basic info")
			return &Backup{
				Key:  uploadedName,
				Size: fileInfo.Size(),
			}, nil
		}
		
		return backup, nil
	}

	// Handle 200/201 responses with backup data
	var backup Backup
	if err := json.Unmarshal(resp.Body(), &backup); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Uploaded backup: %s", backup.Key))
	return &backup, nil
}

// DeleteBackup deletes a backup
func (c *Client) DeleteBackup(backupKey string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}

	utils.PrintDebug(fmt.Sprintf("Deleting backup: %s", backupKey))

	endpoint := fmt.Sprintf("backups/%s", backupKey)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Deleted backup: %s", backupKey))
	return nil
}

// RestoreBackup restores from a backup
func (c *Client) RestoreBackup(backupKey string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("authentication required")
	}

	utils.PrintDebug(fmt.Sprintf("Restoring from backup: %s", backupKey))

	endpoint := fmt.Sprintf("backups/%s/restore", backupKey)
	_, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	utils.PrintDebug(fmt.Sprintf("Restored from backup: %s", backupKey))
	return nil
}

// progressReader wraps an io.Reader and calls a progress callback
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	callback   func(downloaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)
	if pr.callback != nil {
		pr.callback(pr.downloaded, pr.total)
	}
	return n, err
}
