package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v84/github"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/utils"
	"go.uber.org/zap"
)

func (h *GitHubHandler) oauthServer2Server(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {

	// Extract required parameters
	utils.PrintMap(collectedParams)

	privateKey, ok := collectedParams["private_key"].(string)
	if !ok || privateKey == "" {
		return nil, fmt.Errorf("private_key is required")
	}

	appID, ok := collectedParams["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("app_id is required")
	}

	installationID, ok := collectedParams["installation_id"].(string)
	if !ok || installationID == "" {
		return nil, fmt.Errorf("installation_id is required")
	}

	owner, ok := collectedParams["owner"].(string)
	if !ok || owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	repo, ok := collectedParams["repo"].(string)
	if !ok || repo == "" {
		return nil, fmt.Errorf("repo is required")
	}

	// Generate JWT
	jwtToken, err := h.generateJWT(privateKey, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Exchange JWT for installation token
	installToken, expiresAt, err := h.exchangeJWTForToken(ctx, jwtToken, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange JWT for installation token: %w", err)
	}

	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"private_key_key":     privateKey,
		"app_id_key":          appID,
		"installation_id_key": installationID,
		"owner_key":           owner,
		"repo_key":            repo,

		// Generated values (runtime tokens)
		"token":            installToken,
		"token_expires_at": expiresAt.Format(time.RFC3339),
		"token_type":       "installation",
		"generated_at":     time.Now().Format(time.RFC3339),
	}

	h.logger.Info("GitHub App initialization successful",
		zap.String("app_id", appID),
		zap.String("installation_id", installationID),
		zap.Time("token_expires_at", expiresAt),
	)

	return credentialData, nil
}

func (h *GitHubHandler) generateJWT(privateKeyPEM, appID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": appID,
	}

	h.logger.Debug("generating jwttoken for appid", zap.String("appid", appID))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		h.logger.Error("failed to parse key", zap.Error(err))
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		h.logger.Error("failed to parse key", zap.Error(err))
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (h *GitHubHandler) exchangeJWTForToken(ctx context.Context, jwtToken, installationID string) (string, time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	h.logger.Debug("exchange token status code ", zap.Int("status code", resp.StatusCode))
	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, err
	}

	expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse expiry: %w", err)
	}

	return result.Token, expiresAt, nil
}

func (h *GitHubHandler) resolveRuntimeConfig(ctx context.Context,
	econfig *models.ExecutionConfig,
	binding models.IntegrationBinding) (*GitHubRuntimeConfig, error) {
	if binding.Credential == nil || len(binding.Credential.EncryptedData) == 0 {
		return nil, fmt.Errorf("credential is not set for integration binding")
	}

	// Decrypt the credential data
	decryptedData, err := h.encryptionSvc.Decrypt(binding.Credential.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	// Parse the secrets using the storage keys from SecretMapping
	var secrets map[string]interface{}
	if err := json.Unmarshal(decryptedData, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential secrets: %w", err)
	}

	// Extract values using storage keys that match our SecretMapping
	config := &GitHubRuntimeConfig{
		BaseURL:    "https://api.github.com",
		APIVersion: "2022-11-28",
		Credential: binding.Credential,
	}

	// Extract stored values
	if v, ok := secrets["token"].(string); ok {
		config.Token = v
	}
	if v, ok := secrets["token_expires_at"].(string); ok {
		config.TokenExpiresAt, _ = time.Parse(time.RFC3339, v)
		h.logger.Debug("token expiresAt %s", zap.String("", config.TokenExpiresAt.String()))
	}

	// Map storage keys to config fields
	if v, ok := secrets["private_key_key"].(string); ok {
		config.PrivateKey = v
	}
	if v, ok := secrets["app_id_key"].(string); ok {
		config.AppID = v
	}
	if v, ok := secrets["installation_id_key"].(string); ok {
		config.InstallationID = v
	}
	if v, ok := secrets["owner_key"].(string); ok {
		config.Owner = v
	}
	if v, ok := secrets["repo_key"].(string); ok {
		config.Repo = v
	}

	// Validate required fields for GitHub App authentication
	if config.PrivateKey == "" || config.AppID == "" || config.InstallationID == "" {
		return nil, fmt.Errorf("missing required credentials: private_key, app_id, and installation_id are required")
	}

	// Check if token needs refresh
	if time.Until(config.TokenExpiresAt) < 5*time.Minute {
		h.logger.Info("Token expiring soon, refreshing...")
		if err := h.refreshToken(ctx, config, binding); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	} else {
		h.logger.Info("Token still valid...")
	}

	return config, nil
}

func (h *GitHubHandler) buildGitHubClient(config *GitHubRuntimeConfig) (*github.Client, error) {
	// For GitHub App authentication, we need to create a JWT token
	// Then exchange it for an installation access token

	// This is a simplified version - in production you'd use:
	// 1. Create JWT from private key + app_id
	// 2. Use JWT to get installation access token
	// 3. Use installation token for API calls

	// For now, we'll use a placeholder transport that will be implemented
	// with proper GitHub App authentication logic
	httpClient := &http.Client{
		Transport: &githubTokenTransport{
			token:         config.Token,
			apiVersion:    config.APIVersion,
			baseTransport: http.DefaultTransport,
		},
	}

	client := github.NewClient(httpClient)

	// Configure base URL for GitHub Enterprise Server
	if config.BaseURL != "https://api.github.com" && config.BaseURL != "" {
		baseURL, err := url.Parse(config.BaseURL + "/")
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		client.BaseURL = baseURL
	}

	return client, nil
}

func (t *githubTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", t.apiVersion)
	return t.baseTransport.RoundTrip(req)
}

func (h *GitHubHandler) refreshToken(ctx context.Context, config *GitHubRuntimeConfig, binding models.IntegrationBinding) error {
	h.logger.Debug("refreshing github token...")
	jwtToken, err := h.generateJWT(config.PrivateKey, config.AppID)
	if err != nil {
		return err
	}
	h.logger.Debug("got jwt token...")
	newToken, expiresAt, err := h.exchangeJWTForToken(ctx, jwtToken, config.InstallationID)
	if err != nil {
		return err
	}
	h.logger.Debug("new token expiry...", zap.String("", expiresAt.String()))
	// Update config
	config.Token = newToken
	config.TokenExpiresAt = expiresAt

	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"private_key_key":     config.PrivateKey,
		"app_id_key":          config.AppID,
		"installation_id_key": config.InstallationID,
		"owner_key":           config.Owner,
		"repo_key":            config.Repo,

		// Generated values (runtime tokens)
		"token":            newToken,
		"token_expires_at": expiresAt.Format(time.RFC3339),
		"token_type":       "installation",
		"generated_at":     time.Now().Format(time.RFC3339),
	}

	// Marshal secrets to JSON
	secretsJSON, err := json.Marshal(credentialData)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Encrypt the secrets data
	encryptedData, err := h.encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Generate hash for integrity verification
	dataHash := h.encryptionSvc.GenerateDataHash(encryptedData)

	binding.Credential.DataHash = dataHash
	binding.Credential.EncryptedData = encryptedData

	// Credential idx
	err = h.bindingSvc.UpdateCredential(ctx, binding.Credential)
	// Note: In a real implementation, you'd also want to update the stored credential
	if err != nil {
		return fmt.Errorf("failed to update new credential with updated token...")
	}
	// This could be done via a callback or by returning the new token to the caller
	h.logger.Info("Token refreshed and serialized successfully", zap.Time("new_expiry", expiresAt))
	return nil
}
