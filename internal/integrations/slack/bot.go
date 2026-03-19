package slack

import (
	"context"
	"fmt"
	"os"

	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"go.uber.org/zap"
)

func (h *SlackHandler) BotFlow(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {

	client_id, ok := collectedParams["client_id"].(string)
	if !ok || client_id == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	h.logger.Debug("client id found....")

	client_secret, ok := collectedParams["client_secret"].(string)
	if !ok || client_secret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}

	h.logger.Debug("client_secret id found....")

	bot_scope, ok := collectedParams["bot_scopes"].([]interface{})
	if !ok {
		h.logger.Error("bot_scopes not found....")
		return nil, fmt.Errorf("bot_scopes is required")
	}

	h.logger.Debug("bot_scopes id found....")

	long_lived_token, ok := collectedParams["long_lived_token"].(string)
	if !ok {
		return nil, fmt.Errorf("long_lived_token is required")
	}

	h.logger.Info("Initializing for slack client credential flow ...",
		zap.String("credential type", string(config.CredentialBinding.CredentialType)))

	//  follow client credential flow for slack
	authorityUrl := "https://slack.com/api/oauth.v2.exchange"
	if config.CredentialBinding.AuthorityUrl != nil && *config.CredentialBinding.AuthorityUrl != "" {
		// Safe to use the value
		h.logger.Debug(fmt.Sprintf("authorityUrl %s", *config.CredentialBinding.AuthorityUrl))
		authorityUrl = *config.CredentialBinding.AuthorityUrl
	} else {
		h.logger.Debug(fmt.Sprintf("authorityUrl is not set or empty using default %s", authorityUrl))
		fmt.Println("authorityUrl is not set or empty")
	}

	exchangeResp, err := h.exchangeLongLivedToken(client_id, client_secret,
		long_lived_token, authorityUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exchange failed: %v\n", err)
		os.Exit(1)
	}
	h.printTokenInfo("After exchange", exchangeResp)
	accessToken := exchangeResp.AccessToken
	refreshToken := exchangeResp.RefreshToken
	expiresIn := h.ExpiryTimeRFC3339(exchangeResp.ExpiresIn)
	tokenType := exchangeResp.TokenType

	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"client_id_key":        client_id,
		"client_secret_key":    client_secret,
		"bot_scope_key":        bot_scope,
		"long_lived_token_key": long_lived_token,

		// Generated values (runtime tokens)
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    tokenType,
		"expires_in":    expiresIn,
	}

	h.logger.Info("Slack App initialization successful")
	return credentialData, nil

}
