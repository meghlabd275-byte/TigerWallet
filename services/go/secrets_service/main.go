package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Secrets Management - AES-256-GCM encrypted local secret store.
//
// This is a real, self-contained secrets manager: the 32-byte master key is
// read from the SECRETS_MASTER_KEY environment variable (a hex string),
// secrets are encrypted with AES-256-GCM and persisted to an encrypted vault
// file on disk, and a small CLI (set/get/list/delete) operates on the store.
//
// For team/production deployments with rotation, audit, and HA, prefer the
// HashiCorp Vault / AWS KMS / GCP KMS integrations used elsewhere in the
// platform; this tool is for local/single-node encrypted-at-rest storage.

type Secret struct {
	Name      string `json:"name"`
	Encrypted []byte `json:"encrypted"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"created_at"`
	ID        string `json:"id"`
}

type SecretsManager struct {
	mu        sync.RWMutex
	secrets   map[string]Secret
	masterKey []byte
	storePath string
}

func NewSecretsManager(masterKeyHex, storePath string) (*SecretsManager, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid master key (expected hex): %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("master key must be 32 bytes (64 hex chars)")
	}

	sm := &SecretsManager{
		secrets:   make(map[string]Secret),
		masterKey: key,
		storePath: storePath,
	}
	if err := sm.load(); err != nil {
		return nil, err
	}
	return sm, nil
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
	id := newUUIDv4()
	if ok {
		version = existing.Version + 1
		id = existing.ID
	}

	s.secrets[name] = Secret{
		Name:      name,
		Encrypted: encrypted,
		Version:   version,
		CreatedAt: time.Now().Unix(),
		ID:        id,
	}

	return s.persist()
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

	if _, ok := s.secrets[name]; !ok {
		return fmt.Errorf("secret not found: %s", name)
	}
	delete(s.secrets, name)
	return s.persist()
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

// persist writes the encrypted vault to disk atomically. The secret values are
// already AES-256-GCM encrypted in memory, so the file is safe at rest.
func (s *SecretsManager) persist() error {
	if s.storePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.secrets, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.storePath)
}

// newUUIDv4 returns a standards-compliant RFC 4122 version-4 UUID generated
// from crypto/rand (no third-party dependency).
func newUUIDv4() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		// Fall back to a time-based deterministic id if the CSPRNG fails, which
		// is exceptionally unlikely. Correctness of the store does not depend on
		// the id being globally unique, only stable per secret across versions.
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *SecretsManager) load() error {
	if s.storePath == "" {
		return nil
	}
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.secrets)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: secrets_service <set|get|list|delete> [name] [value]")
	fmt.Fprintln(os.Stderr, "  set <name> <value>  - encrypt and store a secret")
	fmt.Fprintln(os.Stderr, "  get <name>          - decrypt and print a secret")
	fmt.Fprintln(os.Stderr, "  list                - list stored secret names")
	fmt.Fprintln(os.Stderr, "  delete <name>       - delete a stored secret")
	fmt.Fprintln(os.Stderr, "env: SECRETS_MASTER_KEY (64 hex chars), SECRETS_STORE (vault file path)")
}

func main() {
	masterKey := strings.TrimSpace(os.Getenv("SECRETS_MASTER_KEY"))
	if masterKey == "" {
		fmt.Fprintln(os.Stderr, "SECRETS_MASTER_KEY environment variable is required (64 hex chars / 32 bytes)")
		os.Exit(1)
	}
	storePath := os.Getenv("SECRETS_STORE")
	if storePath == "" {
		storePath = filepath.Join(os.Getenv("HOME"), ".tigerswap", "secrets.json")
	}

	manager, err := NewSecretsManager(masterKey, storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "set":
		if len(os.Args) != 4 {
			usage()
			os.Exit(1)
		}
		if err := manager.SetSecret(os.Args[2], os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "set failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("stored secret: %s\n", os.Args[2])
	case "get":
		if len(os.Args) != 3 {
			usage()
			os.Exit(1)
		}
		value, err := manager.GetSecret(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "get failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(value)
	case "list":
		for _, name := range manager.ListSecrets() {
			fmt.Println(name)
		}
	case "delete":
		if len(os.Args) != 3 {
			usage()
			os.Exit(1)
		}
		if err := manager.DeleteSecret(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("deleted secret: %s\n", os.Args[2])
	default:
		usage()
		os.Exit(1)
	}
}
