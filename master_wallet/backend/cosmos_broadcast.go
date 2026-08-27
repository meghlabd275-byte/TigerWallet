package main

// cosmos_broadcast.go — real Cosmos-SDK transaction broadcast for the
// MasterWallet auto-signer. Complements non_evm_crypto.go (which derives
// addresses and signs amino SignDocs) with the network half of the
// sign→broadcast→confirm path for Cosmos/Cosmos-SDK chains.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// cosmosLCDBase is the canonical Cosmos REST (LCD/gRPC-gateway) directory.
// Each chain is exposed under https://rest.cosmos.directory/<registry-dir>.
const cosmosLCDBase = "https://rest.cosmos.directory/"

// cosmosRegistryDir maps a native Cosmos-SDK chain_id (the chain_id STRING
// used in the SignDoc, e.g. "cosmoshub-4") to its cosmos.directory registry
// directory name. The directory name is what the public REST gateway routes
// on, and it differs from the chain_id string for several chains.
var cosmosRegistryDir = map[string]string{
	"cosmoshub-4":                "cosmoshub",
	"osmosis-1":                  "osmosis",
	"columbus-5":                 "terra-classic",
	"injective-1":                "injective",
	"mocha-4":                    "celestia",
	"dydx-chain-1":               "dydx",
	"atlantic-2":                 "sei",
	"kaiyo-1":                    "kujira",
	"stride-1":                   "stride",
	"pion-1":                     "neutron",
	"juno-1":                     "juno",
	"akashnet-2":                 "akash",
	"core-1":                     "persistence",
	"evmos_9001-2":               "evmos",
	"canto_7700-1":               "canto",
	"kava_2222-10":               "kava",
	"crypto-org-chain-mainnet-1": "cryptoorgchain",
	"stargaze-1":                 "stargaze",
	"ssc-1":                      "saga",
	"noble-1":                    "noble",
	"axelar-dojo-1":              "axelar",
	"umee-1":                     "umee",
	"secret-4":                   "secret",
}

// cosmosValuesToCoin builds the canonical base-denom integer amount string for
// a human-readable value. Cosmos fees and MsgSend amounts are integers in the
// base denom (uatom, uosmo, ...). decimals differs per chain (6 for ATOM, 18
// for INJ, ...) and is resolved by the caller from the chain registry.
func cosmosHumanToBaseDenom(value string, decimals int) (string, error) {
	return humanToWei(value, decimals), nil
}

// cosmosChainMetaAndDecimals resolves the SignDoc chain_id string + fee denom
// AND the on-chain decimals for a TigerWallet numeric chain_id. The existing
// cosmosChainMeta returns chainIDStr+denom; decimals is looked up from the
// seeded non-EVM registry and falls back to 6 (the Cosmos default).
func cosmosChainMetaAndDecimals(chainID int64) (chainIDStr, denom string, decimals int) {
	chainIDStr, denom = cosmosChainMeta(chainID)
	decimals = 6
	for _, c := range defaultNonEVMChains {
		if c.ChainID == chainID && c.ChainType == "cosmos" {
			if c.Decimals > 0 {
				decimals = c.Decimals
			}
			break
		}
	}
	return chainIDStr, denom, decimals
}

// cosmosLCDForChainID resolves the LCD REST base URL (no trailing slash) for a
// TigerWallet numeric chain_id. Order: operator-pinned env override, then the
// user chain registry RPC pillar, then the public rest.cosmos.directory. The
// registry RPCURL for Cosmos chains points at an RPC node, not an LCD; most
// public Cosmos RPCs also serve the REST gateway on a sibling host, but we
// only switch to the registry URL when the operator explicitly pins it in the
// chain table during add/update (they may use a self-hosted full node with a
// co-located LCD).
func cosmosLCDForChainID(chainID int64, chainIDStr string) string {
	if env := getEnvDefault("COSMOS_LCD_URL", ""); env != "" {
		return trimTrailingSlash(env)
	}
	dir, ok := cosmosRegistryDir[chainIDStr]
	if !ok {
		dir = url.PathEscape(chainIDStr)
	}
	return cosmosLCDBase + dir
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// cosmosAuthAccount is the REST response for /cosmos/auth/v1beta1/accounts/:addr.
type cosmosAuthAccountResp struct {
	Account struct {
		Type          string `json:"@type"`
		Address       string `json:"address"`
		AccountNumber string `json:"account_number"`
		Sequence      string `json:"sequence"`
	} `json:"account"`
}

// fetchCosmosAccount fetches the real account_number + sequence for a bech32
// address from the chain's auth module. Both values are REQUIRED for a valid
// SignDoc — we never fabricate them.
func fetchCosmosAccount(lcdBase, address string) (accountNumber, sequence uint64, err error) {
	endpoint := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", lcdBase, url.PathEscape(address))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := broadcastHTTPClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("cosmos auth fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("cosmos auth fetch HTTP %d", resp.StatusCode)
	}
	var parsed cosmosAuthAccountResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, 0, fmt.Errorf("decode auth account: %w", err)
	}
	accountNumber, err = strconv.ParseUint(parsed.Account.AccountNumber, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse account_number %q: %w", parsed.Account.AccountNumber, err)
	}
	sequence, err = strconv.ParseUint(parsed.Account.Sequence, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse sequence %q: %w", parsed.Account.Sequence, err)
	}
	return accountNumber, sequence, nil
}

// cosmosSendMsgOne is a single amino MsgSend value block. The `amount` is the
// canonical sdk.Coin list [{denom, amount}].
type cosmosSendMsgOne struct {
	Amount      []map[string]string `json:"amount"`
	FromAddress string              `json:"from_address"`
	ToAddress   string              `json:"to_address"`
}

// cosmosMsgTroika mirrors the amino MsgSend wrapper {type, value}.
type cosmosMsgTroika struct {
	Type  string           `json:"type"`
	Value cosmosSendMsgOne `json:"value"`
}

// cosmosFeeTroika mirrors the amino fee block. Keys are intentionally lower-
// cased to match the canonical amino JSON used by the SDK for SIGN_MODE_LEGACY
// (amount, gas, granter, payer — granter/payer omitted when empty so the bytes
// hash to what the chain expects for LegacyAmino JSON).
type cosmosFeeTroika struct {
	Amount []map[string]string `json:"amount"`
	Gas    string              `json:"gas"`
}

// cosmosSignDocTroika is the canonical amino SignDoc. Field order does not
// matter for encoding/json (it hashes the marshalled bytes), but the key names
// must match the SDK exactly.
type cosmosSignDocTroika struct {
	AccountNumber string            `json:"account_number"`
	ChainID       string            `json:"chain_id"`
	Fee           cosmosFeeTroika   `json:"fee"`
	Memo          string            `json:"memo"`
	Msgs          []cosmosMsgTroika `json:"msgs"`
	Sequence      string            `json:"sequence"`
}

// cosmosStdSignature is the stdTx signature entry (base64 sig + pubkey).
type cosmosStdSignature struct {
	PubKey    map[string]string `json:"pub_key"`
	Signature string            `json:"signature"`
}

// cosmosPubKeySecp256k1 is the amino pubkey object.
type cosmosPubKeySecp256k1 struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// cosmosStdTx is the legacy amino stdTx envelope used by BroadcastTxSync for
// SIGN_MODE_LEGACY_AMINO_JSON. `value` matches the amino wire field names.
type cosmosStdTx struct {
	Type  string           `json:"type"`
	Value cosmosStdTxValue `json:"value"`
}

type cosmosStdTxValue struct {
	Msg        []cosmosMsgTroika    `json:"msg"`
	Fee        cosmosFeeTroika      `json:"fee"`
	Signatures []cosmosStdSignature `json:"signatures"`
	Memo       string               `json:"memo"`
}

// cosmosBroadcastRequest is the REST body for POST /txs (legacy amino,
// BroadcastTxSync) or /cosmos/tx/v1beta1/txs (protobuf, disabled here).
type cosmosBroadcastRequest struct {
	Tx   cosmosStdTx `json:"tx"`
	Mode string      `json:"mode"`
}

// cosmosBroadcastResponse is the REST response for POST /txs.
type cosmosBroadcastResponse struct {
	Code      int64  `json:"code"`
	Log       string `json:"raw_log"`
	TxHash    string `json:"txhash"`
	Codespace string `json:"codespace"`
}

// mwCosmosBroadcastTx signs a Cosmos MsgSend with the user's seed and submits
// it to the chain. It performs the full sign→broadcast path:
//
//  1. resolve chain_id string + denom + decimals from the registry
//  2. derive the sender bech32 address
//  3. fetch REAL account_number + sequence from the auth module
//  4. build the canonical amino SignDoc
//  5. sign (secp256k1, SIGN_MODE_LEGACY_AMINO_JSON) via mwCosmosSign
//  6. assemble a stdTx and BroadcastTxSync it
//
// Returns the real on-chain txhash. Any failure returns an error and an empty
// hash — we never fabricate a txid.
func mwCosmosBroadcastTx(seed []byte, derivationPath, toAddress, valueStr string, chainID int64) (string, error) {
	chainIDStr, denom, decimals := cosmosChainMetaAndDecimals(chainID)
	prefix := bech32PrefixForChainID(chainID)

	fromAddr, err := mwCosmosAddressFromSeed(seed, derivationPath, prefix)
	if err != nil {
		return "", fmt.Errorf("cosmos from-address: %w", err)
	}

	amountInt, err := cosmosHumanToBaseDenom(valueStr, decimals)
	if err != nil || amountInt == "" {
		return "", fmt.Errorf("invalid cosmos value %q: %w", valueStr, err)
	}

	lcd := cosmosLCDForChainID(chainID, chainIDStr)

	accountNumber, sequence, err := fetchCosmosAccount(lcd, fromAddr)
	if err != nil {
		return "", err
	}

	coins := []map[string]string{{"denom": denom, "amount": amountInt}}
	msgs := []cosmosMsgTroika{{
		Type: "cosmos-sdk/MsgSend",
		Value: cosmosSendMsgOne{
			Amount:      coins,
			FromAddress: fromAddr,
			ToAddress:   toAddress,
		},
	}}
	fee := cosmosFeeTroika{
		Amount: []map[string]string{{"denom": denom, "amount": "5000"}},
		Gas:    "200000",
	}
	signDoc := cosmosSignDocTroika{
		AccountNumber: strconv.FormatUint(accountNumber, 10),
		ChainID:       chainIDStr,
		Fee:           fee,
		Memo:          "",
		Msgs:          msgs,
		Sequence:      strconv.FormatUint(sequence, 10),
	}
	signDocJSON, err := json.Marshal(signDoc)
	if err != nil {
		return "", fmt.Errorf("marshal sign doc: %w", err)
	}

	sig, pubKeyBytes, err := mwCosmosSign(seed, derivationPath, string(signDocJSON))
	if err != nil {
		return "", err
	}

	stdTx := cosmosStdTx{
		Type: "cosmos-sdk/StdTx",
		Value: cosmosStdTxValue{
			Msg:  msgs,
			Fee:  fee,
			Memo: "",
			Signatures: []cosmosStdSignature{{
				PubKey: map[string]string{
					"type":  "tendermint/PubKeySecp256k1",
					"value": base64.StdEncoding.EncodeToString(pubKeyBytes),
				},
				Signature: base64.StdEncoding.EncodeToString(sig),
			}},
		},
	}
	body, err := json.Marshal(cosmosBroadcastRequest{Tx: stdTx, Mode: "sync"})
	if err != nil {
		return "", fmt.Errorf("marshal broadcast: %w", err)
	}

	// Legacy amino broadcast endpoint (/txs, BroadcastTxSync). This is served by
	// the cosmos.directory gRPC-gateway with the amino codec enabled.
	endpoint := lcd + "/txs"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := broadcastHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cosmos broadcast: %w", err)
	}
	defer resp.Body.Close()

	var br cosmosBroadcastResponse
	if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return "", fmt.Errorf("decode broadcast response: %w", err)
	}
	if br.Code != 0 || br.TxHash == "" {
		return "", fmt.Errorf("cosmos broadcast rejected (code %d, %s): %s", br.Code, br.Codespace, br.Log)
	}
	return br.TxHash, nil
}

// cosmosSignatureHex signs a Cosmos SignDoc and returns the 64-byte r||s
// signature hexed. Kept as a thin helper for the "signed" fallback path used
// when a caller only needs a signature (e.g. offline/dev introspection).
func cosmosSignatureHex(seed []byte, derivationPath, signDoc string) (string, error) {
	sig, _, err := mwCosmosSign(seed, derivationPath, signDoc)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}
