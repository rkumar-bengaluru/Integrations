package actions

import (
	"context"

	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func AddListUsersAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, err := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	var action *models.ActionDefinition
	// initialize action
	action = &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Retrieve a list of users in the Slack workspace",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of users to return",
					"default":     20,
				},
			},
			"required": []string{},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"ok": map[string]interface{}{
					"type":        "boolean",
					"description": "Indicates if the request succeeded",
				},
				"members": map[string]interface{}{
					"type":        "array",
					"description": "List of user objects",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":        map[string]interface{}{"type": "string"},
							"team_id":   map[string]interface{}{"type": "string"},
							"name":      map[string]interface{}{"type": "string"},
							"deleted":   map[string]interface{}{"type": "boolean"},
							"real_name": map[string]interface{}{"type": "string"},
							"tz":        map[string]interface{}{"type": "string"},
							"tz_label":  map[string]interface{}{"type": "string"},
							"tz_offset": map[string]interface{}{"type": "integer"},
							"profile": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"title":        map[string]interface{}{"type": "string"},
									"phone":        map[string]interface{}{"type": "string"},
									"skype":        map[string]interface{}{"type": "string"},
									"real_name":    map[string]interface{}{"type": "string"},
									"display_name": map[string]interface{}{"type": "string"},
									"email":        map[string]interface{}{"type": "string"},
									"image_72":     map[string]interface{}{"type": "string"},
								},
							},
							"is_admin":            map[string]interface{}{"type": "boolean"},
							"is_owner":            map[string]interface{}{"type": "boolean"},
							"is_primary_owner":    map[string]interface{}{"type": "boolean"},
							"is_restricted":       map[string]interface{}{"type": "boolean"},
							"is_ultra_restricted": map[string]interface{}{"type": "boolean"},
							"is_bot":              map[string]interface{}{"type": "boolean"},
							"updated":             map[string]interface{}{"type": "integer"},
						},
					},
				},
				"cache_ts": map[string]interface{}{
					"type":        "integer",
					"description": "Timestamp of cached data",
				},
				"response_metadata": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"next_cursor": map[string]interface{}{
							"type":        "string",
							"description": "Cursor for pagination",
						},
					},
				},
			},
			"required": []interface{}{"ok", "members"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err = integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}
	return action, nil
}

func AddInviteToChannelAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, err := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	var action *models.ActionDefinition
	// initialize action
	action = &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Invite one or more users to a Slack channel",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"channel": map[string]interface{}{
					"type":        "string",
					"description": "ID of the channel to invite users to",
					"source":      "user_input",
				},
				"users": map[string]interface{}{
					"type":        "array",
					"description": "List of user IDs to invite",
					"items":       map[string]interface{}{"type": "string"},
					"source":      "user_input",
				},
			},
			"required": []string{"channel", "users"},
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
						"is_channel":      map[string]interface{}{"type": "boolean"},
						"is_private":      map[string]interface{}{"type": "boolean"},
						"is_archived":     map[string]interface{}{"type": "boolean"},
						"is_general":      map[string]interface{}{"type": "boolean"},
						"is_shared":       map[string]interface{}{"type": "boolean"},
						"is_ext_shared":   map[string]interface{}{"type": "boolean"},
						"is_org_shared":   map[string]interface{}{"type": "boolean"},
						"is_member":       map[string]interface{}{"type": "boolean"},
						"num_members":     map[string]interface{}{"type": "integer"},
						"created":         map[string]interface{}{"type": "integer"},
						"updated":         map[string]interface{}{"type": "integer"},
						"creator":         map[string]interface{}{"type": "string"},
						"name_normalized": map[string]interface{}{"type": "string"},
						"topic": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"value":    map[string]interface{}{"type": "string"},
								"creator":  map[string]interface{}{"type": "string"},
								"last_set": map[string]interface{}{"type": "integer"},
							},
						},
						"purpose": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"value":    map[string]interface{}{"type": "string"},
								"creator":  map[string]interface{}{"type": "string"},
								"last_set": map[string]interface{}{"type": "integer"},
							},
						},
					},
				},
				"error": map[string]interface{}{
					"type":        "string",
					"description": "Error message if request failed",
				},
			},
			"required": []interface{}{"ok", "channel"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err = integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}
	return action, nil
}

func AddListChannelsAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, err := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	var action *models.ActionDefinition
	// initialize action
	action = &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Retrieve a list of channels in the Slack workspace",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of channels to return",
					"default":     20,
					"source":      "user_input",
				},
				"types": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of channel types (public_channel, private_channel, mpim, im)",
					"default":     "public_channels,private_channels, mpim, im",
					"source":      "user_input",
				},
			},
			"required": []string{},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"ok": map[string]interface{}{
					"type":        "boolean",
					"description": "Indicates if the request succeeded",
				},
				"channels": map[string]interface{}{
					"type":        "array",
					"description": "List of channel objects",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":              map[string]interface{}{"type": "string"},
							"name":            map[string]interface{}{"type": "string"},
							"name_normalized": map[string]interface{}{"type": "string"},
							"is_channel":      map[string]interface{}{"type": "boolean"},
							"is_private":      map[string]interface{}{"type": "boolean"},
							"is_archived":     map[string]interface{}{"type": "boolean"},
							"is_general":      map[string]interface{}{"type": "boolean"},
							"is_shared":       map[string]interface{}{"type": "boolean"},
							"is_ext_shared":   map[string]interface{}{"type": "boolean"},
							"is_org_shared":   map[string]interface{}{"type": "boolean"},
							"is_member":       map[string]interface{}{"type": "boolean"},
							"num_members":     map[string]interface{}{"type": "integer"},
							"created":         map[string]interface{}{"type": "integer"},
							"updated":         map[string]interface{}{"type": "integer"},
							"creator":         map[string]interface{}{"type": "string"},
							"topic": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"value":    map[string]interface{}{"type": "string"},
									"creator":  map[string]interface{}{"type": "string"},
									"last_set": map[string]interface{}{"type": "integer"},
								},
							},
							"purpose": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"value":    map[string]interface{}{"type": "string"},
									"creator":  map[string]interface{}{"type": "string"},
									"last_set": map[string]interface{}{"type": "integer"},
								},
							},
						},
					},
				},
				"response_metadata": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"next_cursor": map[string]interface{}{
							"type":        "string",
							"description": "Cursor for pagination",
						},
					},
				},
			},
			"required": []interface{}{"ok", "channels"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err = integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}
	return action, nil
}

func AddPostMessageToChannelAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, err := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	var action *models.ActionDefinition
	// initialize action
	action = &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Send a message to a Slack channel",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"channel": map[string]interface{}{
					"type":        "string",
					"description": "Channel ID to send the message to",
					"source":      "user_input",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text of the message to send",
					"source":      "user_input",
				},
			},
			"required": []string{"channel", "text"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"ok": map[string]interface{}{
					"type":        "boolean",
					"description": "Indicates if the request succeeded",
				},
				"channel": map[string]interface{}{
					"type":        "string",
					"description": "Channel ID where the message was posted",
				},
				"ts": map[string]interface{}{
					"type":        "string",
					"description": "Timestamp of the posted message",
				},
				"message": map[string]interface{}{
					"type":        "object",
					"description": "Message object returned by Slack",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{"type": "string"},
						"user": map[string]interface{}{"type": "string"},
						"text": map[string]interface{}{"type": "string"},
						"ts":   map[string]interface{}{"type": "string"},
					},
				},
				"error": map[string]interface{}{
					"type":        "string",
					"description": "Error message if request failed",
				},
			},
			"required": []interface{}{"ok", "channel", "ts", "message"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err = integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}
	return action, nil
}
