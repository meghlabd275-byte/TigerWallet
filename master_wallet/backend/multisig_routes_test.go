package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// paramLiterals returns the set of literal arguments passed to c.Param(...)
// inside the named function declared in file.
func paramLiterals(t *testing.T, file, funcName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	found := map[string]bool{}
	var fn *ast.FuncDecl
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName {
			continue
		}
		fn = fd
	}
	if fn == nil {
		t.Fatalf("%s not declared in %s", funcName, file)
	}
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Param" || len(call.Args) != 1 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		found[v] = true
		return true
	})
	return found
}

// The multisig sub-routes live under /api/v1/master-wallet/:id/multisig, so :id
// is always the MASTER WALLET id. SignMultisigTransaction and
// ExecuteMultisigTransaction read c.Param("id") as the transaction id, and
// GetMultisigTransactions read c.Param("id") as the multisig wallet id, so every
// lookup ran against the wrong column: signing, executing and listing were all
// inoperable. These assertions pin the parameter each handler must read.
func TestMultisigHandlersReadTheRightRouteParam(t *testing.T) {
	const file = "multisig.go"
	cases := []struct {
		handler string
		want    string
		// mustNotReadAsID is the master wallet id, which these handlers must not
		// use as the transaction or multisig-wallet identifier.
		mustNotReadAsID string
	}{
		{"SignMultisigTransaction", "tid", "id"},
		{"ExecuteMultisigTransaction", "tid", "id"},
		{"GetMultisigTransactions", "wid", "id"},
		{"CreateMultisigTransaction", "wid", "id"},
	}
	for _, tc := range cases {
		got := paramLiterals(t, file, tc.handler)
		if !got[tc.want] {
			t.Errorf("%s must read c.Param(%q); it reads %v", tc.handler, tc.want, keys(got))
		}
		if got[tc.mustNotReadAsID] {
			t.Errorf("%s must not read c.Param(%q) - that is the master wallet id, not the %s", tc.handler, tc.mustNotReadAsID, tc.want)
		}
	}
}

// TestMultisigWalletHandlersAreOwnerScoped asserts that every multisig handler
// that takes :id enforces master-wallet ownership before acting on it.
func TestMultisigWalletHandlersAreOwnerScoped(t *testing.T) {
	for _, handler := range []string{
		"CreateMultisigWallet",
		"GetMultisigWallets",
		"CreateMultisigTransaction",
		"GetMultisigTransactions",
		"SignMultisigTransaction",
		"ExecuteMultisigTransaction",
	} {
		src := funcBody(t, "multisig.go", handler)
		if !contains(src, "requireMasterWalletAccess") {
			t.Errorf("%s does not call requireMasterWalletAccess, so it can act on another tenant's master wallet", handler)
		}
	}
}

func funcBody(t *testing.T, file, funcName string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName {
			continue
		}
		return srcBetween(file, fset.Position(fd.Pos()).Offset, fset.Position(fd.End()).Offset)
	}
	t.Fatalf("%s not found", funcName)
	return ""
}

func srcBetween(file string, start, end int) string {
	b, err := readFile(file)
	if err != nil {
		return ""
	}
	if end > len(b) {
		end = len(b)
	}
	return string(b[start:end])
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && indexOf(haystack, needle) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
