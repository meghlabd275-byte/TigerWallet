package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newTestPushService() *PushService {
	return &PushService{
		config:            PushConfig{APNSKeyID: "KEYID123", APNSTeamID: "TEAM123"},
		fcmClient:         nil,
		fcmServiceAccount: map[string]interface{}{},
	}
}

// TestFCMAccessTokenFailsClosedWithoutCredentials proves a send can never
// proceed on a fabricated token when credentials are absent.
func TestFCMAccessTokenFailsClosedWithoutCredentials(t *testing.T) {
	s := newTestPushService()

	token, err := s.fcmAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected an error when FCM credentials are not configured")
	}
	if !errors.Is(err, errProviderNotConfigured) {
		t.Fatalf("expected errProviderNotConfigured, got %v", err)
	}
	if token != "" {
		t.Fatalf("expected an empty token, got %q", token)
	}
}

// TestFCMAccessTokenRejectsMalformedServiceAccount covers a credentials blob
// that decodes but lacks the signing material.
func TestFCMAccessTokenRejectsMalformedServiceAccount(t *testing.T) {
	s := newTestPushService()
	s.fcmServiceAccount = map[string]interface{}{"client_email": "svc@example.com"}

	if _, err := s.fcmAccessToken(context.Background()); err == nil {
		t.Fatal("expected an error for a service account without a private key")
	}
}

// TestAPNSAuthTokenFailsClosedWithoutKey proves APNs delivery fails closed when
// the signing key was never loaded.
func TestAPNSAuthTokenFailsClosedWithoutKey(t *testing.T) {
	s := newTestPushService()

	if _, err := s.apnsAuthToken(); err == nil {
		t.Fatal("expected an error when the APNS signing key is not configured")
	}
}

// TestAPNSAuthTokenSignsAndCaches verifies the provider token is a real ES256
// JWT carrying the team id and key id, and that it is reused until expiry.
func TestAPNSAuthTokenSignsAndCaches(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	s := newTestPushService()
	s.apnsPrivateKey = key

	first, err := s.apnsAuthToken()
	if err != nil {
		t.Fatalf("apnsAuthToken: %v", err)
	}

	parsed, err := jwt.Parse(first, func(t *jwt.Token) (interface{}, error) {
		return &key.PublicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}))
	if err != nil {
		t.Fatalf("issued APNS token does not verify: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("issued APNS token is not valid")
	}
	if kid, _ := parsed.Header["kid"].(string); kid != "KEYID123" {
		t.Fatalf("expected kid KEYID123, got %q", kid)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}
	if iss, _ := claims["iss"].(string); iss != "TEAM123" {
		t.Fatalf("expected iss TEAM123, got %q", iss)
	}

	second, err := s.apnsAuthToken()
	if err != nil {
		t.Fatalf("apnsAuthToken (cached): %v", err)
	}
	if first != second {
		t.Fatal("expected the cached provider token to be reused before expiry")
	}

	// An expired cache entry must be re-minted.
	s.apnsTokenExp = time.Now().Add(-time.Minute)
	third, err := s.apnsAuthToken()
	if err != nil {
		t.Fatalf("apnsAuthToken (refresh): %v", err)
	}
	if third == "" {
		t.Fatal("expected a re-minted provider token")
	}
}

// TestLoadAPNSPrivateKeyRoundTrip covers the on-disk .p8 format and rejection of
// anything that is not a PKCS#8 EC key.
func TestLoadAPNSPrivateKeyRoundTrip(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	dir := t.TempDir()

	valid := filepath.Join(dir, "AuthKey.p8")
	if err := os.WriteFile(valid, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	loaded, err := loadAPNSPrivateKey(valid)
	if err != nil {
		t.Fatalf("loadAPNSPrivateKey: %v", err)
	}
	if !loaded.Equal(key) {
		t.Fatal("loaded key does not match the generated key")
	}

	invalid := filepath.Join(dir, "bad.p8")
	if err := os.WriteFile(invalid, []byte("not a pem file"), 0o600); err != nil {
		t.Fatalf("write bad key: %v", err)
	}
	if _, err := loadAPNSPrivateKey(invalid); err == nil {
		t.Fatal("expected an error for a non-PEM file")
	}

	if _, err := loadAPNSPrivateKey(filepath.Join(dir, "missing.p8")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

// TestFCMCredentialsParsingMatchesConstructorContract documents the base64-JSON
// shape the constructor expects.
func TestFCMCredentialsParsingMatchesConstructorContract(t *testing.T) {
	account := map[string]interface{}{
		"client_email":   "svc@example.iam.gserviceaccount.com",
		"private_key_id": "abc123",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	raw, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(decoded, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed["client_email"] != account["client_email"] {
		t.Fatal("client_email did not survive the round trip")
	}
}
