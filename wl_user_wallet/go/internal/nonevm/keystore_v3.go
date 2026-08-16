package nonevm

// keystore_v3.go — Web3 Secret Storage V3 (de facto standard) interoperability.
// Ported from the canonical wallet_api keystore_v3.go. Real scrypt KDF,
// AES-128-CTR, keccak256 MAC, real secp256k1 key recovery. Wrong password
// fails the MAC check.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/scrypt"
)

// KeystoreV3 is the Web3 Secret Storage V3 envelope (scrypt variant).
type KeystoreV3 struct {
	Address string   `json:"address"`
	Crypto  CryptoV3 `json:"crypto"`
	ID      string   `json:"id"`
	Version int      `json:"version"`
}

// CryptoV3 holds the cipher + KDF parameters of a V3 keystore.
type CryptoV3 struct {
	Cipher       string          `json:"cipher"`
	CipherText   string          `json:"ciphertext"`
	CipherParams CipherParamsV3  `json:"cipherparams"`
	KDF          string          `json:"kdf"`
	KDFParams    ScryptKDFParams `json:"kdfparams"`
	MAC          string          `json:"mac"`
}

// CipherParamsV3 holds the AES-128-CTR IV.
type CipherParamsV3 struct {
	IV string `json:"iv"`
}

// ScryptKDFParams are the scrypt parameters for the V3 keystore.
type ScryptKDFParams struct {
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	DKLen int    `json:"dklen"`
	Salt  string `json:"salt"`
}

const (
	v3ScryptN     = 1 << 18
	v3ScryptR     = 8
	v3ScryptP     = 1
	v3ScryptDKLen = 32
)

// ExportKeystoreV3 encrypts a secp256k1 private key with a password and
// returns the standard Web3 Secret Storage V3 JSON (scrypt variant).
func ExportKeystoreV3(key *ecdsa.PrivateKey, password string) ([]byte, error) {
	if key == nil || key.D == nil {
		return nil, errors.New("nil private key")
	}
	privBytes := make([]byte, 32)
	key.D.FillBytes(privBytes)

	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt: %w", err)
	}
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("iv: %w", err)
	}

	dk, err := scrypt.Key([]byte(password), salt, v3ScryptN, v3ScryptR, v3ScryptP, v3ScryptDKLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt: %w", err)
	}

	block, err := aes.NewCipher(dk[:16])
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	ct := make([]byte, len(privBytes))
	stream.XORKeyStream(ct, privBytes)

	macInput := append(append([]byte{}, dk[16:32]...), ct...)
	mac := ethcrypto.Keccak256(macInput)

	id, err := randomUUIDv4()
	if err != nil {
		return nil, err
	}

	ks := KeystoreV3{
		Address: ethcrypto.PubkeyToAddress(key.PublicKey).Hex(),
		Crypto: CryptoV3{
			Cipher:     "aes-128-ctr",
			CipherText: hex.EncodeToString(ct),
			CipherParams: CipherParamsV3{
				IV: hex.EncodeToString(iv),
			},
			KDF: "scrypt",
			KDFParams: ScryptKDFParams{
				N:     v3ScryptN,
				R:     v3ScryptR,
				P:     v3ScryptP,
				DKLen: v3ScryptDKLen,
				Salt:  hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
		ID:      id,
		Version: 3,
	}
	return json.MarshalIndent(ks, "", "  ")
}

// ImportKeystoreV3 decrypts a standard Web3 Secret Storage V3 JSON keystore
// (scrypt KDF) with the given password and returns the private key.
func ImportKeystoreV3(keystoreJSON []byte, password string) (*ecdsa.PrivateKey, error) {
	var ks KeystoreV3
	if err := json.Unmarshal(keystoreJSON, &ks); err != nil {
		return nil, fmt.Errorf("parse keystore: %w", err)
	}
	if ks.Version != 3 {
		return nil, fmt.Errorf("unsupported keystore version %d (only v3)", ks.Version)
	}

	salt, err := hex.DecodeString(ks.Crypto.KDFParams.Salt)
	if err != nil {
		return nil, fmt.Errorf("bad salt: %w", err)
	}
	ct, err := hex.DecodeString(ks.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("bad ciphertext: %w", err)
	}
	iv, err := hex.DecodeString(ks.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("bad iv: %w", err)
	}

	var dk []byte
	switch ks.Crypto.KDF {
	case "scrypt":
		dk, err = scrypt.Key([]byte(password), salt,
			ks.Crypto.KDFParams.N, ks.Crypto.KDFParams.R, ks.Crypto.KDFParams.P, ks.Crypto.KDFParams.DKLen)
		if err != nil {
			return nil, fmt.Errorf("scrypt: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported kdf %q (only scrypt supported)", ks.Crypto.KDF)
	}

	macInput := append(append([]byte{}, dk[16:32]...), ct...)
	mac := ethcrypto.Keccak256(macInput)
	expectedMAC, err := hex.DecodeString(ks.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("bad mac: %w", err)
	}
	if !equalConstTime(mac, expectedMAC) {
		return nil, errors.New("invalid password (mac mismatch)")
	}

	block, err := aes.NewCipher(dk[:16])
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	privBytes := make([]byte, len(ct))
	stream.XORKeyStream(privBytes, ct)

	key, err := ethcrypto.ToECDSA(privBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return key, nil
}

func equalConstTime(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func randomUUIDv4() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
