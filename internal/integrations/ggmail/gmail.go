package ggmail

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/ggmail/actions"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository/impl"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateGmailIntegration(ctx context.Context,
	database *gorm.DB,
	logger *zap.Logger,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Integration, error) {

	integrationRepo := impl.NewIntegrationRepository(database, logger)

	// check if integration already exist.
	_, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GmailIntegrationName)
	if err == repository.ErrIntegrationNotFound {
		// get the platform credential id for this integration
		credentials, err := credentialRepo.GetAllCredentials(ctx, GmailVendorName)
		if err != nil {
			credentials, err = AddGmailCredentials(ctx, encryptionSvc, credentialRepo, tenantID)
			if err != nil {
				return nil, fmt.Errorf("Error creating credential %s", GmailOauth2AuthorizationCodeFlowCredentialName)
			}
		}

		platformCredentials, err := commons.ConvertCredentialsToPlatform(credentials, nil)

		if err != nil {
			return nil, fmt.Errorf("Error creating platform credential %s", GmailOauth2AuthorizationCodeFlowCredentialName)
		}
		entityMap := make(models.ExecutionConfigs, 1)
		// Bot key
		entityMap[models.AuthOAuth2] = models.ExecutionConfig{
			CredentialType: models.AuthOAuth2,
			CredentialBinding: &models.CredentialBinding{
				CredentialType: models.AuthOAuth2,
				GrantType:      utils.Ptr(models.GrantTypeAuthorizationFlow),
				AuthorityUrl:   utils.Ptr("https://accounts.google.com/o/oauth2/v2/auth"),
				TokenUrl:       utils.Ptr("https://oauth2.googleapis.com/token"),
				Scopes: []string{
					"https://www.googleapis.com/auth/gmail.modify",
					"https://www.googleapis.com/auth/gmail.compose",
					"https://www.googleapis.com/auth/gmail.readonly",
					"https://www.googleapis.com/auth/gmail.labels",
					"https://www.googleapis.com/auth/userinfo.email",
				},
				SecretMapping: map[string]interface{}{
					"client_idy_key":    "client_id",
					"client_secret_key": "client_secret",
					"redirect_uri_key":  "redirect_uri",
					"scopes_key":        "scopes",
				},
				Notes: utils.Ptr("gmail authorization clode flow"),
			},
			ParamInputSchema: &models.JSONSchema{
				Title:       "Gmail Integration request fields",
				Description: "Fields required to call the Gmail API.",
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
					"redirect_uri": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.redirect_uri",
					},
					"scopes": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.scopes",
					},
				},
				Required: []string{"client_id", "client_secret", "redirect_uri", "scopes"},
			},
			ParamOuputSchema: &models.JSONSchema{
				Title:       "Gmail OAuth v2 Access Response",
				Description: "Fields returned from https://oauth2.googleapis.com/token",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"access_token": {
						Type:        "string",
						Description: "Top-level access token (bot or user)",
						Secret:      true,
						Source:      "$.access_token",
					},
					"refresh_token": {
						Type:        "integer",
						Description: "Lifetime of the top-level access token in seconds",
						Source:      "$.refresh_token",
					},
					"email_id": {
						Type:        "string",
						Description: "Token type: bot or user",
						Source:      "$.email_id",
					},
					"token_expiry": {
						Type:        "string",
						Description: "Scopes granted to the top-level token",
						Source:      "$.token_expiry",
					},
				},
			},
		}

		// create new integration
		integration := &models.Integration{
			Name:        GmailIntegrationName,
			TenantID:    tenantID,
			Description: "Integration with Gmail API's",
			Category:    models.CategoryCommunication,
			SupportedCredentialTypes: []models.CredentialType{
				models.AuthOAuth2,
			},
			PlatformCredentials: platformCredentials,
			ExecutionConfigs:    entityMap,
			Actions: []models.ActionDefinition{
				{
					TenantID:          tenantID,
					Name:              GmailTestActionName,
					Description:       "A simple test action to validate framework wiring",
					Type:              models.ActionType(GmailTestActionType),
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
			return nil, fmt.Errorf("Error creating integration %s, with error %w", GmailIntegrationName, err)
		}

		// create additional integration actions if there are new one added.
		err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
		if err != nil {
			return nil, err
		}

		integration, err = integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GmailIntegrationName)
		if err != nil {
			return nil, err
		}

		return integration, nil
	}

	integration, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GmailIntegrationName)
	if err != nil {
		return nil, err
	}
	// create additional integration actions if there are new one added.
	err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
	if err != nil {
		return nil, err
	}
	// integration already exist.
	logger.Debug(fmt.Sprintf("integration  %s already exit", GmailIntegrationName))
	return integration, nil
}

func AddActions(ctx context.Context,
	tenantID uuid.UUID,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration,
	logger *zap.Logger) error {
	action, err := actions.SendEmailAction(ctx, tenantID,
		GmailSendEmailActionName, GmailSendEmailActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.DraftEmailAction(ctx, tenantID,
		GmailDraftEmailActionName, GmailDraftEmailActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.DeleteEmailAction(ctx, tenantID,
		GmailDeleteEmailActionName, GmailDeleteEmailActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.ListEmailAction(ctx, tenantID,
		GmailLisEmailActionName, GmailListEmailActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	return nil
}
