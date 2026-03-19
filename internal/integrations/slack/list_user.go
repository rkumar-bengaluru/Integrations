package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rkumar-bengaluru/Integrations/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"go.uber.org/zap"
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
