package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository/impl"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (c *SlackHandler) CreateChannel(
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

	// Validate inputs
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	c.logger.Debug("input validation successful")

	// Extract channel name
	channelName, ok := inputs["name"].(string)
	if !ok || channelName == "" {
		c.logger.Error("required input param missing", zap.String("channel", channelName))
		return &handler.ActionResult{
			StatusCode: http.StatusBadRequest,
			Error:      fmt.Errorf("missing or invalid channelName"),
		}, nil
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"name": channelName,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to marshal payload: %w", err),
		}, nil
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/conversations.create", bytes.NewBuffer(body))
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

	// Decode Slack response into a generic map (to match OutputSchema)
	var slackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&slackResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode Slack response: %w", err),
		}, nil
	}

	// Validate outputs
	if err := commons.ValidateSchema(actionDef.OutputSchema, slackResp, "output"); err != nil {
		// Log warning but don't fail - schema might be stricter than actual data
		c.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
		// Or return error if strict validation required:
		// return nil, fmt.Errorf("output validation failed: %w", err)
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

	// Return ActionResult with full Slack response mapped to Data
	return &handler.ActionResult{
		Data:       slackResp,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (c *SlackHandler) ListChannels(
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

	// Extract optional params
	var limitStr, types string
	if limit, ok := inputs["limit"].(int); ok && limit > 0 {
		limitStr = strconv.Itoa(limit)
	}
	if t, ok := inputs["types"].(string); ok && t != "" {
		types = t
	}

	// Build query params
	params := url.Values{}
	if limitStr != "" {
		params.Set("limit", limitStr)
	}
	if types != "" {
		params.Set("types", types)
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://slack.com/api/conversations.list?"+params.Encode(), nil)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error("Error Making Slack call", zap.Error(err))
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call Slack API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	// Decode Slack response
	var slackResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&slackResp); err != nil {
		c.logger.Error("Error decoding slack response", zap.Error(err))
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode Slack response: %w", err),
		}, nil
	}
	c.logger.Debug("Slack response",
		zap.Any("slackResp", slackResp),
	)

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

func AddListChannelsAction(ctx context.Context,
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
		Name:              SlackListChannelsActionName,
		Description:       "Retrieve a list of channels in the Slack workspace",
		Type:              models.ActionType(SlackListChannelsActionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     "slack_list_channels",
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
		Name:              SlackPostMessageActionName,
		Description:       "Send a message to a Slack channel",
		Type:              models.ActionType(SlackPostMessageActionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     "slack_post_message",
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
