package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/integrations/snow" // swap with slack, jira, etc.
	"agent.fabric.com/modules/internal/logger"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/repository/db"
	"agent.fabric.com/modules/internal/repository/impl"
	"agent.fabric.com/modules/internal/service"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Bootstrap sets up common services and repositories
func Bootstrap(ctx context.Context) (
	encryptionSvc encryption.EncryptionService,
	bindingRepo repository.IntegrationBindingRepository,
	credentialRepo repository.CredentialRepository,
	bindingSvc service.IntegrationBindingService,
	logInstance *zap.Logger,
	tenantUID uuid.UUID,
	database *gorm.DB,
) {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: Error loading .env file:", err)
	}

	// Tenant ID
	tenantID := os.Getenv("SYMPHONY_TENANT_ID")
	if tenantID == "" {
		tenantID = "default-key-v1"
	}
	tenantUID = uuid.MustParse(tenantID)

	// Encryption
	keyID := os.Getenv("ENCRYPTION_KEY_ID")
	if keyID == "" {
		keyID = "default-key-v1"
	}
	encryptionSvc = encryption.NewEncryptionService(keyID)

	// Logger
	logInstance = logger.Get(ctx)

	// DB
	conn, dialect := db.CreateDB(ctx, "genei-server")
	defer conn.Close()
	_, database = repository.NewSQLStore(dialect, logInstance)

	// Repos
	bindingRepo = impl.NewIntegrationBindingRepository(database, logInstance)
	credentialRepo = impl.NewCredentialRepository(database, logInstance)

	// Binding service
	validator := service.NewCredentialValidator()
	bindingSvc = service.NewIntegrationBindingService(bindingRepo, credentialRepo, encryptionSvc, &validator)

	return encryptionSvc, bindingRepo, credentialRepo, bindingSvc, logInstance, tenantUID, database
}

func main() {
	ctx := context.Background()

	// Bootstrap common services
	encryptionSvc, bindingRepo, credentialRepo, bindingSvc, logger, tenantUID, database := Bootstrap(ctx)

	// Integration-specific setup (swap snow with slack/jira/etc.)
	integration, err := snow.CreateSnowIntegration(ctx, database, logger, encryptionSvc, credentialRepo, tenantUID)
	if err != nil {
		logger.Fatal("error creating integration", zap.Error(err))
	}
	handlerInstance := snow.NewSnowHandler(encryptionSvc, bindingSvc, logger)

	// Display integration config
	config := commons.DisplaySelectedIntegration(integration)

	// Find binding
	binding, err := bindingRepo.FindIntegrationBinding(ctx, config.CredentialType, integration.ID)
	if err != nil {
		// Create new binding if not found
		handler.CreateNewBinding(ctx, tenantUID, config, integration, encryptionSvc,
			credentialRepo, bindingRepo, bindingSvc, handlerInstance, logger)
		logger.Fatal("error fetching binding credential", zap.Error(err))
	}
	fmt.Printf("found binding for action execution %s\n", binding.Credential.Name)

	// Test connection
	fmt.Printf("\n🔌 Testing connection to %s...\n", integration.Name)
	if err := handlerInstance.TestConnection(ctx, config, *binding); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Connection test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Connection successful!\n")

	// Dynamic Action Menu Loop
	for {
		actionMenu := handler.BuildActionMenu(integration.Actions)

		fmt.Printf("╔════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  %s Test Menu%-*s║\n", integration.Name, 35-len(integration.Name), "")
		fmt.Printf("╠════════════════════════════════════════════════════════════════════╣\n")

		for _, item := range actionMenu {
			fmt.Printf("║  %s\n", item.Display)
		}
		fmt.Printf("║  %2d. 🚪 Exit\n", len(actionMenu)+1)
		fmt.Printf("╚════════════════════════════════════════════════════════════════════╝\n")

		reader := bufio.NewReader(os.Stdin)
		choice := handler.GetInput(reader, "\n👉 Select action: ")
		selection, err := strconv.Atoi(strings.TrimSpace(choice))
		if err != nil || selection < 1 || selection > len(actionMenu)+1 {
			fmt.Println("❌ Invalid choice.")
			continue
		}

		if selection == len(actionMenu)+1 {
			fmt.Println("👋 Goodbye!")
			return
		}

		selectedAction := actionMenu[selection-1].Action
		handler.ExecuteActionFlow(ctx, config, reader, handlerInstance, binding, &selectedAction)

		fmt.Println("\n" + strings.Repeat("─", 70))
	}
}
