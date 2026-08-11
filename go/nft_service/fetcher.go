// fetcher.go — REAL on-chain NFT fetcher (no mock data).
//
// Uses go-ethereum ethclient to perform real eth_call reads against ERC-721 /
// ERC-1155 contracts:
//   - balanceOf(address)            -> uint256
//   - tokenOfOwnerByIndex(address,uint256) -> uint256   (ERC-721Enumerable)
//   - ownerOf(uint256)              -> address            (ERC-721)
//   - tokenURI(uint256)             -> string             (ERC-721)
//   - uri(uint256)                  -> string             (ERC-1155)
//   - name() / symbol()             -> string
//   - totalSupply()                 -> uint256            (ERC-721Enumerable)
//
// Metadata is fetched over HTTP(S) from the tokenURI (supports raw http(s) and
// ipfs:// gateway resolution). Results are cached in Redis (60s TTL) to keep
// on-chain call volume low. If no RPC endpoint is configured (ETH_RPC_URL empty),
// the fetcher reports "unavailable" rather than returning fabricated data.

package main

import (
	"context"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
)

// cryptoKeccak256 wraps go-ethereum's keccak256 for selector computation.
func cryptoKeccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}

// Fetcher performs real on-chain NFT reads.
type Fetcher struct {
	client *ethclient.Client
	redis  *redis.Client
}

// NewFetcher dials the configured RPC. Returns nil (and no error) when no RPC is
// configured; callers must nil-check and report "unavailable" instead of faking.
func NewFetcher(rpcURL, redisAddr string) (*Fetcher, error) {
	if strings.TrimSpace(rpcURL) == "" {
		return nil, nil
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	return &Fetcher{client: client, redis: rdb}, nil
}

// Available reports whether on-chain reads are possible.
func (f *Fetcher) Available() bool { return f != nil && f.client != nil }

// error returned when the fetcher is not configured.
var errFetcherUnavailable = errors.New("on-chain NFT fetcher unavailable: ETH_RPC_URL not set")

// callContract performs a real eth_call against the given contract with manual
// calldata (selector + ABI-encoded args). Returns the raw return bytes.
func (f *Fetcher) callContract(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{To: &to, Data: data}
	return f.client.CallContract(ctx, msg, nil)
}

// selector returns the 4-byte function selector for keccak256(sig)[:4].
func selector(sig string) []byte {
	h := cryptoKeccak256([]byte(sig))
	return h[:4]
}

// addrArg pads a 20-byte address to 32 bytes (left-padded).
func addrArg(a common.Address) []byte {
	b := make([]byte, 32)
	copy(b[12:], a.Bytes())
	return b
}

// uintArg encodes a *big.Int as a 32-byte big-endian arg.
func uintArg(n *big.Int) []byte {
	b := make([]byte, 32)
	n.FillBytes(b)
	return b
}

// ERC-721 / ERC-1155 standard call helpers ------------------------------------

// BalanceOf721 returns the number of ERC-721 tokens owned by owner at contract.
func (f *Fetcher) BalanceOf721(ctx context.Context, contract, owner common.Address) (*big.Int, error) {
	data := append(selector("balanceOf(address)"), addrArg(owner)...)
	out, err := f.callContract(ctx, contract, data)
	if err != nil {
		return nil, err
	}
	if len(out) < 32 {
		return new(big.Int), nil
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// TokenOfOwnerByIndex returns tokenId at the given index for owner (ERC-721Enumerable).
func (f *Fetcher) TokenOfOwnerByIndex(ctx context.Context, contract, owner common.Address, index *big.Int) (*big.Int, error) {
	data := append(selector("tokenOfOwnerByIndex(address,uint256)"), addrArg(owner)...)
	data = append(data, uintArg(index)...)
	out, err := f.callContract(ctx, contract, data)
	if err != nil {
		return nil, err
	}
	if len(out) < 32 {
		return nil, errors.New("empty return")
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// OwnerOf returns the owner of an ERC-721 tokenId.
func (f *Fetcher) OwnerOf(ctx context.Context, contract common.Address, tokenId *big.Int) (common.Address, error) {
	data := append(selector("ownerOf(uint256)"), uintArg(tokenId)...)
	out, err := f.callContract(ctx, contract, data)
	if err != nil {
		return common.Address{}, err
	}
	if len(out) < 32 {
		return common.Address{}, errors.New("empty return")
	}
	return common.BytesToAddress(out[12:32]), nil
}

// TokenURI returns the metadata URI for an ERC-721 token.
func (f *Fetcher) TokenURI(ctx context.Context, contract common.Address, tokenId *big.Int) (string, error) {
	data := append(selector("tokenURI(uint256)"), uintArg(tokenId)...)
	return f.callString(ctx, contract, data)
}

// TokenURI1155 returns the metadata URI for an ERC-1155 token (uri(uint256)).
func (f *Fetcher) TokenURI1155(ctx context.Context, contract common.Address, tokenId *big.Int) (string, error) {
	data := append(selector("uri(uint256)"), uintArg(tokenId)...)
	return f.callString(ctx, contract, data)
}

// Name returns the ERC-721/1155 collection name.
func (f *Fetcher) Name(ctx context.Context, contract common.Address) (string, error) {
	return f.callString(ctx, contract, selector("name()"))
}

// Symbol returns the ERC-721/1155 collection symbol.
func (f *Fetcher) Symbol(ctx context.Context, contract common.Address) (string, error) {
	return f.callString(ctx, contract, selector("symbol()"))
}

// TotalSupply returns the total token count (ERC-721Enumerable).
func (f *Fetcher) TotalSupply(ctx context.Context, contract common.Address) (*big.Int, error) {
	out, err := f.callContract(ctx, contract, selector("totalSupply()"))
	if err != nil {
		return nil, err
	}
	if len(out) < 32 {
		return new(big.Int), nil
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

// callString decodes an ABI-encoded string return value.
func (f *Fetcher) callString(ctx context.Context, contract common.Address, data []byte) (string, error) {
	out, err := f.callContract(ctx, contract, data)
	if err != nil {
		return "", err
	}
	return decodeABIString(out)
}

// decodeABIString decodes a Solidity string return (offset(32) + len(32) + data).
func decodeABIString(out []byte) (string, error) {
	if len(out) < 64 {
		return "", errors.New("malformed string return")
	}
	length := new(big.Int).SetBytes(out[32:64]).Int64()
	if length < 0 || int(length)+64 > len(out) {
		return "", errors.New("string length out of range")
	}
	return string(out[64 : 64+int(length)]), nil
}

// fetchMetadata fetches + decodes NFT metadata JSON from a token URI, resolving
// ipfs:// URIs through a public gateway. Returns name, image, description, attrs.
type nftMetadata struct {
	Name        string         `json:"name"`
	Image       string         `json:"image"`
	Description string         `json:"description"`
	Animation   string         `json:"animation_url"`
	External    string         `json:"external_url"`
	Attributes  []NFTAttribute `json:"attributes"`
}

func fetchMetadata(ctx context.Context, tokenURI string) (*nftMetadata, error) {
	if tokenURI == "" {
		return nil, errors.New("empty token uri")
	}
	url := resolveURI(tokenURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TigerWallet-NFTService/1.0")
	cl := &http.Client{Timeout: 8 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("metadata http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB cap
	if err != nil {
		return nil, err
	}
	var md nftMetadata
	if err := json.Unmarshal(body, &md); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &md, nil
}

// resolveURI rewrites ipfs:// and ipns:// schemes to HTTPS gateway URLs.
func resolveURI(uri string) string {
	switch {
	case strings.HasPrefix(uri, "ipfs://"):
		cid := strings.TrimPrefix(uri, "ipfs://")
		return "https://ipfs.io/ipfs/" + cid
	case strings.HasPrefix(uri, "ipns://"):
		cid := strings.TrimPrefix(uri, "ipns://")
		return "https://ipfs.io/ipns/" + cid
	default:
		return uri
	}
}

// FetchUserNFTs returns every ERC-721 token owned by an address at a contract,
// with on-chain owner + fetched metadata. Cached per (contract,owner) in Redis.
func (f *Fetcher) FetchUserNFTs(ctx context.Context, contract, owner common.Address) ([]*NFT, error) {
	if !f.Available() {
		return nil, errFetcherUnavailable
	}
	cacheKey := fmt.Sprintf("nft:owner:%s:%s", contract.Hex(), owner.Hex())
	if cached, err := f.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var out []*NFT
		if json.Unmarshal(cached, &out) == nil {
			return out, nil
		}
	}

	balance, err := f.BalanceOf721(ctx, contract, owner)
	if err != nil {
		return nil, fmt.Errorf("balanceOf: %w", err)
	}
	count := balance.Int64()
	if count < 0 {
		count = 0
	}
	// Cap to a sane upper bound to avoid unbounded on-chain call loops.
	if count > 200 {
		count = 200
	}

	colName, _ := f.Name(ctx, contract)
	colSym, _ := f.Symbol(ctx, contract)

	results := make([]*NFT, 0, count)
	for i := int64(0); i < count; i++ {
		tokenID, err := f.TokenOfOwnerByIndex(ctx, contract, owner, big.NewInt(i))
		if err != nil {
			continue
		}
		nft := &NFT{
			Chain:           "ethereum",
			ContractAddress: contract.Hex(),
			TokenID:         tokenID.String(),
			Owner:           owner.Hex(),
			Name:            fmt.Sprintf("%s #%s", colName, tokenID.String()),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if colName != "" {
			nft.CollectionID = colName
		}
		_ = colSym

		if uri, err := f.TokenURI(ctx, contract, tokenID); err == nil {
			nft.TokenURI = uri
			if md, err := fetchMetadata(ctx, uri); err == nil {
				if md.Name != "" {
					nft.Name = md.Name
				}
				nft.ImageURL = md.Image
				nft.Description = md.Description
				nft.AnimationURL = md.Animation
				nft.ExternalURL = md.External
				if len(md.Attributes) > 0 {
					nft.Attributes = md.Attributes
				}
			}
		}
		results = append(results, nft)
	}

	if blob, err := json.Marshal(results); err == nil {
		f.redis.Set(ctx, cacheKey, blob, 60*time.Second)
	}
	return results, nil
}

// FetchCollectionStats reads on-chain name/symbol/totalSupply for a contract.
func (f *Fetcher) FetchCollectionStats(ctx context.Context, contract common.Address) (name, symbol, supply string, err error) {
	if !f.Available() {
		return "", "", "", errFetcherUnavailable
	}
	if n, e := f.Name(ctx, contract); e == nil {
		name = n
	}
	if s, e := f.Symbol(ctx, contract); e == nil {
		symbol = s
	}
	if t, e := f.TotalSupply(ctx, contract); e == nil {
		supply = t.String()
	}
	return name, symbol, supply, nil
}
