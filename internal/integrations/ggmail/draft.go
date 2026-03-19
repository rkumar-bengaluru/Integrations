package ggmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"go.uber.org/zap"
	"google.golang.org/api/gmail/v1"
)

// createDraft creates a draft message
func (h *GmailHandler) createDraft(
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

	to := inputs["to"].(string)
	subject := inputs["subject"].(string)
	body := inputs["body"].(string)
	isHTML, _ := inputs["is_html"].(bool)

	// Build raw message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}
	msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n\r\n", contentType))
	msg.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	// Create draft
	draft := &gmail.Draft{
		Message: &gmail.Message{
			Raw: raw,
		},
	}

	// Build Gmail service
	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return nil, err
	}

	userID := h.getUserID(config)
	createdDraft, err := svc.Users.Drafts.Create(userID, draft).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	result := map[string]interface{}{
		"draft_id":    createdDraft.Id,
		"message_id":  createdDraft.Message.Id,
		"thread_id":   createdDraft.Message.ThreadId,
		"status_code": 201,
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
		StatusCode: 201,
	}, nil
}
