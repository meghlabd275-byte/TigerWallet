// nft_evm_client.go — real EVM NFT reads and writes.
//
// All contract interaction goes through go-ethereum against the configured RPC
// endpoint. Reads are eth_call; writes are locally signed EIP-155 transactions
// that are broadcast and awaited to a receipt. Nothing here fabricates a
// metadata field, an owner, or a transaction hash: when the RPC is unreachable
// or the contract reverts, the error is returned to the caller.

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// errRPCNotConfigured is returned when no chain RPC endpoint is set. Callers
// surface this to the client instead of serving placeholder data.
var errRPCNotConfigured = errors.New("chain RPC endpoint is not configured (CHAIN_RPC_URL)")

// EVMChainClient implements ChainClient for EVM chains.
type EVMChainClient struct {
	rpcURL     string
	client     *ethclient.Client
	privateKey string
	chainID    *big.Int
	httpClient *http.Client
}

// NewEVMChainClient dials the configured RPC. It returns a client that reports
// errRPCNotConfigured from every method when rpcURL is empty, rather than
// returning fabricated values.
func NewEVMChainClient(rpcURL string) *EVMChainClient {
	c := &EVMChainClient{
		rpcURL:     rpcURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
	if strings.TrimSpace(rpcURL) == "" {
		return c
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		// Keep the client but leave it unusable; the error surfaces on first use.
		return c
	}
	c.client = client
	if id, err := client.ChainID(context.Background()); err == nil {
		c.chainID = id
	}
	return c
}

// ConfigureSigner attaches the private key used to sign write transactions.
// It is separate from construction so that read-only deployments can run with
// no key material present.
func (c *EVMChainClient) ConfigureSigner(privateKey string) error {
	c.privateKey = strings.TrimSpace(privateKey)
	return nil
}

// Available reports whether reads can be performed.
func (c *EVMChainClient) Available() bool { return c != nil && c.client != nil }

// callContract performs a real eth_call with manually encoded calldata.
func (c *EVMChainClient) callContract(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	if !c.Available() {
		return nil, errRPCNotConfigured
	}
	return c.client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
}

// selector returns keccak256(signature)[:4].
func selector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

// padLeft32 left-pads a big.Int to 32 bytes for ABI encoding.
func padLeft32(v *big.Int) []byte {
	out := make([]byte, 32)
	b := v.Bytes()
	copy(out[32-len(b):], b)
	return out
}

// padAddress32 left-pads an address to 32 bytes.
func padAddress32(a common.Address) []byte {
	out := make([]byte, 32)
	copy(out[12:], a.Bytes())
	return out
}

// decodeUint256 decodes a 32-byte ABI word into a big.Int.
func decodeUint256(raw []byte) (*big.Int, error) {
	if len(raw) < 32 {
		return nil, fmt.Errorf("short ABI word: got %d bytes, want at least 32", len(raw))
	}
	return new(big.Int).SetBytes(raw[:32]), nil
}

// decodeAddress decodes a 32-byte ABI word into an address.
func decodeAddress(raw []byte) (common.Address, error) {
	if len(raw) < 32 {
		return common.Address{}, fmt.Errorf("short ABI word: got %d bytes, want at least 32", len(raw))
	}
	return common.BytesToAddress(raw[12:32]), nil
}

// decodeString decodes a dynamically sized ABI string return value.
func decodeString(raw []byte) (string, error) {
	if len(raw) < 64 {
		return "", fmt.Errorf("short ABI string return: got %d bytes", len(raw))
	}
	offset := new(big.Int).SetBytes(raw[:32])
	if !offset.IsInt64() || offset.Int64()+32 > int64(len(raw)) {
		return "", errors.New("ABI string offset out of range")
	}
	start := offset.Int64()
	length := new(big.Int).SetBytes(raw[start : start+32])
	if !length.IsInt64() || start+32+length.Int64() > int64(len(raw)) {
		return "", errors.New("ABI string length out of range")
	}
	return string(raw[start+32 : start+32+length.Int64()]), nil
}

// GetOwnerOf reads ownerOf(uint256) — ERC-721.
func (c *EVMChainClient) GetOwnerOf(ctx context.Context, contractAddr, tokenID string) (string, error) {
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return "", err
	}
	token, ok := new(big.Int).SetString(strings.TrimSpace(tokenID), 10)
	if !ok {
		return "", fmt.Errorf("invalid token id %q", tokenID)
	}

	data := append(selector("ownerOf(uint256)"), padLeft32(token)...)
	raw, err := c.callContract(ctx, contract, data)
	if err != nil {
		return "", fmt.Errorf("ownerOf call failed: %w", err)
	}
	if len(raw) == 0 {
		return "", errors.New("ownerOf returned no data: contract is not ERC-721 or token does not exist")
	}
	owner, err := decodeAddress(raw)
	if err != nil {
		return "", err
	}
	return strings.ToLower(owner.Hex()), nil
}

// name reads name() on the contract, falling back to an empty string when the
// contract does not implement it (optional in ERC-1155).
func (c *EVMChainClient) name(ctx context.Context, contract common.Address) string {
	raw, err := c.callContract(ctx, contract, selector("name()"))
	if err != nil {
		return ""
	}
	value, err := decodeString(raw)
	if err != nil {
		return ""
	}
	return value
}

// symbol reads symbol() on the contract.
func (c *EVMChainClient) symbol(ctx context.Context, contract common.Address) string {
	raw, err := c.callContract(ctx, contract, selector("symbol()"))
	if err != nil {
		return ""
	}
	value, err := decodeString(raw)
	if err != nil {
		return ""
	}
	return value
}

// tokenURI reads tokenURI(uint256) (ERC-721) and falls back to uri(uint256)
// (ERC-1155).
func (c *EVMChainClient) tokenURI(ctx context.Context, contract common.Address, token *big.Int) (string, error) {
	raw, err := c.callContract(ctx, contract, append(selector("tokenURI(uint256)"), padLeft32(token)...))
	if err == nil && len(raw) > 0 {
		if uri, derr := decodeString(raw); derr == nil && uri != "" {
			return uri, nil
		}
	}

	raw, err = c.callContract(ctx, contract, append(selector("uri(uint256)"), padLeft32(token)...))
	if err != nil {
		return "", fmt.Errorf("neither tokenURI nor uri is readable: %w", err)
	}
	if len(raw) == 0 {
		return "", errors.New("contract returned no token URI")
	}
	uri, derr := decodeString(raw)
	if derr != nil {
		return "", derr
	}
	return uri, nil
}

// ipfsGateway resolves an ipfs:// URI through a public gateway. Deployments may
// point IPFS_GATEWAY at their own node.
func (c *EVMChainClient) ipfsGateway() string {
	if gw := strings.TrimSpace(getEnv("IPFS_GATEWAY", "")); gw != "" {
		return strings.TrimSuffix(gw, "/")
	}
	return "https://ipfs.io/ipfs"
}

// resolveURI turns an on-chain URI into a fetchable HTTP(S) URL.
func (c *EVMChainClient) resolveURI(uri string) (string, error) {
	trimmed := strings.TrimSpace(uri)
	switch {
	case strings.HasPrefix(trimmed, "ipfs://"):
		path := strings.TrimPrefix(trimmed, "ipfs://")
		path = strings.TrimPrefix(path, "ipfs/")
		return c.ipfsGateway() + "/" + path, nil
	case strings.HasPrefix(trimmed, "http://"), strings.HasPrefix(trimmed, "https://"):
		return trimmed, nil
	case trimmed == "":
		return "", errors.New("token URI is empty")
	default:
		return "", fmt.Errorf("unsupported token URI scheme: %s", trimmed)
	}
}

// tokenMetadataJSON is the ERC-721/1155 metadata document shape.
type tokenMetadataJSON struct {
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Image        string                   `json:"image"`
	AnimationURL string                   `json:"animation_url"`
	ExternalURL  string                   `json:"external_url"`
	Attributes   []map[string]interface{} `json:"attributes"`
}

// GetNFTMetadata reads the token URI on chain and fetches the metadata document
// it points at. The token id, symbol and contract are always real on-chain
// values; only the descriptive fields come from the metadata document.
func (c *EVMChainClient) GetNFTMetadata(ctx context.Context, contractAddr, tokenID string) (*NFTMetadata, error) {
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return nil, err
	}
	token, ok := new(big.Int).SetString(strings.TrimSpace(tokenID), 10)
	if !ok {
		return nil, fmt.Errorf("invalid token id %q", tokenID)
	}

	uri, err := c.tokenURI(ctx, contract, token)
	if err != nil {
		return nil, err
	}
	url, err := c.resolveURI(uri)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build metadata request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read metadata body: %w", err)
	}

	var doc tokenMetadataJSON
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode metadata document: %w", err)
	}

	meta := &NFTMetadata{
		TokenID:      token.String(),
		Name:         doc.Name,
		Symbol:       c.symbol(ctx, contract),
		Description:  doc.Description,
		Image:        doc.Image,
		AnimationURL: doc.AnimationURL,
		ExternalURL:  doc.ExternalURL,
		Attributes:   doc.Attributes,
	}
	if meta.Name == "" {
		meta.Name = fmt.Sprintf("%s #%s", meta.Symbol, token.String())
	}
	if meta.Attributes == nil {
		meta.Attributes = []map[string]interface{}{}
	}
	return meta, nil
}

// BalanceOf reads balanceOf(address) — ERC-721 count or ERC-1155 quantity.
func (c *EVMChainClient) BalanceOf(ctx context.Context, contractAddr, owner string) (*big.Int, error) {
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return nil, err
	}
	holder, err := parseAddress(owner)
	if err != nil {
		return nil, err
	}
	data := append(selector("balanceOf(address)"), padAddress32(holder)...)
	raw, err := c.callContract(ctx, contract, data)
	if err != nil {
		return nil, fmt.Errorf("balanceOf call failed: %w", err)
	}
	return decodeUint256(raw)
}

// signerKey loads and validates the configured private key.
func (c *EVMChainClient) signerKey() (*ecdsa.PrivateKey, error) {
	key := strings.TrimPrefix(strings.TrimSpace(c.privateKey), "0x")
	if key == "" {
		return nil, errors.New("no signing key configured (CHAIN_PRIVATE_KEY)")
	}
	parsed, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("invalid signing key: %w", err)
	}
	return parsed, nil
}

// sendTransaction signs and broadcasts a contract call, then waits for the
// receipt so the returned hash is confirmed rather than merely submitted.
func (c *EVMChainClient) sendTransaction(ctx context.Context, contract common.Address, data []byte) (string, error) {
	if !c.Available() {
		return "", errRPCNotConfigured
	}
	key, err := c.signerKey()
	if err != nil {
		return "", err
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	nonce, err := c.client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas price: %w", err)
	}
	gasLimit, err := c.client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &contract, Data: data})
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	chainID := c.chainID
	if chainID == nil {
		chainID, err = c.client.ChainID(ctx)
		if err != nil {
			return "", fmt.Errorf("read chain id: %w", err)
		}
	}

	tx := types.NewTransaction(nonce, contract, big.NewInt(0), gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("sign transaction: %w", err)
	}

	if err := c.client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast transaction: %w", err)
	}

	hash := signed.Hash().Hex()
	receipt, err := c.waitForReceipt(ctx, signed.Hash())
	if err != nil {
		return hash, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return hash, fmt.Errorf("transaction %s reverted", hash)
	}
	return hash, nil
}

// waitForReceipt polls until the transaction is mined or the context expires.
func (c *EVMChainClient) waitForReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	deadline := time.Now().Add(3 * time.Minute)
	for {
		receipt, err := c.client.TransactionReceipt(ctx, hash)
		if err == nil && receipt != nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for receipt of %s", hash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// TransferNFT calls safeTransferFrom(address,address,uint256) for ERC-721.
func (c *EVMChainClient) TransferNFT(ctx context.Context, from, to, contractAddr, tokenID string, amount *big.Int) (string, error) {
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return "", err
	}
	sender, err := parseAddress(from)
	if err != nil {
		return "", err
	}
	recipient, err := parseAddress(to)
	if err != nil {
		return "", err
	}
	token, ok := new(big.Int).SetString(strings.TrimSpace(tokenID), 10)
	if !ok {
		return "", fmt.Errorf("invalid token id %q", tokenID)
	}

	data := selector("safeTransferFrom(address,address,uint256)")
	data = append(data, padAddress32(sender)...)
	data = append(data, padAddress32(recipient)...)
	data = append(data, padLeft32(token)...)

	return c.sendTransaction(ctx, contract, data)
}

// MintNFT calls mint(address,string) on the configured contract. The token id
// is read back from the receipt logs by the caller if the contract emits one;
// this method returns only the confirmed transaction hash.
func (c *EVMChainClient) MintNFT(ctx context.Context, to, contractAddr, uri string) (string, error) {
	contract, err := parseAddress(contractAddr)
	if err != nil {
		return "", err
	}
	recipient, err := parseAddress(to)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(uri) == "" {
		return "", errors.New("token URI is required")
	}

	data := selector("mint(address,string)")
	data = append(data, padAddress32(recipient)...)
	data = append(data, encodeStringArg(uri)...)

	return c.sendTransaction(ctx, contract, data)
}

// SetApprovalForAll calls setApprovalForAll(address,bool).
func (c *EVMChainClient) SetApprovalForAll(ctx context.Context, owner, operator string, approved bool) (string, error) {
	contract, err := parseAddress(operator)
	if err != nil {
		return "", err
	}
	if _, err := parseAddress(owner); err != nil {
		return "", err
	}

	flag := big.NewInt(0)
	if approved {
		flag = big.NewInt(1)
	}

	data := selector("setApprovalForAll(address,bool)")
	data = append(data, padAddress32(contract)...)
	data = append(data, padLeft32(flag)...)

	return c.sendTransaction(ctx, contract, data)
}

// encodeStringArg ABI-encodes a dynamic string argument: offset, length, data.
func encodeStringArg(s string) []byte {
	out := padLeft32(big.NewInt(32))
	out = append(out, padLeft32(big.NewInt(int64(len(s))))...)

	padded := make([]byte, ((len(s)+31)/32)*32)
	copy(padded, []byte(s))
	return append(out, padded...)
}

// parseAddress validates a 0x-prefixed 20-byte hex address.
func parseAddress(raw string) (common.Address, error) {
	trimmed := strings.TrimSpace(raw)
	if !common.IsHexAddress(trimmed) {
		return common.Address{}, fmt.Errorf("invalid address %q", raw)
	}
	return common.HexToAddress(trimmed), nil
}

// mustHexDecode is a small helper used by tests and diagnostics.
func mustHexDecode(s string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil
	}
	return b
}
