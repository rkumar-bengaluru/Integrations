package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"agent.fabric.com/modules/internal/models"
)

// exchangeLongLivedToken calls oauth.v2.exchange to convert legacy long-lived token
func (h *SlackHandler) exchangeLongLivedToken(clientID, clientSecret, longLivedToken, tokenUrl string) (*SlackTokenResponse, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"token":         {longLivedToken},
	}

	h.logger.Debug(fmt.Sprintf("token url for exchange %s", tokenUrl))

	return h.postSlackToken(tokenUrl, data)
}

func (h *SlackHandler) postSlackToken(endpoint string, form url.Values) (*SlackTokenResponse, error) {
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

	var tr SlackTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if !tr.OK {
		return &tr, fmt.Errorf("slack error: %s", tr.Error)
	}

	return &tr, nil
}

func (h *SlackHandler) printTokenInfo(label string, tr *SlackTokenResponse) {
	h.logger.Debug(fmt.Sprintf("\n%s:\n", label))
	h.logger.Debug(fmt.Sprintf("  ok           = %v\n", tr.OK))
	h.logger.Debug(fmt.Sprintf("  access_token = %s\n", h.mask(tr.AccessToken)))
	h.logger.Debug(fmt.Sprintf("  expires_in   = %d s (≈ %.1f h)\n", tr.ExpiresIn, float64(tr.ExpiresIn)/3600))
	h.logger.Debug(fmt.Sprintf("  refresh_token= %s\n", h.mask(tr.RefreshToken)))
	h.logger.Debug(fmt.Sprintf("  token_type   = %s\n", tr.TokenType))
	h.logger.Debug(fmt.Sprintf("  scope        = %s\n", tr.Scope))
	h.logger.Debug(fmt.Sprintf("  team         = %s (%s)\n", tr.Team.Name, tr.Team.ID))
	if tr.BotUserID != "" {
		h.logger.Debug(fmt.Sprintf("  bot_user_id  = %s\n", tr.BotUserID))
	}
	if tr.AuthedUser.ID != "" {
		h.logger.Debug(fmt.Sprintf("  authed_user.id = %s\n", tr.AuthedUser.ID))
	}
}

func (h *SlackHandler) mask(s string) string {
	if len(s) < 12 {
		return "[redacted]"
	}
	return s[:6] + "..." + s[len(s)-6:]
}

// ExpiryTimeRFC3339 takes seconds to expire, adds to current time,
// and returns the result in RFC3339 format.
func (h *SlackHandler) ExpiryTimeRFC3339(seconds int) string {
	expiry := time.Now().Add(time.Duration(seconds) * time.Second)
	return expiry.Format(time.RFC3339)
}

// refreshToken calls oauth.v2.access with grant_type=refresh_token
func (h *SlackHandler) refreshToken(clientID, clientSecret, refreshToken, tokenUrl string) (*SlackTokenResponse, error) {

	// https://slack.com/api/oauth.v2.access
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	// return postSlackToken("https://slack.com/api/oauth.v2.access", data)
	return h.postSlackToken(tokenUrl, data)
}

func (h *SlackHandler) resolveRuntimeConfig(ctx context.Context, config *models.ExecutionConfig, binding models.IntegrationBinding) (*SlackRuntimeConfig, error) {
	if binding.Credential != nil && len(binding.Credential.EncryptedData) > 0 {
		secrets, err := h.decryptCredentialSecrets(binding.Credential)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credential: %w", err)
		}
		if config.CredentialBinding.CredentialType == models.AuthOAuth2 {
			// follow authorization code flow for slack
			return secrets, nil
		} else if config.CredentialBinding.CredentialType == models.AuthAPIKey {

			if secrets.AccessToken == nil && *secrets.AccessToken != "" {
				return nil, fmt.Errorf("unexpected AccessToken is nil")
			}

			if secrets.RefreshToken == nil && *secrets.RefreshToken != "" {
				return nil, fmt.Errorf("unexpected RefreshToken is nil")
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
					resp, err := h.refreshToken(secrets.ClientIDKey, secrets.ClientSecretKey, *secrets.RefreshToken, *config.CredentialBinding.TokenUrl)
					if err != nil {
						return nil, err
					}
					expIn := h.ExpiryTimeRFC3339(resp.ExpiresIn)
					secrets.AccessToken = &resp.AccessToken
					secrets.RefreshToken = &resp.RefreshToken
					secrets.ExpiresIn = &expIn

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

// IsExpired takes an RFC3339 expiry string and returns whether it's expired
// (including "expiring soon" within 5 minutes) and how much time is left.
func (h *SlackHandler) IsExpired(expiryRFC3339 string) (bool, time.Duration, error) {
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

func (h *SlackHandler) decryptCredentialSecrets(credential *models.Credential) (*SlackRuntimeConfig, error) {
	if h == nil || h.encryptionSvc == nil {
		return nil, fmt.Errorf("GmailHandler not initialized with encryption service")
	}
	if credential == nil {
		return nil, fmt.Errorf("credential is nil")
	}

	decrypted, err := h.encryptionSvc.Decrypt(credential.EncryptedData)
	if err != nil {
		return nil, err
	}

	var secrets SlackRuntimeConfig
	if err := json.Unmarshal(decrypted, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential secrets: %w", err)
	}
	return &secrets, nil
}

func (h *SlackHandler) JoinWithoutFirstLast(arr []string) string {
	// If length <= 2, nothing to join
	if len(arr) <= 2 {
		return ""
	}
	// Slice from index 1 up to len-1 (exclusive of last)
	return strings.Join(arr[1:len(arr)-1], ", ")
}

func (h *SlackHandler) handleCallback(w http.ResponseWriter, r *http.Request, codeChan chan<- string, errChan chan<- error) {
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

// validateSchema validates that required fields in the schema are present in the data
// Works for both InputSchema and OutputSchema
func validateSchema(schema models.JSONMap, data map[string]interface{}, schemaType string) error {
	if schema == nil {
		return nil // no schema defined
	}

	// Get properties
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // no properties defined
	}

	// Get required fields
	requiredFields := []string{}
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredFields = append(requiredFields, s)
			}
		}
	}

	// Check required fields exist in data
	for _, key := range requiredFields {
		if _, exists := data[key]; !exists {
			return fmt.Errorf("missing required %s field: %s", schemaType, key)
		}
	}

	// Additional type validation (optional but recommended)
	for key, def := range props {
		defMap, ok := def.(map[string]interface{})
		if !ok {
			continue
		}

		// If field exists, validate type
		if value, exists := data[key]; exists && value != nil {
			expectedType, _ := defMap["type"].(string)
			if err := validateType(key, value, expectedType); err != nil {
				return fmt.Errorf("invalid type for %s field '%s': %w", schemaType, key, err)
			}
		}
	}

	return nil
}

// validateType checks if value matches expected JSON schema type
func validateType(field string, value interface{}, expectedType string) error {
	if expectedType == "" {
		return nil // no type specified
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "integer", "number":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			// ok
		case uint, uint8, uint16, uint32, uint64:
			// ok
		case float32, float64:
			// ok for number type
		default:
			return fmt.Errorf("expected %s, got %T", expectedType, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		// Check if it's a slice (but not a string which is also a slice of bytes)
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		// Check if it's a map or struct
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Map && kind != reflect.Struct {
			return fmt.Errorf("expected object, got %T", value)
		}
	default:
		// Unknown type, skip validation
		return nil
	}

	return nil
}

// PrintCollectedParams prints the key-value pairs of a map[string]interface{} in a sorted and readable format.
// Useful for debugging or logging collected parameters.
func PrintCollectedParams(params map[string]interface{}) {
	if len(params) == 0 {
		fmt.Println("Collected parameters: (empty)")
		return
	}

	fmt.Println("Collected parameters:")

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := params[key]

		// Optional: special formatting for common types
		switch v := value.(type) {
		case string:
			fmt.Printf("  %-24s : %q\n", key, v)
		case int, int64, float64:
			fmt.Printf("  %-24s : %v\n", key, v)
		case bool:
			fmt.Printf("  %-24s : %t\n", key, v)
		case nil:
			fmt.Printf("  %-24s : <nil>\n", key)
		default:
			// For other types (structs, slices, maps, etc.) use %#v for more detail
			fmt.Printf("  %-24s : %#v\n", key, v)
		}
	}
}
