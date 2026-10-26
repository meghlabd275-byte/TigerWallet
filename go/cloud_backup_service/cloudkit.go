// CloudKit Web Services storage for wallet backups.
//
// Talks to Apple's CloudKit Web Services API, the supported server-side
// interface for iCloud data. Requests are signed with an ECDSA P-256 key per
// Apple's server-to-server scheme: the signature covers
// date:base64(sha256(body)):subpath.

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// cloudKitRecordType is the record type used for wallet backups.
const cloudKitRecordType = "WalletBackup"

// cloudKitBaseURL is the CloudKit Web Services endpoint.
const cloudKitBaseURL = "https://api.apple-cloudkit.com"

// cloudKitMaxRecordBytes is the largest payload CloudKit accepts in one record.
const cloudKitMaxRecordBytes = 1024 * 1024

// ErrCloudKitNotConfigured is returned when a CloudKit operation is attempted
// without the credentials it needs.
var ErrCloudKitNotConfigured = errors.New("CloudKit is not configured")

// cloudKitClient talks to CloudKit Web Services.
//
// The zero value is unusable; build one with NewICloudStorage. An instance
// without complete credentials fails every operation with
// ErrCloudKitNotConfigured rather than writing somewhere unexpected.
type cloudKitClient struct {
	container   string
	environment string
	database    string
	keyID       string
	privateKey  *ecdsa.PrivateKey
	baseURL     string
	client      *http.Client
}

// cloudKitOptions holds the credentials CloudKit requires.
type cloudKitOptions struct {
	// Container is the CloudKit container identifier, e.g. iCloud.com.example.app.
	Container string
	// Environment is "development" or "production".
	Environment string
	// Database is "public", "private", or "shared".
	Database string
	// KeyID is the server-to-server key identifier from CloudKit Dashboard.
	KeyID string
	// PrivateKey is the PEM-encoded ECDSA P-256 private key for that key.
	PrivateKey string
	// BaseURL overrides the CloudKit endpoint, for testing.
	BaseURL string
}

// newCloudKitClient builds a CloudKit client.
//
// It returns an error if the container, key ID or private key is missing or
// malformed. A partially configured client is not useful and failing here
// surfaces the problem at startup instead of at the first backup.
func newCloudKitClient(cfg cloudKitOptions) (*cloudKitClient, error) {
	if strings.TrimSpace(cfg.Container) == "" {
		return nil, fmt.Errorf("%w: container is required", ErrCloudKitNotConfigured)
	}
	if strings.TrimSpace(cfg.KeyID) == "" {
		return nil, fmt.Errorf("%w: server-to-server key ID is required", ErrCloudKitNotConfigured)
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, fmt.Errorf("%w: server-to-server private key is required", ErrCloudKitNotConfigured)
	}

	privateKey, err := parseECPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	environment := cfg.Environment
	if environment == "" {
		environment = "production"
	}
	if environment != "development" && environment != "production" {
		return nil, fmt.Errorf("%w: environment must be development or production, got %q",
			ErrCloudKitNotConfigured, environment)
	}

	database := cfg.Database
	if database == "" {
		database = "private"
	}
	switch database {
	case "public", "private", "shared":
	default:
		return nil, fmt.Errorf("%w: database must be public, private or shared, got %q",
			ErrCloudKitNotConfigured, database)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = cloudKitBaseURL
	}

	return &cloudKitClient{
		container:   cfg.Container,
		environment: environment,
		database:    database,
		keyID:       cfg.KeyID,
		privateKey:  privateKey,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		// A bounded timeout keeps a hung provider from holding a request
		// goroutine open indefinitely.
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Configured reports whether the storage has the credentials it needs.
func (i *cloudKitClient) Configured() bool {
	return i != nil && i.privateKey != nil && i.container != "" && i.keyID != ""
}

// parseECPrivateKey accepts a PKCS#8 or SEC1 PEM block.
func parseECPrivateKey(pemData string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("%w: private key is not valid PEM", ErrCloudKitNotConfigured)
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%w: private key is not ECDSA", ErrCloudKitNotConfigured)
		}
		return ecKey, nil
	}

	if ecKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ecKey, nil
	}

	return nil, fmt.Errorf("%w: private key could not be parsed as PKCS#8 or SEC1",
		ErrCloudKitNotConfigured)
}

// subpath builds the URL subpath for a CloudKit operation.
//
// The subpath is part of the signed message, so it must match the request path
// exactly.
func (i *cloudKitClient) subpath(operation string) string {
	return fmt.Sprintf("/database/1/%s/%s/%s/%s",
		i.container, i.environment, i.database, operation)
}

// sign builds the CloudKit authentication headers for a request body.
//
// Apple's scheme concatenates the ISO 8601 date, the base64-encoded SHA-256 of
// the body, and the URL subpath with colons, then signs that string with ECDSA
// P-256. Signed requests are valid for ten minutes, so the timestamp is
// generated per request.
func (i *cloudKitClient) sign(body []byte, subpath string) (map[string]string, error) {
	if !i.Configured() {
		return nil, ErrCloudKitNotConfigured
	}

	// ISO 8601 without milliseconds.
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	digest := sha256.Sum256(body)
	hash := base64.StdEncoding.EncodeToString(digest[:])

	message := date + ":" + hash + ":" + subpath

	signature, err := ecdsa.SignASN1(rand.Reader, i.privateKey, []byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to sign CloudKit request: %w", err)
	}

	return map[string]string{
		"X-Apple-CloudKit-Request-KeyID":       i.keyID,
		"X-Apple-CloudKit-Request-ISO8601Date": date,
		"X-Apple-CloudKit-Request-SignatureV1": base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// call performs a signed CloudKit request and returns the decoded response.
func (i *cloudKitClient) call(ctx context.Context, operation string, body any) (map[string]any, error) {
	if !i.Configured() {
		return nil, ErrCloudKitNotConfigured
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CloudKit request: %w", err)
	}

	subpath := i.subpath(operation)
	headers, err := i.sign(encoded, subpath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.baseURL+subpath, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CloudKit request failed: %w", err)
	}
	defer resp.Body.Close()

	// Bound the response so a hostile or misconfigured endpoint cannot exhaust
	// memory.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*cloudKitMaxRecordBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read CloudKit response: %w", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("CloudKit returned HTTP %d with an unreadable body", resp.StatusCode)
	}

	// CloudKit reports failures in the body, sometimes with a 200 status.
	if code, ok := parsed["serverErrorCode"].(string); ok {
		reason, _ := parsed["reason"].(string)
		if code == "NOT_FOUND" {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("CloudKit error %s: %s", code, reason)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("CloudKit returned HTTP %d", resp.StatusCode)
	}

	return parsed, nil
}

// ErrBackupNotFound is returned when a backup does not exist in the provider.
var ErrBackupNotFound = errors.New("backup not found")

// Upload stores data under filename and returns the CloudKit record name.
func (i *cloudKitClient) Upload(ctx context.Context, data []byte, filename string) (string, error) {
	if len(data) > cloudKitMaxRecordBytes {
		return "", fmt.Errorf("backup is %d bytes, which exceeds the %d byte CloudKit limit",
			len(data), cloudKitMaxRecordBytes)
	}

	recordName, err := cloudKitRecordName(filename)
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256(data)
	checksum := hex.EncodeToString(digest[:])

	// forceUpdate overwrites an existing record with the same name, so a retry
	// after a dropped response does not fail.
	body := map[string]any{
		"operations": []map[string]any{{
			"operationType": "forceUpdate",
			"record": map[string]any{
				"recordType": cloudKitRecordType,
				"recordName": recordName,
				"fields": map[string]any{
					"payload":   map[string]any{"value": base64.StdEncoding.EncodeToString(data)},
					"checksum":  map[string]any{"value": checksum},
					"filename":  map[string]any{"value": filename},
					"createdAt": map[string]any{"value": time.Now().UTC().Format(time.RFC3339)},
				},
			},
		}},
	}

	response, err := i.call(ctx, "records/modify", body)
	if err != nil {
		return "", err
	}

	// records/modify reports per-record failures inside the response.
	records, _ := response["records"].([]any)
	if len(records) == 0 {
		return "", errors.New("CloudKit did not confirm the record was saved")
	}
	if record, ok := records[0].(map[string]any); ok {
		if code, ok := record["serverErrorCode"].(string); ok {
			return "", fmt.Errorf("CloudKit rejected the record: %s", code)
		}
	}

	return recordName, nil
}

// Download retrieves the data stored under filename.
func (i *cloudKitClient) Download(ctx context.Context, filename string) ([]byte, error) {
	recordName, err := cloudKitRecordName(filename)
	if err != nil {
		return nil, err
	}

	response, err := i.call(ctx, "records/lookup", map[string]any{
		"records": []map[string]any{{"recordName": recordName}},
	})
	if err != nil {
		return nil, err
	}

	records, _ := response["records"].([]any)
	if len(records) == 0 {
		return nil, ErrBackupNotFound
	}

	record, ok := records[0].(map[string]any)
	if !ok {
		return nil, errors.New("CloudKit returned a malformed record")
	}

	fields, ok := record["fields"].(map[string]any)
	if !ok {
		return nil, errors.New("CloudKit record has no fields")
	}

	payload, ok := fieldValue(fields, "payload")
	if !ok {
		return nil, errors.New("CloudKit record has no payload")
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("CloudKit payload is not valid base64: %w", err)
	}

	// A checksum mismatch means the stored bytes are not the bytes that were
	// uploaded, so the backup is not trustworthy.
	if expected, ok := fieldValue(fields, "checksum"); ok && expected != "" {
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != expected {
			return nil, errors.New("CloudKit record failed its checksum")
		}
	}

	return data, nil
}

// Delete removes the record stored under filename.
func (i *cloudKitClient) Delete(ctx context.Context, filename string) error {
	recordName, err := cloudKitRecordName(filename)
	if err != nil {
		return err
	}

	_, err = i.call(ctx, "records/modify", map[string]any{
		"operations": []map[string]any{{
			"operationType": "delete",
			"record": map[string]any{
				"recordType": cloudKitRecordType,
				"recordName": recordName,
			},
		}},
	})
	if err != nil && errors.Is(err, ErrBackupNotFound) {
		// Deleting something already gone is not a failure.
		return nil
	}
	return err
}

// RecordURL returns the CloudKit URL for a record.
//
// CloudKit does not issue pre-signed media links for records, so callers that
// need the bytes must use Download.
func (i *cloudKitClient) RecordURL(filename string) (string, error) {
	recordName, err := cloudKitRecordName(filename)
	if err != nil {
		return "", err
	}
	if !i.Configured() {
		return "", ErrCloudKitNotConfigured
	}

	return fmt.Sprintf("%s/database/1/%s/%s/%s/records/resolve?recordName=%s",
		i.baseURL, i.container, i.environment, i.database, recordName), nil
}

// fieldValue reads a CloudKit field's value, which is wrapped as {"value": ...}.
func fieldValue(fields map[string]any, name string) (string, bool) {
	field, ok := fields[name].(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := field["value"].(string)
	return value, ok
}

// cloudKitRecordName derives a CloudKit-safe record name from a filename.
//
// CloudKit record names allow letters, digits, dot, underscore and hyphen, and
// must not begin with an underscore. Validating here keeps a caller-supplied
// name from being interpolated into a URL path unchecked.
func cloudKitRecordName(filename string) (string, error) {
	if filename == "" || len(filename) > 255 {
		return "", errors.New("backup name must be between 1 and 255 characters")
	}
	if strings.HasPrefix(filename, "_") {
		return "", errors.New("backup name must not begin with an underscore")
	}

	var builder strings.Builder
	builder.Grow(len(filename))
	for _, r := range filename {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		default:
			return "", fmt.Errorf("backup name contains an unsupported character %q", r)
		}
	}

	return builder.String(), nil
}
