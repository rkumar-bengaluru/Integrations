package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/integrations/slack"
	"agent.fabric.com/modules/internal/logger"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/repository/db"
	"agent.fabric.com/modules/internal/repository/impl"
	"agent.fabric.com/modules/internal/service"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var (
	clientID     string
	clientSecret string
	redirectURI  string
	tenantID     string
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
	tenantID = os.Getenv("SYMPHONY_TENANT_ID")
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
		CreateNewBinding(ctx, tenantUID, config, integration, encryptionSvc,
			credentialRepo, bindingRepo, bindingSvc, slackHandler, logger)
		logger.Error("error fetching binding credential", zap.Error(err))
		os.Exit(1)
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
		actionMenu := buildActionMenu(integration.Actions)

		fmt.Printf("╔════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  %s Test Menu%-*s║\n", integration.Name,
			35-len(integration.Name), "")
		fmt.Printf("╠════════════════════════════════════════════════════════════════════╣\n")

		for _, item := range actionMenu {
			fmt.Printf("║  %s\n", item.display)
		}
		fmt.Printf("║  %2d. 🚪 Exit\n", len(actionMenu)+1)
		fmt.Printf("╚════════════════════════════════════════════════════════════════════╝\n")
		reader := bufio.NewReader(os.Stdin)
		choice := getInput(reader, "\n👉 Select action: ")
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
		selectedAction := actionMenu[selection-1].action

		executeActionFlow(ctx, config, reader, slackHandler, binding, &selectedAction)

		fmt.Println("\n" + strings.Repeat("─", 70))
	}
}

func CreateNewBinding(
	ctx context.Context,
	tenantUID uuid.UUID,
	config *models.ExecutionConfig,
	integration *models.Integration,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	bindingRepo repository.IntegrationBindingRepository,
	bindingSvc service.IntegrationBindingService,
	slackHhandler handler.IntegrationHandler,
	logger *zap.Logger) error {
	// select the correct platform credential by cred type
	platformCredential, err := slack.FindPlatformCredential(config.CredentialType, integration)

	if err != nil {
		return fmt.Errorf("error fetching platformCredential", zap.Error(err))
	}

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
	binding := handler.NewBindingHandler(slackHhandler, bindingRepo, logger)

	ibinding, err := binding.CreateIntegrationBindingWithOwner(integration, config, params, tenantUID, nil, encryptionSvc)

	if err != nil {
		return fmt.Errorf("failed to create integration binding", zap.Error(err))
	}

	fmt.Println(fmt.Sprintf("create integration binding %s successfully", ibinding.Status))
	return nil
}

func buildActionMenu(actions []models.ActionDefinition) []ActionMenuItem {
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
			action:  action,
			display: display,
			number:  i + 1,
		})
	}

	return menu
}

// ActionMenuItem represents a menu item for an action
type ActionMenuItem struct {
	action  models.ActionDefinition
	display string
	number  int
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

func getInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func executeActionFlow(ctx context.Context,
	config *models.ExecutionConfig, reader *bufio.Reader, handler handler.IntegrationHandler,
	binding *models.IntegrationBinding, action *models.ActionDefinition) {

	fmt.Printf("\n%s %s\n", getActionEmoji(string(action.Type), action.Name), action.Name)
	fmt.Println(strings.Repeat("─", 50))

	if action.Description != "" {
		fmt.Printf("📝 %s\n\n", action.Description)
	}

	// Collect inputs based on action.InputSchema
	//inputs := collectInputsFromSchema(reader, action.InputSchema, binding)
	// Collect inputs - NEW FUNCTION USED HERE
	inputs, userProvided := collectActionInputs(reader, action, binding)

	if userProvided {
		fmt.Println() // Extra line if we asked for input
	}

	// Execute
	fmt.Println("\n🚀 Executing action...")
	start := time.Now()

	result, err := handler.Execute(ctx, config, action, *binding, inputs)
	if err != nil {
		fmt.Printf("❌ Execution failed: %v\n", err)
		return
	}

	if result.Error != nil {
		fmt.Printf("❌ Execution failed: %v\n", result.Error)
		return
	}

	duration := time.Since(start)
	fmt.Printf("✅ Success! (took %v)\n", duration)

	// Display results
	if result.Data != nil {
		fmt.Println("\n📊 Result:")
		data, _ := json.MarshalIndent(result.Data, "", "  ")
		fmt.Println(string(data))
	}

}

// collectActionInputs collects inputs for action execution
// Returns true if any input was collected from user, false if all auto-resolved
// collectActionInputs - REPLACES collectInputsFromSchema
// Returns inputs map and bool indicating if user provided any input
func collectActionInputs(reader *bufio.Reader, action *models.ActionDefinition,
	binding *models.IntegrationBinding) (map[string]interface{}, bool) {

	inputs := make(map[string]interface{})
	userProvided := false

	fmt.Printf("\n🔍 DEBUG: Action = %s\n", action.Name)
	fmt.Printf("🔍 DEBUG: InputSchema = %+v\n\n", action.InputSchema)

	if action.InputSchema == nil {
		fmt.Println("🔍 DEBUG: InputSchema is nil")
		return inputs, userProvided
	}

	schemaBytes, _ := json.Marshal(action.InputSchema)
	var schemaObj map[string]interface{}
	json.Unmarshal(schemaBytes, &schemaObj)

	fmt.Printf("🔍 DEBUG: Parsed schemaObj = %+v\n\n", schemaObj)

	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		fmt.Println("🔍 DEBUG: No properties found in schema")
		return inputs, userProvided
	}

	fmt.Printf("🔍 DEBUG: Found %d properties\n\n", len(properties))

	required := []string{}
	if req, ok := schemaObj["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
	}
	fmt.Printf("🔍 DEBUG: Required fields = %v\n\n", required)

	// Print all properties with their attributes
	fmt.Println("🔍 DEBUG: Property analysis:")
	for propName, propDef := range properties {
		propMap, ok := propDef.(map[string]interface{})
		if !ok {
			fmt.Printf("  %s: ERROR - not a map\n", propName)
			continue
		}

		source := getString(propMap, "source")
		defaultVal := propMap["default"]
		propType := getString(propMap, "type", "string")
		isRequired := isStringInSlice(propName, required)

		fmt.Printf("  %s:\n", propName)
		fmt.Printf("    - type: %s\n", propType)
		fmt.Printf("    - source: %s\n", source)
		fmt.Printf("    - default: %v\n", defaultVal)
		fmt.Printf("    - required: %t\n", isRequired)
	}
	fmt.Println()

	// Check if anything actually needs user input
	needsUserInput := false
	for propName, propDef := range properties {
		propMap, ok := propDef.(map[string]interface{})
		if !ok {
			continue
		}

		source := getString(propMap, "source")
		defaultVal := propMap["default"]
		isRequired := isStringInSlice(propName, required)

		fmt.Printf("🔍 DEBUG: Checking %s - source=%s, default=%v, required=%t\n",
			propName, source, defaultVal, isRequired)

		// Needs user input if:
		// 1. Source is user_input or file_upload
		// 2. Is required AND no default AND no auto-resolve source
		if source == "user_input" || source == "file_upload" {
			fmt.Printf("  -> NEEDS USER INPUT (source=%s)\n", source)
			needsUserInput = true
			break
		}

		if isRequired && defaultVal == nil && !strings.HasPrefix(source, "$.") {
			fmt.Printf("  -> NEEDS USER INPUT (required, no default, not auto-resolve)\n")
			needsUserInput = true
			break
		}

		fmt.Printf("  -> AUTO-RESOLVE\n")
	}

	fmt.Printf("\n🔍 DEBUG: needsUserInput = %t\n\n", needsUserInput)

	// Auto-resolve everything - no user input needed
	if !needsUserInput {
		fmt.Println("🔍 DEBUG: Auto-resolving all parameters...")

		for propName, propDef := range properties {
			propMap, ok := propDef.(map[string]interface{})
			if !ok {
				continue
			}

			defaultVal := propMap["default"]
			source := getString(propMap, "source")

			if defaultVal != nil {
				inputs[propName] = defaultVal
				fmt.Printf("  %s = %v (default)\n", propName, defaultVal)
			} else {
				fmt.Printf("  %s = <will be resolved from %s>\n", propName, source)
			}
		}

		fmt.Println("\n✓ All parameters auto-resolved (using defaults and binding credentials)")

		// DEBUG: Print final inputs map
		fmt.Printf("\n🔍 DEBUG: Final inputs map = %+v\n", inputs)

		return inputs, userProvided
	}

	// Need some user input - show prompt
	fmt.Println("📝 Action inputs:")

	for propName, propDef := range properties {
		propMap, ok := propDef.(map[string]interface{})
		if !ok {
			continue
		}

		description := getString(propMap, "description")
		propType := getString(propMap, "type", "string")
		defaultVal := propMap["default"]
		source := getString(propMap, "source")
		isRequired := isStringInSlice(propName, required)

		// Skip auto-resolved fields
		if strings.HasPrefix(source, "$.bindings.credential.") ||
			strings.HasPrefix(source, "$.platformcredential.") {
			fmt.Printf("  • %s: <will resolve from %s>\n", propName, source)
			continue
		}

		// Use default silently if available and not required
		if !isRequired && defaultVal != nil && source != "user_input" && source != "file_upload" {
			inputs[propName] = defaultVal
			fmt.Printf("  • %s: %v (default)\n", propName, defaultVal)
			continue
		}

		// Need user input
		userProvided = true

		prompt := fmt.Sprintf("  • %s", propName)
		if description != "" {
			prompt += fmt.Sprintf(" (%s)", truncate(description, 40))
		}
		if isRequired {
			prompt += " [required]"
		}
		if defaultVal != nil {
			prompt += fmt.Sprintf(" (default: %v)", defaultVal)
		}
		prompt += ": "

		for {
			fmt.Print(prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				if isRequired && defaultVal == nil {
					fmt.Println("    ⚠️  Required field")
					continue
				}
				if defaultVal != nil {
					inputs[propName] = defaultVal
				}
				break
			}

			value := convertType(input, propType)
			if value == nil {
				fmt.Printf("    ⚠️  Invalid format\n")
				continue
			}

			// Validate enum
			if enumVals, ok := propMap["enum"].([]interface{}); ok {
				valid := false
				for _, ev := range enumVals {
					if fmt.Sprintf("%v", ev) == fmt.Sprintf("%v", value) {
						valid = true
						break
					}
				}
				if !valid {
					fmt.Printf("    ⚠️  Must be one of: %v\n", enumVals)
					continue
				}
			}

			inputs[propName] = value
			break
		}
	}

	// DEBUG: Print final inputs map
	fmt.Printf("\n🔍 DEBUG: Final inputs map = %+v\n", inputs)
	fmt.Printf("🔍 DEBUG: userProvided = %t\n", userProvided)

	return inputs, userProvided
}

// convertType converts a string input to the appropriate type based on targetType
func convertType(input string, targetType string) interface{} {
	switch targetType {
	case "integer":
		val, err := strconv.Atoi(input)
		if err != nil {
			return nil
		}
		return val
	case "number":
		val, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil
		}
		return val
	case "boolean":
		val, err := strconv.ParseBool(input)
		if err != nil {
			return nil
		}
		return val
	case "array":
		// Simple comma-separated values
		if input == "" {
			return []string{}
		}
		return strings.Split(input, ",")
	case "object":
		// Try to parse as JSON
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(input), &obj); err != nil {
			return nil
		}
		return obj
	default:
		// string and any other type
		return input
	}
}

func getString(m map[string]interface{}, key string, defaultVal ...string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func isStringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
