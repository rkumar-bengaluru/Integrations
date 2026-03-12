// dto/integration_binding_dto.go
package api

import (
	"time"

	"github.com/google/uuid"
)

// OAuth2ConfigDTO: Request/response contract for OAuth2 configuration
type OAuth2ConfigDTO struct {
	GrantType string   `json:"grant_type" binding:"required" example:"client_credentials"`
	Authority string   `json:"authority" binding:"required" example:"https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token"`
	ClientID  string   `json:"client_id" binding:"required" example:"your-client-id"`
	Resource  string   `json:"resource,omitempty" example:"https://graph.microsoft.com"`
	Scopes    []string `json:"scopes,omitempty" example:"[\"https://graph.microsoft.com/.default\"]"`
}

// ==================== REQUEST DTOs ====================

type CreateIntegrationBindingRequest struct {
	IntegrationID   uuid.UUID          `json:"integration_id" binding:"required"`
	AgentID         *uuid.UUID         `json:"agent_id,omitempty"`
	UserID          *string            `json:"user_id,omitempty"`
	IsDefault       bool               `json:"is_default"`
	ExecutionConfig ExecutionConfigDTO `json:"execution_config" binding:"required"`

	// Embedded Credential DTO - user provides this to create credential automatically
	Credential CreateCredentialEmbeddedDTO `json:"credential" binding:"required"`
}

type CreateCredentialEmbeddedDTO struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Type        CredentialType `json:"type" binding:"required"`

	// OAuth2 specific (non-secret)
	OAuth2Config *OAuth2ConfigDTO `json:"oauth2_config,omitempty"`

	// Secret data - will be encrypted
	Secrets CredentialSecretsDTO `json:"secrets" binding:"required"`

	// Optional metadata
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Rotation policy
	RotationPolicy *RotationPolicyDTO `json:"rotation_policy,omitempty"`
}

// CredentialSecretsDTO contains the actual secret values based on credential type
type CredentialSecretsDTO struct {
	// OAuth2 / Azure / GCP
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`

	// API Key
	APIKey string `json:"api_key,omitempty"`

	// Bearer Token
	Token string `json:"token,omitempty"`

	// Basic Auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// AWS
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
	Region          string `json:"region,omitempty"`

	// Azure Blob Storage specific
	AccountName        string `json:"account_name,omitempty"`
	AccountKey         string `json:"account_key,omitempty"`
	StroageAccountName string `json:"storage_account_name,omitempty"`

	// Database
	ConnectionString string `json:"connection_string,omitempty"`

	// SSH
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`

	// Generic - catch all for other fields
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

type UpdateIntegrationBindingRequest struct {
	IsDefault       *bool                  `json:"is_default,omitempty"`
	ExecutionConfig *ExecutionConfigDTO    `json:"execution_config,omitempty"`
	Preview         map[string]interface{} `json:"preview,omitempty"`
	Status          *string                `json:"status,omitempty"`

	// Can update credential secrets (creates new version)
	CredentialSecrets *CredentialSecretsDTO `json:"credential_secrets,omitempty"`
}

type ListIntegrationBindingsRequest struct {
	TenantID      *uuid.UUID `form:"tenant_id"`
	IntegrationID *uuid.UUID `form:"integration_id"`
	AgentID       *uuid.UUID `form:"agent_id"`
	UserID        *uuid.UUID `form:"user_id"`
	Status        string     `form:"status"`
	Page          int        `form:"page,default=1"`
	PageSize      int        `form:"page_size,default=20"`
}

// ==================== RESPONSE DTOs ====================

type IntegrationBindingResponse struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         uuid.UUID              `json:"tenant_id"`
	IntegrationID    uuid.UUID              `json:"integration_id"`
	CredentialID     uuid.UUID              `json:"credential_id"`
	AgentID          *uuid.UUID             `json:"agent_id,omitempty"`
	UserID           *string                `json:"user_id,omitempty"`
	IsDefault        bool                   `json:"is_default"`
	ExecutionConfig  ExecutionConfigDTO     `json:"execution_config"`
	Preview          map[string]interface{} `json:"preview,omitempty"`
	Status           string                 `json:"status"`
	ValidationStatus string                 `json:"validation_status"`
	ValidationError  string                 `json:"validation_error,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`

	// Nested credential preview (without secrets)
	Credential CredentialPreviewDTO `json:"credential"`
}

type CredentialPreviewDTO struct {
	ID                uuid.UUID        `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Type              string           `json:"type"`
	OAuth2Config      *OAuth2ConfigDTO `json:"oauth2_config,omitempty"`
	Scopes            []string         `json:"scopes,omitempty"`
	ExpiresAt         *time.Time       `json:"expires_at,omitempty"`
	LastUsedAt        *time.Time       `json:"last_used_at,omitempty"`
	ValidationStatus  string           `json:"validation_status"`
	ValidatedAt       *time.Time       `json:"validated_at,omitempty"`
	HasRotationPolicy bool             `json:"has_rotation_policy"`
}

type IntegrationBindingListResponse struct {
	Items      []IntegrationBindingResponse `json:"items"`
	Total      int64                        `json:"total"`
	Page       int                          `json:"page"`
	PageSize   int                          `json:"page_size"`
	TotalPages int                          `json:"total_pages"`
}

// ==================== VALIDATION DTOs ====================

type ValidateBindingRequest struct {
	IntegrationBindingID uuid.UUID `json:"integration_binding_id" binding:"required"`
}

type ValidateBindingResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
