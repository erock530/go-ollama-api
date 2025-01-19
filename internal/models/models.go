package models

import (
	"database/sql"
	"time"
)

// APIKey represents an API key in the database
type APIKey struct {
	Key         string
	CreatedAt   time.Time
	LastUsed    time.Time
	Tokens      int
	RateLimit   int
	Active      bool
	Description sql.NullString
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID  int64
	URL string
}

// GenerateRequest represents a request to the Ollama API
type GenerateRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Stream bool     `json:"stream"`
	Images []string `json:"images,omitempty"`
	Raw    bool     `json:"raw"`
	APIKey string   `json:"apikey"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Error     string      `json:"error,omitempty"`
	Status    string      `json:"status,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}
