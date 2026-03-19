package snow

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/service"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/utils"
	"go.uber.org/zap"
)

// GitHubHandler implements all GitHub actions using Integration config
type SnowHandler struct {
	encryptionSvc encryption.EncryptionService
	bindingSvc    service.IntegrationBindingService
	logger        *zap.Logger
}

// Ensure GitHubHandler implements IntegrationHandler
var _ handler.IntegrationHandler = (*SnowHandler)(nil)

func NewSnowHandler(
	encryptionSvc encryption.EncryptionService,
	bindingSvc service.IntegrationBindingService, logger *zap.Logger) *SnowHandler {
	return &SnowHandler{
		encryptionSvc: encryptionSvc,
		bindingSvc:    bindingSvc,
		logger:        logger,
	}
}

// Init generates the initial installation token for GitHub App authentication
// It takes the collected parameters (private_key, app_id, installation_id, owner, repo)
// and returns the complete credential data including the generated token
func (h *SnowHandler) Init(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {
	// Extract required parameters
	utils.PrintMap(collectedParams)

	if config.CredentialBinding.CredentialType == models.AuthOAuth2 &&
		models.CredentialType(*config.CredentialBinding.GrantType) == models.CredentialType(models.GrantTypeAuthClientCredential) {
		h.logger.Debug("calling client credential flow...")
		return h.ClientCredentialFlow(ctx, config, collectedParams)
	}

	return nil, fmt.Errorf(fmt.Sprintf("Unknown credential type %s for SnowHandler integration", config.CredentialBinding.CredentialType))
}

// Execute an action with runtime-resolved config
func (h *SnowHandler) Execute(
	ctx context.Context,
	config *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding, inputs map[string]interface{}) (*handler.ActionResult, error) {

	runtimeConfig, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	switch string(actionDef.Type) {
	case SnowTestActionType:
		return h.TestAction(ctx, runtimeConfig, actionDef, binding, inputs)
	case SnowCreateIncidentActionType:
		return h.CreateIncident(ctx, runtimeConfig, actionDef, binding, config, inputs)
	case SnowGetIncidentActionType:
		return h.GetIncidentByNumber(ctx, runtimeConfig, actionDef, binding, config, inputs)
	case SnowSearchIncidentActionType:
		return h.SearchIncidents(ctx, runtimeConfig, actionDef, binding, config, inputs)
	case SnowUpdateIncidentActionType:
		return h.UpdateIncident(ctx, runtimeConfig, actionDef, binding, config, inputs)
	default:
		return nil, fmt.Errorf("unsupported action: %s", actionDef.Type)
	}
}

// Test if binding configuration is valid
func (h *SnowHandler) TestConnection(
	ctx context.Context,
	config *models.ExecutionConfig,
	binding models.IntegrationBinding,
) error {

	secrets, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return fmt.Errorf("Error %w", err)
	}

	switch config.CredentialBinding.CredentialType {
	case models.AuthOAuth2:
		h.logger.Debug("Testing connection for OAuth2 flow",
			zap.String("credential_type", string(config.CredentialBinding.CredentialType)))

		// Validate required fields
		if secrets.AccessToken == nil || *secrets.AccessToken == "" {
			return fmt.Errorf("missing AccessToken for OAuth2 flow")
		}

		// Call Slack API to verify token
		if err := h.testSnowAuth(ctx, *secrets.AccessToken, *config.CredentialBinding.AuthorityUrl); err != nil {
			return fmt.Errorf("OAuth2 token validation failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported credential type: %v",
			config.CredentialBinding.CredentialType)
	}

	return nil
}

func (h *SnowHandler) testSnowAuth(ctx context.Context, token string, authorityURL string) error {
	// Call a lightweight API to verify token and roles

	instanceURL := strings.Split(authorityURL, "/oauth_token.do")[0]
	// instanceURL = "https://dev212340.service-now.com"

	req, err := http.NewRequestWithContext(ctx, "GET", instanceURL+"/api/now/table/incident?sysparm_limit=1", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ServiceNow auth test failed: %s", string(body))
	}

	return nil
}
