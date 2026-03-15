package repository

import (
	"context"
	"errors"

	"agent.fabric.com/modules/internal/api"
	"agent.fabric.com/modules/internal/models"
	"github.com/google/uuid"
)

var (
	ErrCredentialNotFound = errors.New("credential not found")
)

var (
	ErrIntegrationBindingNotFound = errors.New("integration binding not found")
	ErrDuplicateBinding           = errors.New("integration binding already exists")
)

var (
	ErrKnowledgeBaseNotFound  = errors.New("knowledge base not found")
	ErrDocumentNotFound       = errors.New("document not found")
	ErrDuplicateKnowledgeBase = errors.New("knowledge base with same name already exists")
	ErrIntegrationNotFound    = errors.New("integration not found")
	ErrActionHandlerNotFound  = errors.New("action handler not found")
)

type IntegrationBindingRepository interface {
	// CRUD Operations
	Create(ctx context.Context, binding *models.IntegrationBinding) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.IntegrationBinding, error)
	GetByIDWithAssociations(ctx context.Context, id uuid.UUID) (*models.IntegrationBinding, error)
	GetByTenanIDAndIntegrationID(ctx context.Context, tenantId, integrationID uuid.UUID) (*models.IntegrationBinding, error)
	Update(ctx context.Context, binding *models.IntegrationBinding) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List Operations
	List(ctx context.Context, filters api.BindingFilters) ([]models.IntegrationBinding, int64, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.IntegrationBinding, int64, error)
	ListByIntegration(ctx context.Context, integrationID uuid.UUID) ([]models.IntegrationBinding, error)
	ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.IntegrationBinding, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.IntegrationBinding, error)

	// find integration binding by credential type.
	FindIntegrationBinding(ctx context.Context, credType models.CredentialType, integrationID uuid.UUID) (*models.IntegrationBinding, error)

	// Specific Queries
	GetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) (*models.IntegrationBinding, error)
	GetByCredentialID(ctx context.Context, credentialID uuid.UUID) (*models.IntegrationBinding, error)

	// Transaction Support
	WithTransaction(fn func(repo IntegrationBindingRepository) error) error

	// Validation Status
	UpdateValidationStatus(ctx context.Context, id uuid.UUID, status models.ValidationStatus, errorMsg string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// Default binding management
	UnsetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) error
	SetAsDefault(ctx context.Context, id uuid.UUID) error
}

// IntegrationRepository defines all database operations for integrations
type IntegrationRepository interface {
	// Integration operations
	CreateIntegration(ctx context.Context, integration *models.Integration) error
	GetIntegrationByID(ctx context.Context, id uuid.UUID) (*models.Integration, error)
	GetIntegrationByTenantIDAndName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Integration, error)
	GetIntegrationByIDWithActions(ctx context.Context, id uuid.UUID) (*models.Integration, error)
	ListIntegrations(ctx context.Context, tenantID uuid.UUID, category models.IntegrationCategory, isGlobal *bool, search string, page, pageSize int) ([]models.Integration, int64, error)
	UpdateIntegration(ctx context.Context, integration *models.Integration) error
	DeleteIntegration(ctx context.Context, id uuid.UUID) error

	// Get Credential for credential type
	GetCredential(ctx context.Context, id uuid.UUID, credType models.CredentialType) (*models.Credential, error)

	// ActionDefinition operations
	CreateActionDefinition(ctx context.Context, integrationID uuid.UUID, action *models.ActionDefinition) error
	GetActionDefinitionByID(ctx context.Context, id uuid.UUID) (*models.ActionDefinition, error)
	GetActionDefinitionByType(ctx context.Context, actionType models.ActionType, tenantID uuid.UUID) (*models.ActionDefinition, error)
	GetActionDefinitionByName(ctx context.Context, actionType models.ActionType, actionName string, tenantID uuid.UUID) (*models.ActionDefinition, error)
	ListActionDefinitions(ctx context.Context, tenantID uuid.UUID, category models.ActionType, isActive *bool, page, pageSize int) ([]models.ActionDefinition, int64, error)
	UpdateActionDefinition(ctx context.Context, action *models.ActionDefinition) error
	DeleteActionDefinition(ctx context.Context, id uuid.UUID) error
	CheckIfActionDefinitionExist(ctx context.Context,
		integrationRepo IntegrationRepository,
		actionType, actionName string, tenantID uuid.UUID) (*models.ActionDefinition, error)

	// Association operations
	AddActionToIntegration(ctx context.Context, integrationID, actionID uuid.UUID) error
	RemoveActionFromIntegration(ctx context.Context, integrationID, actionID uuid.UUID) error
	GetIntegrationActions(ctx context.Context, integrationID uuid.UUID) ([]models.ActionDefinition, error)

	// Transaction support
	WithTransaction(fn func(repo IntegrationRepository) error) error
}

type CredentialRepository interface {
	Create(ctx context.Context, credential *models.Credential) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Credential, error)
	GetByTenantIDAndName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Credential, error)
	Update(ctx context.Context, credential *models.Credential) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateValidationStatus(ctx context.Context, id uuid.UUID, status models.ValidationStatus, errorMsg string) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// Get All credentials by vendor.
	GetAllCredentials(ctx context.Context, vendor string) ([]*models.Credential, error)
}
