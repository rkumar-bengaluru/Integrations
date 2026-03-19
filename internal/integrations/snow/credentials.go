package snow

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

var (
	SNOW_CLIENT_CREDENTIALS = "client_credentials"
)

func AddSnowCredentials(ctx context.Context,
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

	SNOW_CLIENT_ID := os.Getenv("SNOW_CLIENT_ID")
	if SNOW_CLIENT_ID == "" {
		return nil, fmt.Errorf("no SNOW_CLIENT_ID defined...")
	}

	SNOW_CLIENT_SECRET := os.Getenv("SNOW_CLIENT_SECRET")
	if SNOW_CLIENT_SECRET == "" {
		return nil, fmt.Errorf("no SNOW_CLIENT_SECRET defined...")
	}

	// Prepare secrets for encryption

	bot_secrets := map[string]interface{}{
		"client_id":     SNOW_CLIENT_ID,
		"client_secret": SNOW_CLIENT_SECRET,
		"grant_type":    SNOW_CLIENT_CREDENTIALS,
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

	exist, err := commons.CheckIfCredentialExistByName(ctx, repo, tenantID, SnowOauth2ClientCredentialFlowName)
	if exist != nil {
		return exist, nil
	}

	logger.Get(ctx).Debug("Creating new credential...")

	oAuth2BotCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             SnowOauth2ClientCredentialFlowName,
		Description:      "Details about credential used for auth credentials",
		Provider:         SnowVendorName,
		Type:             models.AuthOAuth2,
		EncryptedData:    botEncryptedData,
		DataHash:         botDataHash,
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, oAuth2BotCredential)
	if err != nil {
		return nil, fmt.Errorf("failed to %s slack credential...", SnowOauth2ClientCredentialFlowName)
	}
	return oAuth2BotCredential, nil
}
