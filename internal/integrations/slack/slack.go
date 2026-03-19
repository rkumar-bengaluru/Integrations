package slack

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/slack/actions"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"github.com/rkumar-bengaluru/Integrations/internal/repository"
	"github.com/rkumar-bengaluru/Integrations/internal/repository/impl"
	"github.com/rkumar-bengaluru/Integrations/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateSlackIntegration(ctx context.Context,
	database *gorm.DB,
	logger *zap.Logger,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Integration, error) {

	integrationRepo := impl.NewIntegrationRepository(database, logger)

	// check if integration already exist.
	_, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SlackIntegrationName)
	if err == repository.ErrIntegrationNotFound {
		// get the platform credential id for this integration
		credentials, err := credentialRepo.GetAllCredentials(ctx, SlackVendorName)
		if err != nil {
			credentials, err = AddCredentials(ctx, encryptionSvc, credentialRepo, tenantID)
			if err != nil {
				return nil, fmt.Errorf("Error creating credential %s", SlackBotCredentialName)
			}
		}

		platformCredentials, err := commons.ConvertCredentialsToPlatform(credentials, nil)

		if err != nil {
			return nil, fmt.Errorf("Error creating platform credential %s", SlackBotCredentialName)
		}
		entityMap := make(models.ExecutionConfigs, 2)
		// Bot key
		entityMap[models.AuthAPIKey] = models.ExecutionConfig{
			CredentialType: models.AuthAPIKey,
			CredentialBinding: &models.CredentialBinding{
				CredentialType: models.AuthAPIKey,
				GrantType:      utils.Ptr(models.GrantTypeAuthAPIKey),
				AuthorityUrl:   utils.Ptr("https://slack.com/api/oauth.v2.exchange"),
				TokenUrl:       utils.Ptr("https://slack.com/api/oauth.v2.access"),
				Scopes: []string{
					"assistant:write",
					"channels:history",
					"channels:join",
					"channels:manage",
					"channels:read",
					"channels:write.invites",
					"chat:write",
					"conversations.connect:manage",
					"conversations.connect:read",
					"conversations.connect:write",
					"im:write",
					"users:read",
					"users:read.email",
				},
				SecretMapping: map[string]interface{}{
					"long_lived_token_key": "long_lived_token",
					"client_id_key":        "client_id",
					"client_secret_key":    "client_secret",
					"user_scope_key":       "user_scopes",
					"bot_scope_key":        "bot_scopes",
					"redirect_url_key":     "redirect_url",
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
					"long_lived_token": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.long_lived_token",
					},
					"user_scopes": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.user_scopes",
					},
					"bot_scopes": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.bot_scopes",
					},
					"redirect_url": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.redirect_url",
					},
					"access_token": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.runtime",
					},
					"refresh_token": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.runtime",
					},
				},
				Required: []string{"client_id", "client_secret", "bot_scopes", "long_lived_token"},
			},
			ParamOuputSchema: &models.JSONSchema{
				Title:       "Slack OAuth v2 Access Response",
				Description: "Fields returned from https://slack.com/api/oauth.v2.access",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"ok": {
						Type:        "boolean",
						Description: "Indicates if the request succeeded",
						Required:    true,
						Source:      "$.ok",
					},
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
					"refresh_token": {
						Type:        "string",
						Description: "Top-level refresh token (if offline.access requested)",
						Secret:      true,
						Source:      "$.refresh_token",
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
					"app_id": {
						Type:        "string",
						Description: "App ID",
						Source:      "$.app_id",
					},
					"team": {
						Type:        "object",
						Description: "Workspace information",
						Required:    true,
						AdditionalProperties: map[string]interface{}{
							"id": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Team ID",
								Source:      "$.team.id",
								Required:    true,
							},
							"name": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Team name",
								Source:      "$.team.name",
								Required:    true,
							},
						},
					},
					"bot_user_id": {
						Type:        "string",
						Description: "Bot user ID (present for bot tokens)",
						Source:      "$.bot_user_id",
					},
					"authed_user": {
						Type:        "object",
						Description: "Information about the authorized user (present for user tokens)",
						AdditionalProperties: map[string]interface{}{
							"id": models.JSONSchemaProperty{
								Type:        "string",
								Description: "User ID",
								Source:      "$.authed_user.id",
								Required:    true,
							},
							"scope": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Scopes granted to the user token",
								Source:      "$.authed_user.scope",
							},
							"access_token": models.JSONSchemaProperty{
								Type:        "string",
								Description: "User access token (xoxe.xoxp-...)",
								Secret:      true,
								Source:      "$.authed_user.access_token",
								Required:    true,
							},
							"expires_in": models.JSONSchemaProperty{
								Type:        "integer",
								Description: "Lifetime of the user access token in seconds",
								Source:      "$.authed_user.expires_in",
							},
							"refresh_token": models.JSONSchemaProperty{
								Type:        "string",
								Description: "User refresh token (if offline.access requested)",
								Secret:      true,
								Source:      "$.authed_user.refresh_token",
							},
							"token_type": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Token type for the user token",
								Source:      "$.authed_user.token_type",
							},
						},
					},
					"error": {
						Type:        "string",
						Description: "Error message if request failed",
						Source:      "$.error",
					},
				},
			},
		}
		// User key
		entityMap[models.AuthOAuth2] = models.ExecutionConfig{
			CredentialType: models.AuthOAuth2,
			CredentialBinding: &models.CredentialBinding{
				CredentialType: models.AuthOAuth2,
				GrantType:      utils.Ptr(models.GrantTypeAuthorizationFlow),
				AuthorityUrl:   utils.Ptr("https://slack.com/oauth/v2/authorize"),
				TokenUrl:       utils.Ptr("https://slack.com/api/oauth.v2.exchange"),
				Scopes: []string{
					"admin",
					"channels:history",
					"channels:read",
					"channels:write",
					"channels:write.invites",
					"channels:write.topic",
					"chat:write",
					"im:write",
					"users:read",
					"users:read.email",
				},
				SecretMapping: map[string]interface{}{
					"client_id_key":     "client_id",
					"client_secret_key": "client_secret",
					"user_scope_key":    "user_scopes",
					"redirect_url_key":  "redirect_url",
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
					"user_scopes": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.user_scopes",
					},
					"redirect_url": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.redirect_url",
					},
					"user_id": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.runtime",
					},
					"access_token": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.runtime",
					},
				},
				Required: []string{"client_id", "client_secret", "user_scopes", "redirect_url"},
			},
			ParamOuputSchema: &models.JSONSchema{
				Title:       "Slack OAuth v2 Access Response",
				Description: "Fields returned from https://slack.com/api/oauth.v2.access",
				Type:        "object",

				Properties: map[string]models.JSONSchemaProperty{
					"ok": {
						Type:        "boolean",
						Description: "Indicates if the request succeeded",
						Secret:      false,
						Source:      "$.ok",
					},
					"authed_user": {
						Type:        "object",
						Description: "Information about the authorized user",
						Required:    true,
						AdditionalProperties: map[string]interface{}{
							"id": models.JSONSchemaProperty{
								Type:        "string",
								Description: "User ID",
								Source:      "$.authed_user.id",
								Required:    true,
							},
							"access_token": models.JSONSchemaProperty{
								Type:        "string",
								Description: "User access token (xoxe.xoxp-...)",
								Secret:      true,
								Source:      "$.authed_user.access_token",
								Required:    true,
							},
							"scope": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Scopes granted to the user token",
								Source:      "$.authed_user.scope",
							},
						},
					},
					"team": {
						Type:        "object",
						Description: "Workspace information",
						Required:    true,
						AdditionalProperties: map[string]interface{}{
							"id": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Team ID",
								Source:      "$.team.id",
								Required:    true,
							},
							"name": models.JSONSchemaProperty{
								Type:        "string",
								Description: "Team name",
								Source:      "$.team.name",
								Required:    true,
							},
						},
					},
				},
				Required: []string{"ok", "authed_user", "team"},
			},
		}

		// create new integration
		integration := &models.Integration{
			Name:        SlackIntegrationName,
			TenantID:    tenantID,
			Description: "Integration with Slack for user communication & bot communication",
			Category:    models.CategoryCommunication,
			SupportedCredentialTypes: []models.CredentialType{
				models.AuthAPIKey,
				models.AuthOAuth2,
			},
			PlatformCredentials: platformCredentials,
			ExecutionConfigs:    entityMap,
			Actions: []models.ActionDefinition{
				{
					TenantID:          tenantID,
					Name:              SlackCreateChannelActionName,
					Description:       "Create a new Slack channel",
					Type:              models.ActionType(SlackCreateChannelActionType),
					SchemaVersion:     "v1",
					SupportsStreaming: false,
					IsInternal:        false,
					Version:           "1.0",
					IsActive:          true,
					ActionHandler:     "slack_create_channel",
					InputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Name of the channel to create",
								"source":      "user_input",
							},
						},
						"required": []string{"name"},
					},
					OutputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"ok": map[string]interface{}{
								"type":        "boolean",
								"description": "Indicates if the request succeeded",
							},
							"channel": map[string]interface{}{
								"type":        "object",
								"description": "Channel object returned by Slack",
								"properties": map[string]interface{}{
									"id":              map[string]interface{}{"type": "string"},
									"name":            map[string]interface{}{"type": "string"},
									"name_normalized": map[string]interface{}{"type": "string"},
									"creator":         map[string]interface{}{"type": "string"},
									"created":         map[string]interface{}{"type": "integer"},
									"updated":         map[string]interface{}{"type": "integer"},
									"is_channel":      map[string]interface{}{"type": "boolean"},
									"is_private":      map[string]interface{}{"type": "boolean"},
									"is_archived":     map[string]interface{}{"type": "boolean"},
									"is_general":      map[string]interface{}{"type": "boolean"},
									"is_shared":       map[string]interface{}{"type": "boolean"},
									"is_ext_shared":   map[string]interface{}{"type": "boolean"},
									"is_org_shared":   map[string]interface{}{"type": "boolean"},
									"is_member":       map[string]interface{}{"type": "boolean"},
									"last_read":       map[string]interface{}{"type": "string"},
									"priority":        map[string]interface{}{"type": "integer"},
									"previous_names":  map[string]interface{}{"type": "array"},
									"shared_team_ids": map[string]interface{}{"type": "array"},
									"purpose": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"value":    map[string]interface{}{"type": "string"},
											"creator":  map[string]interface{}{"type": "string"},
											"last_set": map[string]interface{}{"type": "integer"},
										},
									},
									"topic": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"value":    map[string]interface{}{"type": "string"},
											"creator":  map[string]interface{}{"type": "string"},
											"last_set": map[string]interface{}{"type": "integer"},
										},
									},
								},
							},
							"warning": map[string]interface{}{
								"type":        "string",
								"description": "Warning message if present",
							},
							"response_metadata": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"warnings": map[string]interface{}{
										"type":  "array",
										"items": map[string]interface{}{"type": "string"},
									},
								},
							},
						},
						"required": []interface{}{"ok", "channel"},
					},
				},
			},
		}
		err = integrationRepo.CreateIntegration(ctx, integration)
		if err != nil {
			return nil, fmt.Errorf("Error creating integration %s, with error %w", SlackIntegrationName, err)
		}

		// create additional integration actions if there are new one added.
		err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
		if err != nil {
			return nil, err
		}

		integration, err = integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SlackIntegrationName)
		if err != nil {
			return nil, err
		}

		return integration, nil
	}

	integration, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, SlackIntegrationName)
	if err != nil {
		return nil, err
	}
	// create additional integration actions if there are new one added.
	err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
	if err != nil {
		return nil, err
	}
	// integration already exist.
	logger.Debug(fmt.Sprintf("integration  %s already exit", SlackIntegrationName))
	return integration, nil

}

func AddActions(ctx context.Context,
	tenantID uuid.UUID,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration,
	logger *zap.Logger) error {
	action, err := actions.AddListUsersAction(ctx, tenantID,
		SlackListUsersActionName, SlackListUsersActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.AddInviteToChannelAction(ctx, tenantID,
		SlackInviteUsersActionName, SlackInviteUsersActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.AddListChannelsAction(ctx, tenantID,
		SlackListChannelsActionName, SlackListChannelsActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.AddPostMessageToChannelAction(ctx, tenantID,
		SlackPostMessageActionName, SlackPostMessageActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	return nil
}
