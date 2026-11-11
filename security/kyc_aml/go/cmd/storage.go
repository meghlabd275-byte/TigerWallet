// Identity-document storage.
//
// Documents are stored through a configured backend. When no backend is
// configured the upload fails and the caller is told why, so the service never
// reports a document as stored when nothing was written.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// maxDocumentBytes caps a single upload at 10 MiB.
const maxDocumentBytes = 10 << 20

// documentBackend stores document bytes and returns a retrievable URL.
type documentBackend interface {
	Store(ctx context.Context, applicationID uuid.UUID, docType, side, base64Data string) (string, error)
}

// documentStorage is the backend selected at start-up.
var documentStorage documentBackend = &unconfiguredStorage{}

// unconfiguredStorage fails closed when no backend is configured.
type unconfiguredStorage struct{}

func (u *unconfiguredStorage) Store(context.Context, uuid.UUID, string, string, string) (string, error) {
	return "", fmt.Errorf("no document storage backend is configured (set DOCUMENT_STORAGE_DIR or DOCUMENT_STORAGE_URL)")
}

// filesystemStorage writes documents to a local directory served by the
// operator's web tier.
type filesystemStorage struct {
	dir       string
	publicURL string
}

func (f *filesystemStorage) Store(ctx context.Context, applicationID uuid.UUID, docType, side, base64Data string) (string, error) {
	data, err := decodeDocument(base64Data)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s_%s_%s.bin", applicationID, docType, side)
	fullPath := filepath.Join(f.dir, applicationID.String())
	if err := os.MkdirAll(fullPath, 0o750); err != nil {
		return "", fmt.Errorf("create document directory: %w", err)
	}

	target := filepath.Join(fullPath, name)
	if err := os.WriteFile(target, data, 0o640); err != nil {
		return "", fmt.Errorf("write document: %w", err)
	}

	base := strings.TrimRight(f.publicURL, "/")
	if base == "" {
		return target, nil
	}
	return fmt.Sprintf("%s/%s/%s", base, applicationID, name), nil
}

// remoteStorage forwards documents to an object-storage gateway.
type remoteStorage struct {
	endpoint string
	token    string
	client   *http.Client
}

func (r *remoteStorage) Store(ctx context.Context, applicationID uuid.UUID, docType, side, base64Data string) (string, error) {
	if !strings.HasPrefix(r.endpoint, "https://") {
		return "", fmt.Errorf("DOCUMENT_STORAGE_URL must use https")
	}

	data, err := decodeDocument(base64Data)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(map[string]string{
		"application_id": applicationID.String(),
		"type":           docType,
		"side":           side,
		"content":        base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return "", err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, r.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("document storage call failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("document storage returned HTTP %d: %s", resp.StatusCode, truncateForLog(string(raw), 256))
	}

	var decoded struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("decode document storage response: %w", err)
	}
	if strings.TrimSpace(decoded.URL) == "" {
		return "", fmt.Errorf("document storage accepted the upload without returning a URL")
	}
	return decoded.URL, nil
}

// configureDocumentStorage selects a backend from the environment.
func configureDocumentStorage() error {
	dir := strings.TrimSpace(os.Getenv("DOCUMENT_STORAGE_DIR"))
	endpoint := strings.TrimSpace(os.Getenv("DOCUMENT_STORAGE_URL"))

	switch {
	case dir != "":
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create DOCUMENT_STORAGE_DIR: %w", err)
		}
		documentStorage = &filesystemStorage{
			dir:       dir,
			publicURL: os.Getenv("DOCUMENT_STORAGE_PUBLIC_URL"),
		}
		return nil
	case endpoint != "":
		documentStorage = &remoteStorage{
			endpoint: endpoint,
			token:    os.Getenv("DOCUMENT_STORAGE_TOKEN"),
			client:   &http.Client{Timeout: 60 * time.Second},
		}
		return nil
	default:
		documentStorage = &unconfiguredStorage{}
		return nil
	}
}

// decodeDocument validates and decodes base64 document data.
func decodeDocument(base64Data string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("document data is not valid base64: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("document data is empty")
	}
	if len(data) > maxDocumentBytes {
		return nil, fmt.Errorf("document exceeds the %d byte limit", maxDocumentBytes)
	}
	return data, nil
}

// hashDocumentData returns the SHA-256 of the decoded document, used to detect
// duplicate submissions and to record integrity.
func hashDocumentData(base64Data string) (string, error) {
	data, err := decodeDocument(base64Data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
