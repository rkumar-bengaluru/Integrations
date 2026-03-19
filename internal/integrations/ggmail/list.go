package ggmail

import (
	"context"
	"fmt"

	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"go.uber.org/zap"
)

// listMessages lists messages with optional query filters
func (h *GmailHandler) listMessages(
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

	// Optional filters
	query, _ := inputs["query"].(string)
	labelIDs, _ := inputs["label_ids"].([]interface{})
	maxResults, _ := inputs["max_results"].(int64)
	pageToken, _ := inputs["page_token"].(string)
	includeSpamTrash, _ := inputs["include_spam_trash"].(bool)

	if maxResults == 0 {
		maxResults = 50 // Default
	}

	// Build Gmail service
	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return nil, err
	}

	// Build request
	userID := h.getUserID(config)
	req := svc.Users.Messages.List(userID).
		MaxResults(maxResults).
		IncludeSpamTrash(includeSpamTrash)

	if query != "" {
		req = req.Q(query)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	for _, lid := range labelIDs {
		if id, ok := lid.(string); ok {
			req = req.LabelIds(id)
		}
	}

	// Execute request
	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Format results
	messages := make([]map[string]interface{}, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		msgData := map[string]interface{}{
			"id":        m.Id,
			"thread_id": m.ThreadId,
		}

		// Fetch snippet if available
		if m.ThreadId != "" {
			msg, err := svc.Users.Messages.Get(userID, m.Id).Format("metadata").Do()
			if err == nil && msg.Snippet != "" {
				msgData["snippet"] = msg.Snippet
				msgData["label_ids"] = msg.LabelIds
				msgData["history_id"] = msg.HistoryId
				msgData["internal_date"] = msg.InternalDate
			}
		}

		messages = append(messages, msgData)
	}

	result := map[string]interface{}{
		"messages":             messages,
		"next_page_token":      resp.NextPageToken,
		"result_size_estimate": resp.ResultSizeEstimate,
		"status_code":          200,
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
