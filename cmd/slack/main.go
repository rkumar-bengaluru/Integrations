package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rkumar-bengaluru/Integrations/internal/encryption"
	"github.com/rkumar-bengaluru/Integrations/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/slack"
	"github.com/rkumar-bengaluru/Integrations/internal/logger"
	"github.com/rkumar-bengaluru/Integrations/internal/repository"
	"github.com/rkumar-bengaluru/Integrations/internal/repository/db"
	"github.com/rkumar-bengaluru/Integrations/internal/repository/impl"
	"github.com/rkumar-bengaluru/Integrations/internal/service"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	// Load .env file into environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: Error loading .env file:", err)
		panic(err)
	}

	// Initialize services
	keyID := os.Getenv("ENCRYPTION_KEY_ID")
	if keyID == "" {
		keyID = "default-key-v1"
	}

	// Tenant ID
	tenantID := os.Getenv("SYMPHONY_TENANT_ID")
	if tenantID == "" {
		tenantID = "default-key-v1"
	}

	// Tenant and Integration IDs for test
	tenantUID := uuid.MustParse(tenantID)

	// logger
	logger := logger.Get(ctx)

	encryptionSvc := encryption.NewEncryptionService(keyID)

	conn, dialect := db.CreateDB(ctx, "genei-server")
	defer conn.Close()
	_, database := repository.NewSQLStore(dialect, logger)

	bindingRepo := impl.NewIntegrationBindingRepository(database, logger)
	credentialRepo := impl.NewCredentialRepository(database, logger)

	// create slack integration if not exist.
	integration, err := slack.CreateSlackIntegration(ctx, database, logger, encryptionSvc, credentialRepo, tenantUID)

	if err != nil {
		logger.Error("error fetching integration", zap.Error(err))
	}

	// Initialize repositories
	// slackHandler := slack.NewSlackHandler(encryptionSvc, bindingSvc, logger)

	// Show options to user
	config := commons.DisplaySelectedIntegration(integration)

	// Check if all the credential type bindings are already created.
	// if yes go for actions execution.
	binding, err := bindingRepo.FindIntegrationBinding(ctx, config.CredentialType, integration.ID)

	// handler for slack
	validator := service.NewCredentialValidator()
	bindingSvc := service.NewIntegrationBindingService(bindingRepo, credentialRepo, encryptionSvc, &validator)
	slackHandler := slack.NewSlackHandler(encryptionSvc, bindingSvc, logger)

	if err != nil {
		// try creating a new one.
		handler.CreateNewBinding(ctx, tenantUID, config, integration, encryptionSvc,
			credentialRepo, bindingRepo, bindingSvc, slackHandler, logger)
		logger.Fatal("error fetching binding credential", zap.Error(err))
	}
	fmt.Println(fmt.Sprintf("found the binding for action execution %s", binding.Credential.Name))

	// Test connection
	fmt.Printf("\n🔌 Testing connection to %s...\n", integration.Name)

	if err := slackHandler.TestConnection(ctx, config, *binding); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Connection test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Connection successful!\n")

	// List All actions to test.
	// ============================================
	// DYNAMIC ACTION MENU
	// ============================================

	for {
		// Build dynamic menu from ActionDefinitions
		actionMenu := handler.BuildActionMenu(integration.Actions)

		fmt.Printf("╔════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  %s Test Menu%-*s║\n", integration.Name,
			35-len(integration.Name), "")
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

		// Execute selected action
		selectedAction := actionMenu[selection-1].Action

		handler.ExecuteActionFlow(ctx, config, reader, slackHandler, binding, &selectedAction)

		fmt.Println("\n" + strings.Repeat("─", 70))
	}
}
