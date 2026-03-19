// service/integration_binding_service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/api"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
)

var (
	ErrBindingNotFound       = errors.New("integration binding not found")
	ErrCredentialCreation    = errors.New("failed to create credential")
	ErrInvalidCredentialType = errors.New("invalid credential type")
	ErrValidationFailed      = errors.New("credential validation failed")
)

type IntegrationBindingService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req api.CreateIntegrationBindingRequest) (*api.IntegrationBindingResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*api.IntegrationBindingResponse, error)
	Update(ctx context.Context, id uuid.UUID, req api.UpdateIntegrationBindingRequest) (*api.IntegrationBindingResponse, error)
	UpdateCredential(ctx context.Context, cred *models.Credential) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, req api.ListIntegrationBindingsRequest) (*api.IntegrationBindingListResponse, error)
	ValidateBinding(ctx context.Context, id uuid.UUID) (*api.ValidateBindingResponse, error)
	GetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) (*api.IntegrationBindingResponse, error)
}

type integrationBindingService struct {
	bindingRepo    repository.IntegrationBindingRepository
	credentialRepo repository.CredentialRepository
	encryptionSvc  encryption.EncryptionService
	validator      CredentialValidator
}

func NewIntegrationBindingService(
	bindingRepo repository.IntegrationBindingRepository,
	credentialRepo repository.CredentialRepository,
	encryptionSvc encryption.EncryptionService,
	validator *CredentialValidator,
) IntegrationBindingService {
	return &integrationBindingService{
		bindingRepo:    bindingRepo,
		credentialRepo: credentialRepo,
		encryptionSvc:  encryptionSvc,
		validator:      *validator,
	}
}

func (s *integrationBindingService) UpdateCredential(ctx context.Context, cred *models.Credential) error {
	return s.credentialRepo.Update(ctx, cred)
}

func (s *integrationBindingService) Create(ctx context.Context, tenantID uuid.UUID, req api.CreateIntegrationBindingRequest) (*api.IntegrationBindingResponse, error) {
	var createdBinding *models.IntegrationBinding

	err := s.bindingRepo.WithTransaction(func(txRepo repository.IntegrationBindingRepository) error {
		// 1. Create Credential first from embedded DTO
		credential, err := s.createCredentialFromDTO(ctx, req.Credential)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCredentialCreation, err)
		}

		// 2. Handle default binding logic
		if req.IsDefault {
			if err := txRepo.UnsetDefaultForIntegration(ctx, tenantID, req.IntegrationID); err != nil {
				return err
			}
		}

		// 4. Create Integration Binding
		binding := &models.IntegrationBinding{
			TenantID:         tenantID,
			IntegrationID:    req.IntegrationID,
			CredentialID:     &credential.ID,
			AgentID:          req.AgentID,
			UserID:           req.UserID,
			Status:           "pending",
			ValidationStatus: models.ValidationUnverified,
			Credential:       credential,
		}

		if err := txRepo.Create(ctx, binding); err != nil {
			return err
		}

		createdBinding = binding
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Load associations for response
	fullBinding, err := s.bindingRepo.GetByIDWithAssociations(ctx, createdBinding.ID)
	if err != nil {
		return nil, err
	}

	// Async validation
	go s.asyncValidateBinding(context.Background(), fullBinding)

	return s.toResponse(fullBinding), nil
}

func (s *integrationBindingService) createCredentialFromDTO(ctx context.Context,
	credDto api.CreateCredentialEmbeddedDTO) (*models.Credential, error) {
	// Validate credential type
	if !isValidCredentialTypeDTO(credDto.Type) {
		return nil, ErrInvalidCredentialType
	}

	// Encrypt secrets
	encryptedData, err := s.encryptionSvc.EncryptSecrets(credDto.Secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Generate hash for deduplication detection
	dataHash := s.encryptionSvc.GenerateDataHash(encryptedData)
	v, _ := ToCredentialTypeEntity(string(credDto.Type))
	uuid.MustParse("")
	credential := &models.Credential{
		TenantID:         uuid.New(), // Will be set from binding context
		Name:             credDto.Name,
		Description:      credDto.Description,
		Type:             v,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           credDto.Scopes,
		ExpiresAt:        credDto.ExpiresAt,
		ValidationStatus: models.ValidationUnverified,
	}

	return credential, nil
}

func (s *integrationBindingService) generatePreview(execConfig models.ExecutionConfig, credential *models.Credential) map[string]interface{} {
	preview := make(map[string]interface{})

	// Add credential metadata (non-sensitive)
	preview["credential_type"] = credential.Type
	preview["credential_name"] = credential.Name

	return preview
}

func (s *integrationBindingService) asyncValidateBinding(ctx context.Context, binding *models.IntegrationBinding) {
	// Decrypt credential for validation
	// This would integrate with your validation logic per credential type
	result := s.validator.Validate(ctx, binding.Credential)

	status := models.ValidationValid
	errorMsg := ""
	if !result.Valid {
		status = models.ValidationInvalid
		errorMsg = result.Error
	}

	// Update both credential and binding validation status
	s.credentialRepo.UpdateValidationStatus(ctx, binding.Credential.ID, status, errorMsg)
	s.bindingRepo.UpdateValidationStatus(ctx, binding.ID, status, errorMsg)

	// Update status to active if valid
	if result.Valid {
		s.bindingRepo.UpdateStatus(ctx, binding.ID, "active")
	}
}

func (s *integrationBindingService) GetByID(ctx context.Context, id uuid.UUID) (*api.IntegrationBindingResponse, error) {
	binding, err := s.bindingRepo.GetByIDWithAssociations(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrIntegrationBindingNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return s.toResponse(binding), nil
}

func (s *integrationBindingService) Update(ctx context.Context, id uuid.UUID, req api.UpdateIntegrationBindingRequest) (*api.IntegrationBindingResponse, error) {
	binding, err := s.bindingRepo.GetByIDWithAssociations(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Status != nil {
		binding.Status = *req.Status
	}

	// Update credential secrets if provided
	if req.CredentialSecrets != nil {
		newCredential, err := s.createCredentialFromDTO(ctx, api.CreateCredentialEmbeddedDTO{
			Name:        binding.Credential.Name,
			Description: binding.Credential.Description,
			Type:        api.CredentialType(binding.Credential.Type),
			Secrets:     *req.CredentialSecrets,
		})
		if err != nil {
			return nil, err
		}

		// Create new credential version
		if err := s.credentialRepo.Create(ctx, newCredential); err != nil {
			return nil, err
		}

		// Update binding to point to new credential
		oldCredentialID := binding.CredentialID
		binding.CredentialID = &newCredential.ID

		// Delete old credential
		s.credentialRepo.Delete(ctx, *oldCredentialID)
	}

	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return nil, err
	}

	return s.toResponse(binding), nil
}

func (s *integrationBindingService) Delete(ctx context.Context, id uuid.UUID) error {
	binding, err := s.bindingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete binding first (transaction would be better here)
	if err := s.bindingRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Delete associated credential
	if binding.CredentialID != nil {
		return s.credentialRepo.Delete(ctx, *binding.CredentialID)
	}

	return nil
}

func (s *integrationBindingService) List(ctx context.Context, req api.ListIntegrationBindingsRequest) (*api.IntegrationBindingListResponse, error) {
	filters := api.BindingFilters{
		TenantID:      req.TenantID,
		IntegrationID: req.IntegrationID,
		AgentID:       req.AgentID,
		UserID:        req.UserID,
		Status:        req.Status,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}

	bindings, total, err := s.bindingRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	items := make([]api.IntegrationBindingResponse, len(bindings))
	for i, b := range bindings {
		items[i] = *s.toResponse(&b)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &api.IntegrationBindingListResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *integrationBindingService) ValidateBinding(ctx context.Context, id uuid.UUID) (*api.ValidateBindingResponse, error) {
	binding, err := s.bindingRepo.GetByIDWithAssociations(ctx, id)
	if err != nil {
		return nil, err
	}

	result := s.validator.Validate(ctx, binding.Credential)

	return &api.ValidateBindingResponse{
		Valid:   result.Valid,
		Message: result.Message,
		Error:   result.Error,
	}, nil
}

func (s *integrationBindingService) GetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) (*api.IntegrationBindingResponse, error) {
	binding, err := s.bindingRepo.GetDefaultForIntegration(ctx, tenantID, integrationID)
	if err != nil {
		return nil, err
	}
	return s.toResponse(binding), nil
}

func (s *integrationBindingService) toResponse(binding *models.IntegrationBinding) *api.IntegrationBindingResponse {
	resp := &api.IntegrationBindingResponse{
		ID:               binding.ID,
		TenantID:         binding.TenantID,
		IntegrationID:    binding.IntegrationID,
		CredentialID:     *binding.CredentialID,
		AgentID:          binding.AgentID,
		UserID:           binding.UserID,
		Status:           string(binding.Status),
		ValidationStatus: string(binding.ValidationStatus),
		ValidationError:  binding.ValidationError,
		CreatedAt:        binding.CreatedAt,
		UpdatedAt:        binding.UpdatedAt,
		Credential: api.CredentialPreviewDTO{
			ID:          binding.Credential.ID,
			Name:        binding.Credential.Name,
			Description: binding.Credential.Description,
			Type:        string(binding.Credential.Type),
			// OAuth2Config:      ToOAuth2ConfigDTO(binding.Credential.OAuth2Config),
			ExpiresAt:        binding.Credential.ExpiresAt,
			LastUsedAt:       binding.Credential.LastUsedAt,
			ValidationStatus: string(binding.Credential.ValidationStatus),
			ValidatedAt:      binding.Credential.ValidatedAt,
		},
	}
	return resp
}

func isValidCredentialType(t models.CredentialType) bool {
	validTypes := []models.CredentialType{
		models.AuthOAuth2, models.AuthOAuth2PKCE, models.AuthOAuth2Device,
		models.AuthAPIKey, models.AuthBearerToken, models.AuthBasicAuth,
		models.AuthMTLS, models.AuthMCPSession, models.AuthAWSSigV4,
		models.AuthAzureManaged, models.AuthGCPServiceAcc, models.AuthJWT,
		models.AuthSAML, models.AuthDatabase, models.AuthSSHKey,
	}
	for _, vt := range validTypes {
		if t == vt {
			return true
		}
	}
	return false
}

func isValidCredentialTypeDTO(t api.CredentialType) bool {
	validTypes := []models.CredentialType{
		models.AuthOAuth2, models.AuthOAuth2PKCE, models.AuthOAuth2Device,
		models.AuthAPIKey, models.AuthBearerToken, models.AuthBasicAuth,
		models.AuthMTLS, models.AuthMCPSession, models.AuthAWSSigV4,
		models.AuthAzureManaged, models.AuthGCPServiceAcc, models.AuthJWT,
		models.AuthSAML, models.AuthDatabase, models.AuthSSHKey,
	}
	for _, vt := range validTypes {
		if string(t) == string(vt) {
			return true
		}
	}
	return false
}
