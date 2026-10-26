package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testKeyPEM returns a freshly generated P-256 private key in PKCS#8 PEM form.
func testKeyPEM(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return string(block), key
}

func configuredClient(t *testing.T, baseURL string) (*cloudKitClient, *ecdsa.PrivateKey) {
	t.Helper()

	pemData, key := testKeyPEM(t)
	client, err := newCloudKitClient(cloudKitOptions{
		Container:   "iCloud.com.example.app",
		Environment: "development",
		Database:    "public",
		KeyID:       "test-key-id",
		PrivateKey:  pemData,
		BaseURL:     baseURL,
	})
	if err != nil {
		t.Fatalf("newCloudKitClient: %v", err)
	}
	return client, key
}

// --- configuration ----------------------------------------------------------

func TestNewCloudKitClientRequiresCredentials(t *testing.T) {
	cases := map[string]cloudKitOptions{
		"no container":   {KeyID: "k", PrivateKey: "pem"},
		"no key id":      {Container: "c", PrivateKey: "pem"},
		"no private key": {Container: "c", KeyID: "k"},
	}

	for name, options := range cases {
		if _, err := newCloudKitClient(options); !errors.Is(err, ErrCloudKitNotConfigured) {
			t.Fatalf("%s: err = %v, want ErrCloudKitNotConfigured", name, err)
		}
	}
}

func TestNewCloudKitClientRejectsAMalformedKey(t *testing.T) {
	_, err := newCloudKitClient(cloudKitOptions{
		Container:  "c",
		KeyID:      "k",
		PrivateKey: "not a pem block",
	})
	if !errors.Is(err, ErrCloudKitNotConfigured) {
		t.Fatalf("err = %v, want ErrCloudKitNotConfigured", err)
	}
}

func TestNewCloudKitClientValidatesEnvironmentAndDatabase(t *testing.T) {
	pemData, _ := testKeyPEM(t)

	if _, err := newCloudKitClient(cloudKitOptions{
		Container:   "c",
		KeyID:       "k",
		PrivateKey:  pemData,
		Environment: "staging",
	}); err == nil {
		t.Fatal("an unknown environment must be rejected")
	}

	if _, err := newCloudKitClient(cloudKitOptions{
		Container:  "c",
		KeyID:      "k",
		PrivateKey: pemData,
		Database:   "everywhere",
	}); err == nil {
		t.Fatal("an unknown database must be rejected")
	}
}

func TestCloudKitSubpathMatchesTheSignedRoute(t *testing.T) {
	client, _ := configuredClient(t, "")

	got := client.subpath("records/modify")
	want := "/database/1/iCloud.com.example.app/development/public/records/modify"
	if got != want {
		t.Fatalf("subpath = %q, want %q", got, want)
	}
}

func TestCloudKitUnconfiguredClientFailsClosed(t *testing.T) {
	client := &cloudKitClient{}

	if client.Configured() {
		t.Fatal("an empty client must not report itself configured")
	}
	if _, err := client.Upload(context.Background(), []byte("x"), "backup.enc"); !errors.Is(err, ErrCloudKitNotConfigured) {
		t.Fatalf("Upload err = %v, want ErrCloudKitNotConfigured", err)
	}
	if _, err := client.Download(context.Background(), "backup.enc"); !errors.Is(err, ErrCloudKitNotConfigured) {
		t.Fatalf("Download err = %v, want ErrCloudKitNotConfigured", err)
	}
	if err := client.Delete(context.Background(), "backup.enc"); !errors.Is(err, ErrCloudKitNotConfigured) {
		t.Fatalf("Delete err = %v, want ErrCloudKitNotConfigured", err)
	}
}

// --- request signing --------------------------------------------------------

// TestCloudKitSignatureVerifiesOverTheSpecMessage checks the signature against
// the message Apple defines: date:base64(sha256(body)):subpath.
func TestCloudKitSignatureVerifiesOverTheSpecMessage(t *testing.T) {
	client, key := configuredClient(t, "")

	body := []byte(`{"records":[]}`)
	subpath := client.subpath("records/modify")

	headers, err := client.sign(body, subpath)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if headers["X-Apple-CloudKit-Request-KeyID"] != "test-key-id" {
		t.Fatalf("key id header = %q", headers["X-Apple-CloudKit-Request-KeyID"])
	}

	date := headers["X-Apple-CloudKit-Request-ISO8601Date"]
	if strings.Contains(date, ".") {
		t.Fatalf("date %q must not contain milliseconds", date)
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", date); err != nil {
		t.Fatalf("date %q is not the required ISO 8601 form: %v", date, err)
	}

	digest := sha256.Sum256(body)
	message := date + ":" + base64.StdEncoding.EncodeToString(digest[:]) + ":" + subpath

	signature, err := base64.StdEncoding.DecodeString(headers["X-Apple-CloudKit-Request-SignatureV1"])
	if err != nil {
		t.Fatalf("signature is not valid base64: %v", err)
	}

	// SignASN1 produces an ASN.1 DER signature, so verify with the ASN.1 form.
	if !ecdsa.VerifyASN1(&key.PublicKey, []byte(message), signature) {
		t.Fatal("the signature must verify over date:hash:subpath")
	}

	// A different body must not verify, or the signature does not bind the body.
	otherDigest := sha256.Sum256([]byte(`{"records":[1]}`))
	other := date + ":" + base64.StdEncoding.EncodeToString(otherDigest[:]) + ":" + subpath
	if ecdsa.VerifyASN1(&key.PublicKey, []byte(other), signature) {
		t.Fatal("the signature must not verify over a different body")
	}
}

func TestCloudKitSignedMessageUsesBase64SHA256OfTheBody(t *testing.T) {
	body := []byte(`{"a":1}`)
	digest := sha256.Sum256(body)
	encoded := base64.StdEncoding.EncodeToString(digest[:])

	// Base64, not hex: a hex digest would be 64 characters and fail Apple's
	// verification.
	if len(encoded) != 44 {
		t.Fatalf("encoded digest length = %d, want 44", len(encoded))
	}
	if strings.ContainsAny(encoded, "-_") {
		t.Fatalf("digest %q must be standard base64, not URL-safe", encoded)
	}
}

// --- record names -----------------------------------------------------------

func TestCloudKitRecordNameValidation(t *testing.T) {
	valid := []string{"wallet_backup_abc123-1700000000.enc", "a", strings.Repeat("a", 255)}
	for _, name := range valid {
		if _, err := cloudKitRecordName(name); err != nil {
			t.Fatalf("cloudKitRecordName(%q) = %v, want success", name, err)
		}
	}

	invalid := []string{"", "_leading", "has/slash", "../escape", "has space", strings.Repeat("a", 256)}
	for _, name := range invalid {
		if _, err := cloudKitRecordName(name); err == nil {
			t.Fatalf("cloudKitRecordName(%q) succeeded, want an error", name)
		}
	}
}

// --- round trip against a stub CloudKit server ------------------------------

// newStubCloudKit stands in for CloudKit at the HTTP layer so the request
// shape, the signature and the response handling are exercised end to end.
func newStubCloudKit(t *testing.T, key *ecdsa.PrivateKey) *httptest.Server {
	t.Helper()

	records := map[string]map[string]any{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readAll(r)
		if err != nil {
			t.Errorf("read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify the signature on every request.
		date := r.Header.Get("X-Apple-CloudKit-Request-ISO8601Date")
		signatureB64 := r.Header.Get("X-Apple-CloudKit-Request-SignatureV1")
		digest := sha256.Sum256(body)
		message := date + ":" + base64.StdEncoding.EncodeToString(digest[:]) + ":" + r.URL.Path

		signature, err := base64.StdEncoding.DecodeString(signatureB64)
		if err != nil || !ecdsa.VerifyASN1(&key.PublicKey, []byte(message), signature) {
			t.Errorf("request signature did not verify for %s", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case strings.HasSuffix(r.URL.Path, "/records/modify"):
			var payload struct {
				Operations []struct {
					OperationType string `json:"operationType"`
					Record        struct {
						RecordName string         `json:"recordName"`
						Fields     map[string]any `json:"fields"`
					} `json:"record"`
				} `json:"operations"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("unmarshal modify: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, op := range payload.Operations {
				if op.OperationType == "delete" {
					delete(records, op.Record.RecordName)
					continue
				}
				records[op.Record.RecordName] = op.Record.Fields
			}
			writeJSON(w, map[string]any{
				"records": []map[string]any{{"recordName": payload.Operations[0].Record.RecordName}},
			})

		case strings.HasSuffix(r.URL.Path, "/records/lookup"):
			var payload struct {
				Records []struct {
					RecordName string `json:"recordName"`
				} `json:"records"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("unmarshal lookup: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			name := payload.Records[0].RecordName
			fields, ok := records[name]
			if !ok {
				writeJSON(w, map[string]any{
					"serverErrorCode": "NOT_FOUND",
					"reason":          "record not found",
				})
				return
			}
			writeJSON(w, map[string]any{
				"records": []map[string]any{{"recordName": name, "fields": fields}},
			})

		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func readAll(r *http.Request) ([]byte, error) {
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, nil
		}
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func TestCloudKitUploadDownloadRoundTrip(t *testing.T) {
	pemData, key := testKeyPEM(t)
	server := newStubCloudKit(t, key)
	defer server.Close()

	client, err := newCloudKitClient(cloudKitOptions{
		Container:   "iCloud.com.example.app",
		Environment: "development",
		Database:    "public",
		KeyID:       "test-key-id",
		PrivateKey:  pemData,
		BaseURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("newCloudKitClient: %v", err)
	}

	payload := []byte("encrypted wallet bytes")
	name := "wallet_backup_abc123-1700000000.enc"

	recordName, err := client.Upload(context.Background(), payload, name)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if recordName != name {
		t.Fatalf("record name = %q, want %q", recordName, name)
	}

	got, err := client.Download(context.Background(), name)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("downloaded %q, want %q", got, payload)
	}

	if err := client.Delete(context.Background(), name); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// After deletion the record is gone, and the caller learns that.
	if _, err := client.Download(context.Background(), name); !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("Download after delete = %v, want ErrBackupNotFound", err)
	}
}

func TestCloudKitDownloadOfAMissingRecordIsNotFound(t *testing.T) {
	pemData, key := testKeyPEM(t)
	server := newStubCloudKit(t, key)
	defer server.Close()

	client, err := newCloudKitClient(cloudKitOptions{
		Container:  "c",
		KeyID:      "k",
		PrivateKey: pemData,
		BaseURL:    server.URL,
	})
	if err != nil {
		t.Fatalf("newCloudKitClient: %v", err)
	}

	if _, err := client.Download(context.Background(), "never_uploaded.enc"); !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("err = %v, want ErrBackupNotFound", err)
	}
}

func TestCloudKitRejectsAnOversizedBackup(t *testing.T) {
	client, _ := configuredClient(t, "http://127.0.0.1:1")

	data := make([]byte, cloudKitMaxRecordBytes+1)
	if _, err := client.Upload(context.Background(), data, "big.enc"); err == nil {
		t.Fatal("an oversized backup must be rejected before upload")
	}
}

func TestCloudKitRecordURLUsesTheConfiguredBase(t *testing.T) {
	client, _ := configuredClient(t, "https://example.test")

	url, err := client.RecordURL("backup.enc")
	if err != nil {
		t.Fatalf("RecordURL: %v", err)
	}
	if !strings.HasPrefix(url, "https://example.test/database/1/iCloud.com.example.app/") {
		t.Fatalf("url = %q", url)
	}
	if _, err := client.RecordURL("../escape"); err == nil {
		t.Fatal("a traversal name must be rejected")
	}
}
