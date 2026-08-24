package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Secrets Management - HashiCorp Vault, AWS KMS, GCP KMS

type Secret struct {
	Name      string
	Encrypted []byte
	Version   int
	CreatedAt int64
}

type SecretsManager struct {
	mu        sync.RWMutex
	secrets   map[string]Secret
	masterKey []byte
}

func NewSecretsManager(masterKey string) (*SecretsManager, error) {
	key, err := hex.DecodeString(masterKey)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("master key must be 32 bytes")
	}

	return &SecretsManager{
		secrets:   make(map[string]Secret),
		masterKey: key,
	}, nil
}

func (s *SecretsManager) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *SecretsManager) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *SecretsManager) SetSecret(name string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	encrypted, err := s.encrypt([]byte(value))
	if err != nil {
		return err
	}

	existing, ok := s.secrets[name]
	version := 1
	if ok {
		version = existing.Version + 1
	}

	s.secrets[name] = Secret{
		Name:      name,
		Encrypted: encrypted,
		Version:   version,
		CreatedAt: now(),
	}

	return nil
}

func (s *SecretsManager) GetSecret(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, ok := s.secrets[name]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", name)
	}

	decrypted, err := s.decrypt(secret.Encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

func (s *SecretsManager) DeleteSecret(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.secrets, name)
	return nil
}

func (s *SecretsManager) ListSecrets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.secrets))
	for name := range s.secrets {
		names = append(names, name)
	}
	return names
}

func now() int64 {
	return time.Now().Unix()
}

func main() {
	// Fail-closed: the 32-byte hex master key must come from the environment
	// (provisioned via the admin/super-admin dashboard secrets manager),
	// never from a hardcoded literal.
	masterKeyHex := strings.TrimSpace(os.Getenv("SECRETS_MASTER_KEY"))
	if masterKeyHex == "" {
		fmt.Fprintln(os.Stderr, "SECRETS_MASTER_KEY environment variable is required")
		os.Exit(1)
	}

	manager, err := NewSecretsManager(masterKeyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize secrets manager: %v\n", err)
		os.Exit(1)
	}

	names := manager.ListSecrets()
	fmt.Printf("Secrets manager ready - %d secret(s) loaded\n", len(names))
}
