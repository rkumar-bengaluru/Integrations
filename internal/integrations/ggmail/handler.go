package ggmail

import (
	"context"
	"fmt"

	"github.com/rkumar-bengaluru/Integrations/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"github.com/rkumar-bengaluru/Integrations/internal/service"
	"go.uber.org/zap"
)

// GitHubHandler implements all GitHub actions using Integration config
type GmailHandler struct {
	encryptionSvc encryption.EncryptionService
	bindingSvc    service.IntegrationBindingService
	logger        *zap.Logger
}

// Ensure GitHubHandler implements IntegrationHandler
var _ handler.IntegrationHandler = (*GmailHandler)(nil)

func NewGmailHandler(encryptionSvc encryption.EncryptionService, bindingSvc service.IntegrationBindingService, logger *zap.Logger) *GmailHandler {
	return &GmailHandler{
		encryptionSvc: encryptionSvc,
		bindingSvc:    bindingSvc,
		logger:        logger,
	}
}

// Init generates the initial installation token for GitHub App authentication
// It takes the collected parameters (private_key, app_id, installation_id, owner, repo)
// and returns the complete credential data including the generated token
func (h *GmailHandler) Init(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {
	return h.oauth2AuthorizationCodeflow(ctx, config, collectedParams)
}

// Execute an action with runtime-resolved config
func (h *GmailHandler) Execute(
	ctx context.Context,
	config *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding, inputs map[string]interface{}) (*handler.ActionResult, error) {

	runtimeConfig, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	switch string(actionDef.Type) {
	case GmailTestActionType:
		return h.TestAction(ctx, runtimeConfig, actionDef, binding, inputs)
	case GmailSendEmailActionType:
		return h.sendEmail(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GmailListEmailActionType:
		return h.listMessages(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GmailDraftEmailActionType:
		return h.createDraft(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GmailDeleteEmailActionType:
		return h.deleteMessage(ctx, config, actionDef, runtimeConfig, binding, inputs)
	default:
		return nil, fmt.Errorf("unsupported action: %s", actionDef.Type)
	}
}

// Test if binding configuration is valid
func (h *GmailHandler) TestConnection(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	binding models.IntegrationBinding,
) error {
	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return err
	}

	svc, err := h.buildGmailService(ctx, config)
	if err != nil {
		return err
	}

	userID := h.getUserID(config)
	profile, err := svc.Users.GetProfile(userID).Do()
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	h.logger.Info("Gmail connection test successful",
		zap.String("email", profile.EmailAddress),
		zap.Int64("messages_total", profile.MessagesTotal),
	)

	return nil
}
