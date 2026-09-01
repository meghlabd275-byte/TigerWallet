package main

// non_evm_handlers.go — HTTP handlers for the full 66-chain non-EVM SDK.
// Requests resolve chain_type (or numeric chain_id) through
// non_evm_registry.go and dispatch to the chain-family SDK. Nothing falls
// back to fabrication — invalid or unsupported families fail closed with a
// descriptive error.

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---------- request shapes ----------

type nonEvmSignReq struct {
	WalletID  string `json:"wallet_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Message   string `json:"message" binding:"required"`
	ChainType string `json:"chain_type"`
	ChainID   int64  `json:"chain_id"`
	Prefix    byte   `json:"prefix"`
}

type nonEvmSendReq struct {
	WalletID  string `json:"wallet_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
	ChainType string `json:"chain_type"`
	ChainID   int64  `json:"chain_id"`

	To       string `json:"to"`
	Amount   string `json:"amount"`
	Denom    string `json:"denom"`
	Memo     string `json:"memo"`
	Fee      string `json:"fee"`
	Gas      uint64 `json:"gas"`
	Simulate bool   `json:"simulate"`

	BitcoinInputs  []BTCInput     `json:"bitcoin_inputs"`
	BitcoinOutputs []BTCOutput    `json:"bitcoin_outputs"`
	UTXOInputs     []UTXOInput    `json:"utxo_inputs"`
	UTXOOutputs    []UTXOOutput   `json:"utxo_outputs"`
	CosmosSignDoc  *CosmosSignDoc `json:"cosmos_sign_doc"`

	Sequence  uint64  `json:"sequence"`
	MaxGas    uint64  `json:"max_gas"`
	GenHash   string  `json:"genesis_hash"`
	FirstV    uint64  `json:"first_valid"`
	LastV     uint64  `json:"last_valid"`
	Storage   uint64  `json:"storage"`
	Counter   uint64  `json:"counter"`
	DestTag   *uint32 `json:"dest_tag"`
	WorkHex   string  `json:"work"`
	RemBalHex string  `json:"remaining_balance_hex"`
	Nonce     uint64  `json:"nonce"`
	BHash     string  `json:"block_hash_b58"`
	Prefix    byte    `json:"prefix"`
}

// ---------- address derivation ----------

// handleNonEvmAddress derives the address for the chain-family.
func handleNonEvmAddress(c *gin.Context) {
	var req struct {
		WalletID  string `json:"wallet_id" binding:"required"`
		Password  string `json:"password" binding:"required"`
		ChainType string `json:"chain_type"`
		ChainID   int64  `json:"chain_id"`
		Prefix    byte   `json:"prefix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	family, ct, err := nonEvmResolve(req.ChainType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addr, err := nonEvmAddressFor(family, ct, req.ChainID, seed, wallet.DerivationPath, req.Prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr, "chain_type": ct, "family": string(family)})
}

// nonEvmAddressFor routes address derivation by family.
func nonEvmAddressFor(family nonEvmFamily, ct string, chainID int64, seed []byte, path string, prefix byte) (string, error) {
	switch family {
	case familyUTXO:
		if ct == "zcash" {
			return ZECAddressFromSeed(seed, path)
		}
		return UTXOAddressFromSeed(seed, path, ct)
	case familyCosmos:
		addr, _, err := CosmosAddressFromSeedNew(seed, path, chainID)
		return addr, err
	case familySolana:
		pub, err := edPubKey(seed, path)
		if err != nil {
			return "", err
		}
		return SolanaAddressFromKey(pub)
	case familyAptos:
		return AptosAddress(seed, path)
	case familySui:
		return SuiAddress(seed, path)
	case familyNear:
		return NearAddress(seed, path)
	case familyStellar:
		return StrKeyAddress(seed, path)
	case familyAlgorand:
		return AlgoAddress(seed, path)
	case familyNano:
		return NanoAddress(seed, path)
	case familyMultiversX:
		return MultiversXAddress(seed, path)
	case familyTezos:
		return TezosAddress(seed, path)
	case familyCardano:
		return CardanoAddress(seed, path)
	case familyWaves:
		return WavesAddress(seed, path)
	case familyTron:
		return TronAddress(seed, path)
	case familyVeChain:
		return VETAddress(seed, path)
	case familyRipple:
		return XRPAddress(seed, path)
	case familyICP:
		return ICPAddress(seed, path)
	case familyZilliqa:
		return ZilAddress(seed, path)
	case familyKaspa:
		return KaspaAddress(seed, path)
	case familyNervos:
		return NervosAddress(seed, path)
	case familyFilecoin:
		return FilAddress(seed, path)
	case familySubstrate:
		return SubstrateAddress(seed, prefix)
	case familyAleo, familyHedera, familyFlow:
		return "", nonEvmNotFeasible(family)
	case familyTON:
		return "", errors.New("ton: wallet v4r2 state-init cell derivation is fail-closed (ed25519 message signing stays available)")
	}
	return "", fmt.Errorf("unknown non-EVM chain %q", ct)
}

// ---------- message signing ----------

// handleNonEvmSign signs a message with the family's signer.
func handleNonEvmSign(c *gin.Context) {
	var req nonEvmSignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	family, ct, err := nonEvmResolve(req.ChainType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = ct // chain_type is informational for the sign path
	var sig, pub []byte
	switch family {
	case familyUTXO:
		sig, pub, err = btcSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case familyCosmos:
		doc := &CosmosSignDoc{AccountNumber: "0", Fee: CosmosFee{Gas: "0"}, Memo: req.Message, Sequence: "0"}
		sig, pub, err = CosmosSign(seed, wallet.DerivationPath, doc)
	case familySolana, familyAptos, familySui, familyNear, familyAlgorand,
		familyNano, familyMultiversX, familyTezos, familyStellar, familyTON:
		sig, pub, err = edSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case familyCardano:
		sig, pub, err = CardanoSign(seed, wallet.DerivationPath, []byte(req.Message))
	case familyWaves:
		sig, pub, err = WavesSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case familyTron, familyVeChain, familyRipple, familyICP, familyZilliqa:
		sig, pub, err = secpMessageSign(seed, wallet.DerivationPath, []byte(req.Message))
	case familyKaspa:
		sig, pub, err = KaspaSign(seed, wallet.DerivationPath, []byte(req.Message))
	case familyNervos:
		sig, pub, err = NervosSign(seed, wallet.DerivationPath, []byte(req.Message))
	case familyFilecoin:
		sig, pub, err = FilSign(seed, wallet.DerivationPath, []byte(req.Message))
	case familySubstrate:
		sig, pub, err = substrateSign(seed, wallet.DerivationPath, []byte(req.Message), req.Prefix)
	case familyAleo, familyHedera, familyFlow:
		err = nonEvmNotFeasible(family)
	default:
		err = fmt.Errorf("family %q message-sign not routable", family)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"signature":  hex0x(sig),
		"public_key": hex0x(pub),
		"chain_type": req.ChainType,
		"family":     string(family),
	})
}

// ---------- tx send ----------

// handleNonEvmSend dispatches to the family's builder.
func handleNonEvmSend(c *gin.Context) {
	var req nonEvmSendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	family, ct, err := nonEvmResolve(req.ChainType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw, txHash, famStr, err := nonEvmDispatchSend(c.Request.Context(), family, ct, req, seed, wallet.DerivationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if txHash != "" {
		c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "raw": raw, "family": famStr, "chain_type": ct, "action": "Transaction broadcast to the blockchain network"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"raw_tx": raw, "family": famStr, "chain_type": ct, "action": "broadcast via the chain's node"})
}

// nonEvmDispatchSend executes the send-builder dispatch for every family.
func nonEvmDispatchSend(ctx context.Context, family nonEvmFamily, ct string, req nonEvmSendReq, seed []byte, path string) (raw, txHash, famStr string, err error) {
	famStr = string(family)
	switch family {
	case familyUTXO:
		if len(req.UTXOInputs) == 0 && len(req.BitcoinInputs) > 0 {
			for _, x := range req.BitcoinInputs {
				req.UTXOInputs = append(req.UTXOInputs, UTXOInput{TxID: x.TxID, Vout: x.Vout, ScriptPubKey: x.ScriptPubKey})
			}
			for _, x := range req.BitcoinOutputs {
				req.UTXOOutputs = append(req.UTXOOutputs, UTXOOutput{Address: x.Address, AmountSat: x.AmountSat})
			}
		}
		if ct == "zcash" {
			raw, err := ZECSign(seed, path, req.UTXOInputs, req.UTXOOutputs)
			return raw, "", famStr, err
		}
		raw, err := UTXOSign(seed, path, ct, req.UTXOInputs, req.UTXOOutputs)
		return raw, "", famStr, err

	case familyCosmos:
		if req.ChainID == 0 {
			return "", "", "", fmt.Errorf("cosmos send needs {chain_id} for prefix resolution")
		}
		denom := req.Denom
		if denom == "" {
			denom = defaultDenomOf(req.ChainID, ct)
		}
		resw, err := CosmosExecuteSend(ctx, seed, path, CosmosSendRequest{
			ChainID:   req.ChainID,
			ToAddress: req.To,
			Denom:     denom,
			Amount:    req.Amount,
			GasLimit:  req.Gas,
			Fee:       req.Fee,
			Memo:      req.Memo,
			Simulate:  req.Simulate,
		})
		if err != nil {
			return "", "", famStr, err
		}
		return resw.TxBytes, resw.TxHash, famStr, nil

	case familySolana:
		raw, txHash, err := SolanaBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, parseUintOr(req.Amount), !req.Simulate)
		return raw, txHash, famStr, err

	case familyAptos:
		amount := parseUintOr(req.Amount)
		raw, txHash, err := AptosBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, amount, req.Sequence, req.MaxGas, 100, 0, 0, !req.Simulate)
		return raw, txHash, famStr, err

	case familySui:
		amount := parseUintOr(req.Amount)
		raw, txHash, err := SuiBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, amount, !req.Simulate)
		return raw, txHash, famStr, err

	case familyNear:
		amt := new(big.Int)
		amt.SetString(req.Amount, 10)
		raw, txHash, err := NearBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, amt, NearParams{Nonce: req.Nonce, BlockHashB58: req.BHash}, !req.Simulate)
		return raw, txHash, famStr, err

	case familyStellar:
		amount := parseInt64Or(req.Amount)
		raw, txHash, err := StellarBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), ct, req.To, amount, req.Sequence, 100, !req.Simulate)
		return raw, txHash, famStr, err

	case familyAlgorand:
		raw, txHash, err := AlgoBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), "", req.To, AlgoBuildRequest{
			GenesisHashB64: req.GenHash,
			Fee:            req.Gas,
			FirstValid:     req.FirstV,
			LastValid:      req.LastV,
			Amount:         parseUintOr(req.Amount),
		}, !req.Simulate)
		return raw, txHash, famStr, err

	case familyNano:
		rem, err := hex.DecodeString(req.RemBalHex)
		if err != nil {
			return "", "", famStr, fmt.Errorf("nano remaining_balance_hex malformed: %w", err)
		}
		raw, txHash, err := NanoBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, rem, req.WorkHex, !req.Simulate)
		return raw, txHash, famStr, err

	case familyMultiversX:
		raw, txHash, err := MultiversXBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, req.Amount, !req.Simulate)
		return raw, txHash, famStr, err

	case familyTezos:
		raw, txHash, err := TezosBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, parseUintOr(req.Amount), req.Gas, req.Counter, req.Gas, req.Storage, !req.Simulate)
		return raw, txHash, famStr, err

	case familyTron:
		raw, txHash, err := TronBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, parseUintOr(req.Amount), !req.Simulate)
		return raw, txHash, famStr, err

	case familyVeChain:
		raw, txHash, err := VETBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, parseUintOr(req.Amount), req.Gas, !req.Simulate)
		return raw, txHash, famStr, err

	case familyRipple:
		raw, txHash, err := RippleBuildSend(ctx, seed, path, lookupEndpoint(req.ChainID, ct), req.To, parseUintOr(req.Amount), req.DestTag, !req.Simulate)
		return raw, txHash, famStr, err

	case familyCardano:
		return "", "", "", errors.New("cardano tx build requires CBOR serialization — fail-closed; message signing + address derivation stay complete")

	case familyWaves:
		return "", "", "", errors.New("waves tx build requires protobuf tx shape — fail-closed; address derivation + message signing stay complete")

	case familyTON:
		return "", "", "", errors.New("ton tx build requires BoC serialization — fail-closed")

	case familyICP, familyZilliqa, familyKaspa, familyNervos, familyFilecoin:
		return "", "", "", fmt.Errorf("%s tx build is fail-closed in this build (address + message-sign only)", family)

	case familyAleo, familyHedera, familyFlow:
		return "", "", "", nonEvmNotFeasible(family)

	case familySubstrate:
		return "", "", "", errors.New("substrate tx build requires SCALE encoding — fail-closed; address + message signing stay complete")
	}
	return "", "", "", fmt.Errorf("family %q not routable for send", family)
}

// ---------- small numeric helpers ----------

func parseUintOr(s string) uint64 {
	if s == "" {
		return 0
	}
	v, _ := new(big.Int).SetString(s, 10)
	if v == nil {
		return 0
	}
	return v.Uint64()
}

func parseInt64Or(s string) int64 {
	v := parseUintOr(s)
	return int64(v)
}

// defaultDenomOf resolves the default base denom for a cosmos chain by id.
func defaultDenomOf(chainID int64, ct string) string {
	switch chainID {
	case 9000000118: // Cosmos Hub
		return "uatom"
	case 9000026317: // Osmosis
		return "uosmo"
	case 9000073068: // Injective
		return "inj"
	case 9000012099: // Stride
		return "ustrd"
	case 9000000529: // Secret
		return "uscrt"
	case 9000007183: // Akash
		return "uakt"
	case 9000090063: // Neutron
		return "untrn"
	case 9000073741: // Sei
		return "usei"
	case 9000014648: // Celestia
		return "utia"
	case 9000049823: // dYdX
		return "adydx"
	case 9000005267: // Juno
		return "ujuno"
	case 9000041857: // Kujira
		return "ukuji"
	case 9000000330: // Terra Classic
		return "uluna"
	case 9000018759: // Persistence
		return "uxprt"
	case 9000000160: // Evmos
		return "aevmos"
	case 9000000900: // Kava
		return "ukava"
	case 9000001839: // Cronos
		return "basecro"
	case 9000007017: // Axelar
		return "uaxl"
	case 9000008102: // Band
		return "uband"
	case 9000000438: // IRIS
		return "uiris"
	case 9000008899: // Gravity Bridge
		return "ugraviton"
	case 9000001183: // Stargaze
		return "ustars"
	case 9000002236: // Sommelier
		return "usomm"
	}
	// Fallback: lower-cased chain-type (correct for most SDK chains, where
	// the base denom starts with 'u' — call sites can always pass {denom}).
	return "u" + ct
}

// ---------- shared handler helpers (carried over from the 3-chain build) ----------

// loadOwnedSeed decrypts the wallet seed after verifying ownership + password.
func loadOwnedSeed(c *gin.Context, walletIDStr, password string) ([]byte, *WalletRecord, error) {
	wid, err := uuid.Parse(walletIDStr)
	if err != nil {
		return nil, nil, errInvalidWallet
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		return nil, nil, errWalletNotFound
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		return nil, nil, errNotOwner
	}
	if wallet.IsWatchOnly {
		return nil, nil, fmt.Errorf("watch-only wallet cannot sign")
	}
	seed, err := DecryptSeed(wallet.EncryptedSeed, password)
	if err != nil {
		return nil, nil, errBadPassword
	}
	return seed, wallet, nil
}

var (
	errInvalidWallet  = &publicErr{"invalid wallet_id"}
	errWalletNotFound = &publicErr{"wallet not found"}
	errNotOwner       = &publicErr{"wallet does not belong to user"}
	errBadPassword    = &publicErr{"incorrect password"}
)

type publicErr struct{ msg string }

func (e *publicErr) Error() string { return e.msg }

// hex0x returns the "0x"-prefixed lowercase hex of b.
func hex0x(b []byte) string { return "0x" + hex.EncodeToString(b) }

// btcSignMessage signs a message with the secp256k1 key (r||s, no recovery).
func btcSignMessage(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	full, err := crypto.Sign(message, priv)
	if err != nil {
		return nil, nil, err
	}
	return full[:64], crypto.CompressPubkey(&priv.PublicKey), nil
}

// secpMessageSign signs a message with the secp256k1 key (65-byte r||s||v).
func secpMessageSign(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	h := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
	full, err := crypto.Sign(h.Bytes(), priv)
	if err != nil {
		return nil, nil, err
	}
	return full, crypto.CompressPubkey(&priv.PublicKey), nil
}

// lookupEndpoint resolves the seeded non-EVM endpoint for a chain id; falls
// back to the well-known public endpoint for the chain type otherwise.
func lookupEndpoint(chainID int64, ct string) string {
	for _, c := range nonEVMMainnet {
		if c.ID == chainID {
			return c.RPCEndpoint
		}
	}
	if ct == "pi" {
		return "" // pi endpoint operator-configured (env PI_RPC)
	}
	switch ct {
	case "solana":
		return "https://api.mainnet-beta.solana.com"
	case "aptos":
		return "https://fullnode.mainnet.aptoslabs.com"
	case "sui":
		return "https://fullnode.mainnet.sui.io:443"
	case "near":
		return "https://rpc.mainnet.near.org"
	case "algorand":
		return "https://mainnet-api.algonode.cloud"
	case "nano":
		return "https://proxy.nanos.cc"
	case "tron":
		return "https://api.trongrid.io"
	case "vechain":
		return "https://sync-mainnet.vechain.org"
	case "ripple":
		return "https://s1.ripple.com:51234"
	case "stellar", "pi":
		return "https://api.mainnet.stellar.org"
	case "tezos":
		return "https://mainnet.tezos.ecadinfra.com"
	case "elrond":
		return "https://api.multiversx.com"
	}
	return ""
}
