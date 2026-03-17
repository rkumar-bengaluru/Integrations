package impl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type integrationRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewIntegrationRepository creates a new integration repository
func NewIntegrationRepository(db *gorm.DB, logger *zap.Logger) repository.IntegrationRepository {
	return &integrationRepository{db: db, logger: logger}
}

// WithTransaction executes operations within a database transaction
func (r *integrationRepository) WithTransaction(fn func(repo repository.IntegrationRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &integrationRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *integrationRepository) CheckIfActionDefinitionExist(ctx context.Context,
	integrationRepo repository.IntegrationRepository,
	actionType, actionName string, tenantID uuid.UUID) (*models.ActionDefinition, error) {

	action, err := integrationRepo.GetActionDefinitionByName(ctx, models.ActionType(actionType), actionName, tenantID)

	if err != nil {
		r.logger.Debug("error in finding action definition", zap.String("", actionName))
		return nil, err
	}
	r.logger.Debug("action definition found in database", zap.String("", action.Name))
	return action, nil
}

// CreateIntegration creates a new integration with its actions
func (r *integrationRepository) CreateIntegration(ctx context.Context, integration *models.Integration) error {
	r.logger.Info("creating_integration",
		zap.String("integration_name", integration.Name),
		zap.String("category", string(integration.Category)),
		zap.Int("action_count", len(integration.Actions)),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Create(integration).Error
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_create_integration",
			zap.Error(err),
			zap.String("integration_name", integration.Name),
			zap.Duration("duration_ms", duration),
		)
		return fmt.Errorf("failed to create integration: %w", err)
	}

	r.logger.Info("integration_created_successfully",
		zap.String("integration_id", integration.ID.String()),
		zap.Duration("duration_ms", duration),
		zap.Int("actions_created", len(integration.Actions)),
	)

	return nil
}

func (r *integrationRepository) GetCredential(
	ctx context.Context,
	integrationId uuid.UUID,
	credType models.CredentialType,
) (*models.Credential, error) {
	var ic models.PlatformCredential

	// Query the join table for this integration + credential type
	err := r.db.WithContext(ctx).
		Preload("Credential").
		Where("integration_id = ? AND credential_type = ?", integrationId, credType).
		First(&ic).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no credential found for type %s", credType)
		}
		return nil, fmt.Errorf("failed to load credential for type %s: %w", credType, err)
	}

	return ic.Credential, nil
}

func (r *integrationRepository) GetIntegrationByTenantIDAndName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Integration, error) {
	r.logger.Debug("GetIntegrationByTenantIDAndName", zap.String("tenantID", tenantID.String()))

	var integration models.Integration
	start := time.Now()
	err := r.db.WithContext(ctx).Preload("Actions").Preload("PlatformCredentials").
		Preload("PlatformCredentials.Credential").
		First(&integration, "tenant_id = ? AND name = ?", tenantID, name).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("integration_not_found", zap.String("tenantID", tenantID.String()),
				zap.String(name, name))
			return nil, repository.ErrIntegrationNotFound
		}
		r.logger.Error("failed_to_fetch_integration",
			zap.Error(err),
			zap.String("integration_id", tenantID.String()),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch integration: %w", err)
	}

	r.logger.Debug("integration_fetched",
		zap.String("integration_id", tenantID.String()),
		zap.String("integration_name", integration.Name),
		zap.Duration("duration_ms", duration),
	)

	return &integration, nil
}

// GetIntegrationByID retrieves an integration by ID without actions
func (r *integrationRepository) GetIntegrationByID(ctx context.Context, id uuid.UUID) (*models.Integration, error) {
	r.logger.Debug("fetching_integration_by_id", zap.String("integration_id", id.String()))

	var integration models.Integration
	start := time.Now()
	err := r.db.WithContext(ctx).Preload("PlatformCredentials").First(&integration, "id = ?", id).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("integration_not_found", zap.String("integration_id", id.String()))
			return nil, fmt.Errorf("integration not found: %s", id)
		}
		r.logger.Error("failed_to_fetch_integration",
			zap.Error(err),
			zap.String("integration_id", id.String()),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch integration: %w", err)
	}

	r.logger.Debug("integration_fetched",
		zap.String("integration_id", id.String()),
		zap.String("integration_name", integration.Name),
		zap.Duration("duration_ms", duration),
	)

	return &integration, nil
}

// GetIntegrationByIDWithActions retrieves an integration with its associated actions
func (r *integrationRepository) GetIntegrationByIDWithActions(ctx context.Context, id uuid.UUID) (*models.Integration, error) {
	r.logger.Debug("fetching_integration_with_actions", zap.String("integration_id", id.String()))

	var integration models.Integration
	start := time.Now()
	err := r.db.WithContext(ctx).
		Preload("Actions").
		First(&integration, "id = ?", id).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("integration_not_found", zap.String("integration_id", id.String()))
			return nil, fmt.Errorf("integration not found: %s", id)
		}
		r.logger.Error("failed_to_fetch_integration_with_actions",
			zap.Error(err),
			zap.String("integration_id", id.String()),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch integration with actions: %w", err)
	}

	r.logger.Info("integration_with_actions_fetched",
		zap.String("integration_id", id.String()),
		zap.String("integration_name", integration.Name),
		zap.Int("action_count", len(integration.Actions)),
		zap.Duration("duration_ms", duration),
	)

	return &integration, nil
}

// ListIntegrations retrieves paginated integrations with filtering
func (r *integrationRepository) ListIntegrations(ctx context.Context, tenantID uuid.UUID, category models.IntegrationCategory, isGlobal *bool, search string, page, pageSize int) ([]models.Integration, int64, error) {
	r.logger.Info("listing_integrations",
		zap.String("tenant_id", tenantID.String()),
		zap.String("category_filter", string(category)),
		zap.String("search_term", search),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
	)

	query := r.db.WithContext(ctx).Preload("PlatformCredentials").Model(&models.Integration{})

	// Apply tenant filter (global or specific tenant)
	if tenantID != uuid.Nil {
		query = query.Where("tenant_id = ? OR is_global = ?", tenantID, true)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if isGlobal != nil {
		query = query.Where("is_global = ?", *isGlobal)
	}

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	start := time.Now()
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("failed_to_count_integrations", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count integrations: %w", err)
	}

	offset := (page - 1) * pageSize
	var integrations []models.Integration
	err := query.
		Preload("Actions").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&integrations).Error
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_list_integrations",
			zap.Error(err),
			zap.Duration("duration_ms", duration),
		)
		return nil, 0, fmt.Errorf("failed to list integrations: %w", err)
	}

	r.logger.Info("integrations_listed",
		zap.Int64("total", total),
		zap.Int("returned", len(integrations)),
		zap.Duration("duration_ms", duration),
	)

	return integrations, total, nil
}

// UpdateIntegration updates an existing integration
func (r *integrationRepository) UpdateIntegration(ctx context.Context, integration *models.Integration) error {
	r.logger.Info("updating_integration",
		zap.String("integration_id", integration.ID.String()),
		zap.String("integration_name", integration.Name),
	)

	start := time.Now()
	result := r.db.WithContext(ctx).Model(integration).Updates(map[string]interface{}{
		"name":        integration.Name,
		"description": integration.Description,
		"category":    integration.Category,
		// "supported_credential_types": integration.SupportedCredentialTypes,
		// "execution_config":           integration.ExecutionConfigs,
		"updated_at": time.Now(),
	})
	duration := time.Since(start)

	if result.Error != nil {
		r.logger.Error("failed_to_update_integration",
			zap.Error(result.Error),
			zap.String("integration_id", integration.ID.String()),
			zap.Duration("duration_ms", duration),
		)
		return fmt.Errorf("failed to update integration: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("integration_not_found_for_update", zap.String("integration_id", integration.ID.String()))
		return fmt.Errorf("integration not found: %s", integration.ID)
	}

	r.logger.Info("integration_updated",
		zap.String("integration_id", integration.ID.String()),
		zap.Int64("rows_affected", result.RowsAffected),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// DeleteIntegration soft-deletes an integration
func (r *integrationRepository) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	r.logger.Warn("deleting_integration", zap.String("integration_id", id.String()))

	start := time.Now()
	result := r.db.WithContext(ctx).Delete(&models.Integration{}, "id = ?", id)
	duration := time.Since(start)

	if result.Error != nil {
		r.logger.Error("failed_to_delete_integration",
			zap.Error(result.Error),
			zap.String("integration_id", id.String()),
			zap.Duration("duration_ms", duration),
		)
		return fmt.Errorf("failed to delete integration: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("integration_not_found_for_delete", zap.String("integration_id", id.String()))
		return fmt.Errorf("integration not found: %s", id)
	}

	r.logger.Info("integration_deleted",
		zap.String("integration_id", id.String()),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// CreateActionDefinition creates a new action definition
func (r *integrationRepository) CreateActionDefinition(ctx context.Context, integraitonID uuid.UUID, action *models.ActionDefinition) error {
	r.logger.Info("creating_action_definition",
		zap.String("action_name", action.Name),
		zap.String("action_type", string(action.Type)),
	)

	start := time.Now()
	err := r.db.WithContext(ctx).Create(action).Error
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_create_action",
			zap.Error(err),
			zap.String("action_name", action.Name),
			zap.String("action_type", string(action.Type)),
			zap.Duration("duration_ms", duration),
		)
		return fmt.Errorf("failed to create action definition: %w", err)
	}

	err = r.AddActionToIntegration(ctx, integraitonID, action.ID)

	if err != nil {
		r.logger.Error("failed to AddActionToIntegration",
			zap.Error(err),
			zap.String("action_name", action.Name),
			zap.String("action_type", string(action.Type)),
			zap.Duration("duration_ms", duration),
		)
		return fmt.Errorf("failed to create action definition: %w", err)
	}
	r.logger.Info("action_created",
		zap.String("action_id", action.ID.String()),
		zap.String("action_type", string(action.Type)),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// GetActionDefinitionByID retrieves an action by ID
func (r *integrationRepository) GetActionDefinitionByID(ctx context.Context, id uuid.UUID) (*models.ActionDefinition, error) {
	r.logger.Debug("fetching_action_by_id", zap.String("action_id", id.String()))

	var action models.ActionDefinition
	start := time.Now()
	err := r.db.WithContext(ctx).First(&action, "id = ?", id).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("action_not_found", zap.String("action_id", id.String()))
			return nil, fmt.Errorf("action definition not found: %s", id)
		}
		r.logger.Error("failed_to_fetch_action",
			zap.Error(err),
			zap.String("action_id", id.String()),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch action: %w", err)
	}

	return &action, nil
}

// GetActionDefinitionByType retrieves an action by its unique type identifier
func (r *integrationRepository) GetActionDefinitionByType(ctx context.Context, actionType models.ActionType, tenantID uuid.UUID) (*models.ActionDefinition, error) {
	r.logger.Debug("fetching_action_by_type",
		zap.String("action_type", string(actionType)),
		zap.String("tenant_id", tenantID.String()),
	)

	var action models.ActionDefinition
	query := r.db.WithContext(ctx).Where("type = ?", actionType)

	// If tenant specified, check tenant-specific or global actions
	if tenantID != uuid.Nil {
		query = query.Where("tenant_id = ? OR is_internal = ?", tenantID, true)
	}

	start := time.Now()
	err := query.First(&action).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("action_not_found_by_type", zap.String("action_type", string(actionType)))
			return nil, fmt.Errorf("action type not found: %s", actionType)
		}
		r.logger.Error("failed_to_fetch_action_by_type",
			zap.Error(err),
			zap.String("action_type", string(actionType)),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch action by type: %w", err)
	}

	return &action, nil
}

func (r *integrationRepository) GetActionDefinitionByName(ctx context.Context, actionType models.ActionType, actionName string, tenantID uuid.UUID) (*models.ActionDefinition, error) {
	r.logger.Debug("GetActionDefinitionByName",
		zap.String("action_type", string(actionType)),
		zap.String("name", actionName),
		zap.String("tenant_id", tenantID.String()),
	)

	var action models.ActionDefinition
	query := r.db.WithContext(ctx).Where("type = ? AND name = ? AND tenant_id = ?", actionType, actionName, tenantID)

	start := time.Now()
	err := query.First(&action).Error
	duration := time.Since(start)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warn("action_not_found_by_type", zap.String("action_type", string(actionType)))
			return nil, fmt.Errorf("action type not found: %s", actionType)
		}
		r.logger.Error("failed_to_fetch_action_by_type",
			zap.Error(err),
			zap.String("action_type", string(actionType)),
			zap.Duration("duration_ms", duration),
		)
		return nil, fmt.Errorf("failed to fetch action by type: %w", err)
	}

	return &action, nil
}

// ListActionDefinitions retrieves paginated action definitions
func (r *integrationRepository) ListActionDefinitions(ctx context.Context, tenantID uuid.UUID, category models.ActionType, isActive *bool, page, pageSize int) ([]models.ActionDefinition, int64, error) {
	r.logger.Info("listing_actions",
		zap.String("tenant_id", tenantID.String()),
		zap.String("category", string(category)),
		zap.Int("page", page),
	)

	query := r.db.WithContext(ctx).Model(&models.ActionDefinition{})

	if tenantID != uuid.Nil {
		query = query.Where("tenant_id = ? OR is_internal = ?", tenantID, true)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	var total int64
	start := time.Now()
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("failed_to_count_actions", zap.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var actions []models.ActionDefinition
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&actions).Error
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_list_actions", zap.Error(err), zap.Duration("duration_ms", duration))
		return nil, 0, fmt.Errorf("failed to list actions: %w", err)
	}

	r.logger.Info("actions_listed", zap.Int64("total", total), zap.Int("returned", len(actions)))
	return actions, total, nil
}

// UpdateActionDefinition updates an existing action
func (r *integrationRepository) UpdateActionDefinition(ctx context.Context, action *models.ActionDefinition) error {
	r.logger.Info("updating_action",
		zap.String("action_id", action.ID.String()),
		zap.String("action_name", action.Name),
	)

	start := time.Now()
	result := r.db.WithContext(ctx).Model(action).Updates(map[string]interface{}{
		"name":               action.Name,
		"description":        action.Description,
		"category":           action.Type,
		"supports_streaming": action.SupportsStreaming,
		"is_active":          action.IsActive,
		"input_schema":       action.InputSchema,
		"output_schema":      action.OutputSchema,
		"action_handler":     action.ActionHandler,
		"updated_at":         time.Now(),
	})
	duration := time.Since(start)

	if result.Error != nil {
		r.logger.Error("failed_to_update_action",
			zap.Error(result.Error),
			zap.String("action_id", action.ID.String()),
		)
		return fmt.Errorf("failed to update action: %w", result.Error)
	}

	r.logger.Info("action_updated",
		zap.String("action_id", action.ID.String()),
		zap.Int64("rows_affected", result.RowsAffected),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// DeleteActionDefinition soft-deletes an action
func (r *integrationRepository) DeleteActionDefinition(ctx context.Context, id uuid.UUID) error {
	r.logger.Warn("deleting_action", zap.String("action_id", id.String()))

	start := time.Now()
	result := r.db.WithContext(ctx).Delete(&models.ActionDefinition{}, "id = ?", id)
	duration := time.Since(start)

	if result.Error != nil {
		r.logger.Error("failed_to_delete_action",
			zap.Error(result.Error),
			zap.String("action_id", id.String()),
		)
		return fmt.Errorf("failed to delete action: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("action_not_found_for_delete", zap.String("action_id", id.String()))
		return fmt.Errorf("action not found: %s", id)
	}

	r.logger.Info("action_deleted",
		zap.String("action_id", id.String()),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// AddActionToIntegration associates an action with an integration
func (r *integrationRepository) AddActionToIntegration(ctx context.Context, integrationID, actionID uuid.UUID) error {
	r.logger.Info("adding_action_to_integration",
		zap.String("integration_id", integrationID.String()),
		zap.String("action_id", actionID.String()),
	)

	// GORM many-to-many association
	integration := models.Integration{BaseModel: models.BaseModel{ID: integrationID}}
	action := models.ActionDefinition{BaseModel: models.BaseModel{ID: actionID}}

	start := time.Now()
	err := r.db.WithContext(ctx).Model(&integration).Association("Actions").Append(&action)
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_add_action_to_integration",
			zap.Error(err),
			zap.String("integration_id", integrationID.String()),
			zap.String("action_id", actionID.String()),
		)
		return fmt.Errorf("failed to add action to integration: %w", err)
	}

	r.logger.Info("action_added_to_integration",
		zap.String("integration_id", integrationID.String()),
		zap.String("action_id", actionID.String()),
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// RemoveActionFromIntegration removes an action from an integration
func (r *integrationRepository) RemoveActionFromIntegration(ctx context.Context, integrationID, actionID uuid.UUID) error {
	r.logger.Info("removing_action_from_integration",
		zap.String("integration_id", integrationID.String()),
		zap.String("action_id", actionID.String()),
	)

	integration := models.Integration{BaseModel: models.BaseModel{ID: integrationID}}
	action := models.ActionDefinition{BaseModel: models.BaseModel{ID: actionID}}

	start := time.Now()
	err := r.db.WithContext(ctx).Model(&integration).Association("Actions").Delete(&action)
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_remove_action_from_integration",
			zap.Error(err),
			zap.String("integration_id", integrationID.String()),
			zap.String("action_id", actionID.String()),
		)
		return fmt.Errorf("failed to remove action from integration: %w", err)
	}

	r.logger.Info("action_removed_from_integration",
		zap.Duration("duration_ms", duration),
	)

	return nil
}

// GetIntegrationActions retrieves all actions for an integration
func (r *integrationRepository) GetIntegrationActions(ctx context.Context, integrationID uuid.UUID) ([]models.ActionDefinition, error) {
	r.logger.Debug("fetching_integration_actions", zap.String("integration_id", integrationID.String()))

	var integration models.Integration
	start := time.Now()
	err := r.db.WithContext(ctx).Preload("Actions").First(&integration, "id = ?", integrationID).Error
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("failed_to_fetch_integration_actions",
			zap.Error(err),
			zap.String("integration_id", integrationID.String()),
		)
		return nil, fmt.Errorf("failed to fetch integration actions: %w", err)
	}

	r.logger.Debug("integration_actions_fetched",
		zap.String("integration_id", integrationID.String()),
		zap.Int("action_count", len(integration.Actions)),
		zap.Duration("duration_ms", duration),
	)

	return integration.Actions, nil
}
