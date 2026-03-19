package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"go.uber.org/zap"
)

func (h *SlackHandler) AuthorizationFlow(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {

	client_id, ok := collectedParams["client_id"].(string)
	if !ok || client_id == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	client_secret, ok := collectedParams["client_secret"].(string)
	if !ok || client_secret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}

	rawScopes, ok := collectedParams["user_scopes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("user_scope is required")
	}

	userScopes := make([]string, len(rawScopes))
	for i, v := range rawScopes {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("user_scope must be array of strings")
		}
		userScopes[i] = s
	}

	redirect_url, ok := collectedParams["redirect_url"].(string)
	if !ok || redirect_url == "" {
		return nil, fmt.Errorf("redirect_url is required")
	}

	h.logger.Info("Initializing for slack authorization code flow ...",
		zap.String("credential type", string(config.CredentialBinding.CredentialType)))
	authorityUrl := "https://slack.com/oauth/v2/authorize"
	if config.CredentialBinding.AuthorityUrl != nil && *config.CredentialBinding.AuthorityUrl != "" {
		// Safe to use the value
		h.logger.Debug(fmt.Sprintf("authorityUrl %s", *config.CredentialBinding.AuthorityUrl))
		authorityUrl = *config.CredentialBinding.AuthorityUrl
	} else {
		h.logger.Debug(fmt.Sprintf("authorityUrl is not set or empty using default %s", authorityUrl))
		fmt.Println("authorityUrl is not set or empty")
	}

	user_scopes := h.JoinWithoutFirstLast(userScopes)

	authURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&user_scope=%s&redirect_uri=%s",
		url.QueryEscape(client_id),
		url.QueryEscape(user_scopes), // ← change to valid scopes you actually need
		url.QueryEscape(redirect_url),
	)

	// follow authorization code flow for slack
	h.logger.Info("Opening browser for authorization...")
	h.logger.Info("If browser doesn't open, visit this URL:")
	h.logger.Info(authorityUrl)

	// Open browser automatically
	if err := browser.OpenURL(authURL); err != nil {
		h.logger.Error("Failed to open browser:", zap.Error(err))
		return nil, err
	}

	// Step 3: Start local HTTP server to receive callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		h.handleCallback(w, r, codeChan, errChan)
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

	// Exchange code for token
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", url.Values{
		"client_id":     {client_id},
		"client_secret": {client_secret},
		"code":          {code},
		"redirect_uri":  {redirect_url},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to exchange code:" + err.Error())
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body) // for debugging

	var oauthResp OAuthResponse
	if err := json.Unmarshal(bodyBytes, &oauthResp); err != nil {
		return nil, fmt.Errorf("Failed to decode Slack response:" + err.Error())
	}

	if !oauthResp.OK {
		errMsg := fmt.Sprintf("Slack OAuth failed: error=%s\nRaw response: %s", oauthResp, string(bodyBytes))
		return nil, fmt.Errorf(errMsg)
	}

	if oauthResp.OK {
		accessToken := oauthResp.AuthedUser.AccessToken // this is the one in your JSON
		h.logger.Debug("Exchange the long lived one with refresh token!")
		h.logger.Debug("user id ", zap.String("authd user", oauthResp.AuthedUser.ID))
		// Check if it looks long-lived (no expires_in in response)

		// Build the complete credential data
		// This includes both user-provided values AND generated tokens
		credentialData := map[string]interface{}{
			// User-provided values (stored with _key suffix as per SecretMapping)
			"client_id_key":     client_id,
			"client_secret_key": client_secret,
			"user_scope_key":    userScopes,
			"redirect_url_key":  redirect_url,

			// Generated values (runtime tokens)
			"access_token": accessToken,
			"user_id":      oauthResp.AuthedUser.ID,
		}

		h.logger.Info("Slack App initialization successful")
		return credentialData, nil
	}

	return nil, fmt.Errorf(fmt.Sprintf("Unexpected error in authorization code flow"))

}
