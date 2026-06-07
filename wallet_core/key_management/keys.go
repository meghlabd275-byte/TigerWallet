// ============================================================================
// TIGERWALLET KEY MANAGEMENT
// Secure key storage and management with encryption
// ============================================================================

package key_management

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"
)

// Key represents a cryptographic key
type Key struct {
	ID          string    `json:"id"`
	Type        KeyType   `json:"type"`
	PublicKey   string    `json:"public_key,omitempty"`
	Encrypted  string    `json:"encrypted"`
	Algorithm  string    `json:"algorithm"`
	CreatedAt   int64     `json:"created_at"`
	ModifiedAt int64     `json:"modified_at"`
}

// KeyType represents the type of key
type KeyType string

const (
	KeyTypePrivate     KeyType = "private"
	KeyTypePublic     KeyType = "public"
	KeyTypeSymmetric  KeyType = "symmetric"
	KeyTypeMnemonic   KeyType = "mnemonic"
	KeyTypeKeystore   KeyType = "keystore"
)

// KeyStore represents a secure key store
type KeyStore struct {
	Keys       map[string]Key `json:"keys"`
	MasterKey  []byte        `json:"-"`
}

// Encrypter handles encryption/decryption
type Encrypter struct {
	Cipher   cipher.AEAD
	Nonce   []byte
}

// NewKeyStore creates a new key store
func NewKeyStore(masterPassword string) (*KeyStore, error) {
	masterKey := deriveMasterKey(masterPassword)
	return &KeyStore{
		Keys:      make(map[string]Key),
		MasterKey: masterKey,
	}, nil
}

// deriveMasterKey derives a master key from password
func deriveMasterKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// GenerateKey generates a new key
func GenerateKey(keyType KeyType) (*Key, error) {
	id := generateID()
	
	var keyData []byte
	var err error
	
	switch keyType {
	case KeyTypePrivate:
		keyData, err = generatePrivateKey()
	case KeyTypePublic:
		return nil, errors.New("public keys must be derived from private keys")
	case KeyTypeSymmetric:
		keyData = make([]byte, 32)
		rand.Read(keyData)
	case KeyTypeMnemonic:
		// Mnemonic is generated elsewhere
		return nil, errors.New("use ImportMnemonic to import a mnemonic")
	default:
		return nil, errors.New("unsupported key type")
	}
	
	if err != nil {
		return nil, err
	}
	
	return &Key{
		ID:          id,
		Type:        keyType,
		PublicKey:   "",
		Encrypted:  base64.StdEncoding.EncodeToString(keyData),
		Algorithm:  "AES-256-GCM",
		CreatedAt:   now(),
		ModifiedAt: now(),
	}, nil
}

// ImportMnemonic imports and encrypts a mnemonic
func ImportMnemonic(mnemonic string, password string) (*Key, error) {
	if len(mnemonic) < 1 {
		return nil, errors.New("mnemonic cannot be empty")
	}
	
	id := generateID()
	encrypted := encryptData([]byte(mnemonic), password)
	
	return &Key{
		ID:          id,
		Type:        KeyTypeMnemonic,
		Encrypted:  encrypted,
		Algorithm:  "AES-256-GCM",
		CreatedAt:   now(),
		ModifiedAt: now(),
	}, nil
}

// GenerateRSAKey generates an RSA key pair
func GenerateRSAKey(bits int) (*Key, *Key, error) {
	if bits < 2048 {
		bits = 2048
	}
	
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	
	pubKeyBytes := marshalPublicKey(&privateKey.PublicKey)
	privKeyBytes := marshalPrivateKey(privateKey)
	
	privateKeyEncrypted := base64.StdEncoding.EncodeToString(privKeyBytes)
	publicKeyEncrypted := base64.StdEncoding.EncodeToString(pubKeyBytes)
	
	privateKeyObj := &Key{
		ID:          generateID(),
		Type:        KeyTypePrivate,
		PublicKey:   publicKeyEncrypted,
		Encrypted:  privateKeyEncrypted,
		Algorithm:  "RSA-OAEP",
		CreatedAt:   now(),
		ModifiedAt: now(),
	}
	
	publicKeyObj := &Key{
		ID:          generateID(),
		Type:        KeyTypePublic,
		Encrypted:  publicKeyEncrypted,
		Algorithm:  "RSA-OAEP",
		CreatedAt:   now(),
		ModifiedAt: now(),
	}
	
	return privateKeyObj, publicKeyObj, nil
}

// Encrypt encrypts data with a key
func (k *Key) Encrypt(data []byte) (string, error) {
	switch k.Algorithm {
	case "AES-256-GCM":
		return encryptData(data, k.Encrypted), nil
	case "RSA-OAEP":
		return encryptRSA(data, k.Encrypted)
	default:
		return "", errors.New("unsupported algorithm")
	}
}

// Decrypt decrypts data with a key
func (k *Key) Decrypt(data string) ([]byte, error) {
	switch k.Algorithm {
	case "AES-256-GCM":
		return decryptData(data, k.Encrypted)
	case "RSA-OAEP":
		return decryptRSA(data, k.Encrypted)
	default:
		return nil, errors.New("unsupported algorithm")
	}
}

// generatePrivateKey generates a random private key
func generatePrivateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// generateID generates a unique key ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getCurrentTime returns current timestamp
func getCurrentTime() int64 {
	return time.Now().Unix()
}

// encryptData encrypts data using AES-GCM
func encryptData(data []byte, password string) string {
	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return ""
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	result := append(nonce, ciphertext...)
	
	return base64.StdEncoding.EncodeToString(result)
}

// decryptData decrypts data using AES-GCM
func decryptData(data string, password string) ([]byte, error) {
	key := sha256.Sum256([]byte(password))
	
	dataBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(dataBytes) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	
	nonce, ciphertext := dataBytes[:nonceSize], dataBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// encryptRSA encrypts data using RSA
func encryptRSA(data string, keyData string) (string, error) {
	return "", errors.New("RSA encryption not implemented")
}

// decryptRSA decrypts data using RSA
func decryptRSA(data string, keyData string) ([]byte, error) {
	return nil, errors.New("RSA decryption not implemented")
}

// marshalPublicKey marshals a public key
func marshalPublicKey(pub *rsa.PublicKey) []byte {
	return []byte(fmt.Sprintf("%d,%d", pub.N, pub.E))
}

// marshalPrivateKey marshals a private key
func marshalPrivateKey(priv *rsa.PrivateKey) []byte {
	return []byte(fmt.Sprintf("%d,%d,%s", 
		priv.PublicKey.N, 
		priv.PublicKey.E, 
		priv.D.Text(10)))
}

// KeyManager manages keys
type KeyManager struct {
	Store   *KeyStore
	KeyRing map[string]*Key
}

// NewKeyManager creates a new key manager
func NewKeyManager(password string) (*KeyManager, error) {
	store, err := NewKeyStore(password)
	if err != nil {
		return nil, err
	}
	
	return &KeyManager{
		Store:   store,
		KeyRing: make(map[string]*Key),
	}, nil
}

// StoreKey stores a key
func (km *KeyManager) StoreKey(key *Key) error {
	if key.ID == "" {
		return errors.New("key must have an ID")
	}
	
	km.KeyRing[key.ID] = key
	return nil
}

// GetKey retrieves a key
func (km *KeyManager) GetKey(id string) (*Key, bool) {
	key, ok := km.KeyRing[id]
	return key, ok
}

// DeleteKey deletes a key
func (km *KeyManager) DeleteKey(id string) error {
	if _, ok := km.KeyRing[id]; !ok {
		return errors.New("key not found")
	}
	
	delete(km.KeyRing, id)
	return nil
}

// ListKeys lists all key IDs
func (km *KeyManager) ListKeys() []string {
	ids := make([]string, 0, len(km.KeyRing))
	for id := range km.KeyRing {
		ids = append(ids, id)
	}
	return ids
}