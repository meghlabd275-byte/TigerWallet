package main

// Tests for the prediction-market service. The cases that matter are the ones
// that keep a fabricated value out of a trading response: an unknown outcome
// must have no price, a replayed transaction hash must be refused, and an
// unpermitted origin must not be granted credentialed CORS.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestOriginAllowed(t *testing.T) {
	origins := []string{"https://wallet.example", "https://app.example"}

	if !originAllowed(origins, "https://wallet.example") {
		t.Error("an allowed origin was rejected")
	}
	if originAllowed(origins, "https://evil.example") {
		t.Error("an origin that is not on the list was accepted")
	}
	// A wildcard entry must not be honoured: the service sends credentials.
	if originAllowed([]string{"*"}, "https://evil.example") {
		t.Error("a wildcard allow-list was honoured for a credentialed request")
	}
	// A non-browser client sends no Origin header.
	if !originAllowed(origins, "") {
		t.Error("a request without an Origin header was rejected")
	}
	// An empty allow-list permits nothing from a browser.
	if originAllowed(nil, "https://wallet.example") {
		t.Error("an empty allow-list admitted a browser origin")
	}
}

func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim(" https://a.example , ,https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Errorf("splitAndTrim = %#v", got)
	}
	if len(splitAndTrim("")) != 0 {
		t.Error("an empty value produced entries")
	}
}

func TestPeriodStart(t *testing.T) {
	for _, period := range []string{"24h", "7d", "30d", "all"} {
		if _, err := periodStart(period); err != nil {
			t.Errorf("periodStart(%q): %v", period, err)
		}
	}
	if _, err := periodStart("1y"); err == nil {
		t.Error("an unknown period was accepted")
	}
	if start, _ := periodStart("all"); start != 0 {
		t.Errorf("periodStart(\"all\") = %d, want 0", start)
	}
}

func TestRawNumber(t *testing.T) {
	cases := map[string]float64{
		`123.5`:       123.5,
		`"123.5"`:     123.5,
		`""`:          0,
		`null`:        0,
		`"not-a-num"`: 0,
	}
	for input, want := range cases {
		if got := rawNumber(json.RawMessage(input)); got != want {
			t.Errorf("rawNumber(%s) = %v, want %v", input, got, want)
		}
	}
	if got := rawNumber(nil); got != 0 {
		t.Errorf("rawNumber(nil) = %v, want 0", got)
	}
}

func TestParseRFC3339(t *testing.T) {
	want := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC).Unix()
	for _, layout := range []string{
		"2026-12-31T23:59:00Z",
		"2026-12-31T23:59:00+00:00",
	} {
		if got := parseRFC3339(layout); got != want {
			t.Errorf("parseRFC3339(%q) = %d, want %d", layout, got, want)
		}
	}
	for _, bad := range []string{"", "not-a-date", "31/12/2026"} {
		if got := parseRFC3339(bad); got != 0 {
			t.Errorf("parseRFC3339(%q) = %d, want 0", bad, got)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "c"); got != "c" {
		t.Errorf("firstNonEmpty = %q", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty = %q, want empty", got)
	}
}

// decodePolymarketOutcomes must read the nested JSON strings the Gamma API
// returns, and must not invent an outcome when the arrays are absent.
func TestDecodePolymarketOutcomes(t *testing.T) {
	m := polymarketMarket{
		ID:            "123",
		Outcomes:      `["Yes","No"]`,
		OutcomePrices: `["0.185","0.815"]`,
		Active:        true,
	}

	outcomes := decodePolymarketOutcomes(m)
	if len(outcomes) != 2 {
		t.Fatalf("got %d outcomes, want 2", len(outcomes))
	}
	if outcomes[0].Name != "Yes" || outcomes[0].Price != 0.185 {
		t.Errorf("outcome 0 = %+v", outcomes[0])
	}
	if outcomes[1].Name != "No" || outcomes[1].Price != 0.815 {
		t.Errorf("outcome 1 = %+v", outcomes[1])
	}
	if outcomes[0].ID != "polymarket:123:0" || outcomes[0].MarketID != "polymarket:123" {
		t.Errorf("outcome ids = %q / %q", outcomes[0].ID, outcomes[0].MarketID)
	}

	// No outcome data means no outcomes, never a placeholder.
	if got := decodePolymarketOutcomes(polymarketMarket{ID: "9"}); got != nil {
		t.Errorf("outcomes were invented for a market with no outcome data: %+v", got)
	}
}

func TestDecodePolymarketOutcomesShortPriceList(t *testing.T) {
	// More names than prices: the missing price must be zero, not carried over
	// from the previous entry.
	outcomes := decodePolymarketOutcomes(polymarketMarket{
		ID:            "7",
		Outcomes:      `["A","B","C"]`,
		OutcomePrices: `["0.5"]`,
	})
	if len(outcomes) != 3 {
		t.Fatalf("got %d outcomes, want 3", len(outcomes))
	}
	if outcomes[1].Price != 0 || outcomes[2].Price != 0 {
		t.Errorf("prices = %v / %v, want 0", outcomes[1].Price, outcomes[2].Price)
	}
}

func TestGenerateIDIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := generateID()
		if len(id) != 32 {
			t.Fatalf("id %q is not 16 bytes of hex", id)
		}
		if seen[id] {
			t.Fatalf("generateID produced a duplicate: %s", id)
		}
		seen[id] = true
	}
}

func TestGetDurationEnv(t *testing.T) {
	t.Setenv("TEST_DURATION", "90s")
	if got := getDurationEnv("TEST_DURATION", time.Minute); got != 90*time.Second {
		t.Errorf("got %v, want 90s", got)
	}

	t.Setenv("TEST_DURATION", "not-a-duration")
	if got := getDurationEnv("TEST_DURATION", time.Minute); got != time.Minute {
		t.Errorf("an invalid duration did not fall back to the default: %v", got)
	}

	t.Setenv("TEST_DURATION", "-5s")
	if got := getDurationEnv("TEST_DURATION", time.Minute); got != time.Minute {
		t.Errorf("a negative duration was accepted: %v", got)
	}
}

// A market source must not be registered when its endpoint is not configured,
// so an unconfigured deployment reports "no source" instead of failing.
func TestNewMarketSourceRegistry(t *testing.T) {
	if sources := newMarketSourceRegistry(&Config{}); len(sources) != 0 {
		t.Errorf("a source was registered without an endpoint: %+v", sources)
	}

	cfg := &Config{PolymarketAPI: "https://gamma-api.polymarket.com"}
	sources := newMarketSourceRegistry(cfg)
	if len(sources) != 1 || sources[0].name != "polymarket" {
		t.Fatalf("sources = %+v, want one polymarket source", sources)
	}
	if got := cfg.endpointForSource("polymarket"); got != cfg.PolymarketAPI {
		t.Errorf("endpointForSource = %q", got)
	}
	if got := cfg.endpointForSource("unknown"); got != "" {
		t.Errorf("endpointForSource(unknown) = %q, want empty", got)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("nil was reported as a unique violation")
	}
	if isUniqueViolation(ErrDuplicateTrade) {
		t.Error("a non-database error was reported as a unique violation")
	}
}

func TestMustJSON(t *testing.T) {
	got := string(mustJSON(map[string]int{"a": 1}))
	if !strings.Contains(got, `"a":1`) {
		t.Errorf("mustJSON = %s", got)
	}
}

func TestVerifyTokenRejectsUnsignedAndWrongKey(t *testing.T) {
	secret := []byte("test-signing-key")

	valid := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "user-1"})
	signed, err := valid.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := verifyToken(secret, "Bearer "+signed); err != nil || got != "user-1" {
		t.Errorf("a valid token was rejected: user=%q err=%v", got, err)
	}

	// Signed with a different key.
	other, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "user-1"}).SignedString([]byte("another-key"))
	if _, err := verifyToken(secret, "Bearer "+other); err == nil {
		t.Error("a token signed with the wrong key was accepted")
	}

	// The "none" algorithm must not be accepted.
	unsigned, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "user-1"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := verifyToken(secret, "Bearer "+unsigned); err == nil {
		t.Error("an unsigned token was accepted")
	}

	// A token with no user id is not usable.
	noUser, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{}).SignedString(secret)
	if _, err := verifyToken(secret, "Bearer "+noUser); err == nil {
		t.Error("a token without a user_id was accepted")
	}

	for _, header := range []string{"", "Basic abc", "Bearer ", "Bearer not.a.token"} {
		if _, err := verifyToken(secret, header); err == nil {
			t.Errorf("header %q was accepted", header)
		}
	}
}

func TestResolveUserRefusesAnotherAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := []byte("k")
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "alice"}).SignedString(secret)

	// The real middleware chain is exercised: requireUser populates the
	// context, then the handler resolves the requested identifier against it.
	call := func(header, requested string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			c.Request.Header.Set("Authorization", header)
		}

		requireUser(secret)(c)
		if c.IsAborted() {
			return w
		}

		user, ok := resolveUser(c, requested)
		if ok {
			c.String(http.StatusOK, user)
		}
		return w
	}

	// No token at all.
	if got := call("", "").Code; got != http.StatusUnauthorized {
		t.Errorf("an unauthenticated request returned %d, want 401", got)
	}

	// The caller's own account.
	w := call("Bearer "+token, "alice")
	if w.Code != http.StatusOK || w.Body.String() != "alice" {
		t.Errorf("own account returned %d %q", w.Code, w.Body.String())
	}

	// Another account's identifier must be refused, not honoured.
	if got := call("Bearer "+token, "bob").Code; got != http.StatusForbidden {
		t.Errorf("reading another account returned %d, want 403", got)
	}

	// An empty identifier resolves to the authenticated user.
	if w := call("Bearer "+token, ""); w.Code != http.StatusOK || w.Body.String() != "alice" {
		t.Errorf("empty identifier returned %d %q", w.Code, w.Body.String())
	}

	// A forged token is rejected by the middleware.
	if got := call("Bearer forged.token.value", "alice").Code; got != http.StatusUnauthorized {
		t.Errorf("a forged token returned %d, want 401", got)
	}
}
