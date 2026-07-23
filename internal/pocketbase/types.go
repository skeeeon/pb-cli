package pocketbase

import (
	"encoding/json"
	"fmt"
	"time"

	"pb-cli/internal/utils"
)

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

// Collection represents a PocketBase collection definition.
// Field names match the PocketBase v0.23+ API (the old "schema" key is now "fields").
type Collection struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"` // "base", "auth", or "view"
	System     bool    `json:"system"`
	Fields     []Field `json:"fields"`
	ListRule   *string `json:"listRule"`
	ViewRule   *string `json:"viewRule"`
	CreateRule *string `json:"createRule"`
	UpdateRule *string `json:"updateRule"`
	DeleteRule *string `json:"deleteRule"`
}

// Field represents a single field in a collection's schema.
type Field struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	System      bool   `json:"system"`
	Required    bool   `json:"required"`
	Presentable bool   `json:"presentable"`
}

// Backup represents a PocketBase backup
type Backup struct {
	Key      string `json:"key"`
	Size     int64  `json:"size"`
	Modified PBTime `json:"modified"`
}

// BackupsList represents a list of backups
type BackupsList []Backup

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
		"2006-01-02 15:04:05.999Z", // PocketBase format with space and microseconds
		"2006-01-02 15:04:05Z",     // PocketBase format with space, no microseconds
		time.RFC3339,               // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,           // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05.999Z", // Standard with microseconds
		"2006-01-02T15:04:05Z",     // Standard without microseconds
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

// GetHumanSize returns a human-readable size string
func (b *Backup) GetHumanSize() string {
	return utils.FormatBytes(b.Size)
}

// GetFormattedDate returns a formatted date string
func (b *Backup) GetFormattedDate() string {
	return b.Modified.Time.Format("2006-01-02 15:04:05")
}
