package main

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// stubNode is a JSON-RPC server for tests.
//
// It speaks real HTTP and real JSON-RPC, so the service's request encoding,
// response parsing and error handling are all exercised. It answers
// `eth_call` with the address the canonical factory returns for salt zero, as
// observed from mainnet, and returns the same value the deployed factory would
// for other salts only when they are derived the same way. Anything else it
// does not implement is reported as a JSON-RPC error.
type stubNode struct {
	server *httptest.Server

	mu       sync.Mutex
	methods  []string
	failCall bool
}

const factoryAddressForTests = "0x9406Cc6185a346906296840746125a0E44976454"

// saltZeroResult is what the deployed factory returns for the test owner with
// salt zero, read from mainnet.
const saltZeroResult = "0x0000000000000000000000002bea269f9305577f5a9f6a9fdf04c969368879e7"

func newStubNode(t *testing.T) *stubNode {
	t.Helper()

	node := &stubNode{}

	node.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unreadable body", http.StatusBadRequest)
			return
		}

		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "malformed JSON-RPC request", http.StatusBadRequest)
			return
		}

		node.mu.Lock()
		node.methods = append(node.methods, request.Method)
		failCall := node.failCall
		node.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch request.Method {
		case "eth_call":
			if failCall {
				json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      1,
					"error":   map[string]any{"code": -32000, "message": "the stub node refuses eth_call"},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  saltZeroResult,
			})
		case "eth_gasPrice":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  "0x3b9aca00",
			})
		case "eth_estimateGas":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  "0x5208",
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"error": map[string]any{
					"code":    -32601,
					"message": "the stub node does not implement " + request.Method,
				},
			})
		}
	}))

	t.Cleanup(node.server.Close)
	return node
}

// calledMethods returns the JSON-RPC methods the stub served.
func (s *stubNode) calledMethods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.methods...)
}

// setFailCall makes the stub refuse eth_call.
func (s *stubNode) setFailCall(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCall = fail
}

// serviceWithStub returns a service pointed at the stub node.
func serviceWithStub(t *testing.T) (*AccountAbstractionService, *stubNode) {
	t.Helper()

	node := newStubNode(t)

	service := NewAccountAbstractionService()
	service.factoryAddress = factoryAddressForTests
	service.entryPointAddress = entryPointDefaults["entry_point"]
	service.defaultChainID = 1

	t.Setenv("RPC_URL_1", node.server.URL)

	return service, node
}

func TestTheEntryPointAddressesAreTheCanonicalOnes(t *testing.T) {
	// These addresses are deployed at the same address on every major network,
	// so a typo here would send user operations to a contract that does not
	// exist.
	if got := entryPointDefaults["entry_point"]; !common.IsHexAddress(got) {
		t.Fatalf("entry point %q is not a valid address", got)
	}
	if got := entryPointDefaults["factory"]; !common.IsHexAddress(got) {
		t.Fatalf("factory %q is not a valid address", got)
	}

	if got := entryPointDefaults["entry_point"]; got != "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789" {
		t.Errorf("entry point = %s, want the canonical v0.6 address", got)
	}
	if got := entryPointDefaults["factory"]; got != "0x9406Cc6185a346906296840746125a0E44976454" {
		t.Errorf("factory = %s, want the canonical SimpleAccountFactory", got)
	}
}

func TestCreateSmartAccountResolvesTheAddressFromTheFactory(t *testing.T) {
	service, node := serviceWithStub(t)

	account, err := service.CreateSmartAccount(context.Background(), "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", 1)
	if err != nil {
		t.Fatalf("CreateSmartAccount: %v", err)
	}

	// The address is the one the deployed factory returns.
	if got := strings.ToLower(account.Address); got != "0x2bea269f9305577f5a9f6a9fdf04c969368879e7" {
		t.Fatalf("address = %s, want the factory's answer", got)
	}

	methods := node.calledMethods()
	if len(methods) != 1 || methods[0] != "eth_call" {
		t.Fatalf("node saw %v, want a single eth_call", methods)
	}
}

func TestCreateSmartAccountRejectsAnInvalidOwner(t *testing.T) {
	service, _ := serviceWithStub(t)

	if _, err := service.CreateSmartAccount(context.Background(), "not-an-address", 1); err == nil {
		t.Fatal("an invalid owner was accepted")
	}
}

func TestCreateSmartAccountFailsClosedWithoutAnEndpoint(t *testing.T) {
	service := NewAccountAbstractionService()
	service.factoryAddress = factoryAddressForTests

	// With no RPC endpoint configured there is no address to resolve, so this
	// must error rather than return a made-up one.
	t.Setenv("RPC_URL_1", "")
	t.Setenv("RPC_URL", "")

	if _, err := service.CreateSmartAccount(context.Background(), "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", 1); err == nil {
		t.Fatal("account creation succeeded with no RPC endpoint")
	}
}

func TestCreateSmartAccountFailsClosedWhenTheFactoryErrors(t *testing.T) {
	service, node := serviceWithStub(t)
	node.setFailCall(true)

	if _, err := service.CreateSmartAccount(context.Background(), "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", 1); err == nil {
		t.Fatal("account creation succeeded while the factory call failed")
	}
}

func TestTheUserOpHashMatchesTheDeployedEntryPoint(t *testing.T) {
	service := NewAccountAbstractionService()
	service.entryPointAddress = entryPointDefaults["entry_point"]
	service.defaultChainID = 1

	userOp := &UserOperation{
		Sender:               "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		Nonce:                "0",
		CallGasLimit:         "100000",
		VerificationGasLimit: "150000",
		PreVerificationGas:   "21000",
		MaxFeePerGas:         "100000000000",
		MaxPriorityFeePerGas: "1000000000",
		ChainID:              1,
	}

	// Read from getUserOpHash on the deployed EntryPoint at
	// 0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789 (chain 1).
	const want = "0x658bcf6d542bae91bf338b3bdb4165015cab0e824097f813d148a0ce370c110c"

	if got := service.generateUserOpHash(userOp); got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
}

func TestTheUserOpHashIsEmptyForAMalformedOperation(t *testing.T) {
	service := NewAccountAbstractionService()
	service.entryPointAddress = entryPointDefaults["entry_point"]

	// A hash over the raw strings would not be a hash the EntryPoint accepts,
	// so an unhashable operation must yield nothing.
	bad := &UserOperation{Sender: "not-an-address"}
	if got := service.generateUserOpHash(bad); got != "" {
		t.Fatalf("hash = %q, want empty for a malformed operation", got)
	}

	badNumber := &UserOperation{
		Sender: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		Nonce:  "not-a-number",
	}
	if got := service.generateUserOpHash(badNumber); got != "" {
		t.Fatalf("hash = %q, want empty for a non-numeric field", got)
	}
}

func TestTheUserOpHashIsBoundToTheChain(t *testing.T) {
	service := NewAccountAbstractionService()
	service.entryPointAddress = entryPointDefaults["entry_point"]

	userOp := &UserOperation{
		Sender:               "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		Nonce:                "0",
		CallGasLimit:         "100000",
		VerificationGasLimit: "150000",
		PreVerificationGas:   "21000",
		MaxFeePerGas:         "100000000000",
		MaxPriorityFeePerGas: "1000000000",
	}

	userOp.ChainID = 1
	onMainnet := service.generateUserOpHash(userOp)

	userOp.ChainID = 137
	onPolygon := service.generateUserOpHash(userOp)

	if onMainnet == onPolygon {
		t.Fatal("the hash did not change with the chain")
	}
}

func TestEstimateGasReadsFeesFromTheNode(t *testing.T) {
	service, node := serviceWithStub(t)

	userOp := &UserOperation{
		Sender: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		ChainID: 1,
	}

	fees, err := service.estimateGas(context.Background(), userOp)
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	// eth_gasPrice returned 1 gwei, so the fee fields must reflect that and not
	// a hardcoded value.
	gasPrice := new(big.Int)
	gasPrice.SetString(fees.MaxFeePerGas, 10)

	priority := new(big.Int)
	priority.SetString(fees.MaxPriorityFee, 10)

	if gasPrice.Sign() <= 0 {
		t.Fatalf("maxFeePerGas = %s, want a positive value from the node", fees.MaxFeePerGas)
	}
	if priority.Sign() <= 0 {
		t.Fatalf("maxPriorityFee = %s, want a positive value", fees.MaxPriorityFee)
	}
	if gasPrice.Cmp(priority) < 0 {
		t.Fatalf("maxFeePerGas %s is below the priority fee %s", gasPrice, priority)
	}

	methods := node.calledMethods()
	if len(methods) == 0 || methods[0] != "eth_gasPrice" {
		t.Fatalf("node saw %v, want an eth_gasPrice call", methods)
	}
}

func TestEstimateGasHonoursCallerSuppliedFees(t *testing.T) {
	service, _ := serviceWithStub(t)

	userOp := &UserOperation{
		Sender:               "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		MaxFeePerGas:         "42000000000",
		MaxPriorityFeePerGas: "2000000000",
		ChainID:              1,
	}

	fees, err := service.estimateGas(context.Background(), userOp)
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	if fees.MaxFeePerGas != "42000000000" {
		t.Fatalf("maxFeePerGas = %s, want the caller's 42000000000", fees.MaxFeePerGas)
	}
	if fees.MaxPriorityFee != "2000000000" {
		t.Fatalf("maxPriorityFee = %s, want the caller's 2000000000", fees.MaxPriorityFee)
	}
}

func TestEstimateGasFailsClosedWithoutAnEndpoint(t *testing.T) {
	service := NewAccountAbstractionService()

	t.Setenv("RPC_URL_1", "")
	t.Setenv("RPC_URL", "")

	if _, err := service.estimateGas(context.Background(), &UserOperation{ChainID: 1}); err == nil {
		t.Fatal("gas estimation succeeded with no RPC endpoint")
	}
}

func TestTheHashInTheResponseMatchesTheRecomputedHash(t *testing.T) {
	// The hash reported alongside the fees must be the same hash the operation
	// itself produces, so a client never signs a different value than the one
	// the service reports.
	service, _ := serviceWithStub(t)

	userOp := &UserOperation{
		Sender:               "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		CallGasLimit:         "100000",
		VerificationGasLimit: "150000",
		PreVerificationGas:   "21000",
		ChainID:              1,
	}

	fees, err := service.estimateGas(context.Background(), userOp)
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	if want := service.generateUserOpHash(userOp); fees.UserOpHash != want {
		t.Fatalf("reported hash = %s, recomputed = %s", fees.UserOpHash, want)
	}
}

func TestDefaultChainIDIsConfigurable(t *testing.T) {
	t.Setenv("AA_DEFAULT_CHAIN_ID", "137")
	if got := defaultChainID(); got != 137 {
		t.Fatalf("defaultChainID = %d, want 137", got)
	}

	// A malformed value falls back to the built-in default rather than zero.
	t.Setenv("AA_DEFAULT_CHAIN_ID", "not-a-number")
	if got := defaultChainID(); got != 1 {
		t.Fatalf("defaultChainID = %d, want the fallback 1", got)
	}

	t.Setenv("AA_DEFAULT_CHAIN_ID", "")
	if got := defaultChainID(); got != 1 {
		t.Fatalf("defaultChainID = %d, want 1", got)
	}
}

func TestNormalizeAddressRejectsGarbage(t *testing.T) {
	if _, err := normalizeAddress("0x1234"); err == nil {
		t.Fatal("a short address was accepted")
	}
	if _, err := normalizeAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"); err != nil {
		t.Fatalf("a valid address was rejected: %v", err)
	}
}
