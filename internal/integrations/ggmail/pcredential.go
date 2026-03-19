package ggmail

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/logger"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"github.com/rkumar-bengaluru/Integrations/internal/repository"
)

func AddGmailCredentials(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) ([]*models.Credential, error) {

	botCred, err := AddOauth2ClientCredential(ctx, encryptionSvc, repo, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	// Collect both credentials into a slice
	platformCreds := []*models.Credential{
		botCred,
	}

	// Now you can return or use platformCreds
	return platformCreds, nil
}

func AddOauth2ClientCredential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Credential, error) {

	SYMPHONY_APP_GOOGLE_CLIENT_ID := os.Getenv("SYMPHONY_APP_GOOGLE_CLIENT_ID")
	if SYMPHONY_APP_GOOGLE_CLIENT_ID == "" {
		err := fmt.Errorf("no SYMPHONY_APP_GOOGLE_CLIENT_ID defined...")
		return nil, err
	}
	SYMPHONY_APP_GOOGLE_CLIENT_SECRET := os.Getenv("SYMPHONY_APP_GOOGLE_CLIENT_SECRET")
	if SYMPHONY_APP_GOOGLE_CLIENT_SECRET == "" {
		err := fmt.Errorf("no SYMPHONY_APP_GOOGLE_CLIENT_SECRET defined...")
		return nil, err
	}
	SYMPHONY_APP_GOOGLE_REDIRECT_URL := os.Getenv("SYMPHONY_APP_GOOGLE_REDIRECT_URL")
	if SYMPHONY_APP_GOOGLE_REDIRECT_URL == "" {
		err := fmt.Errorf("no SYMPHONY_APP_GOOGLE_REDIRECT_URL defined...")
		return nil, err
	}

	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"client_id":     SYMPHONY_APP_GOOGLE_CLIENT_ID,
		"client_secret": SYMPHONY_APP_GOOGLE_CLIENT_SECRET,

		"redirect_uri": SYMPHONY_APP_GOOGLE_REDIRECT_URL,
		"scopes": []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/userinfo.email",
		},
	}

	// Encrypt secrets
	botSecretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets...")
	}

	botEncryptedData, err := encryptionSvc.Encrypt(botSecretsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secrets...")
	}

	botDataHash := encryptionSvc.GenerateDataHash(botEncryptedData)

	exist, err := commons.CheckIfCredentialExistByName(ctx, repo, tenantID, GmailOauth2AuthorizationCodeFlowCredentialName)
	if exist != nil {
		return exist, nil
	}

	logger.Get(ctx).Debug("Creating new credential...")

	oAuth2BotCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             GmailOauth2AuthorizationCodeFlowCredentialName,
		Description:      "Details about credential used for auth credentials",
		Provider:         GmailVendorName,
		Type:             models.AuthOAuth2,
		EncryptedData:    botEncryptedData,
		DataHash:         botDataHash,
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, oAuth2BotCredential)
	if err != nil {
		return nil, fmt.Errorf("failed to %s slack credential...", GmailOauth2AuthorizationCodeFlowCredentialName)
	}
	return oAuth2BotCredential, nil
}
