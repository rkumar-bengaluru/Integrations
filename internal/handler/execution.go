package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rkumar-bengaluru/Integrations/internal/models"
)

func ExecuteActionFlow(ctx context.Context,
	config *models.ExecutionConfig, reader *bufio.Reader, handler IntegrationHandler,
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
