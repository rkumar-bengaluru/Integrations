package commons

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/google/uuid"
)

func CheckIfCredentialExistByName(ctx context.Context,
	repo repository.CredentialRepository,
	tenantID uuid.UUID,
	name string) (*models.Credential, error) {

	credential, err := repo.GetByTenantIDAndName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

// ConvertCredentialsToPlatform converts []*models.Credential into []*models.PlatformCredential
func ConvertCredentialsToPlatform(
	creds []*models.Credential,
	integrationID *uuid.UUID,
) ([]*models.PlatformCredential, error) {
	platformCreds := make([]*models.PlatformCredential, 0, len(creds))

	for _, c := range creds {
		if c == nil {
			continue // skip nil entries defensively
		}

		pc := &models.PlatformCredential{
			IntegrationID:  uuid.Nil, // can be uuid.Nil if Integration not yet persisted
			CredentialID:   c.ID,     // link to the credential record
			CredentialType: c.Type,   // assuming Credential has a Type field
			Credential:     c,        // preload the actual credential object
		}

		platformCreds = append(platformCreds, pc)
	}

	return platformCreds, nil
}

// PrintCollectedParams prints the key-value pairs of a map[string]interface{} in a sorted and readable format.
// Useful for debugging or logging collected parameters.
func PrintCollectedParams(params map[string]interface{}) {
	if len(params) == 0 {
		fmt.Println("Collected parameters: (empty)")
		return
	}

	fmt.Println("Collected parameters:")

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := params[key]

		// Optional: special formatting for common types
		switch v := value.(type) {
		case string:
			fmt.Printf("  %-24s : %q\n", key, v)
		case int, int64, float64:
			fmt.Printf("  %-24s : %v\n", key, v)
		case bool:
			fmt.Printf("  %-24s : %t\n", key, v)
		case nil:
			fmt.Printf("  %-24s : <nil>\n", key)
		default:
			// For other types (structs, slices, maps, etc.) use %#v for more detail
			fmt.Printf("  %-24s : %#v\n", key, v)
		}
	}
}

// validateSchema validates that required fields in the schema are present in the data
// Works for both InputSchema and OutputSchema
func ValidateSchema(schema models.JSONMap, data map[string]interface{}, schemaType string) error {
	if schema == nil {
		return nil // no schema defined
	}

	// Get properties
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // no properties defined
	}

	// Get required fields
	requiredFields := []string{}
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredFields = append(requiredFields, s)
			}
		}
	}

	// Check required fields exist in data
	for _, key := range requiredFields {
		if _, exists := data[key]; !exists {
			return fmt.Errorf("missing required %s field: %s", schemaType, key)
		}
	}

	// Additional type validation (optional but recommended)
	for key, def := range props {
		defMap, ok := def.(map[string]interface{})
		if !ok {
			continue
		}

		// If field exists, validate type
		if value, exists := data[key]; exists && value != nil {
			expectedType, _ := defMap["type"].(string)
			if err := validateType(key, value, expectedType); err != nil {
				return fmt.Errorf("invalid type for %s field '%s': %w", schemaType, key, err)
			}
		}
	}

	return nil
}

// validateType checks if value matches expected JSON schema type
func validateType(field string, value interface{}, expectedType string) error {
	if expectedType == "" {
		return nil // no type specified
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "integer", "number":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			// ok
		case uint, uint8, uint16, uint32, uint64:
			// ok
		case float32, float64:
			// ok for number type
		default:
			return fmt.Errorf("expected %s, got %T", expectedType, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		// Check if it's a slice (but not a string which is also a slice of bytes)
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		// Check if it's a map or struct
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Map && kind != reflect.Struct {
			return fmt.Errorf("expected object, got %T", value)
		}
	default:
		// Unknown type, skip validation
		return nil
	}

	return nil
}
