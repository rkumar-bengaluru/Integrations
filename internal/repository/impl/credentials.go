package impl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type credentialRepo struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCredentialRepository(db *gorm.DB, logger *zap.Logger) repository.CredentialRepository {
	return &credentialRepo{db: db, logger: logger}
}

func (r *credentialRepo) Create(ctx context.Context, credential *models.Credential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *credentialRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Credential, error) {
	var cred models.Credential
	if err := r.db.WithContext(ctx).First(&cred, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrCredentialNotFound
		}
		return nil, err
	}
	return &cred, nil
}

func (r *credentialRepo) GetByTenantIDAndName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Credential, error) {
	var cred models.Credential
	if err := r.db.WithContext(ctx).First(&cred, "tenant_id = ? AND name= ?", tenantID, name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrCredentialNotFound
		}
		return nil, err
	}
	return &cred, nil
}

func (r *credentialRepo) Update(ctx context.Context, credential *models.Credential) error {
	return r.db.WithContext(ctx).Save(credential).Error
}

func (r *credentialRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Credential{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrCredentialNotFound
	}
	return nil
}

func (r *credentialRepo) UpdateValidationStatus(ctx context.Context, id uuid.UUID, status models.ValidationStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"validation_status": status,
		"validation_error":  errorMsg,
	}
	if status == models.ValidationValid {
		now := time.Now()
		updates["validated_at"] = &now
	}
	return r.db.WithContext(ctx).
		Model(&models.Credential{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *credentialRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Credential{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}

func (r *credentialRepo) GetAllCredentials(
	ctx context.Context,
	vendor string,
) ([]*models.Credential, error) {
	var creds []*models.Credential

	// Query all credentials for the given vendor
	err := r.db.WithContext(ctx).
		Where("provider = ?", vendor).
		Find(&creds).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch credentials for vendor %s: %w", vendor, err)
	}

	// Defensive check: empty result
	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials found for vendor %s", vendor)
	}

	return creds, nil
}
