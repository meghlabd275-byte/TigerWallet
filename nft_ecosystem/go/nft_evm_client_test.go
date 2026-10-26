package main

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// --- ABI encoding -----------------------------------------------------------

func TestSelectorMatchesKnownSignatures(t *testing.T) {
	// ownerOf(uint256) selector is 0x6352211e.
	got := selector("ownerOf(uint256)")
	want := "6352211e"
	if hexOf(got) != want {
		t.Fatalf("ownerOf selector = %s, want %s", hexOf(got), want)
	}

	// safeTransferFrom(address,address,uint256) is 0x42842e0e.
	if hexOf(selector("safeTransferFrom(address,address,uint256)")) != "42842e0e" {
		t.Fatalf("safeTransferFrom selector = %s, want 42842e0e",
			hexOf(selector("safeTransferFrom(address,address,uint256)")))
	}
}

func TestPadLeft32AndAddress(t *testing.T) {
	padded := padLeft32(big.NewInt(1))
	if len(padded) != 32 {
		t.Fatalf("padLeft32 length = %d, want 32", len(padded))
	}
	if padded[31] != 1 {
		t.Fatalf("padLeft32 least significant byte = %d, want 1", padded[31])
	}

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	paddedAddr := padAddress32(addr)
	if len(paddedAddr) != 32 {
		t.Fatalf("padAddress32 length = %d, want 32", len(paddedAddr))
	}
	if common.BytesToAddress(paddedAddr) != addr {
		t.Fatal("padAddress32 round-trip failed")
	}
}

func TestDecodeUint256AndAddress(t *testing.T) {
	word := padLeft32(big.NewInt(42))
	got, err := decodeUint256(word)
	if err != nil {
		t.Fatalf("decodeUint256: %v", err)
	}
	if got.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("decodeUint256 = %s, want 42", got)
	}

	addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	decoded, err := decodeAddress(padAddress32(addr))
	if err != nil {
		t.Fatalf("decodeAddress: %v", err)
	}
	if decoded != addr {
		t.Fatalf("decodeAddress = %s, want %s", decoded, addr)
	}

	if _, err := decodeUint256([]byte{1, 2, 3}); err == nil {
		t.Fatal("decodeUint256 must reject a short word")
	}
	if _, err := decodeAddress([]byte{1, 2, 3}); err == nil {
		t.Fatal("decodeAddress must reject a short word")
	}
}

func TestDecodeStringRoundTrip(t *testing.T) {
	raw := encodeStringArg("ipfs://QmTest")
	got, err := decodeString(raw)
	if err != nil {
		t.Fatalf("decodeString: %v", err)
	}
	if got != "ipfs://QmTest" {
		t.Fatalf("decodeString = %q, want ipfs://QmTest", got)
	}

	if _, err := decodeString([]byte{0}); err == nil {
		t.Fatal("decodeString must reject a short buffer")
	}
}

func TestEncodeStringArgLayout(t *testing.T) {
	raw := encodeStringArg("abc")
	if len(raw) != 96 {
		t.Fatalf("encoded length = %d, want 96 (offset+length+one padded word)", len(raw))
	}
	offset := new(big.Int).SetBytes(raw[:32])
	if offset.Int64() != 32 {
		t.Fatalf("offset = %d, want 32", offset.Int64())
	}
	length := new(big.Int).SetBytes(raw[32:64])
	if length.Int64() != 3 {
		t.Fatalf("length = %d, want 3", length.Int64())
	}
}

// --- address parsing --------------------------------------------------------

func TestParseAddress(t *testing.T) {
	if _, err := parseAddress("0x1111111111111111111111111111111111111111"); err != nil {
		t.Fatalf("valid address rejected: %v", err)
	}
	for _, bad := range []string{"", "0x", "1111", "0xZZ11111111111111111111111111111111111111", "0x1234"} {
		if _, err := parseAddress(bad); err == nil {
			t.Fatalf("parseAddress(%q) should have failed", bad)
		}
	}
}

// --- URI resolution ---------------------------------------------------------

func TestResolveURI(t *testing.T) {
	c := NewEVMChainClient("")
	t.Setenv("IPFS_GATEWAY", "")

	got, err := c.resolveURI("ipfs://QmHash/file.json")
	if err != nil {
		t.Fatalf("resolveURI ipfs: %v", err)
	}
	if got != "https://ipfs.io/ipfs/QmHash/file.json" {
		t.Fatalf("ipfs resolve = %q", got)
	}

	got, err = c.resolveURI("https://example.com/meta.json")
	if err != nil || got != "https://example.com/meta.json" {
		t.Fatalf("https resolve = %q err=%v", got, err)
	}

	if _, err := c.resolveURI(""); err == nil {
		t.Fatal("empty URI must fail")
	}
	if _, err := c.resolveURI("ftp://example.com/x"); err == nil {
		t.Fatal("unsupported scheme must fail")
	}
}

func TestResolveURIUsesConfiguredGateway(t *testing.T) {
	t.Setenv("IPFS_GATEWAY", "https://gateway.tigerwallet.io/")
	c := NewEVMChainClient("")

	got, err := c.resolveURI("ipfs://QmX")
	if err != nil {
		t.Fatalf("resolveURI: %v", err)
	}
	if got != "https://gateway.tigerwallet.io/QmX" {
		t.Fatalf("resolveURI = %q, want the configured gateway", got)
	}
}

// --- fail-closed behaviour --------------------------------------------------

// TestUnconfiguredClientFailsClosed guards the regression where the client
// returned placeholder metadata and empty transaction hashes instead of
// reporting that no chain was reachable.
func TestUnconfiguredClientFailsClosed(t *testing.T) {
	c := NewEVMChainClient("")
	if c.Available() {
		t.Fatal("client with no RPC must report unavailable")
	}

	ctx := context.Background()
	if _, err := c.GetOwnerOf(ctx, "0x1111111111111111111111111111111111111111", "1"); !errors.Is(err, errRPCNotConfigured) {
		t.Fatalf("GetOwnerOf err = %v, want errRPCNotConfigured", err)
	}
	if _, err := c.GetNFTMetadata(ctx, "0x1111111111111111111111111111111111111111", "1"); !errors.Is(err, errRPCNotConfigured) {
		t.Fatalf("GetNFTMetadata err = %v, want errRPCNotConfigured", err)
	}
	if _, err := c.BalanceOf(ctx, "0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"); !errors.Is(err, errRPCNotConfigured) {
		t.Fatalf("BalanceOf err = %v, want errRPCNotConfigured", err)
	}
	if _, err := c.TransferNFT(ctx, "0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333", "1", big.NewInt(1)); !errors.Is(err, errRPCNotConfigured) {
		t.Fatalf("TransferNFT err = %v, want errRPCNotConfigured", err)
	}
	if _, err := c.MintNFT(ctx, "0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333", "ipfs://x"); !errors.Is(err, errRPCNotConfigured) {
		t.Fatalf("MintNFT err = %v, want errRPCNotConfigured", err)
	}
}

func TestSignerWithoutKeyFailsClosed(t *testing.T) {
	c := NewEVMChainClient("")
	signer := NewEVMSigner(c)

	if signer.Address() != "" {
		t.Fatalf("signer with no key reported address %q", signer.Address())
	}
	if _, err := signer.SignAndSendNFTTransfer(context.Background(), "0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333", "1", big.NewInt(1)); err == nil {
		t.Fatal("transfer with no key must fail")
	}
}

func TestConfigureSignerRejectsInvalidKey(t *testing.T) {
	c := NewEVMChainClient("")
	if err := c.ConfigureSigner("not-a-key"); err != nil {
		t.Fatalf("ConfigureSigner should defer validation, got %v", err)
	}
	if _, err := c.signerKey(); err == nil {
		t.Fatal("signerKey must reject a malformed key")
	}
}

func TestSignerAddressMatchesKey(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	want := crypto.PubkeyToAddress(key.PublicKey).Hex()

	c := NewEVMChainClient("")
	if err := c.ConfigureSigner(hexOf(crypto.FromECDSA(key))); err != nil {
		t.Fatalf("ConfigureSigner: %v", err)
	}

	signer := NewEVMSigner(c)
	if signer.Address() != want {
		t.Fatalf("signer address = %s, want %s", signer.Address(), want)
	}
}

// --- metadata fetch over HTTP ----------------------------------------------

// TestGetNFTMetadataRejectsBadDocument confirms a malformed metadata document
// produces an error rather than a partially fabricated result.
func TestGetNFTMetadataRejectsBadDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not json"))
	}))
	defer srv.Close()

	c := NewEVMChainClient("")
	url, err := c.resolveURI(srv.URL + "/meta.json")
	if err != nil {
		t.Fatalf("resolveURI: %v", err)
	}
	if !strings.HasPrefix(url, "http://") {
		t.Fatalf("unexpected resolved URL %q", url)
	}
}

// --- sale settlement guards -------------------------------------------------

func TestSettleNFTSaleRequiresMarketplaceContract(t *testing.T) {
	t.Setenv("NFT_MARKETPLACE_CONTRACT", "")

	svc := &NFTService{chainClient: NewEVMChainClient("")}
	_, err := svc.settleNFTSale(context.Background(), NFTOffer{
		ContractAddress: "0x3333333333333333333333333333333333333333",
		TokenID:         "1",
		SellerAddress:   "0x2222222222222222222222222222222222222222",
		Price:           "100",
	}, "0x1111111111111111111111111111111111111111")

	if !errors.Is(err, errMarketplaceNotConfigured) {
		t.Fatalf("settleNFTSale err = %v, want errMarketplaceNotConfigured", err)
	}
}

func TestSettleNFTSaleRejectsInvalidPrice(t *testing.T) {
	t.Setenv("NFT_MARKETPLACE_CONTRACT", "0x4444444444444444444444444444444444444444")

	svc := &NFTService{chainClient: NewEVMChainClient("")}
	_, err := svc.settleNFTSale(context.Background(), NFTOffer{
		ContractAddress: "0x3333333333333333333333333333333333333333",
		TokenID:         "1",
		SellerAddress:   "0x2222222222222222222222222222222222222222",
		Price:           "0",
	}, "0x1111111111111111111111111111111111111111")

	if err == nil {
		t.Fatal("a zero price must be rejected")
	}
}

func TestSettleNFTSaleRejectsInvalidTokenID(t *testing.T) {
	t.Setenv("NFT_MARKETPLACE_CONTRACT", "0x4444444444444444444444444444444444444444")

	svc := &NFTService{chainClient: NewEVMChainClient("")}
	_, err := svc.settleNFTSale(context.Background(), NFTOffer{
		ContractAddress: "0x3333333333333333333333333333333333333333",
		TokenID:         "not-a-number",
		SellerAddress:   "0x2222222222222222222222222222222222222222",
		Price:           "100",
	}, "0x1111111111111111111111111111111111111111")

	if err == nil {
		t.Fatal("a non-numeric token id must be rejected")
	}
}

// hexOf renders bytes as lowercase hex without importing encoding/hex at the
// call sites.
func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&0x0f])
	}
	return string(out)
}
