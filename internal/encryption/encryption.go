// service/encryption_service.go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type EncryptionService interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	GenerateDataHash(data []byte) string
	EncryptSecrets(secrets interface{}) ([]byte, error)
}

type encryptionService struct {
	key []byte
}

func NewEncryptionService(masterKey string) EncryptionService {
	// Derive 32-byte key from master key
	hash := sha256.Sum256([]byte(masterKey))
	return &encryptionService{key: hash[:]}
}

func (s *encryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *encryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *encryptionService) GenerateDataHash(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (s *encryptionService) EncryptSecrets(secrets interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets: %w", err)
	}
	return s.Encrypt(jsonData)
}
