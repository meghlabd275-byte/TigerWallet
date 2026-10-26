// secrets.go — authenticated encryption for stored integration credentials.
//
// Third-party API keys and secrets were written to the database in plaintext.
// Anything with read access to the table, including a backup or a replica, could
// then act as the configured provider. Values are now sealed with AES-256-GCM
// under a key derived from SUPER_ADMIN_SECRETS_KEY.

package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	// errSecretsKeyNotConfigured is returned when no encryption key is set.
	errSecretsKeyNotConfigured = errors.New("credential encryption key is not configured (SUPER_ADMIN_SECRETS_KEY)")
	// errCiphertextInvalid is returned for a value that cannot be decrypted.
	errCiphertextInvalid = errors.New("stored credential is not valid ciphertext")
)

// sealedPrefix marks a value that has already been encrypted, so re-saving a
// config does not encrypt it twice.
const sealedPrefix = "enc:v1:"

// secretBox seals and opens stored credentials.
type secretBox struct {
	key []byte
}

// newSecretBox derives a 32-byte key from the environment.
func newSecretBox() *secretBox {
	secret := strings.TrimSpace(os.Getenv("SUPER_ADMIN_SECRETS_KEY"))
	if secret == "" {
		return &secretBox{}
	}
	// Domain-separated so the same env var cannot be reused for another purpose
	// and produce the same key.
	sum := sha256.Sum256([]byte("tigerwallet:integration-secret:v1|" + secret))
	return &secretBox{key: sum[:]}
}

func (b *secretBox) configured() bool {
	return b != nil && len(b.key) == 32
}

// seal encrypts a plaintext value and returns the encoded ciphertext.
func (b *secretBox) seal(plaintext string) (string, error) {
	if !b.configured() {
		return "", errSecretsKeyNotConfigured
	}

	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return sealedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// open decrypts a value produced by seal.
func (b *secretBox) open(ciphertext string) (string, error) {
	if !b.configured() {
		return "", errSecretsKeyNotConfigured
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, sealedPrefix))
	if err != nil {
		return "", errCiphertextInvalid
	}

	block, err := aes.NewCipher(b.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errCiphertextInvalid
	}

	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errCiphertextInvalid
	}
	return string(plaintext), nil
}

// isSealed reports whether a value already carries the ciphertext prefix.
func isSealed(value string) bool {
	return strings.HasPrefix(value, sealedPrefix)
}

// protect seals a credential unless it is empty or already sealed.
//
// Empty values are passed through so an integration without a secret stays
// valid. A sealed value is returned unchanged so saving a config that was read
// back from the database does not double-encrypt it.
func (b *secretBox) protect(value string) (string, error) {
	if value == "" || isSealed(value) {
		return value, nil
	}
	return b.seal(value)
}

// reveal returns the plaintext of a stored credential, tolerating values that
// were written before encryption was introduced.
func (b *secretBox) reveal(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !isSealed(value) {
		return value, nil
	}
	return b.open(value)
}
