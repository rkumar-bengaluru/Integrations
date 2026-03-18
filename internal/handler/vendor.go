package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ActionResult struct {
	Data       interface{}
	StatusCode int
	Error      error
	Metadata   map[string]interface{}
}

// IntegrationHandler is the minimal interface for all integrations
type IntegrationHandler interface {
	// Execute an action with runtime-resolved config
	Execute(ctx context.Context, config *models.ExecutionConfig, actionDef *models.ActionDefinition, binding models.IntegrationBinding, inputs map[string]interface{}) (*ActionResult, error)

	// Test if binding configuration is valid
	TestConnection(ctx context.Context, config *models.ExecutionConfig, binding models.IntegrationBinding) error

	// Init performs integration-specific initialization during binding creation
	// Returns key-value pairs to be stored in Credential.EncryptedData
	// This allows integrations to generate tokens, validate configs, etc.
	Init(ctx context.Context, config *models.ExecutionConfig, collectedParams map[string]interface{}) (map[string]interface{}, error)
}

// IntegrationHandler is the minimal interface for all integrations
type Binding interface {
	CreateIntegrationBindingWithOwner(
		integration *models.Integration,
		credentialConfig *models.ExecutionConfig,
		params []commons.CollectedParam,
		tenantID uuid.UUID,
		userID *string,
		encryptionSvc encryption.EncryptionService,
	) (*models.IntegrationBinding, error)

	CreateIntegrationBinding(
		integration *models.Integration,
		credentialConfig *models.ExecutionConfig,
		params []commons.CollectedParam,
		encryptionSvc encryption.EncryptionService,
	) (*models.IntegrationBinding, error)
}

// GitHubHandler implements all GitHub actions using Integration config
type BindingHandler struct {
	handler IntegrationHandler
	repo    repository.IntegrationBindingRepository
	logger  *zap.Logger
}

// Ensure GitHubHandler implements IntegrationHandler
var _ Binding = (*BindingHandler)(nil)

func NewBindingHandler(handler IntegrationHandler,
	repo repository.IntegrationBindingRepository,
	logger *zap.Logger) *BindingHandler {
	return &BindingHandler{
		handler: handler,
		repo:    repo,
		logger:  logger,
	}
}

// createIntegrationBindingWithOwner creates a binding with explicit ownership
func (h *BindingHandler) CreateIntegrationBindingWithOwner(
	integration *models.Integration,
	credentialConfig *models.ExecutionConfig,
	params []commons.CollectedParam,
	tenantID uuid.UUID,
	userID *string,
	encryptionSvc encryption.EncryptionService,
) (*models.IntegrationBinding, error) {

	binding, err := h.CreateIntegrationBinding(integration, credentialConfig, params, encryptionSvc)
	h.logger.Debug("return binding created for ", zap.String("", integration.Name))

	if err != nil {
		h.logger.Debug("return binding with error for ", zap.String("", integration.Name))
		return nil, err
	}

	binding.TenantID = tenantID
	binding.UserID = userID

	return binding, nil
}

// createIntegrationBinding creates a fully populated models.IntegrationBinding
// using the collected parameters from the binding collector
func (h *BindingHandler) CreateIntegrationBinding(
	integration *models.Integration,
	credentialConfig *models.ExecutionConfig,
	params []commons.CollectedParam,
	encryptionSvc encryption.EncryptionService,
) (*models.IntegrationBinding, error) {

	// Build secrets map from collected parameters
	secrets := make(map[string]interface{})
	for _, param := range params {
		secrets[param.StorageKey] = param.Value
	}

	// Call handler.Init to get complete credential data
	// This is where GitHub generates tokens, Gmail exchanges auth codes, etc.
	ctx := context.Background()
	h.logger.Debug("calling the handler to fetch any handler specific data")
	credentialData, err := h.handler.Init(ctx, credentialConfig, secrets)
	if err != nil {
		h.logger.Error("handler init failed", zap.Error(err))
		return nil, fmt.Errorf("handler initialization failed: %w", err)
	}
	h.logger.Debug("handler init successful.")
	// Marshal secrets to JSON
	secretsJSON, err := json.Marshal(credentialData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Encrypt the secrets data
	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Generate hash for integrity verification
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	// Determine credential type from integration's supported types
	credentialType := credentialConfig.CredentialType
	// if len(integration.SupportedCredentialTypes) > 0 {
	// 	credentialType = integration.SupportedCredentialTypes[0]
	// }

	// Build credential
	credential := &models.Credential{
		Name:          fmt.Sprintf("%s Binding Credentials", integration.Name),
		Description:   fmt.Sprintf("Binding credentials for %s integration", integration.Name),
		Type:          credentialType,
		EncryptedData: encryptedData,
		DataHash:      dataHash,
		Scopes:        credentialConfig.CredentialBinding.Scopes,
		TenantID:      integration.TenantID,
	}

	// Set expiration if token_expiry_key exists in secrets
	if expiryValue, exists := secrets["token_expiry_key"]; exists {
		if expiryStr, ok := expiryValue.(string); ok {
			if expiryTime, err := time.Parse(time.RFC3339, expiryStr); err == nil {
				credential.ExpiresAt = &expiryTime
			}
		}
	}

	// Unmarshal into a generic map for inspection
	var secretsMap map[string]interface{}
	if err := json.Unmarshal(secretsJSON, &secretsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets: %w", err)
	}
	var userID string
	// Check if "user_id" exists
	if val, ok := secretsMap["user_id"]; ok {
		// Take some action
		fmt.Printf("user_id found: %v\n", val)
		// e.g., cast to string if you expect it
		userID, _ = val.(string)
		// do something with userID
	}

	// Build the integration binding
	binding := &models.IntegrationBinding{
		IntegrationID: integration.ID,
		TenantID:      integration.TenantID,
		UserID:        &userID,
		Credential:    credential,
		Status:        string(models.ValidationUnverified),
	}

	// finally save it.
	err = h.repo.Create(ctx, binding)
	return binding, err
}
