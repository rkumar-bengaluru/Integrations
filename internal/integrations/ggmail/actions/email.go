package actions

import (
	"context"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
	"go.uber.org/zap"
)

func SendEmailAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Send an email via Gmail",
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
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Recipient email address",
					"source":      "user_input",
				},
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Email subject line",
					"source":      "user_input",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Plain text email body",
					"source":      "user_input",
				},
			},
			"required": []string{"to", "subject", "body"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Unique ID of the sent message",
				},
				"threadId": map[string]interface{}{
					"type":        "string",
					"description": "Thread ID the message belongs to",
				},
				"labelIds": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Labels applied to the sent message",
				},
			},
			"required": []interface{}{"id", "threadId"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func DraftEmailAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Create a draft email in Gmail",
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
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Recipient email address",
					"source":      "user_input",
				},
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Email subject line",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Email body content",
				},
				"is_html": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the body is HTML (default false)",
					"default":     false,
				},
			},
			"required": []string{"to", "subject", "body"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"draft_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique ID of the created draft",
				},
				"message_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique ID of the draft message",
				},
				"thread_id": map[string]interface{}{
					"type":        "string",
					"description": "Thread ID the draft belongs to",
				},
				"status_code": map[string]interface{}{
					"type":        "integer",
					"description": "HTTP status code of the operation",
				},
			},
			"required": []interface{}{"draft_id", "message_id", "thread_id", "status_code"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func DeleteEmailAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Delete a Gmail message (move to trash or permanently delete)",
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
				"message_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique ID of the Gmail message to delete",
					"source":      "user_input",
				},
				"permanent": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, permanently deletes the message; if false, moves it to trash",
					"default":     false,
				},
			},
			"required": []string{"message_id"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"message_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the deleted message",
				},
				"permanent": map[string]interface{}{
					"type":        "boolean",
					"description": "Indicates whether deletion was permanent",
				},
				"status_code": map[string]interface{}{
					"type":        "integer",
					"description": "HTTP status code of the operation",
				},
			},
			"required": []interface{}{"message_id", "permanent", "status_code"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func ListEmailAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "List Gmail messages with optional query filters",
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
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Gmail search query string (e.g., 'subject:report')",
					"source":      "user_input",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of messages to return",
					"default":     5,
					"maximum":     500,
				},
				"include_spam_trash": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to include messages from Spam and Trash",
					"default":     false,
				},
			},
			"required": []string{},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"messages": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":            map[string]interface{}{"type": "string"},
							"thread_id":     map[string]interface{}{"type": "string"},
							"snippet":       map[string]interface{}{"type": "string"},
							"label_ids":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
							"history_id":    map[string]interface{}{"type": "string"},
							"internal_date": map[string]interface{}{"type": "integer"},
						},
						"required": []interface{}{"id", "thread_id"},
					},
				},
				"next_page_token": map[string]interface{}{
					"type":        "string",
					"description": "Token for the next page of results",
				},
				"result_size_estimate": map[string]interface{}{
					"type":        "integer",
					"description": "Estimated total number of results",
				},
				"status_code": map[string]interface{}{
					"type":        "integer",
					"description": "HTTP status code of the operation",
				},
			},
			"required": []interface{}{"messages", "status_code"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}
