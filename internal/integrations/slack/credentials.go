package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/logger"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
)

var (
	SlackBotCredentialName       = "Symphony Platform Bot Slack Credentials"
	SlackOauthUserCredentialName = "Symphony Platform User Slack Credentials"
)

func AddCredentials(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) ([]*models.Credential, error) {

	userCred, err := AddUserSlackAuthCredential(ctx, encryptionSvc, repo, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	botCred, err := AddBotSlackAuthCredential(ctx, encryptionSvc, repo, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	// Collect both credentials into a slice
	platformCreds := []*models.Credential{
		userCred, botCred,
	}

	// Now you can return or use platformCreds
	return platformCreds, nil
}

func AddUserSlackAuthCredential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Credential, error) {

	SLACK_CLIENT_ID := os.Getenv("SLACK_CLIENT_ID")
	if SLACK_CLIENT_ID == "" {
		return nil, fmt.Errorf("no SLACK_CLIENT_ID defined...")
	}

	SLACK_CLIENT_SECRET := os.Getenv("SLACK_CLIENT_SECRET")
	if SLACK_CLIENT_SECRET == "" {
		return nil, fmt.Errorf("no SLACK_CLIENT_SECRET defined...")
	}

	SLACK_REDIRECT_URL := os.Getenv("SLACK_REDIRECT_URL")
	if SLACK_REDIRECT_URL == "" {
		return nil, fmt.Errorf("no SLACK_REDIRECT_URL defined...")
	}

	UserScopes := []string{
		"admin",
		"channels:history",
		"channels:read",
		"channels:write",
		"channels:write.invites",
		"channels:write.topic",
		"chat:write",
		"im:write",
		"users:read",
		"users:read.email",
	}

	// Prepare secrets for encryption

	user_secrets := map[string]interface{}{
		"client_id":     SLACK_CLIENT_ID,
		"client_secret": SLACK_CLIENT_SECRET,
		"redirect_url":  SLACK_REDIRECT_URL,
		"user_scopes":   UserScopes,
	}

	userSecretsJSON, err := json.Marshal(user_secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	userEncryptedData, err := encryptionSvc.Encrypt(userSecretsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secrets...")
	}

	userDataHash := encryptionSvc.GenerateDataHash(userEncryptedData)

	exist, err := commons.CheckIfCredentialExistByName(ctx, repo, tenantID, SlackOauthUserCredentialName)
	if exist != nil {
		return exist, nil
	}

	if exist != nil {
		return exist, nil
	}

	logger.Get(ctx).Debug("Creating new credential...")

	oAuth2UserCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             SlackOauthUserCredentialName,
		Description:      "Details about credential used for auth credentials",
		Provider:         SlackVendorName,
		Type:             models.AuthOAuth2,
		EncryptedData:    userEncryptedData,
		DataHash:         userDataHash,
		Scopes:           UserScopes,
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, oAuth2UserCredential)
	if err != nil {
		return nil, fmt.Errorf("failed to create slack credential...")
	}
	return oAuth2UserCredential, nil
}

func AddBotSlackAuthCredential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Credential, error) {

	SLACK_CLIENT_ID := os.Getenv("SLACK_CLIENT_ID")
	if SLACK_CLIENT_ID == "" {
		return nil, fmt.Errorf("no SLACK_CLIENT_ID defined...")
	}

	SLACK_CLIENT_SECRET := os.Getenv("SLACK_CLIENT_SECRET")
	if SLACK_CLIENT_SECRET == "" {
		return nil, fmt.Errorf("no SLACK_CLIENT_SECRET defined...")
	}

	SLACK_REDIRECT_URL := os.Getenv("SLACK_REDIRECT_URL")
	if SLACK_REDIRECT_URL == "" {
		return nil, fmt.Errorf("no SLACK_REDIRECT_URL defined...")
	}

	SLACK_BOT_LONG_LIVED_TOKEN := os.Getenv("SLACK_BOT_LONG_LIVED_TOKEN")
	if SLACK_BOT_LONG_LIVED_TOKEN == "" {
		return nil, fmt.Errorf("no SLACK_BOT_LONG_LIVED_TOKEN defined...")
	}

	BotScopes := []string{
		"assistant:write",
		"channels:history",
		"channels:join",
		"channels:manage",
		"channels:read",
		"channels:write.invites",
		"chat:write",
		"conversations.connect:manage",
		"conversations.connect:read",
		"conversations.connect:write",
		"im:write",
		"users:read",
		"users:read.email",
	}
	// Prepare secrets for encryption

	bot_secrets := map[string]interface{}{
		"client_id":        SLACK_CLIENT_ID,
		"client_secret":    SLACK_CLIENT_SECRET,
		"bot_scopes":       BotScopes,
		"long_lived_token": SLACK_BOT_LONG_LIVED_TOKEN,
	}

	// Encrypt secrets
	botSecretsJSON, err := json.Marshal(bot_secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	botEncryptedData, err := encryptionSvc.Encrypt(botSecretsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secrets...")
	}

	botDataHash := encryptionSvc.GenerateDataHash(botEncryptedData)

	exist, err := commons.CheckIfCredentialExistByName(ctx, repo, tenantID, SlackBotCredentialName)
	if exist != nil {
		return exist, nil
	}

	logger.Get(ctx).Debug("Creating new credential...")

	oAuth2BotCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             SlackBotCredentialName,
		Description:      "Details about credential used for auth credentials",
		Provider:         SlackVendorName,
		Type:             models.AuthAPIKey,
		EncryptedData:    botEncryptedData,
		DataHash:         botDataHash,
		Scopes:           BotScopes,
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, oAuth2BotCredential)
	if err != nil {
		return nil, fmt.Errorf("failed to create slack credential...")
	}
	return oAuth2BotCredential, nil
}
