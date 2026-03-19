package impl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/api"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type integrationBindingRepo struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewIntegrationBindingRepository(db *gorm.DB, logger *zap.Logger) repository.IntegrationBindingRepository {
	return &integrationBindingRepo{db: db, logger: logger}
}

func (r *integrationBindingRepo) Create(ctx context.Context, binding *models.IntegrationBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *integrationBindingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.IntegrationBinding, error) {
	var binding models.IntegrationBinding
	if err := r.db.WithContext(ctx).First(&binding, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrIntegrationBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *integrationBindingRepo) GetByTenanIDAndIntegrationID(ctx context.Context, tenantID, integrationID uuid.UUID) (*models.IntegrationBinding, error) {
	var binding models.IntegrationBinding
	if err := r.db.WithContext(ctx).
		Preload("Integration").
		Preload("Integration.PlatformCredential").
		Preload("Integration.Actions").
		Preload("Credential").
		First(&binding, "tenant_id = ? AND integration_id = ?", tenantID, integrationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrIntegrationBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *integrationBindingRepo) FindIntegrationBinding(
	ctx context.Context,
	credType models.CredentialType,
	integrationID uuid.UUID) (*models.IntegrationBinding, error) {

	var bindings []models.IntegrationBinding
	err := r.db.WithContext(ctx).
		Preload("Integration").
		Preload("Credential").
		Where("integration_id = ?", integrationID).
		Find(&bindings).Error

	if err != nil {
		return nil, err
	}

	for _, binding := range bindings {
		if binding.Credential.Type == credType {
			r.logger.Debug("found credential for ", zap.String("credType", string(credType)),
				zap.String("integration", binding.Integration.Name),
				zap.String("credential name", binding.Credential.Name))
			return &binding, nil
		}
	}
	return nil, fmt.Errorf("could not find credential for %s and for type %s", integrationID.String(), credType)
}

func (r *integrationBindingRepo) GetByIDWithAssociations(ctx context.Context, id uuid.UUID) (*models.IntegrationBinding, error) {

	var binding models.IntegrationBinding
	if err := r.db.WithContext(ctx).
		Preload("Integration").
		Preload("Credential").
		First(&binding, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrIntegrationBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *integrationBindingRepo) Update(ctx context.Context, binding *models.IntegrationBinding) error {
	return r.db.WithContext(ctx).Save(binding).Error
}

func (r *integrationBindingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.IntegrationBinding{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrIntegrationBindingNotFound
	}
	return nil
}

func (r *integrationBindingRepo) List(ctx context.Context, filters api.BindingFilters) ([]models.IntegrationBinding, int64, error) {
	var bindings []models.IntegrationBinding
	var total int64

	query := r.db.WithContext(ctx).Model(&models.IntegrationBinding{})

	if filters.TenantID != nil {
		query = query.Where("tenant_id = ?", *filters.TenantID)
	}
	if filters.IntegrationID != nil {
		query = query.Where("integration_id = ?", *filters.IntegrationID)
	}
	if filters.AgentID != nil {
		query = query.Where("agent_id = ?", *filters.AgentID)
	}
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.PageSize
	if err := query.
		Preload("Credential").
		Preload("Integration").
		Order("created_at DESC").
		Offset(offset).
		Limit(filters.PageSize).
		Find(&bindings).Error; err != nil {
		return nil, 0, err
	}

	return bindings, total, nil
}

func (r *integrationBindingRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.IntegrationBinding, int64, error) {
	return r.List(ctx, api.BindingFilters{
		TenantID: &tenantID,
		Page:     page,
		PageSize: pageSize,
	})
}

func (r *integrationBindingRepo) ListByIntegration(ctx context.Context, integrationID uuid.UUID) ([]models.IntegrationBinding, error) {
	var bindings []models.IntegrationBinding
	err := r.db.WithContext(ctx).
		Preload("Credential").
		Where("integration_id = ?", integrationID).
		Find(&bindings).Error
	return bindings, err
}

func (r *integrationBindingRepo) ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.IntegrationBinding, error) {
	var bindings []models.IntegrationBinding
	err := r.db.WithContext(ctx).
		Preload("Credential").
		Where("agent_id = ?", agentID).
		Find(&bindings).Error
	return bindings, err
}

func (r *integrationBindingRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.IntegrationBinding, error) {
	var bindings []models.IntegrationBinding
	err := r.db.WithContext(ctx).
		Preload("Credential").
		Where("user_id = ?", userID).
		Find(&bindings).Error
	return bindings, err
}

func (r *integrationBindingRepo) GetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) (*models.IntegrationBinding, error) {
	var binding models.IntegrationBinding
	if err := r.db.WithContext(ctx).
		Preload("Credential").
		Where("tenant_id = ? AND integration_id = ? AND is_default = ?", tenantID, integrationID, true).
		First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrIntegrationBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *integrationBindingRepo) GetByCredentialID(ctx context.Context, credentialID uuid.UUID) (*models.IntegrationBinding, error) {
	var binding models.IntegrationBinding
	if err := r.db.WithContext(ctx).
		Where("credential_id = ?", credentialID).
		First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrIntegrationBindingNotFound
		}
		return nil, err
	}
	return &binding, nil
}

func (r *integrationBindingRepo) WithTransaction(fn func(repo repository.IntegrationBindingRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &integrationBindingRepo{db: tx}
		return fn(txRepo)
	})
}

func (r *integrationBindingRepo) UpdateValidationStatus(ctx context.Context, id uuid.UUID, status models.ValidationStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"validation_status": status,
		"validation_error":  errorMsg,
	}
	if status == models.ValidationValid {
		now := time.Now()
		updates["validated_at"] = &now
	}
	return r.db.WithContext(ctx).
		Model(&models.IntegrationBinding{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *integrationBindingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.IntegrationBinding{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *integrationBindingRepo) UnsetDefaultForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.IntegrationBinding{}).
		Where("tenant_id = ? AND integration_id = ? AND is_default = ?", tenantID, integrationID, true).
		Update("is_default", false).Error
}

func (r *integrationBindingRepo) SetAsDefault(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.IntegrationBinding{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}
