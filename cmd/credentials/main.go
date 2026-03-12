// cmd/gmail/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/repository/db"
	"agent.fabric.com/modules/internal/repository/impl"
	"agent.fabric.com/modules/internal/utils"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: Error loading .env file")
	}

	// Initialize services
	keyID := os.Getenv("ENCRYPTION_KEY_ID")
	if keyID == "" {
		keyID = "default-key-v1"
	}

	logger, err := utils.NewFileLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	encryptionSvc := encryption.NewEncryptionService(keyID)

	// Setup database
	conn, dialect := db.CreateDB(ctx, "genei-server")
	defer conn.Close()
	_, dbConn := repository.NewSQLStore(dialect, logger)

	repo := impl.NewCredentialRepository(dbConn, logger)

	// Tenant and Integration IDs - adjust these for your Gmail integration
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	create_gmail_credential(ctx, encryptionSvc, repo, tenantID)
	create_azure_storage_credential(ctx, encryptionSvc, repo, tenantID)
	create_openai_credential(ctx, encryptionSvc, repo, tenantID)
	create_github_credential(ctx, encryptionSvc, repo, tenantID)
	create_qdrant_credential(ctx, encryptionSvc, repo, tenantID)
	create_azure_sharepoint_credential(ctx, encryptionSvc, repo, tenantID)
}

func check_if_credential_exists(ctx context.Context,
	repo repository.CredentialRepository,
	tenantID uuid.UUID,
	name string) bool {

	_, err := repo.GetByTenantIDAndName(ctx, tenantID, name)
	if err == repository.ErrCredentialNotFound {
		return true
	}

	return false
}

func create_azure_sharepoint_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {

	SHAREPOINT_TENANT_ID := os.Getenv("SHAREPOINT_TENANT_ID")
	if SHAREPOINT_TENANT_ID == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_TENANT_ID defined..."))
		os.Exit(1)
	}

	SHAREPOINT_CLIENT_ID := os.Getenv("SHAREPOINT_CLIENT_ID")
	if SHAREPOINT_CLIENT_ID == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_CLIENT_ID defined..."))
		os.Exit(1)
	}

	SHAREPOINT_CLIENT_SECRET := os.Getenv("SHAREPOINT_CLIENT_SECRET")
	if SHAREPOINT_CLIENT_SECRET == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_CLIENT_SECRET defined..."))
		os.Exit(1)
	}

	SHAREPOINT_SITE_ID := os.Getenv("SHAREPOINT_SITE_ID")
	if SHAREPOINT_SITE_ID == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_SITE_ID defined..."))
		os.Exit(1)
	}

	SHAREPOINT_DRIVE_ID := os.Getenv("SHAREPOINT_DRIVE_ID")
	if SHAREPOINT_DRIVE_ID == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_DRIVE_ID defined..."))
		os.Exit(1)
	}

	SHAREPOINT_SCOPE := os.Getenv("SHAREPOINT_SCOPE")
	if SHAREPOINT_SCOPE == "" {
		fmt.Println(fmt.Errorf("no SHAREPOINT_SCOPE defined..."))
		os.Exit(1)
	}
	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"client_id":     SHAREPOINT_CLIENT_ID,
		"client_secret": SHAREPOINT_CLIENT_SECRET,
		"tenant_id":     SHAREPOINT_TENANT_ID,
		"site_id":       SHAREPOINT_SITE_ID,
		"drive_id":      SHAREPOINT_DRIVE_ID,
		"scope":         SHAREPOINT_SCOPE,
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony Azure Sharepoint Credential"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	githubPlatformCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             credentialName,
		Description:      "Details about credential used for azure sharepoint",
		Type:             models.AuthOAuth2,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           []string{},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, githubPlatformCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create github credential: %v", err))
	}

}

func create_azure_storage_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {

	AZURE_CLIENT_ID := os.Getenv("AZURE_CLIENT_ID")
	if AZURE_CLIENT_ID == "" {
		fmt.Println(fmt.Errorf("no AZURE_CLIENT_ID defined..."))
		os.Exit(1)
	}

	AZURE_SECRET_ID := os.Getenv("AZURE_SECRET_ID")
	if AZURE_SECRET_ID == "" {
		fmt.Println(fmt.Errorf("no AZURE_SECRET_ID defined..."))
		os.Exit(1)
	}

	AZURE_CLIENT_SECRET := os.Getenv("AZURE_CLIENT_SECRET")
	if AZURE_CLIENT_SECRET == "" {
		fmt.Println(fmt.Errorf("no AZURE_CLIENT_SECRET defined..."))
		os.Exit(1)
	}

	AZURE_TENANT_ID := os.Getenv("AZURE_TENANT_ID")
	if AZURE_TENANT_ID == "" {
		fmt.Println(fmt.Errorf("no AZURE_TENANT_ID defined..."))
		os.Exit(1)
	}

	STORAGE_ACCOUNT_NAME := os.Getenv("STORAGE_ACCOUNT_NAME")
	if STORAGE_ACCOUNT_NAME == "" {
		fmt.Println(fmt.Errorf("no STORAGE_ACCOUNT_NAME defined..."))
		os.Exit(1)
	}
	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"client_id":            AZURE_CLIENT_ID,
		"client_secret":        AZURE_CLIENT_SECRET,
		"secret_id":            AZURE_SECRET_ID,
		"tenant_id":            AZURE_TENANT_ID,
		"storage_account_name": STORAGE_ACCOUNT_NAME,
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony Azure Storage Credential"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	githubPlatformCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             credentialName,
		Description:      "Details about credential used for azure storage blob",
		Type:             models.AuthOAuth2,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           []string{},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, githubPlatformCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create github credential: %v", err))
	}

}

func create_openai_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {

	OPENAI_KEY := os.Getenv("OPENAI_KEY")
	if OPENAI_KEY == "" {
		fmt.Println(fmt.Errorf("no OPENAI_KEY defined..."))
		os.Exit(1)
	}

	OPENAI_EMBEDDING_MODEL := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if OPENAI_KEY == "" {
		fmt.Println(fmt.Errorf("no OPENAI_EMBEDDING_MODEL defined..."))
		os.Exit(1)
	}

	OPENAI_EMBEDDING_DIMENSION := os.Getenv("OPENAI_EMBEDDING_DIMENSION")
	if OPENAI_KEY == "" {
		fmt.Println(fmt.Errorf("no OPENAI_EMBEDDING_DIMENSION defined..."))
		os.Exit(1)
	}

	OPENAI_EMBEDDING_DISTANCE := os.Getenv("OPENAI_EMBEDDING_DISTANCE")
	if OPENAI_KEY == "" {
		fmt.Println(fmt.Errorf("no OPENAI_EMBEDDING_DISTANCE defined..."))
		os.Exit(1)
	}
	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"api_key":             OPENAI_KEY,
		"embedding_model":     OPENAI_EMBEDDING_MODEL,
		"embedding_dimension": OPENAI_EMBEDDING_DIMENSION,
		"embedding_distance":  OPENAI_EMBEDDING_DISTANCE,
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony OpenAI Credential"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	githubPlatformCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             credentialName,
		Description:      "Details about credential used for openai embeddings.",
		Type:             models.AuthOAuth2,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           []string{},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, githubPlatformCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create github credential: %v", err))
	}

}

func create_github_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {

	SYMPHONY_GITHUB_APP_ID := os.Getenv("SYMPHONY_GITHUB_APP_ID")
	if SYMPHONY_GITHUB_APP_ID == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_GITHUB_APP_ID defined..."))
		os.Exit(1)
	}
	SYMPHONY_GITHUB_CLIENT_ID := os.Getenv("SYMPHONY_APP_GOOGLE_CLIENT_SECRET")
	if SYMPHONY_GITHUB_CLIENT_ID == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_GITHUB_CLIENT_ID defined..."))
		os.Exit(1)
	}

	SYMPHONY_GITHIB_INSTALLATION_ID := os.Getenv("SYMPHONY_GITHIB_INSTALLATION_ID")
	if SYMPHONY_GITHIB_INSTALLATION_ID == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_GITHIB_INSTALLATION_ID defined..."))
		os.Exit(1)
	}

	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"app_id":          SYMPHONY_GITHUB_APP_ID,
		"client_id":       SYMPHONY_GITHUB_CLIENT_ID,
		"installation_id": SYMPHONY_GITHIB_INSTALLATION_ID,
		"redirect_uri":    "http://localhost:8080/callback",
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony Github Credential"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	githubPlatformCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             credentialName,
		Description:      "Details about application created in github to associate repos to this platform",
		Type:             models.AuthGitubServiceAcc,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           []string{},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, githubPlatformCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create github credential: %v", err))
	}

}

func create_gmail_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {
	SYMPHONY_APP_GOOGLE_CLIENT_ID := os.Getenv("SYMPHONY_APP_GOOGLE_CLIENT_ID")
	if SYMPHONY_APP_GOOGLE_CLIENT_ID == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_APP_GOOGLE_CLIENT_ID defined..."))
		os.Exit(1)
	}
	SYMPHONY_APP_GOOGLE_CLIENT_SECRET := os.Getenv("SYMPHONY_APP_GOOGLE_CLIENT_SECRET")
	if SYMPHONY_APP_GOOGLE_CLIENT_SECRET == "" {
		fmt.Println(fmt.Errorf("no SYMPHONY_APP_GOOGLE_CLIENT_SECRET defined..."))
		os.Exit(1)
	}

	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"client_id":     SYMPHONY_APP_GOOGLE_CLIENT_ID,
		"client_secret": SYMPHONY_APP_GOOGLE_CLIENT_SECRET,
		"redirect_uri":  "http://localhost:8080/callback",
		"scopes": []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/userinfo.email",
		},
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony Gmail Credential"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	gmailPlatformCredential := &models.Credential{
		TenantID:      tenantID,
		Name:          credentialName,
		Description:   "Details about application created in google to associate email id's to this platform",
		Type:          models.AuthOAuth2,
		EncryptedData: encryptedData,
		DataHash:      dataHash,
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, gmailPlatformCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create gmail credential: %v", err))
	}

}

func create_qdrant_credential(ctx context.Context,
	encryptionSvc encryption.EncryptionService,
	repo repository.CredentialRepository,
	tenantID uuid.UUID) {
	QDRANT_API_KEY := os.Getenv("QDRANT_API_KEY")
	if QDRANT_API_KEY == "" {
		fmt.Println(fmt.Errorf("no QDRANT_API_KEY defined..."))
		os.Exit(1)
	}
	QDRANT_CONNECTION_STRING := os.Getenv("QDRANT_CONNECTION_STRING")
	if QDRANT_CONNECTION_STRING == "" {
		fmt.Println(fmt.Errorf("no QDRANT_CONNECTION_STRING defined..."))
		os.Exit(1)
	}

	// Prepare secrets for encryption
	secrets := map[string]interface{}{
		"api_key":           QDRANT_API_KEY,
		"connection_string": QDRANT_CONNECTION_STRING,
	}

	// Encrypt secrets
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal qdrant secrets: %v", err))
	}

	encryptedData, err := encryptionSvc.Encrypt(secretsJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt secrets: %v", err))
	}
	dataHash := encryptionSvc.GenerateDataHash(encryptedData)

	credentialName := "Symphony Qdrant Integration Credentials"

	if !check_if_credential_exists(ctx, repo, tenantID, credentialName) {
		fmt.Println(fmt.Sprintf("Credential %s already exist", credentialName))
		return
	}

	qdrantCredential := &models.Credential{
		TenantID:         tenantID,
		Name:             credentialName,
		Description:      "Details about qdrant credential details",
		Type:             models.AuthAPIKey,
		EncryptedData:    encryptedData,
		DataHash:         dataHash,
		Scopes:           []string{},
		ValidationStatus: models.ValidationUnverified,
	}

	err = repo.Create(ctx, qdrantCredential)
	if err != nil {
		panic(fmt.Sprintf("failed to create qdrant Credential: %v", err))
	}

}
