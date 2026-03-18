package github

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/integrations/github/actions"
	"agent.fabric.com/modules/internal/logger"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/repository/impl"
	"agent.fabric.com/modules/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateGithubIntegration(ctx context.Context,
	database *gorm.DB,
	logger *zap.Logger,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Integration, error) {

	integrationRepo := impl.NewIntegrationRepository(database, logger)

	// check if integration already exist.
	_, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GithubIntegrationName)
	if err == repository.ErrIntegrationNotFound {
		// get the platform credential id for this integration
		credentials, err := credentialRepo.GetAllCredentials(ctx, GithubVendorName)
		if err != nil {
			credentials, err = AddGithubCredentials(ctx, encryptionSvc, credentialRepo, tenantID)
			if err != nil {
				return nil, fmt.Errorf("Error creating credential %s", GithubOauth2AuthorizationCodeFlowCredentialName)
			}
		}

		platformCredentials, err := commons.ConvertCredentialsToPlatform(credentials, nil)

		if err != nil {
			return nil, fmt.Errorf("Error creating platform credential %s", GithubOauth2AuthorizationCodeFlowCredentialName)
		}
		entityMap := make(models.ExecutionConfigs, 1)
		// Bot key
		entityMap[models.AuthOAuth2] = models.ExecutionConfig{
			CredentialType: models.AuthOAuth2,
			CredentialBinding: &models.CredentialBinding{
				CredentialType: models.AuthGitubServiceAcc,
				GrantType:      utils.Ptr(models.GrantGitubServiceAcc),
				AuthorityUrl:   utils.Ptr("https://api.github.com/app/installations/{installation_id}/access_tokens"),
				SecretMapping: map[string]interface{}{
					"private_key_key":     "private_key",
					"app_id_key":          "app_id",
					"installation_id_key": "installation_id",
					"owner_key":           "owner",
					"repo_key":            "repo",
				},
				Notes: utils.Ptr("github authorization clode flow"),
			},
			ParamInputSchema: &models.JSONSchema{
				Title:       "Github Integration request fields",
				Description: "Fields required to call the Github API.",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"private_key": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.private_key",
					},
					"app_id": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "$.platformcredential.app_id",
					},
					"installation_id": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "user_input",
					},
					"owner": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "user_input",
					},
					"repo": {
						Type:        "string",
						Description: "slack xoxb bot secret",
						Secret:      true,
						Source:      "user_input",
					},
				},
				Required: []string{"private_key", "app_id", "installation_id", "owner", "repo"},
			},
			ParamOuputSchema: &models.JSONSchema{
				Title:       "Snow OAuth v2 Access Response",
				Description: "Fields returned from https://dev212340.service-now.com//oauth_token.do",
				Type:        "object",
				Properties: map[string]models.JSONSchemaProperty{
					"token": {
						Type:        "string",
						Description: "Top-level access token (bot or user)",
						Secret:      true,
						Source:      "$.access_token",
					},
					"token_expires_at": {
						Type:        "integer",
						Description: "Lifetime of the top-level access token in seconds",
						Source:      "$.token_expires_at",
					},
					"token_type": {
						Type:        "string",
						Description: "Token type: bot or user",
						Source:      "$.token_type",
					},
					"generated_at": {
						Type:        "string",
						Description: "Scopes granted to the top-level token",
						Source:      "$.generated_at",
					},
				},
			},
		}

		// create new integration
		integration := &models.Integration{
			Name:        GithubIntegrationName,
			TenantID:    tenantID,
			Description: "Integration with Snow for ITSM",
			Category:    models.CategoryCommunication,
			SupportedCredentialTypes: []models.CredentialType{
				models.AuthOAuth2,
			},
			PlatformCredentials: platformCredentials,
			ExecutionConfigs:    entityMap,
			Actions: []models.ActionDefinition{
				{
					TenantID:          tenantID,
					Name:              GithubTestActionName,
					Description:       "A simple test action to validate framework wiring",
					Type:              models.ActionType(GithubTestActionType),
					SchemaVersion:     "v1",
					SupportsStreaming: false,
					IsInternal:        false,
					Version:           "1.0",
					IsActive:          true,
					ActionHandler:     "test_action_handler",
					InputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"message": map[string]interface{}{
								"type":        "string",
								"description": "A sample input message to test the action",
								"source":      "user_input",
							},
						},
						"required": []string{"message"},
					},
					OutputSchema: models.JSONMap{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]interface{}{
								"type":        "boolean",
								"description": "Indicates if the test action executed successfully",
							},
							"echo": map[string]interface{}{
								"type":        "string",
								"description": "Echoes back the input string",
							},
						},
						"required": []interface{}{"success", "echo"},
					},
				},
			},
		}

		err = integrationRepo.CreateIntegration(ctx, integration)
		if err != nil {
			return nil, fmt.Errorf("Error creating integration %s, with error %w", GithubIntegrationName, err)
		}

		// create additional integration actions if there are new one added.
		err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
		if err != nil {
			return nil, err
		}

		integration, err = integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GithubIntegrationName)
		if err != nil {
			return nil, err
		}

		return integration, nil
	}

	integration, err := integrationRepo.GetIntegrationByTenantIDAndName(ctx, tenantID, GithubIntegrationName)
	if err != nil {
		return nil, err
	}
	// create additional integration actions if there are new one added.
	err = AddActions(ctx, tenantID, integrationRepo, integration, logger)
	if err != nil {
		return nil, err
	}
	// integration already exist.
	logger.Debug(fmt.Sprintf("integration  %s already exit", GithubIntegrationName))
	return integration, nil
}

func AddActions(ctx context.Context,
	tenantID uuid.UUID,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration,
	logger *zap.Logger) error {
	action, err := actions.ListPullRequestAction(ctx, tenantID,
		GithubPullOpenRequestActionName, GithubPullOpenRequestActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.GetPullRequestByNumberAction(ctx, tenantID,
		GithubGetPullRequestByNumberActionName, GithubGetPullRequestByNumberActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.ListCommitsByPRAction(ctx, tenantID,
		GithubListCommitsActionName, GithubListCommitsActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.GetCommitsByShaAction(ctx, tenantID,
		GithubGetCommitsActionName, GithubGetCommitsActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	action, err = actions.CreateReviewCommentAction(ctx, tenantID,
		GithubAddReviewCommentActionName, GithubAddReviewCommentActionType, integrationRepo, integration)
	if err != nil {
		fmt.Errorf("error creating action %w", err)
	}
	logger.Debug("created action successfully", zap.String(action.Name, action.Name))

	return nil
}

func AddGithubCredentials(ctx context.Context,
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

func validateRSAPrivateKey(key string) error {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return fmt.Errorf("failed to parse PEM block")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return fmt.Errorf("expected RSA PRIVATE KEY, got %s", block.Type)
	}
	return nil
}

func AddOauth2ClientCredential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) (*models.Credential, error) {

	SYMPHONY_GITHUB_APP_ID := os.Getenv("SYMPHONY_GITHUB_APP_ID")
	if SYMPHONY_GITHUB_APP_ID == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_GITHUB_APP_ID defined..."))
		os.Exit(1)
	}

	SYMPHONY_GITHUB_APP_SECRET_FILE := os.Getenv("SYMPHONY_GITHUB_APP_SECRET_FILE")
	if SYMPHONY_GITHUB_APP_SECRET_FILE == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_GITHUB_APP_SECRET_FILE defined..."))
		os.Exit(1)
	}

	if _, err := os.Stat(SYMPHONY_GITHUB_APP_SECRET_FILE); os.IsNotExist(err) {
		fmt.Printf("   ⚠️  File not found: %s\n", SYMPHONY_GITHUB_APP_SECRET_FILE)
		return nil, err
	}

	content, err := os.ReadFile(SYMPHONY_GITHUB_APP_SECRET_FILE)
	if err != nil {
		fmt.Printf("   ❌ Error reading: %v\n", err)
		return nil, err
	}
	fmt.Printf("   ✅ Loaded (%d bytes)\n", len(content))
	private_key := string(content)

	err = validateRSAPrivateKey(private_key)

	if err != nil {
		return nil, err
	}

	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"app_id":      SYMPHONY_GITHUB_APP_ID,
		"private_key": private_key,
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

	exist, err := commons.CheckIfCredentialExistByName(ctx, repo, tenantID, GithubOauth2AuthorizationCodeFlowCredentialName)
	if exist != nil {
		return exist, nil
	}

	logger.Get(ctx).Debug("Creating new credential...")

	oAuth2BotCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             GithubOauth2AuthorizationCodeFlowCredentialName,
		Description:      "Details about credential used for auth credentials",
		Provider:         GithubVendorName,
		Type:             models.AuthOAuth2,
		EncryptedData:    botEncryptedData,
		DataHash:         botDataHash,
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, oAuth2BotCredential)
	if err != nil {
		return nil, fmt.Errorf("failed to %s slack credential...", GithubOauth2AuthorizationCodeFlowCredentialName)
	}
	return oAuth2BotCredential, nil
}
