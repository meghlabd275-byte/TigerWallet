package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// LicenseToken is the cryptographically-signed payload a WL product presents
// to prove it is authorized. It is signed with the control plane's Ed25519
// private key and verified by the WL product with the public key (distributed
// out-of-band). Tampering with any field invalidates the signature.
type LicenseToken struct {
	LicenseID    string    `json:"license_id"`
	WLClientID   string    `json:"wl_client_id"`
	Product      string    `json:"product"`
	Plan         string    `json:"plan"`
	Status       string    `json:"status"` // active | suspended | revoked | expired | halted
	ValidFrom    int64     `json:"valid_from"`
	ValidUntil   int64     `json:"valid_until"`
	MaxUsers     int       `json:"max_users"`
	MaxWallets   int       `json:"max_wallets"`
	MaxBots      int       `json:"max_bots"`
	Features     []string  `json:"features"`
	IssuedAt     int64     `json:"issued_at"`
	NotBefore    int64     `json:"not_before"`
	ExpiresAt    int64     `json:"expires_at"` // token (not license) expiry — short-lived
	Nonce        string    `json:"nonce"`
}

// SignedLicenseToken is a LicenseToken plus its Ed25519 signature.
type SignedLicenseToken struct {
	Token     LicenseToken `json:"token"`
	Signature string       `json:"signature"` // hex-encoded Ed25519
	PublicKey string       `json:"public_key"` // hex-encoded verifier key
}

// KeyPair holds the control plane's Ed25519 signing key pair.
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// GenerateKeyPair creates a fresh Ed25519 key pair (used when no env key set).
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	return &KeyPair{Private: priv, Public: pub}, nil
}

// LoadKeyPair loads from hex seed (64 hex chars) or generates if empty.
func LoadKeyPair(seedHex, pubHex string) (*KeyPair, error) {
	if seedHex != "" {
		seed, err := hex.DecodeString(seedHex)
		if err != nil || len(seed) != ed25519.SeedSize {
			return nil, errors.New("invalid LICENSE_SIGN_KEY_HEX (expected 64 hex chars)")
		}
		priv := ed25519.NewKeyFromSeed(seed)
		pub := priv.Public().(ed25519.PublicKey)
		if pubHex != "" {
			want, err := hex.DecodeString(pubHex)
			if err != nil || !equal(pub, want) {
				return nil, errors.New("LICENSE_VERIFY_KEY_HEX does not match signing key")
			}
		}
		return &KeyPair{Private: priv, Public: pub}, nil
	}
	return GenerateKeyPair()
}

// PublicKeyHex returns the hex-encoded public key for distribution to WL
// products (they verify license tokens with it, out-of-band).
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.Public)
}

// SignToken signs a LicenseToken with the private key.
func (kp *KeyPair) SignToken(t LicenseToken) (*SignedLicenseToken, error) {
	payload, err := canonicalJSON(t)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(kp.Private, payload)
	return &SignedLicenseToken{
		Token:     t,
		Signature: hex.EncodeToString(sig),
		PublicKey: hex.EncodeToString(kp.Public),
	}, nil
}

// VerifyToken verifies a SignedLicenseToken against the provided public key
// (hex). Returns an error if the signature is invalid, the token has expired,
// or the license status is not active.
func VerifyToken(slt *SignedLicenseToken, pubHex string) error {
	pub, err := hex.DecodeString(pubHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid verifier public key")
	}
	sig, err := hex.DecodeString(slt.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("malformed signature")
	}
	payload, err := canonicalJSON(slt.Token)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return errors.New("license token signature invalid (tampered or forged)")
	}
	now := time.Now().Unix()
	if slt.Token.ExpiresAt > 0 && now > slt.Token.ExpiresAt {
		return errors.New("license token expired (renew from control plane)")
	}
	if slt.Token.NotBefore > 0 && now < slt.Token.NotBefore {
		return errors.New("license token not yet valid")
	}
	if slt.Token.Status != "active" {
		return fmt.Errorf("license status is %q, not active", slt.Token.Status)
	}
	if slt.Token.ValidUntil > 0 && now > slt.Token.ValidUntil {
		return errors.New("license validity period has ended")
	}
	return nil
}

// NewToken builds a fresh short-lived token from a license record.
func NewToken(licenseID, wlClientID, product, plan, status string,
	validFrom, validUntil int64, maxU, maxW, maxB int, features []string,
	ttl time.Duration) LicenseToken {
	now := time.Now().Unix()
	return LicenseToken{
		LicenseID:  licenseID,
		WLClientID: wlClientID,
		Product:    product,
		Plan:       plan,
		Status:     status,
		ValidFrom:  validFrom,
		ValidUntil: validUntil,
		MaxUsers:   maxU,
		MaxWallets: maxW,
		MaxBots:    maxB,
		Features:   features,
		IssuedAt:   now,
		NotBefore:  now,
		ExpiresAt:  now + int64(ttl.Seconds()),
		Nonce:      uuid.New().String(),
	}
}

// canonicalJSON produces deterministic JSON (sorted keys, no whitespace) so the
// signature is stable across encoders.
func canonicalJSON(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
