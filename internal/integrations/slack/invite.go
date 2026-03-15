package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository/impl"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (c *SlackHandler) InviteToChannel(
	ctx context.Context,
	runtimeConfig *SlackRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	c.logger.Debug("finally action time for slack", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate required fields
	if runtimeConfig.AccessToken == nil || *runtimeConfig.AccessToken == "" {
		return nil, fmt.Errorf("missing AccessToken for bot flow")
	}

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	c.logger.Debug("input validation successful")

	// Extract required params
	channelID, ok := inputs["channel"].(string)
	if !ok || channelID == "" {
		return &handler.ActionResult{
			StatusCode: http.StatusBadRequest,
			Error:      fmt.Errorf("missing or invalid channel ID"),
		}, nil
	}

	users, ok := inputs["users"].([]string)
	if !ok || len(users) == 0 {
		return &handler.ActionResult{
			StatusCode: http.StatusBadRequest,
			Error:      fmt.Errorf("missing or invalid users list"),
		}, nil
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"channel": channelID,
		"users":   strings.Join(users, ","),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to marshal payload: %w", err),
		}, nil
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://slack.com/api/conversations.invite", bytes.NewBuffer(body))
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call Slack API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	// Decode Slack response
	var slackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&slackResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode Slack response: %w", err),
		}, nil
	}

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, slackResp, "output"); err != nil {
		c.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	}
	c.logger.Debug("Output validation successful")

	// Check for Slack error
	if okVal, exists := slackResp["ok"].(bool); exists && !okVal {
		return &handler.ActionResult{
			Data:       slackResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("slack error: %v", slackResp["error"]),
		}, nil
	}

	// Return ActionResult with Slack response
	return &handler.ActionResult{
		Data:       slackResp,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func AddInviteToChannelAction(ctx context.Context,
	database *gorm.DB,
	logger *zap.Logger,
	tenantID uuid.UUID,
	actionName, actionType, integrationName string) (*models.ActionDefinition, error) {

	integrationRepo := impl.NewIntegrationRepository(database, logger)
	resp, err := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	// check if integration already exist.
	integration, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, integrationName)

	if err != nil {
		return nil, fmt.Errorf("integration %s missing", integrationName)
	}

	var action *models.ActionDefinition
	// initialize action
	action = &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              SlackInviteUsersActionName,
		Description:       "Invite one or more users to a Slack channel",
		Type:              models.ActionType(SlackInviteUsersActionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     "slack_invite_to_channel",
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
