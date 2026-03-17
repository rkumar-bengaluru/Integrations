package snow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"agent.fabric.com/modules/internal/models"
	"go.uber.org/zap"
)

func (h *SnowHandler) ClientCredentialFlow(
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

	grant_type, ok := collectedParams["grant_type"].(string)
	if !ok {
		h.logger.Error("grant_type not found....")
		return nil, fmt.Errorf("grant_type is required")
	}

	h.logger.Debug("grant_type id found....")

	h.logger.Info("Initializing for slack client credential flow ...",
		zap.String("credential type", string(config.CredentialBinding.CredentialType)))

	//  follow client credential flow for slack
	var authorityUrl string
	if config.CredentialBinding.AuthorityUrl != nil && *config.CredentialBinding.AuthorityUrl != "" {
		// Safe to use the value
		h.logger.Debug(fmt.Sprintf("authorityUrl %s", *config.CredentialBinding.AuthorityUrl))
		authorityUrl = *config.CredentialBinding.AuthorityUrl
	} else {
		h.logger.Debug(fmt.Sprintf("authorityUrl is not set or empty using default %s", authorityUrl))
		return nil, fmt.Errorf("authority Url not found for snow integration")
	}

	exchangeResp, err := h.exchangeToken(client_id, client_secret,
		grant_type, authorityUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exchange failed: %v\n", err)
		os.Exit(1)
	}
	h.printTokenInfo("After exchange", exchangeResp)
	access_token := exchangeResp.AccessToken
	expiresIn := h.ExpiryTimeRFC3339(exchangeResp.ExpiresIn)
	tokenType := exchangeResp.TokenType

	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"client_id_key":     client_id,
		"client_secret_key": client_secret,
		"grant_type":        grant_type,

		// Generated values (runtime tokens)
		"access_token": access_token,
		"scope":        exchangeResp.Scope,
		"token_type":   tokenType,
		"expires_in":   expiresIn,
	}

	h.logger.Info("Snow App initialization successful")
	return credentialData, nil

}

// exchangeLongLivedToken calls oauth.v2.exchange to convert legacy long-lived token
func (h *SnowHandler) exchangeToken(clientID, clientSecret, grantType, authorityUrl string) (*SnowTokenResponse, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {grantType},
	}

	h.logger.Debug(fmt.Sprintf("authorityUrl url for exchange %s", authorityUrl))

	return h.postSnowToken(authorityUrl, data)
}

func (h *SnowHandler) postSnowToken(endpoint string, form url.Values) (*SnowTokenResponse, error) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Debug: print raw response
	if strings.HasPrefix(strings.TrimSpace(string(body)), "<html") {
		return nil, fmt.Errorf("instance is hibernating or returned HTML instead of JSON")
	}

	var tr SnowTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &tr, nil
}

func (h *SnowHandler) printTokenInfo(label string, tr *SnowTokenResponse) {
	h.logger.Debug(fmt.Sprintf("\n%s:\n", label))
	h.logger.Debug(fmt.Sprintf("  access_token = %s\n", h.mask(tr.AccessToken)))
	h.logger.Debug(fmt.Sprintf("  expires_in   = %d s (≈ %.1f h)\n", tr.ExpiresIn, float64(tr.ExpiresIn)/3600))
	h.logger.Debug(fmt.Sprintf("  token_type   = %s\n", tr.TokenType))
	h.logger.Debug(fmt.Sprintf("  scope        = %s\n", tr.Scope))

}

func (h *SnowHandler) mask(s string) string {
	if len(s) < 12 {
		return "[redacted]"
	}
	return s[:6] + "..." + s[len(s)-6:]
}

// ExpiryTimeRFC3339 takes seconds to expire, adds to current time,
// and returns the result in RFC3339 format.
func (h *SnowHandler) ExpiryTimeRFC3339(seconds int) string {
	expiry := time.Now().Add(time.Duration(seconds) * time.Second)
	return expiry.Format(time.RFC3339)
}

func (h *SnowHandler) resolveRuntimeConfig(ctx context.Context,
	config *models.ExecutionConfig,
	binding models.IntegrationBinding) (*SnowRuntimeConfig, error) {
	if binding.Credential != nil && len(binding.Credential.EncryptedData) > 0 {
		secrets, err := h.decryptCredentialSecrets(binding.Credential)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credential: %w", err)
		}

		if config.CredentialBinding.CredentialType == models.AuthOAuth2 &&
			models.CredentialType(*config.CredentialBinding.GrantType) == models.CredentialType(models.GrantTypeAuthClientCredential) {
			h.logger.Debug("calling client credential flow...")
			if secrets.AccessToken == nil && *secrets.AccessToken != "" {
				return nil, fmt.Errorf("unexpected AccessToken is nil")
			}

			// Parse expiry if provided
			if secrets.ExpiresIn != nil && *secrets.ExpiresIn != "" {
				// Safe to use the value
				h.logger.Debug(fmt.Sprintf("secrets.ExpiresIn %s", *secrets.ExpiresIn))
				expired, _, err := h.IsExpired(*secrets.ExpiresIn)
				if err != nil {
					return nil, fmt.Errorf(fmt.Sprintf("Failed to check if token expired %s", *secrets.ExpiresIn))
				}

				if expired {
					resp, err := h.exchangeToken(secrets.ClientIDKey, secrets.ClientSecretKey,
						SNOW_CLIENT_CREDENTIALS,
						*config.CredentialBinding.AuthorityUrl)
					if err != nil {
						return nil, err
					}
					expIn := h.ExpiryTimeRFC3339(resp.ExpiresIn)
					secrets.AccessToken = &resp.AccessToken
					secrets.ExpiresIn = &expIn
					secrets.Scope = resp.Scope

					// Marshal secrets to JSON
					secretsJSON, err := json.Marshal(secrets)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal secrets: %w", err)
					}

					// Encrypt the secrets data
					encryptedData, err := h.encryptionSvc.Encrypt(secretsJSON)
					if err != nil {
						return nil, fmt.Errorf("failed to encrypt secrets: %w", err)
					}

					// Generate hash for integrity verification
					dataHash := h.encryptionSvc.GenerateDataHash(encryptedData)

					binding.Credential.DataHash = dataHash
					binding.Credential.EncryptedData = encryptedData
					// save it back to database for future use.
					h.bindingSvc.UpdateCredential(ctx, binding.Credential)
					h.logger.Debug("crdential saved to database...")
					return secrets, nil
				} else {
					h.logger.Debug("token is not expired...")
				}

			} else {
				return nil, fmt.Errorf("unexpected expiresIn is nil")
			}
		}
		return secrets, nil
	}

	return nil, fmt.Errorf("credential is not set for integration %s", binding.Integration.Name)
}

func (h *SnowHandler) decryptCredentialSecrets(credential *models.Credential) (*SnowRuntimeConfig, error) {
	if h == nil || h.encryptionSvc == nil {
		return nil, fmt.Errorf("SnowRuntimeConfig not initialized with encryption service")
	}
	if credential == nil {
		return nil, fmt.Errorf("credential is nil")
	}

	decrypted, err := h.encryptionSvc.Decrypt(credential.EncryptedData)
	if err != nil {
		return nil, err
	}

	var secrets SnowRuntimeConfig
	if err := json.Unmarshal(decrypted, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential secrets: %w", err)
	}
	return &secrets, nil
}

// IsExpired takes an RFC3339 expiry string and returns whether it's expired
// (including "expiring soon" within 5 minutes) and how much time is left.
func (h *SnowHandler) IsExpired(expiryRFC3339 string) (bool, time.Duration, error) {
	expiryTime, err := time.Parse(time.RFC3339, expiryRFC3339)
	if err != nil {
		return false, 0, err
	}

	remaining := time.Until(expiryTime)

	// Treat as expired if already past OR less than 5 minutes left
	if remaining <= 5*time.Minute {
		return true, remaining, nil
	}
	return false, remaining, nil
}
