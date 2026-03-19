package ggmail

import (
	"context"
	"fmt"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"go.uber.org/zap"
)

// deleteMessage moves a message to trash or permanently deletes it
func (h *GmailHandler) deleteMessage(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	config *GmailRuntimeConfig,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("finally action time for slack", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	messageID := inputs["message_id"].(string)
	permanent, _ := inputs["permanent"].(bool)

	// Build Gmail service
	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return nil, err
	}

	userID := h.getUserID(config)

	if permanent {
		err = svc.Users.Messages.Delete(userID, messageID).Do()
	} else {
		_, err = svc.Users.Messages.Trash(userID, messageID).Do()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}

	result := map[string]interface{}{
		"message_id":  messageID,
		"permanent":   permanent,
		"status_code": 200,
	}

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, result, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	}
	h.logger.Debug("Output validation successful")
	return &handler.ActionResult{
		Data:       result,
		StatusCode: 200,
	}, nil
}
