package api

import "github.com/google/uuid"

type BindingFilters struct {
	TenantID      *uuid.UUID
	IntegrationID *uuid.UUID
	AgentID       *uuid.UUID
	UserID        *uuid.UUID
	Status        string
	Page          int
	PageSize      int
}

type CredentialType string

const (
	AuthOAuth2           CredentialType = "oauth2"
	AuthOAuth2PKCE       CredentialType = "oauth2_pkce"
	AuthOAuth2Device     CredentialType = "oauth2_device"
	AuthAPIKey           CredentialType = "api_key"
	AuthBearerToken      CredentialType = "bearer_token"
	AuthBasicAuth        CredentialType = "basic_auth"
	AuthMTLS             CredentialType = "mtls"
	AuthMCPSession       CredentialType = "mcp_session"
	AuthAWSSigV4         CredentialType = "aws_sigv4"
	AuthAzureManaged     CredentialType = "azure_managed_identity"
	AuthGCPServiceAcc    CredentialType = "gcp_service_account"
	AuthGithubSevicebAcc CredentialType = "github_service_account"
	AuthJWT              CredentialType = "jwt"
	AuthSAML             CredentialType = "saml_assertion"
	AuthDatabase         CredentialType = "database_connection"
	AuthSSHKey           CredentialType = "ssh_key"
)

type Status string

const (
	Active   Status = "active"
	InActive Status = "inactive"
)

// ExecutionConfig: Flexible JSON-serializable config for actions
type ExecutionConfigDTO struct {
	Mode                 string                `json:"mode,omitempty"`
	CredentialBinding    *CredentialBindingDTO `json:"credential_binding,omitempty"`
	ParamSchema          *JSONSchemaDTO        `json:"param_schema,omitempty"`
	PreviewKeys          []string              `json:"preview_keys,omitempty"`
	DefaultTimeoutSecond int                   `json:"default_timeout_seconds,omitempty"`
	// allow arbitrary extra fields if needed
	Extras map[string]interface{} `json:"-"`
}

type CredentialBindingDTO struct {
	Type          string                 `json:"type,omitempty"`
	SecretMapping map[string]interface{} `json:"secrets_mapping"`
	GrantType     *string                `json:"grant_type,omitempty"` // api_key, authorization_code, client_credential
	AuthorityUrl  *string                `json:"auth_url,omitempty"`
	Notes         *string                `json:"notes,omitempty"`
}

type JSONSchemaDTO struct {
	Title       string                           `json:"title,omitempty"`
	Description string                           `json:"description,omitempty"`
	Type        string                           `json:"type,omitempty"`
	Properties  map[string]JSONSchemaPropertyDTO `json:"properties,omitempty"`
	Required    []string                         `json:"required,omitempty"`
}

type JSONSchemaPropertyDTO struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
	// allow nested schema or additionalProperties
	AdditionalProperties map[string]interface{} `json:"additionalProperties,omitempty"`
}

// RotationPolicyDTO is the transport-friendly version of RotationPolicy
type RotationPolicyDTO struct {
	Enabled            bool                    `json:"enabled"`
	RotationInterval   string                  `json:"rotation_interval"` // e.g., "90d"
	ExpiryBuffer       string                  `json:"expiry_buffer"`     // e.g., "7d"
	Strategy           string                  `json:"strategy"`          // "automatic", "manual", "maintenance_window"
	GracePeriod        string                  `json:"grace_period"`      // e.g., "24h"
	NotificationConfig RotationNotificationDTO `json:"notification_config"`
	// Constraints        *RotationConstraintsDTO `json:"constraints"`
}

// RotationNotificationDTO is the DTO for RotationNotification
type RotationNotificationDTO struct {
	NotifyBefore string `json:"notify_before"` // e.g., ["7d", "1d", "1h"]
	Channels     string `json:"channels"`      // "email", "slack", etc.
	WebhookURL   string `json:"webhook_url,omitempty"`
}

// RotationConstraintsDTO is the DTO for RotationConstraints
type RotationConstraintsDTO struct {
	MaintenanceWindows TimeWindowDTO `json:"maintenance_windows,omitempty"`
	MaxRetries         int           `json:"max_retries"`
	RequireApproval    bool          `json:"require_approval"`
	AllowedDays        int           `json:"allowed_days"` // 0=Sunday, 6=Saturday
}

// TimeWindowDTO is the DTO for TimeWindow
type TimeWindowDTO struct {
	DayOfWeek int    `json:"day_of_week"` // 0-6, -1 for daily
	StartTime string `json:"start_time"`  // "15:00"
	EndTime   string `json:"end_time"`    // "17:00"
	Timezone  string `json:"timezone"`    // e.g., "America/New_York"
}
