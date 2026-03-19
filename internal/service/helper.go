package service

import (
	"fmt"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
)

// ToCredentialTypeEntity converts a DTO string into a model CredentialType
func ToCredentialTypeEntity(s string) (models.CredentialType, error) {
	switch s {
	case string(models.AuthOAuth2):
		return models.AuthOAuth2, nil
	case string(models.AuthOAuth2PKCE):
		return models.AuthOAuth2PKCE, nil
	case string(models.AuthOAuth2Device):
		return models.AuthOAuth2Device, nil
	case string(models.AuthAPIKey):
		return models.AuthAPIKey, nil
	case string(models.AuthBearerToken):
		return models.AuthBearerToken, nil
	case string(models.AuthBasicAuth):
		return models.AuthBasicAuth, nil
	case string(models.AuthMTLS):
		return models.AuthMTLS, nil
	case string(models.AuthMCPSession):
		return models.AuthMCPSession, nil
	case string(models.AuthAWSSigV4):
		return models.AuthAWSSigV4, nil
	case string(models.AuthAzureManaged):
		return models.AuthAzureManaged, nil
	case string(models.AuthGCPServiceAcc):
		return models.AuthGCPServiceAcc, nil
	case string(models.AuthJWT):
		return models.AuthJWT, nil
	case string(models.AuthSAML):
		return models.AuthSAML, nil
	case string(models.AuthDatabase):
		return models.AuthDatabase, nil
	case string(models.AuthSSHKey):
		return models.AuthSSHKey, nil
	default:
		return "", fmt.Errorf("invalid credential type: %s", s)
	}
}
