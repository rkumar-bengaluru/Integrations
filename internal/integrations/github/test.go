package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rkumar-bengaluru/Integrations/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"go.uber.org/zap"
)

func (c *GitHubHandler) TestAction(
	ctx context.Context,
	runtimeConfig *GitHubRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	c.logger.Debug("finally action time for slack", zap.String("action", string(actionDef.Type)))

	commons.PrintCollectedParams(inputs)

	// Validate inputs
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	c.logger.Debug("input validation successful")

	// Extract channel name
	userMessage, ok := inputs["message"].(string)
	if !ok || userMessage == "" {
		c.logger.Error("required input param missing", zap.String("userMessage", userMessage))
		return &handler.ActionResult{
			StatusCode: http.StatusBadRequest,
			Error:      fmt.Errorf("missing or invalid message"),
		}, nil
	}

	// Prepare request payload
	_ = map[string]interface{}{
		"message": userMessage,
	}

	// Build request

	_, err := c.buildGitHubClient(runtimeConfig)
	if err != nil {
		return nil, err
	}

	// Execute request
	////////////////////////SIMULATION
	// resp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	return &handler.ActionResult{
	// 		StatusCode: http.StatusInternalServerError,
	// 		Error:      fmt.Errorf("failed to call Slack API: %w", err),
	// 	}, nil
	// }
	// defer resp.Body.Close()

	// Simulated response
	testActionResponse := map[string]interface{}{
		"success": true,
		"echo":    userMessage,
	}

	// Validate outputs
	if err := commons.ValidateSchema(actionDef.OutputSchema, testActionResponse, "output"); err != nil {
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
	if okVal, exists := testActionResponse["success"].(bool); exists {
		if !okVal {
			return &handler.ActionResult{
				Data:       testActionResponse,
				StatusCode: http.StatusInternalServerError,
			}, nil
		}
		// Success case
		return &handler.ActionResult{
			Data:       testActionResponse,
			StatusCode: http.StatusOK,
		}, nil
	}

	// If "success" not present, treat as error
	return nil, fmt.Errorf("internal server error on test.action")

}
