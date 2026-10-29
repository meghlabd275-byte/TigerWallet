package main

// WebAuthn/FIDO2 verification primitives: a CBOR reader (RFC 8949), COSE key
// parsing (RFC 8152), authenticator-data parsing, and signature verification.
//
// These are implemented directly rather than through a helper library so the
// verification path stays auditable and dependency-free.

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"strings"
)

// ---------------------------------------------------------------------------
// CBOR (RFC 8949)
// ---------------------------------------------------------------------------

const (
	cborMajorUnsigned = 0
	cborMajorNegative = 1
	cborMajorBytes    = 2
	cborMajorText     = 3
	cborMajorArray    = 4
	cborMajorMap      = 5
	cborMajorTag      = 6
	cborMajorSimple   = 7

	cborBreak = 0xff
)

// cborDecode parses a single CBOR item that must span the whole input.
func cborDecode(data []byte) (interface{}, error) {
	value, next, err := cborDecodeAt(data, 0)
	if err != nil {
		return nil, err
	}
	if next != len(data) {
		return nil, fmt.Errorf("cbor: %d trailing byte(s) after the item", len(data)-next)
	}
	return value, nil
}

// cborItemLength returns the encoded length of the CBOR item starting at
// data[0], which is used to slice nested items out of authenticator data.
func cborItemLength(data []byte) (int, error) {
	_, next, err := cborDecodeAt(data, 0)
	if err != nil {
		return 0, err
	}
	return next, nil
}

func cborDecodeAt(data []byte, off int) (interface{}, int, error) {
	if off >= len(data) {
		return nil, 0, errors.New("cbor: unexpected end of input")
	}
	initial := data[off]
	major := initial >> 5
	additional := initial & 0x1f
	off++

	// Indefinite length: only byte/text strings, arrays and maps may use it.
	if additional == 31 {
		switch major {
		case cborMajorBytes, cborMajorText:
			var buf []byte
			for {
				if off >= len(data) {
					return nil, 0, errors.New("cbor: unterminated indefinite-length string")
				}
				if data[off] == cborBreak {
					off++
					break
				}
				chunk, next, err := cborDecodeAt(data, off)
				if err != nil {
					return nil, 0, err
				}
				piece, ok := chunk.([]byte)
				if !ok {
					return nil, 0, errors.New("cbor: indefinite-length string contains a non-string chunk")
				}
				buf = append(buf, piece...)
				off = next
			}
			if major == cborMajorText {
				return string(buf), off, nil
			}
			return buf, off, nil
		case cborMajorArray:
			items := []interface{}{}
			for {
				if off >= len(data) {
					return nil, 0, errors.New("cbor: unterminated indefinite-length array")
				}
				if data[off] == cborBreak {
					off++
					break
				}
				item, next, err := cborDecodeAt(data, off)
				if err != nil {
					return nil, 0, err
				}
				items = append(items, item)
				off = next
			}
			return items, off, nil
		case cborMajorMap:
			result := map[interface{}]interface{}{}
			for {
				if off >= len(data) {
					return nil, 0, errors.New("cbor: unterminated indefinite-length map")
				}
				if data[off] == cborBreak {
					off++
					break
				}
				key, next, err := cborDecodeAt(data, off)
				if err != nil {
					return nil, 0, err
				}
				value, next, err := cborDecodeAt(data, next)
				if err != nil {
					return nil, 0, err
				}
				result[key] = value
				off = next
			}
			return result, off, nil
		}
		return nil, 0, fmt.Errorf("cbor: indefinite length is not valid for major type %d", major)
	}

	var value uint64
	switch {
	case additional < 24:
		value = uint64(additional)
	case additional == 24:
		if off+1 > len(data) {
			return nil, 0, errors.New("cbor: truncated 8-bit argument")
		}
		value = uint64(data[off])
		off++
	case additional == 25:
		if off+2 > len(data) {
			return nil, 0, errors.New("cbor: truncated 16-bit argument")
		}
		value = uint64(binary.BigEndian.Uint16(data[off:]))
		off += 2
	case additional == 26:
		if off+4 > len(data) {
			return nil, 0, errors.New("cbor: truncated 32-bit argument")
		}
		value = uint64(binary.BigEndian.Uint32(data[off:]))
		off += 4
	case additional == 27:
		if off+8 > len(data) {
			return nil, 0, errors.New("cbor: truncated 64-bit argument")
		}
		value = binary.BigEndian.Uint64(data[off:])
		off += 8
	default:
		return nil, 0, fmt.Errorf("cbor: reserved additional information %d", additional)
	}

	switch major {
	case cborMajorUnsigned:
		return value, off, nil

	case cborMajorNegative:
		if value > math.MaxInt64 {
			return nil, 0, errors.New("cbor: negative integer out of range")
		}
		return -1 - int64(value), off, nil

	case cborMajorBytes:
		if uint64(off)+value > uint64(len(data)) {
			return nil, 0, errors.New("cbor: truncated byte string")
		}
		out := make([]byte, value)
		copy(out, data[off:off+int(value)])
		return out, off + int(value), nil

	case cborMajorText:
		if uint64(off)+value > uint64(len(data)) {
			return nil, 0, errors.New("cbor: truncated text string")
		}
		return string(data[off : off+int(value)]), off + int(value), nil

	case cborMajorArray:
		items := make([]interface{}, 0, value)
		for i := uint64(0); i < value; i++ {
			item, next, err := cborDecodeAt(data, off)
			if err != nil {
				return nil, 0, err
			}
			items = append(items, item)
			off = next
		}
		return items, off, nil

	case cborMajorMap:
		result := make(map[interface{}]interface{}, value)
		for i := uint64(0); i < value; i++ {
			key, next, err := cborDecodeAt(data, off)
			if err != nil {
				return nil, 0, err
			}
			item, next, err := cborDecodeAt(data, next)
			if err != nil {
				return nil, 0, err
			}
			result[key] = item
			off = next
		}
		return result, off, nil

	case cborMajorTag:
		// Semantic tags carry no meaning for the structures parsed here.
		return cborDecodeAt(data, off)

	case cborMajorSimple:
		switch additional {
		case 20:
			return false, off, nil
		case 21:
			return true, off, nil
		case 22, 23:
			return nil, off, nil
		}
		return nil, 0, fmt.Errorf("cbor: unsupported simple value %d", additional)
	}

	return nil, 0, fmt.Errorf("cbor: unsupported major type %d", major)
}

// ---------------------------------------------------------------------------
// COSE keys (RFC 8152)
// ---------------------------------------------------------------------------

// COSE algorithm identifiers used by WebAuthn.
const (
	coseAlgES256 = -7
	coseAlgEdDSA = -8
	coseAlgRS256 = -257
)

// COSE key type identifiers.
const (
	coseKtyOKP = 1
	coseKtyEC2 = 2
	coseKtyRSA = 3
)

// coseLookup resolves a COSE label. Positive CBOR integers decode as uint64
// while negative ones decode as int64, so both key forms are tried rather than
// assuming one representation.
func coseLookup(m map[interface{}]interface{}, label int64) (interface{}, bool) {
	if raw, ok := m[label]; ok {
		return raw, true
	}
	if label >= 0 {
		if raw, ok := m[uint64(label)]; ok {
			return raw, true
		}
	}
	return nil, false
}

func coseInt(m map[interface{}]interface{}, label int64) (int64, bool) {
	raw, ok := coseLookup(m, label)
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int64:
		return v, true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	}
	return 0, false
}

func coseBytes(m map[interface{}]interface{}, label int64) []byte {
	if raw, ok := coseLookup(m, label); ok {
		if b, ok := raw.([]byte); ok {
			return b
		}
	}
	return nil
}

// coseKeyToPublicKey converts a COSE_Key into a Go public key and reports the
// COSE algorithm identifier it declares.
func coseKeyToPublicKey(rawKey []byte) (crypto.PublicKey, int, error) {
	decoded, err := cborDecode(rawKey)
	if err != nil {
		return nil, 0, fmt.Errorf("credential public key is not valid CBOR: %w", err)
	}
	m, ok := decoded.(map[interface{}]interface{})
	if !ok {
		return nil, 0, errors.New("credential public key is not a COSE map")
	}

	kty, ok := coseInt(m, 1)
	if !ok {
		return nil, 0, errors.New("COSE key has no key type (label 1)")
	}
	alg, _ := coseInt(m, 3)

	switch kty {
	case coseKtyEC2:
		crv, ok := coseInt(m, -1)
		if !ok {
			return nil, 0, errors.New("EC2 key has no curve (label -1)")
		}
		var curve elliptic.Curve
		switch crv {
		case 1:
			curve = elliptic.P256()
		case 2:
			curve = elliptic.P384()
		case 3:
			curve = elliptic.P521()
		default:
			return nil, 0, fmt.Errorf("unsupported EC curve %d", crv)
		}
		x, y := coseBytes(m, -2), coseBytes(m, -3)
		if len(x) == 0 || len(y) == 0 {
			return nil, 0, errors.New("EC2 key is missing a coordinate")
		}
		pub := &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}
		if !curve.IsOnCurve(pub.X, pub.Y) {
			return nil, 0, errors.New("EC2 public key is not on its declared curve")
		}
		if alg == 0 {
			alg = coseAlgES256
		}
		return pub, int(alg), nil

	case coseKtyRSA:
		n, e := coseBytes(m, -1), coseBytes(m, -2)
		if len(n) == 0 || len(e) == 0 {
			return nil, 0, errors.New("RSA key is missing a modulus or exponent")
		}
		exponent := new(big.Int).SetBytes(e)
		if !exponent.IsInt64() || exponent.Int64() <= 0 {
			return nil, 0, errors.New("RSA public exponent is out of range")
		}
		if alg == 0 {
			alg = coseAlgRS256
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())}, int(alg), nil

	case coseKtyOKP:
		crv, ok := coseInt(m, -1)
		if !ok {
			return nil, 0, errors.New("OKP key has no curve (label -1)")
		}
		if crv != 6 {
			return nil, 0, fmt.Errorf("unsupported OKP curve %d", crv)
		}
		x := coseBytes(m, -2)
		if len(x) != ed25519.PublicKeySize {
			return nil, 0, fmt.Errorf("Ed25519 public key must be %d bytes, got %d", ed25519.PublicKeySize, len(x))
		}
		if alg == 0 {
			alg = coseAlgEdDSA
		}
		return ed25519.PublicKey(x), int(alg), nil
	}

	return nil, 0, fmt.Errorf("unsupported COSE key type %d", kty)
}

// ---------------------------------------------------------------------------
// Authenticator data
// ---------------------------------------------------------------------------

const (
	flagUserPresent            = 0x01
	flagUserVerified           = 0x04
	flagBackupEligible         = 0x08
	flagBackupState            = 0x10
	flagAttestedCredentialData = 0x40
	flagExtensionData          = 0x80

	authenticatorDataMinLen = 37
)

type authenticatorData struct {
	Raw              []byte
	RPIDHash         []byte
	Flags            byte
	SignCount        uint32
	AAGUID           []byte
	CredentialID     []byte
	COSEKey          []byte
	Extensions       []byte
	HasAttestedCreds bool
	UserPresent      bool
	UserVerified     bool
	BackupEligible   bool
	BackupState      bool
}

func parseAuthenticatorData(data []byte) (*authenticatorData, error) {
	if len(data) < authenticatorDataMinLen {
		return nil, fmt.Errorf("authenticator data must be at least %d bytes, got %d", authenticatorDataMinLen, len(data))
	}
	ad := &authenticatorData{
		Raw:            data,
		RPIDHash:       data[0:32],
		Flags:          data[32],
		SignCount:      binary.BigEndian.Uint32(data[33:37]),
		UserPresent:    data[32]&flagUserPresent != 0,
		UserVerified:   data[32]&flagUserVerified != 0,
		BackupEligible: data[32]&flagBackupEligible != 0,
		BackupState:    data[32]&flagBackupState != 0,
	}

	off := authenticatorDataMinLen
	if ad.Flags&flagAttestedCredentialData != 0 {
		ad.HasAttestedCreds = true
		if len(data) < off+18 {
			return nil, errors.New("attested credential data is truncated")
		}
		ad.AAGUID = data[off : off+16]
		off += 16
		credLen := int(binary.BigEndian.Uint16(data[off : off+2]))
		off += 2
		if len(data) < off+credLen {
			return nil, errors.New("credential id is truncated")
		}
		ad.CredentialID = data[off : off+credLen]
		off += credLen

		keyLen, err := cborItemLength(data[off:])
		if err != nil {
			return nil, fmt.Errorf("credential public key: %w", err)
		}
		if len(data) < off+keyLen {
			return nil, errors.New("credential public key is truncated")
		}
		ad.COSEKey = data[off : off+keyLen]
		off += keyLen
	}

	if ad.Flags&flagExtensionData != 0 {
		extLen, err := cborItemLength(data[off:])
		if err != nil {
			return nil, fmt.Errorf("authenticator extension data: %w", err)
		}
		if len(data) < off+extLen {
			return nil, errors.New("authenticator extension data is truncated")
		}
		ad.Extensions = data[off : off+extLen]
		off += extLen
	}

	if off != len(data) {
		return nil, fmt.Errorf("authenticator data has %d unexpected trailing byte(s)", len(data)-off)
	}
	return ad, nil
}

// ---------------------------------------------------------------------------
// Client data
// ---------------------------------------------------------------------------

const (
	clientDataTypeCreate = "webauthn.create"
	clientDataTypeGet    = "webauthn.get"
)

// verifyClientData checks the ceremony type, the challenge echo and the origin
// of the client data produced by the authenticator's host.
func verifyClientData(raw []byte, wantType, wantChallenge, wantOrigin string) (*ClientDataJSON, error) {
	var clientData ClientDataJSON
	if err := json.Unmarshal(raw, &clientData); err != nil {
		return nil, fmt.Errorf("client data is not valid JSON: %w", err)
	}
	if clientData.Type != wantType {
		return nil, fmt.Errorf("client data type is %q, want %q", clientData.Type, wantType)
	}
	if clientData.Challenge == "" {
		return nil, errors.New("client data has no challenge")
	}
	if subtle.ConstantTimeCompare([]byte(clientData.Challenge), []byte(wantChallenge)) != 1 {
		return nil, errors.New("client data challenge does not match the issued challenge")
	}
	if !originMatches(clientData.Origin, wantOrigin) {
		return nil, fmt.Errorf("client data origin %q does not match the configured origin %q", clientData.Origin, wantOrigin)
	}
	return &clientData, nil
}

// originMatches compares an origin against the configured one on scheme, host
// and effective port. A prefix test would accept look-alike hosts such as
// https://tigerwallet.com.attacker.example.
func originMatches(actual, expected string) bool {
	a, err := url.Parse(actual)
	if err != nil {
		return false
	}
	e, err := url.Parse(expected)
	if err != nil {
		return false
	}
	if !strings.EqualFold(a.Scheme, e.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), e.Hostname()) {
		return false
	}
	return effectivePort(a) == effectivePort(e)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}

// ---------------------------------------------------------------------------
// Attestation
// ---------------------------------------------------------------------------

// parseAttestationObject extracts the statement format and the authenticator
// data from a CBOR attestation object.
func parseAttestationObject(raw []byte) (format string, authData []byte, err error) {
	decoded, err := cborDecode(raw)
	if err != nil {
		return "", nil, fmt.Errorf("attestation object is not valid CBOR: %w", err)
	}
	m, ok := decoded.(map[interface{}]interface{})
	if !ok {
		return "", nil, errors.New("attestation object is not a CBOR map")
	}
	format, _ = m["fmt"].(string)
	authData, _ = m["authData"].([]byte)
	if len(authData) == 0 {
		return "", nil, errors.New("attestation object has no authenticator data")
	}
	if format == "" {
		return "", nil, errors.New("attestation object has no statement format")
	}
	return format, authData, nil
}

// ---------------------------------------------------------------------------
// Signature verification
// ---------------------------------------------------------------------------

// verifyAssertionSignature verifies the authenticator signature over
// authenticatorData || SHA-256(clientDataJSON).
func verifyAssertionSignature(pub crypto.PublicKey, alg int, authData, clientDataJSON, signature []byte) error {
	if len(signature) == 0 {
		return errors.New("assertion has no signature")
	}
	clientHash := sha256.Sum256(clientDataJSON)
	signed := make([]byte, 0, len(authData)+len(clientHash))
	signed = append(signed, authData...)
	signed = append(signed, clientHash[:]...)

	switch alg {
	case coseAlgES256:
		key, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("credential declares ES256 but its stored key is not an EC key")
		}
		// WebAuthn encodes ECDSA signatures as fixed-width r || s, not DER.
		if len(signature) != 64 {
			return fmt.Errorf("ES256 signature must be 64 bytes, got %d", len(signature))
		}
		digest := sha256.Sum256(signed)
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:])
		if !ecdsa.Verify(key, digest[:], r, s) {
			return errors.New("ES256 signature verification failed")
		}
		return nil

	case coseAlgRS256:
		key, ok := pub.(*rsa.PublicKey)
		if !ok {
			return errors.New("credential declares RS256 but its stored key is not an RSA key")
		}
		digest := sha256.Sum256(signed)
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
			return fmt.Errorf("RS256 signature verification failed: %w", err)
		}
		return nil

	case coseAlgEdDSA:
		key, ok := pub.(ed25519.PublicKey)
		if !ok {
			return errors.New("credential declares EdDSA but its stored key is not an Ed25519 key")
		}
		if !ed25519.Verify(key, signed, signature) {
			return errors.New("EdDSA signature verification failed")
		}
		return nil
	}

	return fmt.Errorf("unsupported COSE algorithm %d", alg)
}

// verifyRPIDHash checks that the authenticator was scoped to this relying party.
func verifyRPIDHash(ad *authenticatorData, rpID string) error {
	expected := sha256.Sum256([]byte(rpID))
	if subtle.ConstantTimeCompare(ad.RPIDHash, expected[:]) != 1 {
		return errors.New("authenticator data was produced for a different relying party")
	}
	return nil
}
