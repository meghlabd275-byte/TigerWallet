package repository

import (
	"os"
	"strings"
	"testing"
)

// filter.SortBy and filter.Order arrived from the request and were interpolated
// straight into the SQL text:
//
//	fmt.Sprintf(`... %s ORDER BY %s %s LIMIT $%d OFFSET $%d`, baseQuery, sortBy, order, ...)
//
// An ORDER BY column cannot be a bind parameter, so these tests pin the
// validation that replaced the raw interpolation.

func TestSafeSortColumnsAcceptsAllowListedColumns(t *testing.T) {
	got, err := safeSortColumns([]string{"created_at"}, clientSortColumns, "created_at")
	if err != nil {
		t.Fatalf("rejected an allow-listed column: %v", err)
	}
	if got != "created_at" {
		t.Errorf("expected created_at, got %q", got)
	}
}

func TestSafeSortColumnsRejectsUnknownColumn(t *testing.T) {
	for _, bad := range []string{
		"password_hash",
		"pg_shadow",
		"1",
		"(SELECT 1)",
	} {
		if _, err := safeSortColumns([]string{bad}, clientSortColumns, "created_at"); err == nil {
			t.Errorf("accepted %q as a sort column", bad)
		}
	}
}

// An allow-listed name still has to look like a plain identifier. Without this
// rule a future allow-list entry such as "sort_order, created_at" would be taken
// as two expressions and the injection would come back through the front door.
func TestSafeSortColumnsRejectsNonIdentifierEvenIfAllowListed(t *testing.T) {
	allowed := map[string]bool{
		"created_at":                         true,
		"created_at; DROP TABLE clients":     true,
		"created_at, (SELECT password_hash)": true,
		"created_at --":                      true,
		"created_at)":                        true,
	}
	for col := range allowed {
		if col == "created_at" {
			continue
		}
		if _, err := safeSortColumns([]string{col}, allowed, "created_at"); err == nil {
			t.Errorf("accepted %q: it is allow-listed but not a plain identifier", col)
		}
	}
}

// A qualified name is a plain identifier too, so table.column is allowed.
func TestSafeSortColumnsAllowsQualifiedIdentifier(t *testing.T) {
	allowed := map[string]bool{"l.created_at": true}
	got, err := safeSortColumns([]string{"l.created_at"}, allowed, "l.created_at")
	if err != nil {
		t.Fatalf("rejected a qualified identifier: %v", err)
	}
	if got != "l.created_at" {
		t.Errorf("expected l.created_at, got %q", got)
	}
}

func TestSafeSortColumnsDeduplicatesAndKeepsOrder(t *testing.T) {
	got, err := safeSortColumns([]string{"sort_order", "created_at", "sort_order"}, productSortColumns, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sort_order, created_at" {
		t.Errorf("expected \"sort_order, created_at\", got %q", got)
	}
}

func TestSafeSortColumnsFallsBackWhenNothingRequested(t *testing.T) {
	got, err := safeSortColumns(nil, clientSortColumns, "created_at")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "created_at" {
		t.Errorf("expected the fallback, got %q", got)
	}
}

func TestNormalizeSortDirection(t *testing.T) {
	for in, want := range map[string]string{"": "DESC", "asc": "ASC", "DESC": "DESC", " Asc ": "ASC"} {
		got, err := normalizeSortDirection(in, "DESC")
		if err != nil {
			t.Errorf("normalizeSortDirection(%q) returned %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalizeSortDirection(%q) = %q, want %q", in, got, want)
		}
	}
	// The direction is appended after the columns, so anything with a statement
	// separator or a subquery has to be refused.
	for _, bad := range []string{
		"DESC; DROP TABLE white_label_clients",
		"DESC, (SELECT 1)",
		"DESC --",
		"RANDOM()",
	} {
		if _, err := normalizeSortDirection(bad, "DESC"); err == nil {
			t.Errorf("accepted %q as a sort direction", bad)
		}
	}
}

func TestBuildOrderBy(t *testing.T) {
	got, err := buildOrderBy("created_at", "asc", clientSortColumns, "created_at")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ORDER BY created_at ASC" {
		t.Errorf("got %q", got)
	}

	// The classic payload, end to end.
	if _, err := buildOrderBy("id", "DESC; DROP TABLE white_label_clients", clientSortColumns, "created_at"); err == nil {
		t.Error("accepted a stacked statement in the direction")
	}
	if _, err := buildOrderBy("id) FROM white_label_clients --", "DESC", clientSortColumns, "created_at"); err == nil {
		t.Error("accepted an injected expression as the column")
	}

	// Nothing requested: the caller's fallback, with the direction applied.
	got, err = buildOrderBy("", "", productSortColumns, "sort_order, created_at")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ORDER BY sort_order, created_at DESC" {
		t.Errorf("got %q", got)
	}
}

// Every allow-list must name columns the matching SELECT actually returns,
// otherwise the endpoint would 400 on a sort it should support.
func TestSortAllowListsMatchTheSelectedColumns(t *testing.T) {
	src := readRepositorySource(t)
	cases := []struct {
		fn    string
		cols  map[string]bool
		extra []string // columns produced by an expression, not a bare identifier
	}{
		{"ListClients", clientSortColumns, nil},
		{"ListAdmins", adminSortColumns, nil},
		{"ListProducts", productSortColumns, nil},
	}
	for _, tc := range cases {
		body := functionBody(t, src, tc.fn)
		// Anchor on the page-data query, not the COUNT(*) query built earlier in
		// the same function.
		anchor := strings.Index(body, "selectQuery := fmt.Sprintf(`")
		if anchor < 0 {
			t.Fatalf("%s: could not locate selectQuery", tc.fn)
		}
		rest := body[anchor:]
		i := strings.Index(rest, "SELECT ")
		if i < 0 {
			t.Fatalf("%s: could not locate the SELECT list", tc.fn)
		}
		// The FROM clause lives inside baseQuery, so the column list ends at the
		// first format verb.
		rest = rest[i+len("SELECT "):]
		j := strings.Index(rest, "%s")
		if j < 0 {
			t.Fatalf("%s: could not find the end of the SELECT list", tc.fn)
		}
		selected := rest[:j]
		for col := range tc.cols {
			if !strings.Contains(selected, col) {
				t.Errorf("%s allow-list names %q but the SELECT does not return it", tc.fn, col)
			}
		}
	}
}

// Pin that the three list queries route the sort through buildOrderBy and no
// longer interpolate filter.SortBy / filter.Order themselves.
func TestListQueriesNoLongerInterpolateRawSortInput(t *testing.T) {
	src := readRepositorySource(t)
	if strings.Contains(src, "baseQuery, sortBy, order, argCount") {
		t.Error("a list query still interpolates sortBy/order directly into the SQL")
	}
	if strings.Contains(src, "%s ORDER BY %s %s LIMIT") {
		t.Error("a list query still builds ORDER BY with raw format arguments")
	}
	for _, fn := range []string{"ListClients", "ListAdmins", "ListProducts"} {
		if !strings.Contains(functionBody(t, src, fn), "buildOrderBy(") {
			t.Errorf("%s does not route its sort through buildOrderBy", fn)
		}
	}
}

func readRepositorySource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("read repository.go: %v", err)
	}
	return string(b)
}

func functionBody(t *testing.T, src, name string) string {
	t.Helper()
	marker := "func (r *Repository) " + name + "("
	i := strings.Index(src, marker)
	if i < 0 {
		t.Fatalf("%s not found", name)
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}
