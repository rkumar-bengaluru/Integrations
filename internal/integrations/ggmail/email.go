package ggmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"go.uber.org/zap"
	"google.golang.org/api/gmail/v1"
)

// sendEmail sends an email with optional attachments
func (h *GmailHandler) sendEmail(
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

	// Required inputs
	to := inputs["to"].(string)
	subject := inputs["subject"].(string)
	body := inputs["body"].(string)

	// Optional inputs
	cc, _ := inputs["cc"].(string)
	bcc, _ := inputs["bcc"].(string)
	isHTML, _ := inputs["is_html"].(bool)
	attachments, _ := inputs["attachments"].([]interface{})

	// Build Gmail service
	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return nil, err
	}

	// Construct raw email message
	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	if cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		msg.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}

	// Content-Type
	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	// Handle attachments
	if len(attachments) > 0 {
		boundary := uuid.New().String()
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n\r\n", contentType))
		msg.WriteString(body)
		msg.WriteString("\r\n")

		// Process attachments
		for _, att := range attachments {
			attMap, ok := att.(map[string]interface{})
			if !ok {
				continue
			}

			filename, _ := attMap["filename"].(string)
			contentTypeAtt, _ := attMap["content_type"].(string)
			content, err := extractFileReader(attMap["content"])
			if err != nil {
				h.logger.Warn("Failed to extract attachment", zap.Error(err))
				continue
			}

			data, err := io.ReadAll(content)
			if err != nil {
				return nil, fmt.Errorf("failed to read attachment: %w", err)
			}

			msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			msg.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentTypeAtt))
			msg.WriteString("Content-Transfer-Encoding: base64\r\n")
			msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename))
			msg.WriteString(base64.StdEncoding.EncodeToString(data))
			msg.WriteString("\r\n")
		}

		msg.WriteString(fmt.Sprintf("--%s--", boundary))
	} else {
		msg.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n\r\n", contentType))
		msg.WriteString(body)
	}

	// Encode message
	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	// Create Gmail message
	gmailMsg := &gmail.Message{
		Raw: raw,
	}

	// Send message
	userID := h.getUserID(config)
	sentMsg, err := svc.Users.Messages.Send(userID, gmailMsg).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	result := map[string]interface{}{
		"message_id":  sentMsg.Id,
		"thread_id":   sentMsg.ThreadId,
		"label_ids":   sentMsg.LabelIds,
		"history_id":  sentMsg.HistoryId,
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

func extractFileReader(v interface{}) (io.ReadCloser, error) {
	switch f := v.(type) {

	case io.ReadCloser:
		return f, nil

	case io.Reader:
		// You can wrap plain io.Reader if needed, but usually better to reject
		return nil, fmt.Errorf("plain io.Reader not supported (must be io.ReadCloser)")

	case *multipart.FileHeader:
		file, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open uploaded file: %w", err)
		}
		return file, nil

	case string:
		file, err := os.Open(f)
		if err != nil {
			return nil, fmt.Errorf("failed to open local file %q: %w", f, err)
		}
		return file, nil

	default:
		return nil, fmt.Errorf("unsupported file_content type: %T", v)
	}
}

// getMessage retrieves a specific message by ID
func (h *GmailHandler) getMessage(
	ctx context.Context,
	actionDef models.ActionDefinition,
	config *GmailRuntimeConfig,
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
	format, _ := inputs["format"].(string)
	if format == "" {
		format = "full"
	}

	// Build Gmail service
	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return nil, err
	}

	userID := h.getUserID(config)
	msg, err := svc.Users.Messages.Get(userID, messageID).Format(format).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Parse headers
	headers := make(map[string]string)
	for _, h := range msg.Payload.Headers {
		headers[h.Name] = h.Value
	}

	// Extract body
	body := h.extractBody(msg.Payload)

	result := map[string]interface{}{
		"id":            msg.Id,
		"thread_id":     msg.ThreadId,
		"label_ids":     msg.LabelIds,
		"snippet":       msg.Snippet,
		"history_id":    msg.HistoryId,
		"internal_date": msg.InternalDate,
		"size_estimate": msg.SizeEstimate,
		"headers":       headers,
		"body":          body,
		"status_code":   200,
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

// extractBody recursively extracts text body from message parts
func (h *GmailHandler) extractBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// Check if this part is text/plain or text/html
	if part.MimeType == "text/plain" || part.MimeType == "text/html" {
		if part.Body != nil && part.Body.Data != "" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				// Try URL-safe variant
				data, _ = base64.RawURLEncoding.DecodeString(part.Body.Data)
			}
			return string(data)
		}
	}

	// Recurse into parts
	for _, p := range part.Parts {
		body := h.extractBody(p)
		if body != "" {
			return body
		}
	}

	return ""
}
