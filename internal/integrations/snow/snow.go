package snow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/snow/actions"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"github.com/rkumar-bengaluru/Integrations/internal/repository"
	"github.com/rkumar-bengaluru/Integrations/internal/repository/impl"
	"github.com/rkumar-bengaluru/Integrations/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateSnowIntegration(ctx context.Context,
	database *gorm.DB,
	logger *zap.Logger,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Integration, error) {

	integrationRepo := impl.NewIntegrationRepository(database, logger)

	// check if integration already exist.
	_, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SnowIntegrationName)
	if err == repository.ErrIntegrationNotFound {
		// get the platform credential id for this integration
		credentials, err := credentialRepo.GetAllCredentials(ctx, SnowVendorName)
		if err != nil {
			credentials, err = AddSnowCredentials(ctx, encryptionSvc, credentialRepo, tenantID)
			if err != nil {
				return nil, fmt.Errorf("Error creating credential %s", SnowOauth2ClientCredentialFlowName)
			}
		}

		platformCredentials, err := commons.ConvertCredentialsToPlatform(credentials, nil)

		if err != nil {
			return nil, fmt.Errorf("Error creating platform credential %s", SnowOauth2ClientCredentialFlowName)
		}
		entityMap := make(models.ExecutionConfigs, 1)
		// Bot key
		entityMap[models.AuthOAuth2] = models.ExecutionConfig{
			CredentialType: models.AuthOAuth2,
			CredentialBinding: &models.CredentialBinding{
				CredentialType: models.AuthOAuth2,
				GrantType:      utils.Ptr(models.GrantTypeAuthClientCredential),
				AuthorityUrl:   utils.Ptr("https://dev212340.service-now.com//oauth_token.do"),
				SecretMapping: map[string]interface{}{
					"client_id_key":     "client_id",
					"client_secret_key": "client_secret",
					"grant_type_key":    "grant_type",
				},
				Notes: utils.Ptr("Just grab the xoxb key for bot integration"),
			},
			ParamInputSchema: &models.JSONSchema{
				Title:       "OpenAI Embedding request fields",
				Description: "Fields required to call the OpenAI Embeddings API.",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"client_id": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.client_id",
					},
					"client_secret": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.client_secret",
					},
					"grant_type": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.grant_type",
					},
				},
				Required: []string{"client_id", "client_secret", "grant_type"},
			},
			ParamOuputSchema: &models.JSONSchema{
				Title:       "Snow OAuth v2 Access Response",
				Description: "Fields returned from https://dev212340.service-now.com//oauth_token.do",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"access_token": {
						Type:        "string",
						Description: "Top-level access token (bot or user)",
						Secret:      true,
						Source:      "$.access_token",
					},
					"expires_in": {
						Type:        "integer",
						Description: "Lifetime of the top-level access token in seconds",
						Source:      "$.expires_in",
					},
					"token_type": {
						Type:        "string",
						Description: "Token type: bot or user",
						Source:      "$.token_type",
					},
					"scope": {
						Type:        "string",
						Description: "Scopes granted to the top-level token",
						Source:      "$.scope",
					},
					"error": {
						Type:        "string",
						Description: "Error message if request failed",
						Source:      "$.error",
					},
				},
			},
		}

		// create new integration
		integration := &models.Integration{
			Name:        SnowIntegrationName,
			TenantID:    tenantID,
			Description: "Integration with Snow for ITSM",
			Category:    models.CategoryCommunication,
			SupportedCredentialTypes: []models.CredentialType{
				models.AuthOAuth2,
			},
			PlatformCredentials: platformCredentials,
			ExecutionConfigs:    entityMap,
			Actions: []models.ActionDefinition{
				{
					TenantID:          tenantID,
					Name:              SnowTestActionName,
					Description:       "A simple test action to validate framework wiring",
					Type:              models.ActionType(SnowTestActionType),
					SchemaVersion:     "v1",
					SupportsStreaming: false,
					IsInternal:        false,
					Version:           "1.0",
					IsActive:          true,
					ActionHandler:     "test_action_handler",
					InputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"message": map[string]interface{}{
								"type":        "string",
								"description": "A sample input message to test the action",
								"source":      "user_input",
							},
						},
						"required": []string{"message"},
					},
					OutputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]interface{}{
								"type":        "boolean",
								"description": "Indicates if the test action executed successfully",
							},
							"echo": map[string]interface{}{
								"type":        "string",
								"description": "Echoes back the input string",
							},
						},
						"required": []interface{}{"success", "echo"},
					},
				},
			},
		}

		err = integrationRepo.CreateIntegration(ctx, integration)
		if err != nil {
			return nil, fmt.Errorf("Error creating integration %s, with error %w", SnowIntegrationName, err)
		}

		// create additional integration actions if there are new one added.
		err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
		if err != nil {
			return nil, err
		}

		integration, err = integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SnowIntegrationName)
		if err != nil {
			return nil, err
		}

		return integration, nil
	}

	integration, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SnowIntegrationName)
	if err != nil {
		return nil, err
	}
	// create additional integration actions if there are new one added.
	err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
	if err != nil {
		return nil, err
	}
	// integration already exist.
	logger.Debug(fmt.Sprintf("integration  %s already exit", SnowIntegrationName))
	return integration, nil
}

func AddActions(ctx context.Context,
	tenantID uuid.UUID,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration,
	logger *zap.Logger) error {
	action, err := actions.CreateIncidentAction(ctx, tenantID,
		SnowCreateIncidentActionName, SnowCreateIncidentActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.GetIncidentByNumberAction(ctx, tenantID,
		SnowGetIncidentActionName, SnowGetIncidentActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.SearchSimilarIncidentAction(ctx, tenantID,
		SnowSearchIncidentActionName, SnowSearchIncidentActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.UpdateIncidentAction(ctx, tenantID,
		SnowUpdateIncidentActionName, SnowUpdateIncidentActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	return nil
}
