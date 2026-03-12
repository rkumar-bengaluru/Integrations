package models

import (
	"github.com/google/uuid"
)

type JSONSchema struct {
	Title       string                        `json:"title,omitempty"`
	Description string                        `json:"description,omitempty"`
	Type        string                        `json:"type,omitempty"`
	Properties  map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required    []string                      `json:"required,omitempty"`
}

type JSONSchemaProperty struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Source      string `json:"source,omitempty"`
	FileType    string `json:"file_type,omitempty"`

	// allow nested schema or additionalProperties
	AdditionalProperties map[string]interface{} `json:"additionalProperties,omitempty"`
}

type CredentialBinding struct {
	CredentialType CredentialType `json:"type,omitempty"`
	GrantType      *GrantType     `json:"grant_type,omitempty"`
	AuthorityUrl   *string        `json:"authority_url,omitempty"`
	TokenUrl       *string        `json:"token_url,omitempty"`
	Scopes         []string       `json:"scopes,omitempty"`
	Notes          *string        `json:"notes,omitempty"`
	SecretMapping  JSONMap        `json:"secrets_mapping,omitempty"`
}

// ExecutionConfig: Flexible JSON-serializable config for actions
type ExecutionConfig struct {
	CredentialType    CredentialType     `json:"mode,omitempty"`
	CredentialBinding *CredentialBinding `json:"credential_binding,omitempty"`
	ParamInputSchema  *JSONSchema        `json:"param_input_schema,omitempty"`
	ParamOuputSchema  *JSONSchema        `json:"param_output_schema,omitempty"`
	// allow arbitrary extra fields if needed
	Extras map[string]interface{} `json:"-"`
}

// Map credential type → execution config
type ExecutionConfigs map[CredentialType]ExecutionConfig

type CredentialTypes []CredentialType

type IntegrationCategory string

const (
	CategoryVectorStore   IntegrationCategory = "vector_store"
	CategoryEnterprise    IntegrationCategory = "enterprise"    // Confluence, SharePoint, etc.
	CategoryCommunication IntegrationCategory = "communication" // Gmail, Slack, Teams
	CategoryCalendar      IntegrationCategory = "calendar"      // Google Cal, Outlook
	CategoryDatabase      IntegrationCategory = "database"
	CategoryAPI           IntegrationCategory = "api"    // Generic REST/GraphQL
	CategoryMCP           IntegrationCategory = "mcp"    // Model Context Protocol
	CategoryAgent         IntegrationCategory = "agent"  // Sub-agent delegation
	CategoryCustom        IntegrationCategory = "custom" // User-defined code
)

type PlatformCredential struct {
	BaseModel
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	IntegrationID  uuid.UUID      `gorm:"column:integration_id;index" json:"integration_id"`
	CredentialID   uuid.UUID      `gorm:"column:credential_id;index" json:"credential_id"`
	CredentialType CredentialType `gorm:"column:credential_type;index" json:"credential_type"`

	Credential *Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
}

// Integration: Model for grouping actions into packages
type Integration struct {
	BaseModel
	TenantID    uuid.UUID           `gorm:"column:tenant_id;index" json:"tenant_id"` // Zero UUID for global
	Name        string              `gorm:"column:name;not null" json:"name"`        // E.g., "Gmail Integration", SharePoint Etc
	Description string              `gorm:"column:description;type:text" json:"description"`
	Category    IntegrationCategory `gorm:"column:category;index" json:"category"`
	Actions     []ActionDefinition  `gorm:"many2many:integration_actions;" json:"actions"`

	// supported credential types
	SupportedCredentialTypes CredentialTypes `gorm:"column:supported_credential_types;type:jsonb" json:"supported_credential_types"`
	// ExecutionConfig, store a map keyed per credential type
	ExecutionConfigs ExecutionConfigs `gorm:"column:execution_configs;type:jsonb" json:"execution_configs"`
	// Credential for each supported credential types.
	PlatformCredentials []*PlatformCredential `gorm:"foreignKey:IntegrationID" json:"credentials"`
}
