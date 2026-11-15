package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// bot_core accepts a start request only if the JSON `kind` tag matches one of
// its StartReq variants. Go and Rust disagree about the spelling of two of
// them (market_maker -> "marketmaker", mean_reversion -> "meanreversion"),
// because the Rust tags come from `rename_all = "lowercase"` applied to the
// variant name. Nothing in either compiler catches a mismatch: bot_core simply
// answers 422 and the bot never runs.
//
// These tests read the Rust enum out of the source tree and compare it with the
// Go mapping, so adding a runner on one side without the other fails the build
// instead of failing silently in production.

// botCoreRelPath locates bot_core's main.rs relative to the test's working
// directory. `go test` runs with the package directory as CWD, but the depth
// from there to the repository root depends on where the module sits, so walk
// up until the file is found rather than hardcoding a fixed number of "..".
const botCoreRelPath = "mm_bot_platform/bot_core/src/main.rs"

func botCoreMainRS(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, botCoreRelPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("%s not found in any parent of the test directory", botCoreRelPath)
		}
		dir = parent
	}
}

// rustStartReqTags parses the `kind` tag of every variant in bot_core's
// StartReq enum. A variant uses an explicit #[serde(rename = "...")] when one is
// present, otherwise the variant name lowercased (rename_all = "lowercase").
func rustStartReqTags(t *testing.T) map[string]bool {
	t.Helper()

	path := botCoreMainRS(t)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(source)

	// Isolate the StartReq enum body.
	start := strings.Index(text, "enum StartReq {")
	if start < 0 {
		t.Fatalf("StartReq enum not found in %s", path)
	}
	body := text[start:]
	end := strings.Index(body, "\n}")
	if end < 0 {
		t.Fatalf("unterminated StartReq enum in %s", path)
	}
	body = body[:end]

	tags := map[string]bool{}
	rename := regexp.MustCompile(`rename\s*=\s*"([a-z_]+)"`)
	variant := regexp.MustCompile(`(?m)^\s*([A-Z][A-Za-z0-9]*)(\(|,)`)

	// Split on variants, keeping each variant's attributes with it.
	lines := strings.Split(body, "\n")
	var pending []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#[serde") || strings.HasPrefix(trimmed, "#[") {
			pending = append(pending, line)
			continue
		}
		if m := variant.FindStringSubmatch(line); m != nil {
			name := m[1]
			tag := strings.ToLower(name)
			for _, attr := range pending {
				if r := rename.FindStringSubmatch(attr); r != nil {
					tag = r[1]
					break
				}
			}
			tags[tag] = true
			pending = nil
			continue
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			pending = nil
		}
	}

	if len(tags) == 0 {
		t.Fatalf("parsed no StartReq tags from %s", path)
	}
	return tags
}

// TestDispatchKindMatchesBotCore is the guard that matters: every executable
// type bot_api advertises must resolve to a tag bot_core actually accepts.
func TestDispatchKindMatchesBotCore(t *testing.T) {
	tags := rustStartReqTags(t)

	executable := executableBotTypes()
	if len(executable) != 16 {
		t.Fatalf("expected 16 executable bot types, got %d", len(executable))
	}

	for _, botType := range executable {
		kind, ok := dispatchKind(botType)
		if !ok {
			t.Errorf("bot type %q is advertised as executable but dispatchKind rejects it", botType)
			continue
		}
		if !tags[kind] {
			t.Errorf("bot type %q maps to kind %q, which is not a StartReq variant in bot_core", botType, kind)
		}
	}
}

// TestEveryStartReqVariantIsReachable is the converse: a runner added to
// bot_core must be startable through bot_api, otherwise it is dead code.
func TestEveryStartReqVariantIsReachable(t *testing.T) {
	tags := rustStartReqTags(t)

	reachable := map[string]bool{}
	for _, botType := range executableBotTypes() {
		if kind, ok := dispatchKind(botType); ok {
			reachable[kind] = true
		}
	}
	// bot_core accepts "mev_bot" as an alias of front_run; bot_api sends
	// "front_run", which is the canonical tag.
	reachable["mev_bot"] = true

	var missing []string
	for tag := range tags {
		if !reachable[tag] {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("bot_core accepts kinds no bot_type can reach: %v", missing)
	}
}

// TestDispatchKindRejectsNonExecutableTypes pins the fail-closed set: the
// signal-only types must be refused rather than dispatched to a runner that
// does not exist.
func TestDispatchKindRejectsNonExecutableTypes(t *testing.T) {
	for _, botType := range []string{"sandwich", "flash_loan", ""} {
		if kind, ok := dispatchKind(botType); ok {
			t.Errorf("dispatchKind(%q) = %q, true; want fail-closed", botType, kind)
		}
	}
}

// TestExecutableTypesAreValidBotTypes keeps the advertised set inside the
// create-time allowlist, so a bot can never be started for a type the API
// would refuse to create.
func TestExecutableTypesAreValidBotTypes(t *testing.T) {
	for _, botType := range executableBotTypes() {
		if !validBotType(botType) {
			t.Errorf("executable bot type %q is not in the create allowlist", botType)
		}
	}
}
