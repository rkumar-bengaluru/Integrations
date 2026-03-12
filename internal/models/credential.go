package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CredentialType string

const (
	AuthOAuth2          CredentialType = "oauth2"
	AuthOAuth2PKCE      CredentialType = "oauth2_pkce"
	AuthOAuth2Device    CredentialType = "oauth2_device"
	AuthAPIKey          CredentialType = "api_key"
	AuthBearerToken     CredentialType = "bearer_token"
	AuthBasicAuth       CredentialType = "basic_auth"
	AuthMTLS            CredentialType = "mtls"
	AuthMCPSession      CredentialType = "mcp_session"
	AuthAWSSigV4        CredentialType = "aws_sigv4"
	AuthAzureManaged    CredentialType = "azure_managed_identity"
	AuthGCPServiceAcc   CredentialType = "gcp_service_account"
	AuthGitubServiceAcc CredentialType = "github_service_account"
	AuthJWT             CredentialType = "jwt"
	AuthSAML            CredentialType = "saml_assertion"
	AuthDatabase        CredentialType = "database_connection"
	AuthSSHKey          CredentialType = "ssh_key"
)

type GrantType string

const (
	GrantTypeAuthorizationFlow    GrantType = "authorization_flow"
	GrantTypeAuthClientCredential GrantType = "client_credenial"
	GrantTypeAuthAPIKey           GrantType = "api_key"
)

type Status string

const (
	Active   Status = "active"
	InActive Status = "inactive"
)

type ValidationStatus string

const (
	ValidationUnverified ValidationStatus = "unverified"
	ValidationValid      ValidationStatus = "valid"
	ValidationInvalid    ValidationStatus = "invalid"
	ValidationExpired    ValidationStatus = "expired"
)

type Credential struct {
	BaseModel
	TenantID    uuid.UUID      `gorm:"column:tenant_id;not null;index" json:"tenant_id"`
	Name        string         `gorm:"column:name;not null" json:"name"`
	Description string         `gorm:"column:description" json:"description"`
	Type        CredentialType `gorm:"column:type;type:varchar(50);not null" json:"type"`
	Provider    string         `gorm:"column:provider" json:"provider"`

	// Encrypted secret blob stored as ciphertext
	EncryptedData []byte `gorm:"column:encrypted_data" json:"-"`

	// Deterministic hash of normalized plaintext for dedupe detection
	DataHash string `gorm:"column:data_hash;index" json:"-"`

	// Optional non secret metadata for UI and lookup
	Scopes     datatypes.JSONSlice[string] `gorm:"column:scopes" json:"scopes,omitempty"`
	ExpiresAt  *time.Time                  `gorm:"column:expires_at" json:"expires_at,omitempty"`
	LastUsedAt *time.Time                  `gorm:"column:last_used_at" json:"last_used_at,omitempty"`

	ValidationStatus ValidationStatus `gorm:"column:validation_status;type:varchar(32);default:'unverified'" json:"validation_status"`
	ValidatedAt      *time.Time       `gorm:"column:validated_at" json:"validated_at,omitempty"`
	ValidationError  string           `gorm:"column:validation_error;type:text" json:"validation_error,omitempty"`
}
