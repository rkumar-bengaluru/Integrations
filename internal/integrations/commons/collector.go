package commons

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BindingCollector handles parameter collection for integration bindings
type BindingCollector struct {
	reader         *bufio.Reader
	encryptionSvc  encryption.EncryptionService
	credentialRepo repository.CredentialRepository
	logger         *zap.Logger
}

// NewBindingCollector creates a new binding collector
func NewBindingCollector(
	reader *bufio.Reader,
	encryptionSvc encryption.EncryptionService,
	credentialRepo repository.CredentialRepository,
	logger *zap.Logger,
) *BindingCollector {
	return &BindingCollector{
		reader:         reader,
		encryptionSvc:  encryptionSvc,
		credentialRepo: credentialRepo,
		logger:         logger,
	}
}

func DisplaySelectedIntegration(integration *models.Integration) *models.ExecutionConfig {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ✅ INTEGRATION SELECTED                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("  📛 Name:        %s\n", integration.Name)
	fmt.Printf("  📂 Category:    %s\n", integration.Category)
	fmt.Printf("  📝 Description: %s\n", integration.Description)
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for i, t := range integration.SupportedCredentialTypes {
		fmt.Printf("  📝 Supported Credential Type: %d: %s\n", i+1, t)
	}
	// Interactive selection
	reader := bufio.NewReader(os.Stdin)
	selected := SelectIntegrationBinding(reader, integration)
	fmt.Println(fmt.Sprintf("selected config %s ", selected.CredentialType))
	return selected
}

// ============================================
// SELECTION & VALIDATION FUNCTIONS
// ============================================
func SelectIntegrationBinding(reader *bufio.Reader, integration *models.Integration) *models.ExecutionConfig {
	for {
		fmt.Printf("🔍 Enter the number of the binding (1-%d), or 'q' to quit: ", len(integration.SupportedCredentialTypes))
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "q" {
			fmt.Println("👋 Goodbye!")
			os.Exit(0)
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(integration.SupportedCredentialTypes) {
			fmt.Printf("⚠️ Invalid selection. Please enter a number between 1 and %d.\n\n", len(integration.SupportedCredentialTypes))
			continue
		}

		selected := integration.SupportedCredentialTypes[choice-1]

		fmt.Printf("Selected: %s\n", selected)

		// Look up the config directly — no loop needed if CredentialType is map key
		if config, ok := integration.ExecutionConfigs[selected]; ok {
			fmt.Printf("Found matching config for %s\n", selected)
			return &config
		}

		// If not found (should rarely happen if SupportedCredentialTypes is consistent)
		fmt.Printf("⚠️ No execution config found for credential type: %s\n", selected)
		// You could continue the outer loop here to let user try again
		// or return nil / error depending on your needs
	}
}

// ValidationResult holds the validation outcome between SecretMapping and ParamSchema
type ValidationResult struct {
	Valid            bool
	MissingInSchema  []string // Parameters in SecretMapping but not in ParamSchema
	MissingInMapping []string // Required parameters in ParamSchema but not in SecretMapping
	Errors           []string
}

func ValidateSecretMapping(secretMapping map[string]interface{}, paramSchema *models.JSONSchema) ValidationResult {
	result := ValidationResult{
		Valid:            true,
		MissingInSchema:  []string{},
		MissingInMapping: []string{},
		Errors:           []string{},
	}

	for storageKey, paramNameValue := range secretMapping {
		paramName := fmt.Sprintf("%v", paramNameValue)

		if _, exists := paramSchema.Properties[paramName]; !exists {
			result.MissingInSchema = append(result.MissingInSchema, paramName)
			result.Errors = append(result.Errors,
				fmt.Sprintf("SecretMapping references '%s' (storage: %s) but not found in ParamSchema",
					paramName, storageKey))
			result.Valid = false
		}
	}

	requiredParams := make(map[string]bool)
	for _, req := range paramSchema.Required {
		requiredParams[req] = true
	}

	mappedParams := make(map[string]bool)
	for _, paramNameValue := range secretMapping {
		paramName := fmt.Sprintf("%v", paramNameValue)
		mappedParams[paramName] = true
	}

	for reqParam := range requiredParams {
		if !mappedParams[reqParam] {
			result.MissingInMapping = append(result.MissingInMapping, reqParam)
			result.Errors = append(result.Errors,
				fmt.Sprintf("Required parameter '%s' in ParamSchema but not mapped in SecretMapping",
					reqParam))
			result.Valid = false
		}
	}

	return result
}

// CollectedParam represents a single collected parameter
type CollectedParam struct {
	StorageKey  string      `json:"storage_key"`
	ParamName   string      `json:"param_name"`
	Value       interface{} `json:"value"`
	IsSecret    bool        `json:"is_secret"`
	Source      string      `json:"source"`
	Description string      `json:"description"`
}

// ============================================
// COLLECTION FUNCTIONS
// ============================================

func (bc *BindingCollector) CollectParameters(
	ctx context.Context,
	platformCredential *models.PlatformCredential,
	integration *models.Integration,
	credentialBinding *models.CredentialBinding,
	paramSchema *models.JSONSchema,
) []CollectedParam {
	var collected []CollectedParam

	for storageKey, paramNameValue := range credentialBinding.SecretMapping {
		paramName := fmt.Sprintf("%v", paramNameValue)
		property := paramSchema.Properties[paramName]

		isRequired := IsParamRequired(paramName, paramSchema.Required)

		fmt.Printf("📌 Collecting: %s (stored as: %s)\n", paramName, storageKey)
		if property.Description != "" {
			fmt.Printf("   Description: %s\n", property.Description)
		}
		fmt.Printf("   Type: %s | Required: %t | Secret: %t | Source: %s\n",
			property.Type, isRequired, property.Secret, property.Source)

		value := bc.collectValue(ctx, platformCredential, integration, property, isRequired)

		if value != nil {
			collected = append(collected, CollectedParam{
				StorageKey:  paramName,
				ParamName:   paramName,
				Value:       value,
				IsSecret:    property.Secret,
				Source:      property.Source,
				Description: property.Description,
			})
		}
		fmt.Println()
	}

	return collected
}

func (bc *BindingCollector) collectValue(
	ctx context.Context,
	platformCredential *models.PlatformCredential,
	integration *models.Integration,
	property models.JSONSchemaProperty,
	isRequired bool,
) interface{} {
	switch {
	case strings.HasPrefix(property.Source, "$.platformcredential"):
		return bc.collectFromCredential(ctx, platformCredential, integration, property, isRequired)
	case property.Source == "file_upload":
		return bc.collectFromFile(property, isRequired)
	case property.Source == "user_input":
		return bc.CollectFromUser(property, isRequired)
	default:
		return bc.collectUnknownSource(property, isRequired)
	}
}

func (bc *BindingCollector) collectFromCredential(
	ctx context.Context,
	platformCredential *models.PlatformCredential,
	integration *models.Integration,
	property models.JSONSchemaProperty,
	isRequired bool,
) interface{} {
	value, err := ExtractFromPlatformCredential(ctx, platformCredential, bc.encryptionSvc, integration, property.Source, bc.logger)
	if err != nil {
		if isRequired {
			fmt.Printf("   ❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   ⚠️  Warning: %v (optional, skipping)\n", err)
		return nil
	}
	fmt.Printf("   ✅ Extracted: %v\n", maskIfSecret(fmt.Sprintf("%v", value), property.Secret))
	return value
}

func (bc *BindingCollector) collectFromFile(property models.JSONSchemaProperty, isRequired bool) interface{} {
	for {
		fmt.Printf("   📁 File path")
		if property.FileType != "" {
			fmt.Printf(" (%s)", property.FileType)
		}
		fmt.Print(": ")

		filePath, err := bc.reader.ReadString('\n')
		if err != nil {
			fmt.Println("   ❌ Error reading input, try again")
			continue
		}
		filePath = strings.TrimSpace(filePath)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("   ⚠️  File not found: %s\n", filePath)
			if isRequired {
				continue
			}
			return nil
		}

		if property.Secret {
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("   ❌ Error reading: %v\n", err)
				if isRequired {
					continue
				}
				return nil
			}
			fmt.Printf("   ✅ Loaded (%d bytes)\n", len(content))
			return string(content)
		}

		fmt.Printf("   ✅ Recorded: %s\n", filepath.Base(filePath))
		return filePath
	}
}

func (bc *BindingCollector) CollectFromUser(property models.JSONSchemaProperty, isRequired bool) interface{} {
	for {
		prompt := "   ⌨️  Value"
		if isRequired {
			prompt += " (required)"
		} else {
			prompt += " (optional)"
		}
		fmt.Printf("%s: ", prompt)

		input, err := bc.reader.ReadString('\n')
		if err != nil {
			fmt.Println("   ❌ Error reading, try again")
			continue
		}
		input = strings.TrimSpace(input)

		if input == "" {
			if isRequired {
				fmt.Println("   ⚠️  Required field")
				continue
			}
			fmt.Println("   ⏭️  Skipped")
			return nil
		}

		value := ConvertType(input, property.Type)
		if value == nil {
			fmt.Println("   ⚠️  Invalid format, try again")
			continue
		}

		fmt.Printf("   ✅ Recorded: %v\n", maskIfSecret(fmt.Sprintf("%v", value), property.Secret))
		return value
	}
}

func (bc *BindingCollector) collectUnknownSource(property models.JSONSchemaProperty, isRequired bool) interface{} {
	fmt.Printf("   ⚠️  Unknown source: %s\n", property.Source)
	if !isRequired {
		return nil
	}

	fmt.Print("   Manual input: ")
	input, _ := bc.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// ============================================
// UTILITY FUNCTIONS
// ============================================

func IsParamRequired(paramName string, required []string) bool {
	for _, req := range required {
		if req == paramName {
			return true
		}
	}
	return false
}

func ConvertType(input string, targetType string) interface{} {
	switch targetType {
	case "integer":
		val, err := strconv.Atoi(input)
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
		return strings.Split(input, ",")
	default:
		return input
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func ExtractFromPlatformCredential(
	ctx context.Context,
	platformCredential *models.PlatformCredential,
	encryptionSvc encryption.EncryptionService,
	integration *models.Integration,
	source string,
	logger *zap.Logger,
) (interface{}, error) {

	if platformCredential.ID == uuid.Nil || platformCredential == nil {
		return nil, fmt.Errorf("no platform credential configured")
	}

	credential := platformCredential.Credential

	if credential == nil {
		return nil, fmt.Errorf("platform.credential is nil:")
	}

	decryptedData, err := encryptionSvc.Decrypt(credential.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	field := strings.TrimPrefix(source, "$.platformcredential.")
	value, ok := data[field]
	if !ok {
		return nil, fmt.Errorf("field '%s' not found", field)
	}

	return value, nil
}

func maskIfSecret(value string, isSecret bool) string {
	if !isSecret {
		return value
	}
	return maskString(value)
}

func maskString(s string) string {
	if len(s) <= 6 {
		return "****"
	}
	return s[:3] + "****" + s[len(s)-3:]
}

func DisplayCollectedParameters(integrationName string, params []CollectedParam) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          📋 COLLECTED BINDING DATA                         ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("  Integration: %s\n", integrationName)
	fmt.Printf("  Total Parameters: %d\n\n", len(params))

	finalMap := make(map[string]interface{})
	for _, param := range params {
		finalMap[param.StorageKey] = param.Value

		valueStr := fmt.Sprintf("%v", param.Value)
		if param.IsSecret && param.Value != nil {
			valueStr = maskString(valueStr)
		}
		fmt.Printf("  🔑 %-25s → %-20s: %s\n", param.StorageKey, param.ParamName, valueStr)
	}

	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("🔍 Final Data Map (JSON format):")
	jsonData, _ := json.MarshalIndent(finalMap, "", "  ")
	fmt.Println(string(jsonData))
	fmt.Println()
}
