package slack

import (
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

func (c *SlackHandler) ListUsers(
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
	var limitStr, cursor string
	if limit, ok := inputs["limit"].(int); ok && limit > 0 {
		limitStr = strconv.Itoa(limit)
	}
	if cur, ok := inputs["cursor"].(string); ok && cur != "" {
		cursor = cur
	}

	// Build query params
	params := url.Values{}
	if limitStr != "" {
		params.Set("limit", limitStr)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://slack.com/api/users.list?"+params.Encode(), nil)
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

	c.logger.Debug("Raw Response", zap.Any("response", slackResp))

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

func AddListUsersAction(ctx context.Context,
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
		Name:              SlackListUsersActionName,
		Description:       "Retrieve a list of users in the Slack workspace",
		Type:              models.ActionType(SlackListUsersActionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     "slack_list_users",
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
