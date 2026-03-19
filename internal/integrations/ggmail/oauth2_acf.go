package ggmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func (h *GmailHandler) oauth2AuthorizationCodeflow(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {

	// Extract required parameters
	utils.PrintMap(collectedParams)

	client_id, ok := collectedParams["client_id"].(string)
	if !ok || client_id == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	client_secret, ok := collectedParams["client_secret"].(string)
	if !ok || client_secret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}

	redirect_uri, ok := collectedParams["redirect_uri"].(string)
	if !ok || redirect_uri == "" {
		return nil, fmt.Errorf("redirect_uri is required")
	}

	scopes, err := GetStringSlice(collectedParams, "scopes")
	if err != nil {
		return nil, fmt.Errorf("invalid scopes: %w", err)
	}

	// now scopes is []string and safe to use

	// Step 1: Generate OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     client_id,
		ClientSecret: client_secret,
		RedirectURL:  redirect_uri,
		Scopes:       cleanScopes(scopes),
		Endpoint:     google.Endpoint,
	}

	endpoint := oauth2.Endpoint{
		AuthURL:  google.Endpoint.AuthURL,
		TokenURL: google.Endpoint.TokenURL,
	}
	oauth2Config.Endpoint = endpoint

	// Step 2: Generate authorization URL with offline access
	authURL := oauth2Config.AuthCodeURL(
		generateState(),
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	h.logger.Info("Opening browser for authorization...")
	h.logger.Info("If browser doesn't open, visit this URL:")
	h.logger.Info(authURL)

	// Open browser automatically
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Println("Please open the URL manually.")
	}

	// Step 3: Start local HTTP server to receive callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		handleCallback(w, r, codeChan, errChan)
	})

	go func() {
		h.logger.Info("Waiting for OAuth2 callback on http://localhost:8080/callback ...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for authorization code or error
	var code string
	select {
	case code = <-codeChan:
		h.logger.Info("Authorization code received!")
	case err := <-errChan:
		h.logger.Error("Error: ", zap.Error(err))
		return nil, err
	case <-time.After(5 * time.Minute):
		h.logger.Info("Timeout waiting for authorization")
		return nil, fmt.Errorf("Timeout waiting for authorization")
	}

	// Shutdown server
	defer server.Shutdown(ctx)

	// Step 4: Exchange code for tokens
	h.logger.Info("Exchanging code for tokens...", zap.String("code", code))
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("Error: ", zap.Error(err))
		return nil, err
	}

	h.logger.Info("✓ Token obtained successfully!")

	h.logger.Info("  Expiry", zap.String("", token.Expiry.Format(time.RFC3339)))

	// Step 5: Fetch user email
	emailID := fetchUserEmail(token.AccessToken)
	if emailID != "" {
		h.logger.Info("✓ Fetched user email: ", zap.String("", emailID))
	}
	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"client_id":     client_id,
		"client_secret": client_secret,
		"redirect_uri":  redirect_uri,
		"scopes":        scopes,

		// Generated values (runtime tokens)
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"email_id":      emailID,
		"token_expiry":  token.Expiry.Format(time.RFC3339),
	}

	h.logger.Info("Gmail App initialization successful",
		zap.String("email_id", emailID),
		zap.String("token_expires_at", token.Expiry.Format(time.RFC3339)),
	)

	return credentialData, nil
}

// GetStringSlice safely extracts a []string from map[string]interface{}
// Returns error if key missing, wrong type, empty, or contains non-strings
func GetStringSlice(m map[string]interface{}, key string) ([]string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("%s is required", key)
	}

	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("%s cannot be empty", key)
	}

	result := make([]string, 0, len(items))
	for i, v := range items {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string, got %T", key, i, v)
		}
		result = append(result, s)
	}

	return result, nil
}

func cleanScopes(scopes []string) []string {
	cleaned := make([]string, len(scopes))
	for i, s := range scopes {
		cleaned[i] = cleanURL(s)
	}
	return cleaned
}

func cleanURL(s string) string {
	// Remove trailing spaces and normalize
	s = strings.TrimSpace(s)
	return s
}

func generateState() string {
	return uuid.New().String()
}

func handleCallback(w http.ResponseWriter, r *http.Request, codeChan chan<- string, errChan chan<- error) {
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errChan <- fmt.Errorf("OAuth error: %s", errorParam)
		http.Error(w, "Authorization failed: "+errorParam, http.StatusBadRequest)
		return
	}

	if code == "" {
		errChan <- fmt.Errorf("no code in callback")
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<html>
		<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
			<h1 style="color: #4CAF50;">✓ Authorization Successful</h1>
			<p>You can close this window and return to the terminal.</p>
		</body>
		</html>
	`))

	codeChan <- code
}

func fetchUserEmail(accessToken string) string {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return ""
	}

	return userInfo.Email
}

func (h *GmailHandler) resolveRuntimeConfig(ctx context.Context,
	econfig *models.ExecutionConfig,
	binding models.IntegrationBinding) (*GmailRuntimeConfig, error) {
	if binding.Credential != nil && len(binding.Credential.EncryptedData) > 0 {
		if binding.Credential == nil || len(binding.Credential.EncryptedData) == 0 {
			return nil, fmt.Errorf("credential is not set for integration binding")
		}

		// Decrypt the credential data
		decryptedData, err := h.encryptionSvc.Decrypt(binding.Credential.EncryptedData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credential: %w", err)
		}

		var secrets GmailRuntimeConfig
		if err := json.Unmarshal(decryptedData, &secrets); err != nil {
			return nil, fmt.Errorf("failed to unmarshal credential secrets: %w", err)
		}

		// Check if token needs refresh
		if time.Until(*secrets.TokenExpiry) < 5*time.Minute {
			h.logger.Info("Token expiring soon, refreshing...")
			if err := h.refreshToken(ctx, &secrets, binding); err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}
		} else {
			h.logger.Info("Token still valid...")
		}

		h.logger.Debug("client id from platform config", zap.String("client id", secrets.ClientID))
		return &secrets, nil
	}

	return nil, fmt.Errorf("credential is not set for integration %s", binding.Integration.Name)
}

func (h *GmailHandler) refreshToken(ctx context.Context, config *GmailRuntimeConfig, binding models.IntegrationBinding) error {
	h.logger.Debug("refreshing gmail token...")

	// Build the same oauth2.Config you used during initial authorization
	oauthCfg := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		// RedirectURL:  config.RedirectURI,
		Scopes:   config.Scopes,
		Endpoint: google.Endpoint,
	}

	// Load the previously stored token (from DB / your credential storage)
	storedToken := &oauth2.Token{
		AccessToken:  *config.AccessToken,  // from storage
		RefreshToken: *config.RefreshToken, // critical — must exist
		TokenType:    "Bearer",
		Expiry:       *config.TokenExpiry, // time.Time from storage
	}

	h.logger.Debug("Loaded stored Gmail token",
		zap.Time("stored_expiry", storedToken.Expiry),
		zap.Bool("has_refresh", storedToken.RefreshToken != ""),
	)

	if !storedToken.Valid() && storedToken.RefreshToken == "" {
		return fmt.Errorf("no refresh token available — re-authorization required")
	}

	// Create a TokenSource that auto-refreshes using refresh_token
	tokenSource := oauthCfg.TokenSource(ctx, storedToken)

	// Get a valid token (will refresh automatically if expired or near expiry)
	freshToken, err := tokenSource.Token()
	if err != nil {
		h.logger.Error("Failed to refresh Gmail token", zap.Error(err))
		return fmt.Errorf("token refresh failed: %w", err)
	}

	expiresAt := freshToken.Expiry
	if expiresAt.IsZero() {
		// Some refresh responses don't return new expiry → estimate conservatively
		expiresAt = time.Now().Add(50 * time.Minute)
	}

	h.logger.Debug("Gmail token refreshed/valid",
		zap.String("access_token_prefix", freshToken.AccessToken[:10]+"..."),
		zap.Time("new_expiry", expiresAt),
	)

	// Build the complete credential data
	// This includes both user-provided values AND generated tokens
	credentialData := map[string]interface{}{
		// User-provided values (stored with _key suffix as per SecretMapping)
		"client_id":     config.ClientID,
		"client_secret": config.ClientSecret,
		"redirect_uri":  "",
		"scopes":        config.Scopes,

		// Generated values (runtime tokens)
		"access_token":  freshToken.AccessToken,
		"refresh_token": freshToken.RefreshToken,
		"email_id":      config.UserEmail,
		"token_expiry":  freshToken.Expiry.Format(time.RFC3339),
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

func (h *GmailHandler) buildGmailService(ctx context.Context, config *GmailRuntimeConfig) (*gmail.Service, error) {
	// Create OAuth2 config
	h.logger.Debug("cllent id ", zap.String("client_id", config.ClientID))
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			gmail.GmailSendScope,
			gmail.GmailReadonlyScope,
			gmail.GmailModifyScope,
			gmail.GmailComposeScope,
		},
	}

	// Create token from stored credentials
	token := &oauth2.Token{
		AccessToken:  *config.AccessToken,
		RefreshToken: *config.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       *config.TokenExpiry,
	}

	// Create HTTP client with automatic token refresh
	httpClient := oauthConfig.Client(ctx, token)

	// Create Gmail service
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return svc, nil
}

func (h *GmailHandler) getUserID(config *GmailRuntimeConfig) string {
	if config.UserEmail != nil {
		return *config.UserEmail
	}
	return "me"
}
