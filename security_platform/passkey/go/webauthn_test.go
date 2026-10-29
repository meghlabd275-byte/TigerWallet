package main

// Tests for the WebAuthn verification primitives. A software authenticator is
// built with real key material and real CBOR so the verification path is
// exercised end to end rather than asserted against fixed fixtures.

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Minimal CBOR encoder (test-only) so vectors can be constructed in-memory.
// ---------------------------------------------------------------------------

func cborHead(major byte, n uint64) []byte {
	switch {
	case n < 24:
		return []byte{major<<5 | byte(n)}
	case n <= 0xff:
		return []byte{major<<5 | 24, byte(n)}
	case n <= 0xffff:
		b := []byte{major<<5 | 25, 0, 0}
		binary.BigEndian.PutUint16(b[1:], uint16(n))
		return b
	default:
		b := []byte{major<<5 | 26, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(b[1:], uint32(n))
		return b
	}
}

func cborText(s string) []byte  { return append(cborHead(3, uint64(len(s))), s...) }
func cborBytes(b []byte) []byte { return append(cborHead(2, uint64(len(b))), b...) }
func cborUint(n uint64) []byte  { return cborHead(0, n) }

func cborNegInt(n int64) []byte { return cborHead(1, uint64(-1-n)) }

func cborMap(pairs ...[]byte) []byte {
	out := cborHead(5, uint64(len(pairs)/2))
	for _, p := range pairs {
		out = append(out, p...)
	}
	return out
}

// coseKeyEC2 builds a COSE_Key for a P-256 public key.
func coseKeyEC2(pub *ecdsa.PublicKey) []byte {
	size := (pub.Curve.Params().BitSize + 7) / 8
	x := make([]byte, size)
	y := make([]byte, size)
	pub.X.FillBytes(x)
	pub.Y.FillBytes(y)
	return cborMap(
		cborUint(1), cborUint(2), // kty: EC2
		cborUint(3), cborNegInt(coseAlgES256), // alg: ES256
		cborNegInt(-1), cborUint(1), // crv: P-256
		cborNegInt(-2), cborBytes(x), // x
		cborNegInt(-3), cborBytes(y), // y
	)
}

func coseKeyOKP(pub ed25519.PublicKey) []byte {
	return cborMap(
		cborUint(1), cborUint(1), // kty: OKP
		cborUint(3), cborNegInt(coseAlgEdDSA), // alg: EdDSA
		cborNegInt(-1), cborUint(6), // crv: Ed25519
		cborNegInt(-2), cborBytes(pub),
	)
}

func coseKeyRSA(pub *rsa.PublicKey) []byte {
	e := big.NewInt(int64(pub.E)).Bytes()
	return cborMap(
		cborUint(1), cborUint(3), // kty: RSA
		cborUint(3), cborNegInt(coseAlgRS256), // alg: RS256
		cborNegInt(-1), cborBytes(pub.N.Bytes()), // n
		cborNegInt(-2), cborBytes(e), // e
	)
}

const testRPID = "tigerwallet.com"

func testAuthData(rpID string, flags byte, signCount uint32) []byte {
	hash := sha256.Sum256([]byte(rpID))
	out := make([]byte, 0, authenticatorDataMinLen)
	out = append(out, hash[:]...)
	out = append(out, flags)
	sc := make([]byte, 4)
	binary.BigEndian.PutUint32(sc, signCount)
	return append(out, sc...)
}

func testClientData(t *testing.T, typ, challenge, origin string) []byte {
	t.Helper()
	raw, err := json.Marshal(ClientDataJSON{Type: typ, Challenge: challenge, Origin: origin})
	if err != nil {
		t.Fatalf("marshal client data: %v", err)
	}
	return raw
}

// ---------------------------------------------------------------------------
// CBOR
// ---------------------------------------------------------------------------

func TestCBORRoundTrip(t *testing.T) {
	encoded := cborMap(
		cborText("fmt"), cborText("none"),
		cborText("authData"), cborBytes([]byte{1, 2, 3}),
		cborUint(7), cborNegInt(coseAlgES256),
	)
	decoded, err := cborDecode(encoded)
	if err != nil {
		t.Fatalf("cborDecode: %v", err)
	}
	m, ok := decoded.(map[interface{}]interface{})
	if !ok {
		t.Fatalf("expected a map, got %T", decoded)
	}
	if m["fmt"] != "none" {
		t.Errorf("fmt = %v, want none", m["fmt"])
	}
	if !bytes.Equal(m["authData"].([]byte), []byte{1, 2, 3}) {
		t.Errorf("authData = %v", m["authData"])
	}
	if m[uint64(7)] != int64(coseAlgES256) {
		t.Errorf("alg = %v (%T), want %d", m[uint64(7)], m[uint64(7)], coseAlgES256)
	}
}

func TestCBORRejectsMalformedInput(t *testing.T) {
	cases := map[string][]byte{
		"truncated byte string":  {0x43, 0x01, 0x02},
		"truncated text string":  {0x62, 0x61},
		"trailing content":       {0x01, 0x02},
		"truncated 32-bit arg":   {0x1a, 0x00, 0x00},
		"empty input":            {},
		"reserved additional":    {0x1c},
		"unterminated indet map": {0xbf},
	}
	for name, raw := range cases {
		if _, err := cborDecode(raw); err == nil {
			t.Errorf("%s: expected an error, got none", name)
		}
	}
}

// ---------------------------------------------------------------------------
// COSE key parsing
// ---------------------------------------------------------------------------

func TestCOSEParsesES256Key(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pub, alg, err := coseKeyToPublicKey(coseKeyEC2(&priv.PublicKey))
	if err != nil {
		t.Fatalf("coseKeyToPublicKey: %v", err)
	}
	if alg != coseAlgES256 {
		t.Errorf("alg = %d, want %d", alg, coseAlgES256)
	}
	parsed, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("expected *ecdsa.PublicKey, got %T", pub)
	}
	if parsed.X.Cmp(priv.PublicKey.X) != 0 || parsed.Y.Cmp(priv.PublicKey.Y) != 0 {
		t.Error("parsed public key does not match the generated key")
	}
}

func TestCOSERejectsOffCurvePoint(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	size := 32
	x := make([]byte, size)
	y := make([]byte, size)
	priv.PublicKey.X.FillBytes(x)
	priv.PublicKey.Y.FillBytes(y)
	y[31] ^= 0xff // corrupt the point so it is no longer on the curve

	raw := cborMap(
		cborUint(1), cborUint(2),
		cborUint(3), cborNegInt(coseAlgES256),
		cborNegInt(-1), cborUint(1),
		cborNegInt(-2), cborBytes(x),
		cborNegInt(-3), cborBytes(y),
	)
	if _, _, err := coseKeyToPublicKey(raw); err == nil {
		t.Fatal("expected an off-curve point to be rejected")
	}
}

// ---------------------------------------------------------------------------
// Authenticator data
// ---------------------------------------------------------------------------

func TestParseAuthenticatorData(t *testing.T) {
	raw := testAuthData(testRPID, flagUserPresent|flagUserVerified, 42)
	ad, err := parseAuthenticatorData(raw)
	if err != nil {
		t.Fatalf("parseAuthenticatorData: %v", err)
	}
	if ad.SignCount != 42 {
		t.Errorf("SignCount = %d, want 42", ad.SignCount)
	}
	if !ad.UserPresent || !ad.UserVerified {
		t.Error("expected user present and verified flags")
	}
	if err := verifyRPIDHash(ad, testRPID); err != nil {
		t.Errorf("verifyRPIDHash: %v", err)
	}
	if err := verifyRPIDHash(ad, "evil.example"); err == nil {
		t.Error("expected a different rpId to be rejected")
	}
}

func TestParseAuthenticatorDataRejectsShortInput(t *testing.T) {
	if _, err := parseAuthenticatorData(make([]byte, 36)); err == nil {
		t.Fatal("expected a short authenticator data buffer to be rejected")
	}
}

// ---------------------------------------------------------------------------
// Signature verification
// ---------------------------------------------------------------------------

func TestES256AssertionVerifiesAndRejectsTampering(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	authData := testAuthData(testRPID, flagUserPresent|flagUserVerified, 5)
	clientData := testClientData(t, clientDataTypeGet, "challenge-abc", "https://tigerwallet.com")

	clientHash := sha256.Sum256(clientData)
	signed := append(append([]byte{}, authData...), clientHash[:]...)
	digest := sha256.Sum256(signed)

	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	pub, alg, err := coseKeyToPublicKey(coseKeyEC2(&priv.PublicKey))
	if err != nil {
		t.Fatalf("coseKeyToPublicKey: %v", err)
	}
	if err := verifyAssertionSignature(pub, alg, authData, clientData, sig); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}

	// A flipped bit in the signature must fail.
	tampered := append([]byte{}, sig...)
	tampered[0] ^= 0x01
	if err := verifyAssertionSignature(pub, alg, authData, clientData, tampered); err == nil {
		t.Error("tampered signature was accepted")
	}

	// A different challenge (hence different client data) must fail.
	otherClient := testClientData(t, clientDataTypeGet, "challenge-xyz", "https://tigerwallet.com")
	if err := verifyAssertionSignature(pub, alg, authData, otherClient, sig); err == nil {
		t.Error("signature for a different challenge was accepted")
	}

	// Altered authenticator data must fail.
	altered := append([]byte{}, authData...)
	altered[32] ^= 0x04
	if err := verifyAssertionSignature(pub, alg, altered, clientData, sig); err == nil {
		t.Error("signature over altered authenticator data was accepted")
	}

	// A DER-encoded signature must be rejected: WebAuthn uses raw r||s.
	if err := verifyAssertionSignature(pub, alg, authData, clientData, []byte{0x30, 0x44}); err == nil {
		t.Error("a non-64-byte ES256 signature was accepted")
	}
}

func TestEd25519AssertionVerifies(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	authData := testAuthData(testRPID, flagUserPresent, 1)
	clientData := testClientData(t, clientDataTypeGet, "c1", "https://tigerwallet.com")
	clientHash := sha256.Sum256(clientData)
	signed := append(append([]byte{}, authData...), clientHash[:]...)

	sig := ed25519.Sign(priv, signed)
	parsed, alg, err := coseKeyToPublicKey(coseKeyOKP(pub))
	if err != nil {
		t.Fatalf("coseKeyToPublicKey: %v", err)
	}
	if err := verifyAssertionSignature(parsed, alg, authData, clientData, sig); err != nil {
		t.Fatalf("valid EdDSA signature rejected: %v", err)
	}
	if err := verifyAssertionSignature(parsed, alg, authData, clientData, append([]byte{0}, sig[1:]...)); err == nil {
		t.Error("tampered EdDSA signature was accepted")
	}
}

func TestRS256AssertionVerifies(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	authData := testAuthData(testRPID, flagUserPresent, 1)
	clientData := testClientData(t, clientDataTypeGet, "c1", "https://tigerwallet.com")
	clientHash := sha256.Sum256(clientData)
	signed := append(append([]byte{}, authData...), clientHash[:]...)
	digest := sha256.Sum256(signed)

	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parsed, alg, err := coseKeyToPublicKey(coseKeyRSA(&priv.PublicKey))
	if err != nil {
		t.Fatalf("coseKeyToPublicKey: %v", err)
	}
	if err := verifyAssertionSignature(parsed, alg, authData, clientData, sig); err != nil {
		t.Fatalf("valid RS256 signature rejected: %v", err)
	}
	if err := verifyAssertionSignature(parsed, alg, authData, clientData, make([]byte, 256)); err == nil {
		t.Error("an all-zero RS256 signature was accepted")
	}
}

func TestSignatureRejectedForWrongKeyType(t *testing.T) {
	// A credential that declares ES256 but stores an Ed25519 key must fail
	// rather than silently skipping verification.
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	authData := testAuthData(testRPID, flagUserPresent, 1)
	clientData := testClientData(t, clientDataTypeGet, "c1", "https://tigerwallet.com")
	clientHash := sha256.Sum256(clientData)
	signed := append(append([]byte{}, authData...), clientHash[:]...)
	sig := ed25519.Sign(priv, signed)

	if err := verifyAssertionSignature(priv.Public(), coseAlgES256, authData, clientData, sig); err == nil {
		t.Error("an ES256 declaration backed by an Ed25519 key was accepted")
	}
}

// ---------------------------------------------------------------------------
// Client data
// ---------------------------------------------------------------------------

func TestVerifyClientData(t *testing.T) {
	good := testClientData(t, clientDataTypeGet, "chal", "https://tigerwallet.com")
	if _, err := verifyClientData(good, clientDataTypeGet, "chal", "https://tigerwallet.com"); err != nil {
		t.Fatalf("valid client data rejected: %v", err)
	}

	cases := []struct {
		name string
		raw  []byte
	}{
		{"wrong ceremony type", testClientData(t, clientDataTypeCreate, "chal", "https://tigerwallet.com")},
		{"wrong challenge", testClientData(t, clientDataTypeGet, "other", "https://tigerwallet.com")},
		{"wrong origin", testClientData(t, clientDataTypeGet, "chal", "https://evil.example")},
		{"look-alike host", testClientData(t, clientDataTypeGet, "chal", "https://tigerwallet.com.evil.example")},
		{"scheme downgrade", testClientData(t, clientDataTypeGet, "chal", "http://tigerwallet.com")},
		{"empty challenge", testClientData(t, clientDataTypeGet, "", "https://tigerwallet.com")},
		{"not json", []byte("nope")},
	}
	for _, tc := range cases {
		if _, err := verifyClientData(tc.raw, clientDataTypeGet, "chal", "https://tigerwallet.com"); err == nil {
			t.Errorf("%s: expected rejection, got none", tc.name)
		}
	}
}

func TestOriginMatchesPortHandling(t *testing.T) {
	cases := []struct {
		actual, expected string
		want             bool
	}{
		{"https://tigerwallet.com", "https://tigerwallet.com", true},
		{"https://tigerwallet.com:443", "https://tigerwallet.com", true},
		{"https://tigerwallet.com:8443", "https://tigerwallet.com:8443", true},
		{"https://tigerwallet.com:8443", "https://tigerwallet.com", false},
		{"http://localhost:3000", "http://localhost:3000", true},
	}
	for _, tc := range cases {
		if got := originMatches(tc.actual, tc.expected); got != tc.want {
			t.Errorf("originMatches(%q, %q) = %v, want %v", tc.actual, tc.expected, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Attestation object
// ---------------------------------------------------------------------------

func TestParseAttestationObject(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	credID := []byte("credential-id-bytes")

	authData := testAuthData(testRPID, flagUserPresent|flagAttestedCredentialData, 0)
	authData = append(authData, make([]byte, 16)...) // AAGUID
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(credID)))
	authData = append(authData, lenBuf...)
	authData = append(authData, credID...)
	authData = append(authData, coseKeyEC2(&priv.PublicKey)...)

	obj := cborMap(
		cborText("fmt"), cborText("none"),
		cborText("authData"), cborBytes(authData),
	)

	format, gotAuthData, err := parseAttestationObject(obj)
	if err != nil {
		t.Fatalf("parseAttestationObject: %v", err)
	}
	if format != "none" {
		t.Errorf("fmt = %q, want none", format)
	}

	ad, err := parseAuthenticatorData(gotAuthData)
	if err != nil {
		t.Fatalf("parseAuthenticatorData: %v", err)
	}
	if !ad.HasAttestedCreds {
		t.Fatal("expected attested credential data")
	}
	if !bytes.Equal(ad.CredentialID, credID) {
		t.Errorf("credential id = %x, want %x", ad.CredentialID, credID)
	}
	if len(ad.AAGUID) != 16 {
		t.Errorf("AAGUID length = %d, want 16", len(ad.AAGUID))
	}
	if _, _, err := coseKeyToPublicKey(ad.COSEKey); err != nil {
		t.Errorf("embedded COSE key is unusable: %v", err)
	}
}

func TestParseAttestationObjectRejectsGarbage(t *testing.T) {
	if _, _, err := parseAttestationObject([]byte("not cbor at all")); err == nil {
		t.Error("expected non-CBOR input to be rejected")
	}
	if _, _, err := parseAttestationObject(cborMap(cborText("fmt"), cborText("none"))); err == nil {
		t.Error("expected a missing authData field to be rejected")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestChallengeIsRandomAndUnique(t *testing.T) {
	svc := &PasskeyService{config: &Config{Timeout: time.Minute}}
	seen := make(map[string]bool, 256)
	for i := 0; i < 256; i++ {
		c, err := svc.GenerateChallenge()
		if err != nil {
			t.Fatalf("GenerateChallenge: %v", err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(c)
		if err != nil {
			t.Fatalf("challenge is not raw-url base64: %v", err)
		}
		if len(raw) != 32 {
			t.Fatalf("challenge is %d bytes, want 32", len(raw))
		}
		if seen[c] {
			t.Fatalf("duplicate challenge generated: %s", c)
		}
		seen[c] = true
	}
}
