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
	"go.uber.org/zap"
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
