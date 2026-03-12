package models

import (
	"time"

	"github.com/google/uuid"
)

// this is the run time binding of an integration with the credential.
type IntegrationBinding struct {
	BaseModel

	TenantID      uuid.UUID  `gorm:"column:tenant_id;not null;index:uniq_tenant_integration_agent_user,unique"`
	IntegrationID uuid.UUID  `gorm:"column:integration_id;not null;index:uniq_tenant_integration_agent_user,unique"`
	AgentID       *uuid.UUID `gorm:"column:agent_id;index:uniq_tenant_integration_agent_user,unique"`
	UserID        *string    `gorm:"column:user_id;index:uniq_tenant_integration_agent_user,unique"`

	Status           string           `gorm:"column:status;type:varchar(32);default:'pending';index" json:"status"`
	ValidationStatus ValidationStatus `gorm:"column:validation_status;type:varchar(32);default:'unverified'" json:"validation_status"`
	ValidationError  string           `gorm:"column:validation_error;type:text" json:"validation_error,omitempty"`
	ValidatedAt      *time.Time       `gorm:"column:validated_at" json:"validated_at,omitempty"`

	Integration *Integration `gorm:"foreignKey:IntegrationID" json:"integration,omitempty"`

	CredentialID *uuid.UUID  `gorm:"column:credential_id;not null;index" json:"credential_id"`
	Credential   *Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
}
