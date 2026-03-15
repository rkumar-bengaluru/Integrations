package snow

// SnowTokenResponse represents the common shape of successful responses
// from both oauth.v2.exchange and oauth.v2.access (with token rotation enabled)
type SnowTokenResponse struct {
	AccessToken string `json:"access_token"` // xoxe. prefix, expires in 12h
	ExpiresIn   int    `json:"expires_in"`   // usually 43200 (12 hours)
	TokenType   string `json:"token_type"`   // "bot" or "user"
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
}

// RuntimeConfig holds resolved values from binding

type SnowRuntimeConfig struct {
	// User-provided values
	ClientIDKey     string `json:"client_id_key"`
	ClientSecretKey string `json:"client_secret_key"`
	GrantTypeKey    string `json:"grant_type"`

	// Generated values (runtime tokens)
	AccessToken *string `json:"access_token"`
	TokenType   *string `json:"token_type"`
	ExpiresIn   *string `json:"expires_in"` // RFC3339 string
	Scope       string  `json:"scope"`
}
