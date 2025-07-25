package pocketbase

import (
	"encoding/json"
	"fmt"
	"time"

	"pb-cli/internal/utils"
)

// Collection represents a PocketBase collection definition
type Collection struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "base" or "auth"
	System     bool                   `json:"system"`
	Schema     []Field                `json:"schema"`
	ListRule   *string                `json:"listRule"`
	ViewRule   *string                `json:"viewRule"`
	CreateRule *string                `json:"createRule"`
	UpdateRule *string                `json:"updateRule"`
	DeleteRule *string                `json:"deleteRule"`
	Options    map[string]interface{} `json:"options"`
	Created    time.Time              `json:"created"`
	Updated    time.Time              `json:"updated"`
}

// Field represents a collection field definition
type Field struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	System       bool                   `json:"system"`
	Required     bool                   `json:"required"`
	Presentable  bool                   `json:"presentable"`
	Unique       bool                   `json:"unique,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// RecordsList represents a paginated list of records
type RecordsList struct {
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
	TotalItems int                      `json:"totalItems"`
	TotalPages int                      `json:"totalPages"`
	Items      []map[string]interface{} `json:"items"`
}

// ListOptions represents options for listing records
type ListOptions struct {
	Page    int      `json:"page,omitempty"`
	PerPage int      `json:"perPage,omitempty"`
	Sort    string   `json:"sort,omitempty"`
	Filter  string   `json:"filter,omitempty"`
	Fields  []string `json:"fields,omitempty"`
	Expand  []string `json:"expand,omitempty"`
}

// Record represents a generic PocketBase record
type Record map[string]interface{}

// GetID returns the record ID
func (r Record) GetID() string {
	if id, ok := r["id"].(string); ok {
		return id
	}
	return ""
}

// GetString returns a string field value
func (r Record) GetString(field string) string {
	if value, ok := r[field].(string); ok {
		return value
	}
	return ""
}

// GetBool returns a boolean field value
func (r Record) GetBool(field string) bool {
	if value, ok := r[field].(bool); ok {
		return value
	}
	return false
}

// GetInt returns an integer field value
func (r Record) GetInt(field string) int {
	switch value := r[field].(type) {
	case int:
		return value
	case float64:
		return int(value)
	}
	return 0
}

// GetFloat returns a float field value
func (r Record) GetFloat(field string) float64 {
	switch value := r[field].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	}
	return 0
}

// GetTime returns a time field value with multiple format support
func (r Record) GetTime(field string) *time.Time {
	if timeStr, ok := r[field].(string); ok {
		// Try multiple time formats that PocketBase might use
		formats := []string{
			time.RFC3339,                    // "2006-01-02T15:04:05Z07:00"
			time.RFC3339Nano,                // "2006-01-02T15:04:05.999999999Z07:00"
			"2006-01-02 15:04:05.999Z",      // PocketBase format with space
			"2006-01-02 15:04:05Z",          // PocketBase format without microseconds
			"2006-01-02T15:04:05.999Z",      // Standard with microseconds
			"2006-01-02T15:04:05Z",          // Standard without microseconds
		}
		
		for _, format := range formats {
			if t, err := time.Parse(format, timeStr); err == nil {
				return &t
			}
		}
	}
	return nil
}

// GetCreated returns the record creation time
func (r Record) GetCreated() *time.Time {
	return r.GetTime("created")
}

// GetUpdated returns the record update time
func (r Record) GetUpdated() *time.Time {
	return r.GetTime("updated")
}

// GetArray returns an array field value
func (r Record) GetArray(field string) []interface{} {
	if value, ok := r[field].([]interface{}); ok {
		return value
	}
	return nil
}

// GetObject returns an object field value
func (r Record) GetObject(field string) map[string]interface{} {
	if value, ok := r[field].(map[string]interface{}); ok {
		return value
	}
	return nil
}

// HasField checks if a field exists in the record
func (r Record) HasField(field string) bool {
	_, exists := r[field]
	return exists
}

// GetDisplayName attempts to get a human-readable display name for the record
// Tries common name fields in order of preference
func (r Record) GetDisplayName() string {
	// Try common name fields
	nameFields := []string{"name", "title", "display_name", "full_name"}
	for _, field := range nameFields {
		if name := r.GetString(field); name != "" {
			return name
		}
	}
	
	// Try combining first/last name
	firstName := r.GetString("first_name")
	lastName := r.GetString("last_name")
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	} else if firstName != "" {
		return firstName
	} else if lastName != "" {
		return lastName
	}
	
	// Try email or username
	if email := r.GetString("email"); email != "" {
		return email
	}
	if username := r.GetString("username"); username != "" {
		return username
	}
	
	// Fallback to ID
	if id := r.GetID(); id != "" {
		return fmt.Sprintf("ID: %s", id)
	}
	
	return "Unknown"
}

// Backup Management Types

// Backup represents a PocketBase backup
type Backup struct {
	Key      string `json:"key"`
	Size     int64  `json:"size"`
	Modified PBTime `json:"modified"`
}

// PBTime handles PocketBase's time format
type PBTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for PocketBase time format
func (pbt *PBTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	timeStr := string(data)
	if len(timeStr) >= 2 && timeStr[0] == '"' && timeStr[len(timeStr)-1] == '"' {
		timeStr = timeStr[1 : len(timeStr)-1]
	}

	// Try multiple time formats that PocketBase might use
	formats := []string{
		"2006-01-02 15:04:05.999Z",      // PocketBase format with space and microseconds
		"2006-01-02 15:04:05Z",          // PocketBase format with space, no microseconds
		time.RFC3339,                    // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,                // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05.999Z",      // Standard with microseconds
		"2006-01-02T15:04:05Z",          // Standard without microseconds
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			pbt.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse time: %s", timeStr)
}

// MarshalJSON implements custom JSON marshaling
func (pbt PBTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(pbt.Time.Format(time.RFC3339))
}

// BackupsList represents a list of backups
type BackupsList []Backup

// GetHumanSize returns a human-readable size string
func (b *Backup) GetHumanSize() string {
	return utils.FormatBytes(b.Size)
}

// GetFormattedDate returns a formatted date string
func (b *Backup) GetFormattedDate() string {
	return b.Modified.Time.Format("2006-01-02 15:04:05")
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// RequestOptions represents common request options
type RequestOptions struct {
	Headers map[string]string `json:"headers,omitempty"`
	Timeout time.Duration     `json:"timeout,omitempty"`
}

// HealthStatus represents the PocketBase health status
type HealthStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		CanBackup bool `json:"canBackup"`
	} `json:"data"`
}

// FileUpload represents a file upload for a record
type FileUpload struct {
	FieldName string `json:"field_name"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileBytes []byte `json:"file_bytes,omitempty"`
}

// BatchOperation represents a batch operation request
type BatchOperation struct {
	Method     string                 `json:"method"`     // GET, POST, PATCH, DELETE
	Collection string                 `json:"collection"`
	ID         string                 `json:"id,omitempty"` // For single record operations
	Data       map[string]interface{} `json:"data,omitempty"`
	Filter     string                 `json:"filter,omitempty"`
}

// BatchResponse represents a batch operation response
type BatchResponse struct {
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// CollectionStats represents statistics for a collection
type CollectionStats struct {
	Name        string `json:"name"`
	TotalRecords int   `json:"total_records"`
	Type        string `json:"type"`
	System      bool   `json:"system"`
}

// SchemaField represents a collection schema field for validation
type SchemaField struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Required bool                   `json:"required"`
	Unique   bool                   `json:"unique"`
	Options  map[string]interface{} `json:"options"`
}

// AuthMethods represents available authentication methods for a collection
type AuthMethods struct {
	EmailPassword bool     `json:"emailPassword"`
	Username      bool     `json:"username"`
	OAuth         []string `json:"oauth"`
}

// CollectionRule represents a collection rule (create, read, update, delete)
type CollectionRule struct {
	Rule        *string `json:"rule"`
	Description string  `json:"description,omitempty"`
}

// CollectionInfo represents detailed information about a collection
type CollectionInfo struct {
	Collection  Collection         `json:"collection"`
	Stats       CollectionStats    `json:"stats,omitempty"`
	AuthMethods *AuthMethods       `json:"auth_methods,omitempty"`
	Rules       map[string]string  `json:"rules,omitempty"`
}

// FilterBuilder helps build PocketBase filter expressions
type FilterBuilder struct {
	conditions []string
}

// NewFilterBuilder creates a new filter builder
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		conditions: make([]string, 0),
	}
}

// Equal adds an equality condition
func (fb *FilterBuilder) Equal(field, value string) *FilterBuilder {
	fb.conditions = append(fb.conditions, fmt.Sprintf("%s='%s'", field, value))
	return fb
}

// NotEqual adds a not equal condition
func (fb *FilterBuilder) NotEqual(field, value string) *FilterBuilder {
	fb.conditions = append(fb.conditions, fmt.Sprintf("%s!='%s'", field, value))
	return fb
}

// Like adds a LIKE condition
func (fb *FilterBuilder) Like(field, value string) *FilterBuilder {
	fb.conditions = append(fb.conditions, fmt.Sprintf("%s~'%s'", field, value))
	return fb
}

// In adds an IN condition
func (fb *FilterBuilder) In(field string, values []string) *FilterBuilder {
	quotedValues := make([]string, len(values))
	for i, v := range values {
		quotedValues[i] = fmt.Sprintf("'%s'", v)
	}
	fb.conditions = append(fb.conditions, fmt.Sprintf("%s?=[%s]", field, fmt.Sprintf("%s", quotedValues)))
	return fb
}

// And adds an AND operator
func (fb *FilterBuilder) And() *FilterBuilder {
	if len(fb.conditions) > 0 {
		fb.conditions = append(fb.conditions, "&&")
	}
	return fb
}

// Or adds an OR operator
func (fb *FilterBuilder) Or() *FilterBuilder {
	if len(fb.conditions) > 0 {
		fb.conditions = append(fb.conditions, "||")
	}
	return fb
}

// Build builds the final filter string
func (fb *FilterBuilder) Build() string {
	if len(fb.conditions) == 0 {
		return ""
	}
	
	result := ""
	for i, condition := range fb.conditions {
		if i > 0 {
			result += " "
		}
		result += condition
	}
	
	return result
}
