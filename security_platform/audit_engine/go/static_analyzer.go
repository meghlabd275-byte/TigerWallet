package main

// static_analyzer.go — a real static analyzer for Solidity source.
//
// Findings are produced only from patterns actually present in the submitted
// source, with the line number and the offending snippet. Nothing is inferred
// from the scan's name, type or size, and an empty result means the analyzer
// found nothing rather than that it did not run.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// StaticFinding is one issue located in the analyzed source.
type StaticFinding struct {
	Title       string
	Description string
	Category    string
	Severity    string
	CWEID       string
	CVSSScore   float64
	LineNumber  int
	Snippet     string
}

// severityOf maps a severity label to the CVSS base score reported for it.
// The score is a property of the finding class, not of the individual hit.
func severityOf(severity string) float64 {
	switch severity {
	case "critical":
		return 9.1
	case "high":
		return 8.1
	case "medium":
		return 5.3
	case "low":
		return 3.7
	default:
		return 0.0
	}
}

// ---------------------------------------------------------------------------
// Source normalization
// ---------------------------------------------------------------------------

// stripCommentsAndStrings blanks out comments and string literals while
// preserving every byte offset and line break, so reported line numbers and
// snippets still line up with the original file. Analyzing raw text would
// otherwise match patterns inside comments and string literals.
func stripCommentsAndStrings(src string) string {
	out := []byte(src)
	n := len(out)
	i := 0
	for i < n {
		c := out[i]
		switch {
		case c == '/' && i+1 < n && out[i+1] == '/':
			for i < n && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		case c == '/' && i+1 < n && out[i+1] == '*':
			out[i], out[i+1] = ' ', ' '
			i += 2
			for i < n {
				if out[i] == '*' && i+1 < n && out[i+1] == '/' {
					out[i], out[i+1] = ' ', ' '
					i += 2
					break
				}
				if out[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
		case c == '"' || c == '\'':
			quote := c
			out[i] = ' '
			i++
			for i < n {
				if out[i] == '\\' && i+1 < n {
					out[i] = ' '
					if out[i+1] != '\n' {
						out[i+1] = ' '
					}
					i += 2
					continue
				}
				if out[i] == quote {
					out[i] = ' '
					i++
					break
				}
				if out[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
		default:
			i++
		}
	}
	return string(out)
}

// ---------------------------------------------------------------------------
// Structure extraction
// ---------------------------------------------------------------------------

type solidityFunction struct {
	Name      string
	Signature string
	Modifiers string
	StartLine int
	BodyStart int // byte offset of the opening brace
	BodyEnd   int // byte offset just past the closing brace
}

type stateVariable struct {
	Name string
	Line int
}

var (
	reFunctionDecl = regexp.MustCompile(`\b(function|constructor|fallback|receive)\b([^{;]*)`)
	reStateVarDecl = regexp.MustCompile(`(?m)^[ \t]*(?:mapping[ \t]*\([^)]*\)|[A-Za-z_][A-Za-z0-9_]*(?:\[[^\]]*\])?)[ \t]+(?:public|private|internal|constant|immutable|override|virtual|[ \t])*([A-Za-z_][A-Za-z0-9_]*)[ \t]*(?:=[^;]*)?;`)
	reIdentifier   = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

// lineOf returns the 1-based line number containing byte offset off.
func lineOf(src string, off int) int {
	if off > len(src) {
		off = len(src)
	}
	return strings.Count(src[:off], "\n") + 1
}

// lineText returns the trimmed text of the given 1-based line.
func lineText(src string, line int) string {
	lines := strings.Split(src, "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	text := strings.TrimSpace(lines[line-1])
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return text
}

// matchBrace returns the offset just past the brace that closes the block
// opened at openOff, or -1 when the block is unterminated.
func matchBrace(src string, openOff int) int {
	depth := 0
	for i := openOff; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// contractBodies returns the byte ranges of every contract/library/interface
// body in the source.
func contractBodies(src string) [][2]int {
	var bodies [][2]int
	re := regexp.MustCompile(`\b(contract|library|interface)\s+[A-Za-z_][A-Za-z0-9_]*[^{;]*\{`)
	for _, loc := range re.FindAllStringIndex(src, -1) {
		open := loc[1] - 1
		end := matchBrace(src, open)
		if end < 0 {
			end = len(src)
		}
		bodies = append(bodies, [2]int{open, end})
	}
	return bodies
}

// extractFunctions finds every function-like declaration with a body.
func extractFunctions(src string) []solidityFunction {
	var fns []solidityFunction
	for _, loc := range reFunctionDecl.FindAllStringSubmatchIndex(src, -1) {
		headerEnd := loc[3]
		// The declaration ends at the opening brace of the body, or at the
		// terminating semicolon for an interface/abstract declaration.
		open := strings.IndexByte(src[headerEnd:], '{')
		semi := strings.IndexByte(src[headerEnd:], ';')
		if open < 0 {
			continue
		}
		if semi >= 0 && semi < open {
			continue
		}
		bodyStart := headerEnd + open
		bodyEnd := matchBrace(src, bodyStart)
		if bodyEnd < 0 {
			bodyEnd = len(src)
		}

		kind := src[loc[2]:loc[3]]
		header := strings.TrimSpace(src[loc[3]:bodyStart])
		name := kind
		if kind == "function" {
			if m := reIdentifier.FindString(header); m != "" {
				name = m
			}
		}

		fns = append(fns, solidityFunction{
			Name:      name,
			Signature: strings.TrimSpace(src[loc[2]:bodyStart]),
			Modifiers: header,
			StartLine: lineOf(src, loc[2]),
			BodyStart: bodyStart,
			BodyEnd:   bodyEnd,
		})
	}
	return fns
}

// extractStateVariables collects contract-level variable declarations.
func extractStateVariables(src string, bodies [][2]int) []stateVariable {
	var vars []stateVariable
	for _, body := range bodies {
		// Only consider declarations at the contract's top level.
		inner := src[body[0]+1 : body[1]-1]
		depth := 0
		lineStart := 0
		for i := 0; i < len(inner); i++ {
			switch inner[i] {
			case '{':
				depth++
			case '}':
				depth--
			case '\n':
				if depth == 0 {
					stmt := inner[lineStart:i]
					// The declaration pattern has a single capture group holding
					// the variable name.
					if m := reStateVarDecl.FindStringSubmatch(stmt); len(m) > 1 {
						vars = append(vars, stateVariable{
							Name: m[1],
							Line: lineOf(src, body[0]+1+lineStart),
						})
					}
				}
				lineStart = i + 1
			}
		}
	}
	return vars
}

// ---------------------------------------------------------------------------
// Detectors
// ---------------------------------------------------------------------------

var (
	reExternalCall  = regexp.MustCompile(`\.\s*(call|delegatecall|callcode|staticcall|send|transfer)\s*[({]`)
	reUncheckedCall = regexp.MustCompile(`\.\s*(call|delegatecall|staticcall)\s*[({]`)
	reStateWrite    = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]*\])?\s*(?:=[^=]|\+=|-=|\*=|/=|\|=|&=|\^=)`)
	reDelete        = regexp.MustCompile(`^\s*delete\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rePush          = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]*\])?\s*\.\s*(push|pop)\s*\(`)
)

// isStateWrite reports whether a statement writes to one of the contract's
// state variables, returning the variable name.
func isStateWrite(stmt string, stateVars map[string]bool) (string, bool) {
	if m := reDelete.FindStringSubmatch(stmt); m != nil {
		if stateVars[m[1]] {
			return m[1], true
		}
	}
	if m := rePush.FindStringSubmatch(stmt); m != nil {
		if stateVars[m[1]] {
			return m[1], true
		}
	}
	if m := reStateWrite.FindStringSubmatch(stmt); m != nil {
		name := m[1]
		// Skip local declarations such as "uint256 x = 1;" and control flow.
		if stateVars[name] {
			return name, true
		}
	}
	return "", false
}

// statement is one statement located within the analyzed source.
type statement struct {
	Text string
	Off  int // byte offset of the statement's first non-space character
	End  int // byte offset just past the terminating semicolon
}

// statementsIn splits a function body into statements with their offsets. The
// range must be the body's interior, between the braces. The reported End is
// the position after the semicolon, so a caller can resume scanning from it
// without re-deriving the statement's extent from its trimmed text.
func statementsIn(src string, start, end int) []statement {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if start >= end {
		return nil
	}
	body := src[start:end]
	var out []statement
	stmtStart := 0
	depth := 0
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '{', '(':
			depth++
		case '}', ')':
			depth--
		case ';':
			if depth == 0 {
				raw := body[stmtStart:i]
				trimmed := strings.TrimSpace(raw)
				if trimmed != "" {
					offset := start + stmtStart + (len(raw) - len(strings.TrimLeft(raw, " \t\r\n")))
					out = append(out, statement{Text: trimmed, Off: offset, End: start + i + 1})
				}
				stmtStart = i + 1
			}
		}
	}
	return out
}

// AnalyzeSolidity runs every detector over the source and returns the findings
// ordered by severity then line.
func AnalyzeSolidity(src string) []StaticFinding {
	clean := stripCommentsAndStrings(src)
	bodies := contractBodies(clean)
	fns := extractFunctions(clean)

	stateVarSet := map[string]bool{}
	for _, v := range extractStateVariables(clean, bodies) {
		stateVarSet[v.Name] = true
	}

	var findings []StaticFinding
	add := func(f StaticFinding) {
		if f.CVSSScore == 0 {
			f.CVSSScore = severityOf(f.Severity)
		}
		f.Snippet = lineText(src, f.LineNumber)
		findings = append(findings, f)
	}

	// --- Reentrancy: an external call followed by a state write, in a
	// function that is not guarded. ---
	for _, fn := range fns {
		guarded := strings.Contains(fn.Modifiers, "nonReentrant")
		for _, st := range statementsIn(clean, fn.BodyStart+1, fn.BodyEnd-1) {
			if !reExternalCall.MatchString(st.Text) {
				continue
			}
			// Look for a state write later in the same function.
			for _, later := range statementsIn(clean, st.End, fn.BodyEnd-1) {
				if name, ok := isStateWrite(later.Text, stateVarSet); ok {
					if !guarded {
						add(StaticFinding{
							Title: fmt.Sprintf("Reentrancy: external call before state update in %s()", fn.Name),
							Description: fmt.Sprintf(
								"%s() performs an external call and then writes state variable %q. Control returns to the callee before the write, so a malicious callee can re-enter and observe the stale state. Apply a reentrancy guard or follow checks-effects-interactions.",
								fn.Name, name),
							Category:   "reentrancy",
							Severity:   "critical",
							CWEID:      "CWE-841",
							LineNumber: lineOf(clean, later.Off),
						})
					}
					break
				}
			}
		}
	}

	// --- Unchecked low-level call return value. ---
	for _, fn := range fns {
		for _, st := range statementsIn(clean, fn.BodyStart+1, fn.BodyEnd-1) {
			if !reUncheckedCall.MatchString(st.Text) {
				continue
			}
			text := st.Text
			checked := strings.Contains(text, "require(") ||
				strings.Contains(text, "assert(") ||
				strings.Contains(text, "revert(") ||
				strings.Contains(text, "if (") ||
				strings.Contains(text, "if(") ||
				strings.Contains(text, "success") ||
				strings.Contains(text, "ok)") ||
				strings.Contains(text, "bool ")
			if checked {
				continue
			}
			add(StaticFinding{
				Title: fmt.Sprintf("Unchecked low-level call return value in %s()", fn.Name),
				Description: fmt.Sprintf(
					"%s() ignores the success flag of a low-level call. A failed call does not revert the transaction, so execution continues as if it succeeded. Check the returned flag and revert on failure.",
					fn.Name),
				Category:   "unchecked_return",
				Severity:   "medium",
				CWEID:      "CWE-252",
				LineNumber: lineOf(clean, st.Off),
			})
		}
	}

	// --- tx.origin used for authorization. ---
	reTxOrigin := regexp.MustCompile(`tx\s*\.\s*origin`)
	for _, loc := range reTxOrigin.FindAllStringIndex(clean, -1) {
		add(StaticFinding{
			Title:       "Authorization uses tx.origin",
			Description: "tx.origin is the transaction's original sender, not the immediate caller. A contract called through a phishing intermediary inherits the victim's tx.origin, so this check can be bypassed. Use msg.sender.",
			Category:    "access_control",
			Severity:    "high",
			CWEID:       "CWE-477",
			LineNumber:  lineOf(clean, loc[0]),
		})
	}

	// --- delegatecall to a target that is not a compile-time constant. ---
	reDelegatecall := regexp.MustCompile(`\.\s*delegatecall\s*\(`)
	for _, loc := range reDelegatecall.FindAllStringIndex(clean, -1) {
		line := lineOf(clean, loc[0])
		text := lineText(clean, line)
		add(StaticFinding{
			Title:       "delegatecall in execution context",
			Description: "delegatecall runs the callee's code against this contract's storage and balance. If the target address can be influenced, an attacker can execute arbitrary code in this contract's context. Restrict the target to a trusted, immutable address.",
			Category:    "delegatecall",
			Severity:    "high",
			CWEID:       "CWE-829",
			LineNumber:  line,
			Snippet:     text,
		})
	}

	// --- selfdestruct. ---
	reSelfdestruct := regexp.MustCompile(`\b(selfdestruct|suicide)\s*\(`)
	for _, loc := range reSelfdestruct.FindAllStringIndex(clean, -1) {
		line := lineOf(clean, loc[0])
		// Locate the enclosing function to judge whether it is guarded.
		guarded := false
		for _, fn := range fns {
			if loc[0] >= fn.BodyStart && loc[0] < fn.BodyEnd {
				if strings.Contains(fn.Modifiers, "onlyOwner") ||
					strings.Contains(fn.Modifiers, "onlyAdmin") ||
					strings.Contains(fn.Modifiers, "onlyRole") ||
					strings.Contains(fn.Signature, "onlyOwner") {
					guarded = true
				}
				break
			}
		}
		severity := "high"
		title := "Contract can be destroyed"
		if !guarded {
			severity = "critical"
			title = "selfdestruct reachable without an access-control modifier"
		}
		add(StaticFinding{
			Title:       title,
			Description: "selfdestruct permanently removes the contract's code and forwards its balance. If the call is reachable by an unauthorized caller the contract can be destroyed and its funds drained. Gate it behind access control or remove it.",
			Category:    "destructible",
			Severity:    severity,
			CWEID:       "CWE-284",
			LineNumber:  line,
		})
	}

	// --- Weak randomness derived from block properties. ---
	reWeakRandom := regexp.MustCompile(`\b(block\s*\.\s*(timestamp|difficulty|prevrandao|number|blockhash)|now)\b`)
	for _, loc := range reWeakRandom.FindAllStringIndex(clean, -1) {
		line := lineOf(clean, loc[0])
		// Only meaningful when the value feeds randomness; look at the line.
		text := lineText(clean, line)
		lower := strings.ToLower(text)
		if !strings.Contains(lower, "keccak") && !strings.Contains(lower, "sha256") &&
			!strings.Contains(lower, "random") && !strings.Contains(lower, "seed") &&
			!strings.Contains(lower, "%") {
			continue
		}
		add(StaticFinding{
			Title:       "Weak randomness derived from block properties",
			Description: "Block properties are known to miners/validators and can be influenced when producing a block, so a value derived from them is predictable. Use a verifiable randomness source such as a commit-reveal scheme or an oracle.",
			Category:    "weak_randomness",
			Severity:    "high",
			CWEID:       "CWE-330",
			LineNumber:  line,
		})
	}

	// --- Arithmetic in an unchecked block (pre-0.8 or explicit unchecked). ---
	reUncheckedBlock := regexp.MustCompile(`\bunchecked\s*\{`)
	for _, loc := range reUncheckedBlock.FindAllStringIndex(clean, -1) {
		end := matchBrace(clean, loc[1]-1)
		if end < 0 {
			end = len(clean)
		}
		region := clean[loc[1]:end]
		reArith := regexp.MustCompile(`[A-Za-z0-9_\)\]]\s*(\+|\-|\*)\s*[A-Za-z0-9_\(]`)
		if m := reArith.FindStringIndex(region); m != nil {
			add(StaticFinding{
				Title:       "Arithmetic inside an unchecked block",
				Description: "Arithmetic in an unchecked block is not overflow-checked, so an operation that exceeds the type's range wraps silently and can corrupt balances or accounting. Verify the operands' bounds or remove the block.",
				Category:    "arithmetic",
				Severity:    "high",
				CWEID:       "CWE-190",
				LineNumber:  lineOf(clean, loc[1]+m[0]),
			})
		}
	}

	// --- Initializer without the initializer modifier. ---
	reInitFn := regexp.MustCompile(`\bfunction\s+(initialize|init)\s*\(`)
	for _, loc := range reInitFn.FindAllStringIndex(clean, -1) {
		for _, fn := range fns {
			if fn.BodyStart < loc[1] {
				continue
			}
			if strings.Contains(fn.Modifiers, "initializer") || strings.Contains(fn.Modifiers, "reinitializer") {
				break
			}
			add(StaticFinding{
				Title:       fmt.Sprintf("%s() is not protected by the initializer modifier", fn.Name),
				Description: "The function name indicates one-time initialization but it is not guarded by an initializer modifier, so it can be called again to reset ownership or configuration. Use OpenZeppelin's initializer or an explicit initialized flag.",
				Category:    "access_control",
				Severity:    "high",
				CWEID:       "CWE-665",
				LineNumber:  lineOf(clean, loc[0]),
			})
			break
		}
	}

	// --- Hardcoded secrets. ---
	reSecret := regexp.MustCompile(`(?i)\b(private[_\s]?key|secret|mnemonic|seed[_\s]?phrase)\s*(=|:)\s*0x[0-9a-fA-F]{32,}`)
	for _, loc := range reSecret.FindAllStringIndex(clean, -1) {
		add(StaticFinding{
			Title:       "Hardcoded secret in source",
			Description: "A private key or secret appears as a literal in the source. On-chain data is public and repository history is permanent, so the value is compromised. Load it from a secret store at deploy time.",
			Category:    "secret_management",
			Severity:    "critical",
			CWEID:       "CWE-798",
			LineNumber:  lineOf(clean, loc[0]),
		})
	}

	// --- Deprecated builtins. ---
	reDeprecated := regexp.MustCompile(`\b(suicide|sha3|throw)\s*[({;]`)
	for _, loc := range reDeprecated.FindAllStringIndex(clean, -1) {
		add(StaticFinding{
			Title:       "Use of a deprecated Solidity builtin",
			Description: "suicide, sha3 and throw were removed from modern Solidity. Their presence indicates code written against an old compiler with known issues. Migrate to selfdestruct/keccak256/require or revert.",
			Category:    "code_quality",
			Severity:    "informational",
			CWEID:       "CWE-477",
			LineNumber:  lineOf(clean, loc[0]),
		})
	}

	// --- Floating pragma. ---
	rePragma := regexp.MustCompile(`(?m)^\s*pragma\s+solidity\s+([^;]+);`)
	if m := rePragma.FindStringSubmatchIndex(clean); m != nil {
		version := strings.TrimSpace(clean[m[2]:m[3]])
		if strings.HasPrefix(version, "^") || strings.HasPrefix(version, ">") || strings.Contains(version, ">") {
			add(StaticFinding{
				Title:       "Floating pragma allows a different compiler version",
				Description: fmt.Sprintf("The pragma %q permits a range of compiler versions. A different compiler can change code generation and diagnostics, so a contract audited under one version may not behave identically under another. Pin an exact version.", version),
				Category:    "code_quality",
				Severity:    "low",
				CWEID:       "CWE-1103",
				LineNumber:  lineOf(clean, m[0]),
			})
		}
	}

	// --- Missing zero-address check on ownership transfer. ---
	reTransferOwnership := regexp.MustCompile(`\bfunction\s+(transferOwnership|setOwner)\s*\([^)]*\)[^{]*\{`)
	for _, loc := range reTransferOwnership.FindAllStringIndex(clean, -1) {
		open := strings.LastIndexByte(clean[:loc[1]], '{')
		end := matchBrace(clean, open)
		if end < 0 {
			end = len(clean)
		}
		body := clean[open:end]
		if !strings.Contains(body, "address(0)") && !strings.Contains(body, "!= address(0)") {
			add(StaticFinding{
				Title:       "Ownership transfer does not reject the zero address",
				Description: "Setting the owner to the zero address would permanently remove the ability to call owner-restricted functions, locking the contract. Require the new owner to be non-zero.",
				Category:    "access_control",
				Severity:    "low",
				CWEID:       "CWE-20",
				LineNumber:  lineOf(clean, loc[0]),
			})
		}
	}

	// --- State-changing external function with no access control and no
	// caller check. ---
	for _, fn := range fns {
		if fn.Name == "constructor" || fn.Name == "receive" || fn.Name == "fallback" {
			continue
		}
		mods := fn.Modifiers
		if !strings.Contains(mods, "public") && !strings.Contains(mods, "external") {
			continue
		}
		// View/pure functions cannot change state.
		if strings.Contains(mods, "view") || strings.Contains(mods, "pure") {
			continue
		}
		if strings.Contains(mods, "only") || strings.Contains(mods, "initializer") ||
			strings.Contains(mods, "nonReentrant") || strings.Contains(mods, "auth") {
			continue
		}
		body := clean[fn.BodyStart:fn.BodyEnd]
		if strings.Contains(body, "msg.sender") || strings.Contains(body, "onlyRole") ||
			strings.Contains(body, "_checkOwner") || strings.Contains(body, "hasRole(") {
			continue
		}
		// Require that the function actually writes state.
		writes := false
		for _, st := range statementsIn(clean, fn.BodyStart+1, fn.BodyEnd-1) {
			if _, ok := isStateWrite(st.Text, stateVarSet); ok {
				writes = true
				break
			}
		}
		if !writes {
			continue
		}
		add(StaticFinding{
			Title:       fmt.Sprintf("%s() writes state but has no access control", fn.Name),
			Description: "The function is externally callable, modifies contract state, and neither carries an access-control modifier nor inspects msg.sender. Any account can call it. Restrict it to the intended role.",
			Category:    "access_control",
			Severity:    "high",
			CWEID:       "CWE-284",
			LineNumber:  fn.StartLine,
		})
	}

	// --- Missing zero-address check on plain ERC-20 transfer. ---
	reTransfer := regexp.MustCompile(`\bfunction\s+transfer\s*\(\s*address\s+(\w+)\s*,`)
	for _, loc := range reTransfer.FindAllStringSubmatchIndex(clean, -1) {
		open := strings.IndexByte(clean[loc[1]:], '{')
		if open < 0 {
			continue
		}
		open += loc[1]
		end := matchBrace(clean, open)
		if end < 0 {
			end = len(clean)
		}
		body := clean[open:end]
		if !strings.Contains(body, "address(0)") {
			add(StaticFinding{
				Title:       "transfer() does not reject the zero address",
				Description: "A transfer to the zero address would burn the tokens irrecoverably. Require the recipient to be non-zero.",
				Category:    "input_validation",
				Severity:    "low",
				CWEID:       "CWE-20",
				LineNumber:  lineOf(clean, loc[0]),
			})
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		ri, rj := severityRank(findings[i].Severity), severityRank(findings[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return findings[i].LineNumber < findings[j].LineNumber
	})
	return findings
}

// severityRank orders severities from most to least severe.
func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

// CountBySeverity tallies findings per severity label.
func CountBySeverity(findings []StaticFinding) map[string]int {
	counts := map[string]int{
		"critical": 0, "high": 0, "medium": 0, "low": 0, "informational": 0,
	}
	for _, f := range findings {
		counts[f.Severity]++
	}
	return counts
}
