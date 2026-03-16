package handler

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func CreateNewBinding(
	ctx context.Context,
	tenantUID uuid.UUID,
	config *models.ExecutionConfig,
	integration *models.Integration,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	bindingRepo repository.IntegrationBindingRepository,
	bindingSvc service.IntegrationBindingService,
	slackHhandler IntegrationHandler,
	logger *zap.Logger) error {
	// select the correct platform credential by cred type
	logger.Debug("find platform credential")
	platformCredential, err := FindPlatformCredential(config.CredentialType, integration)

	if err != nil {
		return fmt.Errorf("error fetching platformCredential", zap.Error(err))
	}
	logger.Debug("found platform credential")
	// Validate SecretMapping against ParamSchema
	validation := commons.ValidateSecretMapping(config.CredentialBinding.SecretMapping, config.ParamInputSchema)
	if !validation.Valid {
		fmt.Println("❌ Validation failed between SecretMapping and ParamSchema:")
		for _, err := range validation.Errors {
			fmt.Printf("   • %s\n", err)
		}
		return fmt.Errorf("❌ Validation failed between SecretMapping and ParamSchema:", zap.Error(err))
	}

	fmt.Println("🔧 Collecting required parameters...")
	fmt.Printf("   SecretMapping defines %d parameters\n", len(config.CredentialBinding.SecretMapping))
	fmt.Printf("   ParamSchema validates %d properties\n\n", len(config.ParamInputSchema.Properties))

	// Interactive selection
	reader := bufio.NewReader(os.Stdin)
	// Initialize collector and collect parameters
	collector := commons.NewBindingCollector(reader, encryptionSvc, credentialRepo, logger)
	params := collector.CollectParameters(ctx, platformCredential, integration, config.CredentialBinding, config.ParamInputSchema)

	// Display results
	commons.DisplayCollectedParameters(integration.Name, params)

	// ============================================
	// CREATE INTEGRATION BINDING
	// ============================================

	fmt.Println("📝 Creating integration binding...")
	binding := NewBindingHandler(slackHhandler, bindingRepo, logger)

	ibinding, err := binding.CreateIntegrationBindingWithOwner(integration, config, params, tenantUID, nil, encryptionSvc)

	if err != nil {
		return fmt.Errorf("failed to create integration binding", zap.Error(err))
	}

	fmt.Println(fmt.Sprintf("create integration binding %s successfully", ibinding.Status))
	return nil
}

func FindPlatformCredential(
	credType models.CredentialType,
	integration *models.Integration,
) (*models.PlatformCredential, error) {
	if integration == nil {
		return nil, fmt.Errorf("integration is nil")
	}

	for _, pc := range integration.PlatformCredentials {
		fmt.Println(fmt.Sprintf("available credential type %s", pc.CredentialType))
		fmt.Println(fmt.Sprintf("crdential name %s", pc.Credential.Name))
		if pc.CredentialType == credType {
			return pc, nil
		}
	}

	return nil, fmt.Errorf("no platform credential found for type %s", credType)
}

func BuildActionMenu(actions []models.ActionDefinition) []ActionMenuItem {
	var menu []ActionMenuItem
	fmt.Printf("length of actions %d\n", len(actions))
	for i, action := range actions {
		fmt.Println(fmt.Sprintf("action action %s is active %s", action.Name, action.IsActive))
		if !action.IsActive {
			continue // Skip inactive actions
		}

		// Build emoji based on action category/name
		emoji := getActionEmoji(string(action.Type), action.Name)

		// Format display string
		name := truncate(action.Name, 28)
		category := truncate(string(action.Type), 15)

		display := fmt.Sprintf("%2d. %s %-28s [%s]", i+1, emoji, name, category)

		menu = append(menu, ActionMenuItem{
			Action:  action,
			Display: display,
			Number:  i + 1,
		})
	}

	return menu
}

func getActionEmoji(category, name string) string {
	catLower := strings.ToLower(category)
	nameLower := strings.ToLower(name)

	// Category-based emojis
	switch {
	case strings.Contains(catLower, "email") || strings.Contains(catLower, "gmail"):
		return "📧"
	case strings.Contains(catLower, "developer") || strings.Contains(catLower, "github"):
		return "🐙"
	case strings.Contains(catLower, "crm"):
		return "👥"
	case strings.Contains(catLower, "storage"):
		return "📦"
	case strings.Contains(catLower, "calendar"):
		return "📅"
	case strings.Contains(catLower, "chat"):
		return "💬"
	default:
		// Name-based fallback
		switch {
		case strings.Contains(nameLower, "list") || strings.Contains(nameLower, "get"):
			return "📋"
		case strings.Contains(nameLower, "create") || strings.Contains(nameLower, "add"):
			return "➕"
		case strings.Contains(nameLower, "update") || strings.Contains(nameLower, "edit"):
			return "✏️"
		case strings.Contains(nameLower, "delete") || strings.Contains(nameLower, "remove"):
			return "🗑️"
		case strings.Contains(nameLower, "send"):
			return "📤"
		case strings.Contains(nameLower, "search"):
			return "🔍"
		default:
			return "⚡"
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func GetInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
