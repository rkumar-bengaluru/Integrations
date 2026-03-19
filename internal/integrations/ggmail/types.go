package ggmail

import (
	"time"
)

// RuntimeConfig holds resolved values from binding
type GmailRuntimeConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopres"`
	RedirectUrl  string   `json:"redirect_url"`

	AccessToken  *string    `json:"access_token"`
	RefreshToken *string    `json:"refresh_token"`
	TokenExpiry  *time.Time `json:"token_expiry"`
	UserEmail    *string    `json:"user_email"` // Delegation email (optional, defaults to "me")
}
