package slack

import (
	"context"

	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/models"
	"go.uber.org/zap"
)

func (c *SlackHandler) CreateChannel(ctx context.Context,
	sharepointClient *SlackRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	inputs map[string]interface{}) (*handler.ActionResult, error) {

	c.logger.Debug("finally action time for slack", zap.String("action", string(actionDef.Type)))
	return nil, nil
}
