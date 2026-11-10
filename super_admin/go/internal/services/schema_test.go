package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/tigerwallet/super-admin/internal/config"
	"github.com/tigerwallet/super-admin/internal/database"
)

// Every SQL statement in the handler layer is prepared against the real schema.
// PostgreSQL resolves columns and tables at prepare time, so a query naming a
// column that does not exist fails here instead of failing for an operator in
// production. Set TEST_DATABASE_URL to run it; without that the test skips.
func TestHandlerStatementsMatchSchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping SQL schema test")
	}

	cfg := &config.Config{
		DatabaseURL:      dsn,
		DatabaseMaxConns: 4,
		DatabaseMinConns: 1,
	}
	if err := database.Initialize(cfg); err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Statements are gathered from the handler files rather than a fixture, so
	// a query added later is checked automatically.
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	files, err := goSourceFiles(moduleRoot)
	if err != nil {
		t.Fatalf("list source files: %v", err)
	}
	statements, err := collectStatements(files)
	if err != nil {
		t.Fatalf("collect statements: %v", err)
	}
	if len(statements) < 50 {
		t.Fatalf("only %d statements collected; the extractor is probably broken", len(statements))
	}

	var failures []string
	for i, stmt := range statements {
		name := fmt.Sprintf("chk_%d", i)
		if _, err := database.Pool.Exec(ctx, "PREPARE "+name+" AS "+stmt); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "could not determine data type") {
				// A parameter PostgreSQL cannot type on its own is not a schema
				// mismatch.
				continue
			}
			failures = append(failures, msg+"\n    SQL: "+firstLine(stmt))
		}
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("%d of %d statements do not match the schema:\n\n%s",
			len(failures), len(statements), strings.Join(failures, "\n\n"))
	}
}

func firstLine(stmt string) string {
	if i := strings.IndexByte(stmt, '\n'); i >= 0 {
		return strings.TrimSpace(stmt[:i]) + " ..."
	}
	return strings.TrimSpace(stmt)
}

// endsIncomplete reports whether the statement stops on a keyword that needs
// more input, which happens when a query is built by concatenation.
var danglingTail = regexp.MustCompile(`(?i)\b(set|select|from|where|and|or|values|into|join|order\s+by|group\s+by|limit|returning|,\s*)$`)

func endsIncomplete(stmt string) bool {
	return danglingTail.MatchString(strings.TrimSpace(stmt))
}

var (
	backtickSQL = regexp.MustCompile("(?s)`\\s*((?:SELECT|INSERT|UPDATE|DELETE)[^`]*)`")
	quotedSQL   = regexp.MustCompile(`"(\\s*(?:SELECT|INSERT|UPDATE|DELETE)[^"]{20,})"`)
	// A bare statement assembled as "SELECT " + columns + " FROM table" leaves
	// the suffix in a separate literal, so those fragments are skipped.
	isWholeStatement = regexp.MustCompile(`(?is)^(select|insert|update|delete)\b.*\b(from|into|set|values)\b`)
)

// goSourceFiles returns every non-test Go file under root, skipping vendor and
// generated directories. Test files are excluded because their SQL belongs to
// fixtures rather than to the handlers under check.
func goSourceFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// collectStatements pulls SQL out of the given Go files. Queries built by
// concatenating a constant are reconstructed from the constant and the literal
// that follows it.
func collectStatements(files []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string

	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		text := string(src)

		var candidates []string
		for _, m := range backtickSQL.FindAllStringSubmatch(text, -1) {
			candidates = append(candidates, m[1])
		}
		for _, m := range quotedSQL.FindAllStringSubmatch(text, -1) {
			candidates = append(candidates, m[1])
		}

		// Constants used in concatenated queries, e.g. slaPolicyColumns.
		constants := map[string]string{}
		for _, m := range regexp.MustCompile("(?s)([A-Za-z][A-Za-z0-9]*)\\s*=\\s*`([^`]+)`").
			FindAllStringSubmatch(text, -1) {
			constants[m[1]] = m[2]
		}
		// "SELECT " + constName + " FROM x" -> substitute the constant.
		for name, value := range constants {
			needle := `+ ` + name + ` +`
			if !strings.Contains(text, needle) {
				continue
			}
			for _, m := range regexp.MustCompile(`"((?:SELECT|INSERT|UPDATE|DELETE)[^"]*)"\s*\+\s*`+name).
				FindAllStringSubmatch(text, -1) {
				candidates = append(candidates, m[1]+" "+value)
			}
		}

		for _, c := range candidates {
			stmt := strings.TrimSpace(c)
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.Count(stmt, ";") > 0 {
				continue
			}
			if !isWholeStatement.MatchString(stmt) {
				continue
			}
			// A query built with fmt.Sprintf cannot be prepared until the caller
			// supplies the table name.
			if strings.Contains(stmt, "%s") || strings.Contains(stmt, "%d") {
				continue
			}
			if seen[stmt] {
				continue
			}
			// A statement that stops on a dangling keyword is the prefix of a
			// query assembled with concatenation, not a statement of its own.
			if endsIncomplete(stmt) {
				continue
			}
			seen[stmt] = true
			out = append(out, stmt)
		}
	}
	return out, nil
}

// The extractor is pure string work, so its behaviour is pinned without a
// database.
func TestCollectStatements(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "handlers.go")
	src := "package x\n" +
		"const cols = `id, name`\n" +
		"func a() {\n" +
		"\tdbExec(c, `SELECT id FROM users WHERE id=$1`)\n" +
		"\tdbQuery(c, \"SELECT \"+cols+\" FROM accounts\")\n" +
		"\tdbExec(c, `FROM accounts`)\n" +
		"\tdbExec(c, `UPDATE users SET name=$1 WHERE id=$2`)\n" +
		"}\n"
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stmts, err := collectStatements([]string{file})
	if err != nil {
		t.Fatalf("collectStatements: %v", err)
	}

	joined := strings.Join(stmts, "\n---\n")
	for _, want := range []string{
		"SELECT id FROM users WHERE id=$1",
		"UPDATE users SET name=$1 WHERE id=$2",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing statement %q in:\n%s", want, joined)
		}
	}
	// A bare fragment must not be treated as a statement, and a query assembled
	// from a separate column constant is left out rather than half-built.
	for _, stmt := range stmts {
		if strings.HasPrefix(strings.TrimSpace(stmt), "FROM ") {
			t.Errorf("collected a fragment: %q", stmt)
		}
		if strings.Contains(stmt, "FROM accounts") {
			t.Errorf("collected an incomplete concatenated query: %q", stmt)
		}
	}
}

// A tiny guard so an accidentally empty file list cannot make the schema test
// pass vacuously.
func TestCollectStatementsFindsRealQueries(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	stmts, err := collectStatements([]string{filepath.Join(moduleRoot, "main.go")})
	if err != nil {
		t.Fatalf("collectStatements: %v", err)
	}
	if len(stmts) < 50 {
		t.Fatalf("collected only %d statements from main.go", len(stmts))
	}
}
