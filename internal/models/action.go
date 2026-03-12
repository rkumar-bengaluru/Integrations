package models

import "github.com/google/uuid"

type ActionType string

const (
	ActionTypeFunction    ActionType = "function"     // Native function call, "sharepoint_upload", "gmail_create_meeting", "mongodb_query", "child_agent_invoke"
	ActionTypeMCP         ActionType = "mcp"          // MCP Server integration
	ActionTypeHTTP        ActionType = "http"         // Generic HTTP API
	ActionTypeDatabase    ActionType = "database"     // SQL/NoSQL queries
	ActionTypeAgent       ActionType = "agent"        // Delegate to another agent
	ActionTypeVectorQuery ActionType = "vector_query" // Direct vector search
	ActionTypeScript      ActionType = "script"       // Python/JS/WASM execution
	ActionTypeComposite   ActionType = "composite"    // Workflow of multiple actions
)

// ActionDefinition: Core model for individual actions
type ActionDefinition struct {
	BaseModel                    // Includes ID, CreatedAt, UpdatedAt, DeletedAt
	TenantID          uuid.UUID  `gorm:"column:tenant_id;index" json:"tenant_id"`
	Name              string     `gorm:"column:name;not null" json:"name"` // E.g., "Upload Document to SharePoint"
	Description       string     `gorm:"column:description;type:text" json:"description"`
	Type              ActionType `gorm:"column:type;not null;" json:"type"`                        // Unique identifier for plugin action
	SchemaVersion     string     `gorm:"column:schema_version;default:'v1'" json:"schema_version"` // For schema evolution
	SupportsStreaming bool       `gorm:"column:supports_streaming;default:false" json:"supports_streaming"`
	IsInternal        bool       `gorm:"column:is_internal;default:false" json:"is_internal"` // Core vs. plugin-provided
	Version           string     `gorm:"column:version;default:'1.0.0'" json:"version"`
	PreviousVersionID *uuid.UUID `gorm:"column:previous_version_id" json:"previous_version_id,omitempty"`
	IsActive          bool       `gorm:"column:is_active;default:true" json:"is_active"`

	InputSchema  JSONMap `gorm:"column:input_schema;type:jsonb" json:"input_schema"`
	OutputSchema JSONMap `gorm:"column:output_schema;type:jsonb" json:"output_schema"`

	// Action handler
	ActionHandler string `gorm:"column:action_handler;default:'plugins.gmail.GMailActionHandler'" json:"action_handler"`
}
