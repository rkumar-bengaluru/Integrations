package slack

// SlackTokenResponse represents the common shape of successful responses
// from both oauth.v2.exchange and oauth.v2.access (with token rotation enabled)
type SlackTokenResponse struct {
	OK           bool   `json:"ok"`
	AccessToken  string `json:"access_token"` // xoxe. prefix, expires in 12h
	ExpiresIn    int    `json:"expires_in"`   // usually 43200 (12 hours)
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"` // "bot" or "user"
	Scope        string `json:"scope"`
	AppID        string `json:"app_id"`
	Team         struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	// Bot-specific fields (present for bot tokens)
	BotUserID string `json:"bot_user_id,omitempty"`
	// User-specific fields (present for user tokens)
	AuthedUser struct {
		ID           string `json:"id"`
		Scope        string `json:"scope"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
	} `json:"authed_user,omitempty"`
	Error string `json:"error,omitempty"`
}

// RuntimeConfig holds resolved values from binding

type SlackRuntimeConfig struct {
	// User-provided values
	ClientIDKey       string   `json:"client_id_key"`
	ClientSecretKey   string   `json:"client_secret_key"`
	UserScopeKey      []string `json:"user_scope_key"`
	BotScopeKey       []string `json:"bot_scope_key"`
	RedirectURLKey    string   `json:"redirect_url_key"`
	LongLivedTokenKey string   `json:"long_lived_token_key"`

	// Generated values (runtime tokens)
	AccessToken  *string `json:"access_token"`
	RefreshToken *string `json:"refresh_token"`
	TokenType    *string `json:"token_type"`
	ExpiresIn    *string `json:"expires_in"` // RFC3339 string
	UserID       *string `json:"user_id"`
}

// Slack OAuth response structure
type OAuthResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token,omitempty"` // bot token if requested
	TokenType   string `json:"token_type,omitempty"`
	Scope       string `json:"scope,omitempty"`
	AuthedUser  struct {
		ID          string `json:"id"`
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	} `json:"authed_user"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}
