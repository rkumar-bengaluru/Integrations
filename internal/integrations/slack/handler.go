package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/service"
	"agent.fabric.com/modules/internal/utils"
	"go.uber.org/zap"
)

// GitHubHandler implements all GitHub actions using Integration config
type SlackHandler struct {
	encryptionSvc encryption.EncryptionService
	bindingSvc    service.IntegrationBindingService
	logger        *zap.Logger
}

// Ensure GitHubHandler implements IntegrationHandler
var _ handler.IntegrationHandler = (*SlackHandler)(nil)

func NewSlackHandler(
	encryptionSvc encryption.EncryptionService,
	bindingSvc service.IntegrationBindingService, logger *zap.Logger) *SlackHandler {
	return &SlackHandler{
		encryptionSvc: encryptionSvc,
		bindingSvc:    bindingSvc,
		logger:        logger,
	}
}

// Init generates the initial installation token for GitHub App authentication
// It takes the collected parameters (private_key, app_id, installation_id, owner, repo)
// and returns the complete credential data including the generated token
func (h *SlackHandler) Init(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {
	// Extract required parameters
	utils.PrintMap(collectedParams)

	if config.CredentialBinding.CredentialType == models.AuthOAuth2 {
		h.logger.Debug("calling authorization flow...")
		return h.AuthorizationFlow(ctx, config, collectedParams)
	} else if config.CredentialBinding.CredentialType == models.AuthAPIKey {
		h.logger.Debug("calling api key flow")
		return h.BotFlow(ctx, config, collectedParams)
	}

	return nil, fmt.Errorf(fmt.Sprintf("Unknown credential type %s for slack integration", config.CredentialBinding.CredentialType))
}

// Execute an action with runtime-resolved config
func (h *SlackHandler) Execute(
	ctx context.Context,
	config *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding, inputs map[string]interface{}) (*handler.ActionResult, error) {

	runtimeConfig, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	switch actionDef.Type {
	case "create_channel":
		return h.CreateChannel(ctx, runtimeConfig, actionDef, binding, inputs)
	default:
		return nil, fmt.Errorf("unsupported action: %s", actionDef.Type)
	}
}

// Test if binding configuration is valid
func (h *SlackHandler) TestConnection(
	ctx context.Context,
	config *models.ExecutionConfig,
	binding models.IntegrationBinding,
) error {

	secrets, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return fmt.Errorf("failed to decrypt credential: %w", err)
	}

	switch config.CredentialBinding.CredentialType {
	case models.AuthOAuth2:
		h.logger.Debug("Testing connection for OAuth2 flow",
			zap.String("credential_type", string(config.CredentialBinding.CredentialType)))

		// Validate required fields
		if secrets.AccessToken == nil || *secrets.AccessToken == "" {
			return fmt.Errorf("missing AccessToken for OAuth2 flow")
		}
		if secrets.UserID == nil || *secrets.UserID == "" {
			return fmt.Errorf("missing UserID for OAuth2 flow")
		}

		// Call Slack API to verify token
		if err := h.testSlackAuth(ctx, *secrets.AccessToken); err != nil {
			return fmt.Errorf("OAuth2 token validation failed: %w", err)
		}

	case models.AuthAPIKey:
		h.logger.Debug("Testing connection for Bot flow",
			zap.String("credential_type", string(config.CredentialBinding.CredentialType)))

		// Validate required fields
		if secrets.AccessToken == nil || *secrets.AccessToken == "" {
			return fmt.Errorf("missing AccessToken for bot flow")
		}
		if secrets.RefreshToken == nil || *secrets.RefreshToken == "" {
			return fmt.Errorf("missing RefreshToken for bot flow")
		}
		if secrets.ExpiresIn == nil || *secrets.ExpiresIn == "" {
			return fmt.Errorf("missing ExpiresIn for bot flow")
		}

		// Call Slack API to verify bot token
		if err := h.testSlackAuth(ctx, *secrets.AccessToken); err != nil {
			return fmt.Errorf("Bot token validation failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported credential type: %v",
			config.CredentialBinding.CredentialType)
	}

	return nil
}

// Helper to call Slack auth.test
func (h *SlackHandler) testSlackAuth(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("slack auth.test failed: %s", result.Error)
	}
	return nil
}
