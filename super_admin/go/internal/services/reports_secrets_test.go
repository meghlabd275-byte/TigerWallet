package services

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- credential encryption --------------------------------------------------

func TestSecretBoxRequiresKey(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "")

	box := newSecretBox()
	if box.configured() {
		t.Fatal("a box without a key must not report itself configured")
	}
	if _, err := box.seal("secret"); err != errSecretsKeyNotConfigured {
		t.Fatalf("seal err = %v, want errSecretsKeyNotConfigured", err)
	}
}

func TestSecretBoxRoundTrip(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "unit-test-key")

	box := newSecretBox()
	plaintext := "sk-live-abcdef0123456789"

	sealed, err := box.seal(plaintext)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed == plaintext {
		t.Fatal("sealed value must not equal the plaintext")
	}
	if !strings.HasPrefix(sealed, sealedPrefix) {
		t.Fatalf("sealed value %q is missing the prefix", sealed)
	}
	if strings.Contains(sealed, plaintext) {
		t.Fatal("ciphertext must not contain the plaintext")
	}

	opened, err := box.open(sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != plaintext {
		t.Fatalf("open = %q, want %q", opened, plaintext)
	}
}

// TestSecretBoxNonceIsRandom ensures the same plaintext encrypts differently,
// so identical keys across integrations are not visible in the database.
func TestSecretBoxNonceIsRandom(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "unit-test-key")
	box := newSecretBox()

	first, err := box.seal("same-value")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	second, err := box.seal("same-value")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if first == second {
		t.Fatal("two seals of the same plaintext must differ")
	}
}

func TestSecretBoxRejectsWrongKey(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "key-one")
	sealed, err := newSecretBox().seal("secret")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "key-two")
	if _, err := newSecretBox().open(sealed); err != errCiphertextInvalid {
		t.Fatalf("open err = %v, want errCiphertextInvalid", err)
	}
}

func TestSecretBoxRejectsTamperedCiphertext(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "unit-test-key")
	box := newSecretBox()

	sealed, err := box.seal("secret")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	// Flip a character in the base64 payload; GCM must reject it.
	body := []byte(sealed)
	last := len(body) - 1
	if body[last] == 'A' {
		body[last] = 'B'
	} else {
		body[last] = 'A'
	}
	if _, err := box.open(string(body)); err == nil {
		t.Fatal("a tampered ciphertext must not decrypt")
	}
}

// TestSecretBoxProtectIdempotent covers re-saving a config that was read back,
// which must not encrypt the ciphertext a second time.
func TestSecretBoxProtectIdempotent(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "unit-test-key")
	box := newSecretBox()

	sealed, err := box.protect("secret")
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	again, err := box.protect(sealed)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	if again != sealed {
		t.Fatal("protect must leave an already-sealed value unchanged")
	}

	if empty, err := box.protect(""); err != nil || empty != "" {
		t.Fatalf("protect(\"\") = %q, %v; want empty and no error", empty, err)
	}
}

// TestSecretBoxRevealToleratesPlaintext covers rows written before encryption
// was introduced.
func TestSecretBoxRevealToleratesPlaintext(t *testing.T) {
	t.Setenv("SUPER_ADMIN_SECRETS_KEY", "unit-test-key")
	box := newSecretBox()

	value, err := box.reveal("legacy-plaintext")
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if value != "legacy-plaintext" {
		t.Fatalf("reveal = %q, want the value unchanged", value)
	}
}

// --- CSV encoding -----------------------------------------------------------

func parseCSV(t *testing.T, content []byte) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(string(content))).ReadAll()
	if err != nil {
		t.Fatalf("generated CSV does not parse: %v", err)
	}
	return records
}

func TestEncodeCSVFromSlice(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "1", "email": "a@example.com"},
		{"id": "2", "email": "b@example.com"},
	}

	records := parseCSV(t, mustEncodeCSV(t, data))
	if len(records) != 3 {
		t.Fatalf("got %d rows, want a header plus 2 rows", len(records))
	}
	if records[0][0] != "email" || records[0][1] != "id" {
		t.Fatalf("header = %v, want sorted columns", records[0])
	}
	if records[1][0] != "a@example.com" {
		t.Fatalf("first row = %v", records[1])
	}
}

func TestEncodeCSVFromNestedObject(t *testing.T) {
	data := map[string]interface{}{
		"total": 3,
		"users": []interface{}{
			map[string]interface{}{"id": "1"},
			map[string]interface{}{"id": "2"},
		},
	}

	records := parseCSV(t, mustEncodeCSV(t, data))
	if len(records) != 3 {
		t.Fatalf("got %d rows, want the list-valued key expanded to 2 rows", len(records))
	}
}

// TestEncodeCSVQuotesSpecialCharacters ensures a value containing a separator
// does not shift the columns.
func TestEncodeCSVQuotesSpecialCharacters(t *testing.T) {
	data := []map[string]interface{}{
		{"note": "a,b", "other": "c\"d"},
	}

	records := parseCSV(t, mustEncodeCSV(t, data))
	if records[1][0] != "a,b" {
		t.Fatalf("comma-containing value = %q, want it preserved in one field", records[1][0])
	}
	if records[1][1] != "c\"d" {
		t.Fatalf("quote-containing value = %q", records[1][1])
	}
}

// TestEncodeCSVNeutralisesFormulaInjection covers spreadsheet formula injection:
// an exported value starting with = must not be executed when opened.
func TestEncodeCSVNeutralisesFormulaInjection(t *testing.T) {
	data := []map[string]interface{}{
		{"name": "=HYPERLINK(\"http://evil.test\")"},
		{"name": "+1+1"},
		{"name": "-2+3"},
		{"name": "@SUM(A1)"},
	}

	records := parseCSV(t, mustEncodeCSV(t, data))
	for i, record := range records[1:] {
		if record[0] == "" || !strings.HasPrefix(record[0], "'") {
			t.Fatalf("row %d value %q was not neutralised", i, record[0])
		}
	}
}

func TestEncodeCSVEmptyData(t *testing.T) {
	content, err := encodeCSV([]map[string]interface{}{})
	if err != nil {
		t.Fatalf("encodeCSV: %v", err)
	}
	if len(content) != 0 {
		t.Fatalf("empty input produced %q", content)
	}
}

func TestFormatCSVCellFormatsTimeAndNumbers(t *testing.T) {
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := formatCSVCell(stamp); got != "2026-01-02T03:04:05Z" {
		t.Fatalf("time cell = %q", got)
	}
	if got := formatCSVCell(12.5); got != "12.5" {
		t.Fatalf("number cell = %q", got)
	}
	if got := formatCSVCell(nil); got != "" {
		t.Fatalf("nil cell = %q, want empty", got)
	}
}

func mustEncodeCSV(t *testing.T, data interface{}) []byte {
	t.Helper()
	content, err := encodeCSV(data)
	if err != nil {
		t.Fatalf("encodeCSV: %v", err)
	}
	return content
}

// --- report file writing ----------------------------------------------------

func TestWriteReportFileRequiresDirectory(t *testing.T) {
	t.Setenv("SUPER_ADMIN_REPORT_DIR", "")

	if _, err := writeReportFile("abc", "json", []byte("{}")); err == nil {
		t.Fatal("writing without a configured directory must fail")
	}
}

func TestWriteReportFileCreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUPER_ADMIN_REPORT_DIR", dir)

	path, err := writeReportFile("11111111-2222-3333-4444-555555555555", "csv", []byte("a,b\n"))
	if err != nil {
		t.Fatalf("writeReportFile: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(content) != "a,b\n" {
		t.Fatalf("content = %q", content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Reports hold user and financial data.
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("permissions = %o, want 600", perm)
	}
}

// TestWriteReportFileStaysInDirectory ensures a crafted id cannot escape.
func TestWriteReportFileStaysInDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUPER_ADMIN_REPORT_DIR", dir)

	if _, err := writeReportFile("../../etc/passwd", "json", []byte("{}")); err == nil {
		t.Fatal("a traversal id must be rejected")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("directory should be empty, found %d entries", len(entries))
	}
}

func TestWriteCSVReportProducesParsableFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUPER_ADMIN_REPORT_DIR", dir)

	service := NewReportService()
	path, err := service.writeCSVReport("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", []map[string]interface{}{
		{"id": "1"},
	})
	if err != nil {
		t.Fatalf("writeCSVReport: %v", err)
	}
	if filepath.Ext(path) != ".csv" {
		t.Fatalf("path = %q, want a .csv file", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if records := parseCSV(t, content); len(records) != 2 {
		t.Fatalf("got %d rows, want a header plus 1 row", len(records))
	}
}

// TestGenerateGenericReportFails ensures an unknown report type does not
// produce a placeholder document.
func TestGenerateGenericReportFails(t *testing.T) {
	if _, err := NewReportService().generateGenericReport(nil, nil); err == nil {
		t.Fatal("an unsupported report type must return an error")
	}
}
