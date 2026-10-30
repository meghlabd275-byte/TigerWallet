package main

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func testService(t *testing.T, cfg WCConfig) *WalletConnectService {
	t.Helper()
	svc := &WalletConnectService{config: cfg, upgrader: walletConnectUpgrader(cfg.AllowedOrigins)}
	if cfg.SignerPrivateKey != "" {
		key, err := crypto.HexToECDSA(trimHexPrefix(cfg.SignerPrivateKey))
		if err != nil {
			t.Fatalf("test key: %v", err)
		}
		svc.signer = key
	}
	return svc
}

// Broadcasting must fail closed rather than invent a transaction hash. The
// previous implementation returned "0x" + random bytes and reported success.
func TestBroadcastFailsClosedWithoutRPC(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	svc := testService(t, WCConfig{
		SignerPrivateKey: hex.EncodeToString(crypto.FromECDSA(key)),
		RPCURL:           "",
	})

	hash, err := svc.broadcastTransaction("topic", map[string]interface{}{
		"to":    "0x0000000000000000000000000000000000000001",
		"value": "0x1",
	})
	if err == nil {
		t.Fatalf("broadcast without an RPC endpoint returned a hash: %s", hash)
	}
	if hash != "" {
		t.Fatalf("expected empty hash on failure, got %q", hash)
	}
	if !strings.Contains(err.Error(), "rpc_url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// With no signer configured the call must also fail rather than fabricate.
func TestBroadcastFailsClosedWithoutSigner(t *testing.T) {
	svc := testService(t, WCConfig{RPCURL: "http://127.0.0.1:1"})
	hash, err := svc.broadcastTransaction("topic", map[string]interface{}{
		"to": "0x0000000000000000000000000000000000000001",
	})
	if err == nil {
		t.Fatalf("broadcast without a signer returned a hash: %s", hash)
	}
	if hash != "" {
		t.Fatalf("expected empty hash on failure, got %q", hash)
	}
}

// An invalid destination address must be rejected before any network call.
func TestBroadcastRejectsInvalidAddress(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	svc := testService(t, WCConfig{
		SignerPrivateKey: hex.EncodeToString(crypto.FromECDSA(key)),
		RPCURL:           "http://127.0.0.1:1",
	})
	_, err = svc.broadcastTransaction("topic", map[string]interface{}{"to": "not-an-address"})
	if err == nil {
		t.Fatal("invalid address accepted")
	}
	if !strings.Contains(err.Error(), "address") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Origin policy: unknown browser origins are refused, non-browser clients pass.
func TestWalletConnectOriginPolicy(t *testing.T) {
	svc := testService(t, WCConfig{AllowedOrigins: []string{"https://dapp.example"}})

	req := httptest.NewRequest(http.MethodGet, "/ws/topic", nil)
	req.Header.Set("Origin", "https://evil.example")
	if svc.upgrader.CheckOrigin(req) {
		t.Fatal("origin outside the allowlist was accepted")
	}

	req.Header.Set("Origin", "https://dapp.example")
	if !svc.upgrader.CheckOrigin(req) {
		t.Fatal("allowlisted origin was rejected")
	}

	req.Header.Del("Origin")
	if !svc.upgrader.CheckOrigin(req) {
		t.Fatal("non-browser client was rejected")
	}
}

func TestSplitOrigins(t *testing.T) {
	got := splitOrigins(" https://a.example , ,https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("splitOrigins = %#v", got)
	}
	if len(splitOrigins("")) != 0 {
		t.Fatal("empty input must yield no origins")
	}
}

func TestTxFieldParsing(t *testing.T) {
	v, err := txFieldBigInt(map[string]interface{}{"value": "0xde0b6b3a7640000"}, "value")
	if err != nil {
		t.Fatalf("txFieldBigInt: %v", err)
	}
	if v.String() != "1000000000000000000" {
		t.Fatalf("value = %s, want 1e18", v)
	}

	// A missing value is zero, not an error.
	v, err = txFieldBigInt(map[string]interface{}{}, "value")
	if err != nil || v.Sign() != 0 {
		t.Fatalf("missing value = %v, %v; want 0, nil", v, err)
	}

	if _, err := txFieldBigInt(map[string]interface{}{"value": "zzz"}, "value"); err == nil {
		t.Fatal("invalid quantity accepted")
	}

	data, err := txFieldBytes(map[string]interface{}{"data": "0xdeadbeef"}, "data")
	if err != nil {
		t.Fatalf("txFieldBytes: %v", err)
	}
	if len(data) != 4 || data[0] != 0xde {
		t.Fatalf("data = %x", data)
	}

	if _, err := txFieldBytes(map[string]interface{}{"data": "0xzz"}, "data"); err == nil {
		t.Fatal("invalid hex accepted")
	}

	if _, err := txFieldString(map[string]interface{}{}, "to"); err == nil {
		t.Fatal("missing 'to' accepted")
	}
}
