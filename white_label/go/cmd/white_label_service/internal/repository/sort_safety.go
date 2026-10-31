package repository

import (
	"errors"
	"regexp"
	"strings"
)

// ORDER BY cannot use bind parameters: PostgreSQL resolves a sort column at
// parse time, so the identifier has to be part of the SQL text. The list
// queries therefore interpolated filter.SortBy and filter.Order straight into
// the statement, which let a caller put arbitrary SQL there.
//
// The rule enforced here is:
//
//   - every column must be on the caller's allow-list, so an injected expression
//     cannot name anything the endpoint was not written to expose;
//   - every column must look like a plain identifier, optionally qualified as
//     table.column, so even an allow-listed name cannot carry a subquery,
//     comment, comma or quote;
//   - the direction must be ASC or DESC and nothing else.
//
// The first rule is the security boundary, the second stops a future allow-list
// entry such as "sort_order, created_at" from being taken as two expressions,
// and the third blocks the classic "ORDER BY id DESC; DROP TABLE ..." payload.

var sortIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// errInvalidSortIdentifier is returned for a column that is not a plain
// identifier.
var errInvalidSortIdentifier = errors.New("sort column must be a plain identifier, optionally qualified as table.column")

// errInvalidSortDirection is returned for anything other than ASC or DESC.
var errInvalidSortDirection = errors.New("sort direction must be ASC or DESC")

// normalizeSortDirection upper-cases and validates the requested direction,
// falling back to fallback when the caller did not ask for one.
func normalizeSortDirection(requested, fallback string) (string, error) {
	d := strings.TrimSpace(requested)
	if d == "" {
		return fallback, nil
	}
	d = strings.ToUpper(d)
	if d != "ASC" && d != "DESC" {
		return "", errInvalidSortDirection
	}
	return d, nil
}

// safeSortColumns validates each requested column against allowed and returns
// the ORDER BY clause body. Columns are de-duplicated so a caller cannot force
// the database to sort on the same expression repeatedly.
//
// When the caller asks for nothing, fallback (which must already be a validated
// literal) is returned unchanged.
func safeSortColumns(requested []string, allowed map[string]bool, fallback string) (string, error) {
	if len(requested) == 0 {
		return fallback, nil
	}
	seen := make(map[string]bool, len(requested))
	cols := make([]string, 0, len(requested))
	for _, raw := range requested {
		col := strings.TrimSpace(raw)
		if col == "" {
			continue
		}
		if !allowed[col] {
			return "", errors.New("sort column is not allowed: " + col)
		}
		if !sortIdentifierPattern.MatchString(col) {
			return "", errInvalidSortIdentifier
		}
		if seen[col] {
			continue
		}
		seen[col] = true
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return fallback, nil
	}
	return strings.Join(cols, ", "), nil
}

// parseSortBy turns the request's sortBy string into a column list. A comma is
// accepted as a separator so callers can ask for a tie-breaker, but each
// resulting part is still validated individually.
func parseSortBy(sortBy string) []string {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		return nil
	}
	parts := strings.Split(sortBy, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// --- Per-query allow-lists -------------------------------------------------
//
// Each list names exactly the columns the corresponding SELECT exposes. A
// column that is not selected cannot be sorted on, which also keeps the error
// messages from leaking the schema.

var clientSortColumns = map[string]bool{
	"id": true, "name": true, "domain": true, "subdomain": true,
	"status": true, "plan": true, "max_users": true, "current_users": true,
	"fee_percent": true, "created_at": true, "updated_at": true,
	"approved_at": true, "expires_at": true,
}

var adminSortColumns = map[string]bool{
	"id": true, "client_id": true, "email": true, "name": true, "role": true,
	"status": true, "two_factor_enabled": true, "last_login": true,
	"login_attempts": true, "locked_until": true, "created_at": true,
	"updated_at": true,
}

var productSortColumns = map[string]bool{
	"id": true, "client_id": true, "name": true, "type": true, "status": true,
	"fee": true, "min_deposit": true, "max_deposit": true, "min_withdrawal": true,
	"max_withdrawal": true, "sort_order": true, "created_at": true,
	"updated_at": true,
}

// buildOrderBy validates the caller's sort request for one of the list queries.
// It returns the complete "ORDER BY ..." fragment, or an error describing what
// was rejected.
func buildOrderBy(sortBy, order string, allowed map[string]bool, fallback string) (string, error) {
	cols, err := safeSortColumns(parseSortBy(sortBy), allowed, fallback)
	if err != nil {
		return "", err
	}
	dir, err := normalizeSortDirection(order, "DESC")
	if err != nil {
		return "", err
	}
	return "ORDER BY " + cols + " " + dir, nil
}
