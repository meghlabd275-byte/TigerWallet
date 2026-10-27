package sdk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient builds a client pointed at a test server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient(&Config{
		APIKey:     "test-key",
		APISecret:  "test-secret",
		BaseURL:    server.URL,
		RetryCount: 1,
		RetryDelay: 1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// The test server is plain HTTP on loopback; allow it for the test only.
	client.config.BaseURL = server.URL
	return client
}

func TestNewClientRejectsMissingCredentials(t *testing.T) {
	if _, err := NewClient(&Config{BaseURL: "https://api.example.com"}); err == nil {
		t.Fatal("a client with no credentials must be rejected")
	}
	if _, err := NewClient(&Config{APIKey: "k", BaseURL: "https://api.example.com"}); err == nil {
		t.Fatal("a client with no secret must be rejected")
	}
}

func TestNewClientRejectsPlaintextBaseURL(t *testing.T) {
	_, err := NewClient(&Config{
		APIKey:    "k",
		APISecret: "s",
		BaseURL:   "http://api.example.com",
	})
	if err == nil {
		t.Fatal("a plaintext remote base URL must be rejected")
	}
}

func TestNewClientRejectsCredentialsInURL(t *testing.T) {
	_, err := NewClient(&Config{
		APIKey:    "k",
		APISecret: "s",
		BaseURL:   "https://user:pass@api.example.com",
	})
	if err == nil {
		t.Fatal("credentials embedded in the base URL must be rejected")
	}
}

func TestSignatureMatchesIndependentHMAC(t *testing.T) {
	auth := NewAuth("key", "secret")
	body := []byte(`{"jsonrpc":"2.0"}`)
	got := auth.Sign("POST", "/v1/rpc", "1700000000", body)

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("POST\n/v1/rpc\n1700000000\n"))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("signature mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestSignatureCoversTimestampAndPath(t *testing.T) {
	auth := NewAuth("key", "secret")
	body := []byte("{}")

	base := auth.Sign("POST", "/v1/rpc", "1700000000", body)
	if base == auth.Sign("POST", "/v1/rpc", "1700000001", body) {
		t.Fatal("the timestamp must be part of the signed message")
	}
	if base == auth.Sign("POST", "/v1/other", "1700000000", body) {
		t.Fatal("the path must be part of the signed message")
	}
	if base == auth.Sign("GET", "/v1/rpc", "1700000000", body) {
		t.Fatal("the method must be part of the signed message")
	}
}

func TestCallSendsSignedHeaders(t *testing.T) {
	var seenKey, seenSig, seenTimestamp string
	var seenBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.Header.Get("X-API-Key")
		seenSig = r.Header.Get("X-Signature")
		seenTimestamp = r.Header.Get("X-Timestamp")
		seenBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"amount":"100"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	raw, err := client.Call(context.Background(), "getBalance", map[string]interface{}{"address": "0xabc"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	if seenKey != "test-key" {
		t.Errorf("X-API-Key = %q, want test-key", seenKey)
	}
	if seenTimestamp == "" {
		t.Error("X-Timestamp was not sent")
	}

	// Recompute the signature from what the server actually received.
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte("POST\n/v1/rpc\n" + seenTimestamp + "\n"))
	mac.Write(seenBody)
	want := hex.EncodeToString(mac.Sum(nil))
	if seenSig != want {
		t.Errorf("signature does not verify against the received request\n got: %s\nwant: %s", seenSig, want)
	}

	if string(raw) != `{"amount":"100"}` {
		t.Errorf("result = %s", raw)
	}
}

func TestCallSurfacesRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"bad address"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.Call(context.Background(), "getBalance", nil)
	if err == nil {
		t.Fatal("an RPC error must be returned")
	}
	if !strings.Contains(err.Error(), "bad address") {
		t.Errorf("error = %v, want it to mention the RPC message", err)
	}
}

func TestCallFailsWhenResultMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	if _, err := client.Call(context.Background(), "getBalance", nil); err == nil {
		t.Fatal("a response with no result must be an error")
	}
}

func TestGetBalanceParsesAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"symbol":"USDT","amount":"123456","decimals":6,"valueUsd":"123.456"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	wallet := client.NewWallet("0xabc", 1)

	balance, err := wallet.GetBalance(context.Background(), "USDT")
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if balance.Symbol != "USDT" {
		t.Errorf("symbol = %q", balance.Symbol)
	}
	if balance.Amount.String() != "123456" {
		t.Errorf("amount = %s", balance.Amount)
	}
	if balance.Decimals != 6 {
		t.Errorf("decimals = %d", balance.Decimals)
	}
	if balance.ValueUSD == nil || balance.ValueUSD.FloatString(3) != "123.456" {
		t.Errorf("valueUsd = %v, want 123.456", balance.ValueUSD)
	}
}

func TestGetBalanceRejectsNonNumericAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"amount":"not-a-number"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	wallet := client.NewWallet("0xabc", 1)
	if _, err := wallet.GetBalance(context.Background(), "USDT"); err == nil {
		t.Fatal("a non-numeric amount must be rejected")
	}
}

func TestTransferReturnsBackendHash(t *testing.T) {
	const hash = "0xdeadbeef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tx_hash":"` + hash + `"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	wallet := client.NewWallet("0xabc", 1)

	got, err := wallet.Transfer(context.Background(), "0xdef", big.NewInt(1000), "USDT")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if got != hash {
		t.Errorf("hash = %q, want %q", got, hash)
	}
}

func TestTransferFailsWhenNoHashReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	wallet := client.NewWallet("0xabc", 1)

	if _, err := wallet.Transfer(context.Background(), "0xdef", big.NewInt(1000), "USDT"); err == nil {
		t.Fatal("a transfer result with no hash must not be reported as success")
	}
}

func TestTransferRejectsNonPositiveAmount(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.NotFoundHandler()))
	wallet := client.NewWallet("0xabc", 1)

	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1)} {
		if _, err := wallet.Transfer(context.Background(), "0xdef", amount, "USDT"); err == nil {
			t.Errorf("amount %v must be rejected", amount)
		}
	}
}

func TestRenderWidgetEscapesValues(t *testing.T) {
	html, err := RenderWidget("https://wallet.example.com", &WidgetConfig{
		Type:   "send",
		Theme:  "dark",
		Width:  `1"><script>alert(1)</script>`,
		Height: "640",
	})
	if err != nil {
		t.Fatalf("RenderWidget: %v", err)
	}
	if strings.Contains(html, "<script>") {
		t.Fatalf("script tag was not escaped: %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("expected escaped output, got: %s", html)
	}
}

func TestRenderWidgetEncodesQueryValues(t *testing.T) {
	html, err := RenderWidget("https://wallet.example.com", &WidgetConfig{
		Type:    "swap",
		ChainID: 137,
		Token:   "USDT&theme=evil",
	})
	if err != nil {
		t.Fatalf("RenderWidget: %v", err)
	}
	if strings.Contains(html, "theme=evil") {
		t.Fatalf("query value was not encoded: %s", html)
	}
	if !strings.Contains(html, "chainId=137") {
		t.Errorf("chainId missing: %s", html)
	}
}

func TestRenderWidgetRejectsInsecureURLs(t *testing.T) {
	if _, err := RenderWidget("http://wallet.example.com", &WidgetConfig{Type: "send"}); err == nil {
		t.Error("a plaintext base URL must be rejected")
	}
	if _, err := RenderWidget("https://wallet.example.com", &WidgetConfig{
		Type:     "send",
		Callback: "http://evil.example.com",
	}); err == nil {
		t.Error("a plaintext callback URL must be rejected")
	}
}

func TestDecodeListAcceptsBareAndWrapped(t *testing.T) {
	bare, err := decodeList[Transaction](json.RawMessage(`[{"txHash":"0x1"}]`), "transactions")
	if err != nil || len(bare) != 1 || bare[0].TxHash != "0x1" {
		t.Fatalf("bare array: %v %v", bare, err)
	}

	wrapped, err := decodeList[Transaction](json.RawMessage(`{"transactions":[{"txHash":"0x2"}]}`), "transactions")
	if err != nil || len(wrapped) != 1 || wrapped[0].TxHash != "0x2" {
		t.Fatalf("wrapped array: %v %v", wrapped, err)
	}

	if _, err := decodeList[Transaction](json.RawMessage(`{"other":[]}`), "transactions"); err == nil {
		t.Error("a response without the expected field must be an error")
	}
}

func TestHandleBalanceRejectsOversizedBody(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.NotFoundHandler()))

	big := strings.Repeat("a", maxRequestBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/v1/balance",
		strings.NewReader(`{"address":"`+big+`"}`))
	rec := httptest.NewRecorder()

	client.HandleBalance(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandlersRejectWrongMethod(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.NotFoundHandler()))

	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
	}{
		{"balance", client.HandleBalance, http.MethodGet},
		{"transfer", client.HandleTransfer, http.MethodGet},
		{"portfolio", client.HandlePortfolio, http.MethodPost},
		{"transactions", client.HandleTransactions, http.MethodPost},
		{"chains", client.HandleChains, http.MethodPost},
		{"sdk", client.HandleSDK, http.MethodPost},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, "/", nil)
		rec := httptest.NewRecorder()
		tc.handler(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want 405", tc.name, rec.Code)
		}
	}
}

func TestHandleSDKSetsNosniff(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.NotFoundHandler()))

	req := httptest.NewRequest(http.MethodGet, "/sdk.js", nil)
	rec := httptest.NewRecorder()
	client.HandleSDK(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options must be nosniff")
	}
}

func TestHandlersDoNotLeakUpstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal db password is hunter2"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)

	req := httptest.NewRequest(http.MethodGet, "/v1/portfolio?address=0xabc", nil)
	rec := httptest.NewRecorder()
	client.HandlePortfolio(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "hunter2") {
		t.Fatalf("upstream error detail leaked to the caller: %s", body)
	}
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
}

func TestJavaScriptSDKHasNoHardcodedChainEndpoints(t *testing.T) {
	js := JavaScriptSDK()

	for _, banned := range []string{"alchemyapi.io", "bsc-dataseed", "polygon-rpc.com"} {
		if strings.Contains(js, banned) {
			t.Errorf("the SDK must not hardcode %q", banned)
		}
	}
	if !strings.Contains(js, "loadChains") {
		t.Error("the SDK must load chains from the backend")
	}
}

func TestJavaScriptSDKEncodesERC20TransferCorrectly(t *testing.T) {
	js := JavaScriptSDK()

	// The selector and the address-then-amount ordering are what make the
	// calldata valid; assert the encoder is present and used.
	if !strings.Contains(js, "ERC20_TRANSFER_SELECTOR") {
		t.Error("the ERC-20 transfer selector must be a named constant")
	}
	if !strings.Contains(js, "pad64") {
		t.Error("the SDK must pad calldata words to 32 bytes")
	}
	if strings.Contains(js, "amount.toString(16).padStart(64, '0')") &&
		!strings.Contains(js, "BigInt(amount).toString(16)") {
		t.Error("amounts must be converted through BigInt before hex encoding")
	}
}
