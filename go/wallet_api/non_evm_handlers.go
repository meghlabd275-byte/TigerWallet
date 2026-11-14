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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/chainsdk"
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

	BitcoinInputs  []chainsdk.BTCInput     `json:"bitcoin_inputs"`
	BitcoinOutputs []chainsdk.BTCOutput    `json:"bitcoin_outputs"`
	UTXOInputs     []chainsdk.UTXOInput    `json:"utxo_inputs"`
	UTXOOutputs    []chainsdk.UTXOOutput   `json:"utxo_outputs"`
	CosmosSignDoc  *chainsdk.CosmosSignDoc `json:"cosmos_sign_doc"`

	// Cardano selects its own UTXO set: a Shelley body is built from explicit
	// inputs, so the caller passes the exact outputs to spend and the fee.
	CardanoInputs  []chainsdk.CardanoUTXO   `json:"cardano_inputs"`
	CardanoOutputs []chainsdk.CardanoOutput `json:"cardano_outputs"`
	FeeLovelace    uint64                   `json:"fee_lovelace"`

	// Substrate (Polkadot/Kusama) runtime values. Spec version, transaction
	// version, genesis hash and the era anchor all come from the node's runtime
	// metadata because they change on every runtime upgrade. The genesis hash
	// reuses GenHash below, which carries the same value for Algorand.
	SpecVersion uint32 `json:"spec_version"`
	TxVersion   uint32 `json:"tx_version"`
	BlockHash   string `json:"block_hash"`
	Tip         uint64 `json:"tip"`
	EraPeriod   uint64 `json:"era_period"`
	// PalletIndex and CallIndex are the balances.transferKeepAlive dispatch
	// indices from the runtime metadata. They are runtime-specific, so the
	// caller supplies them rather than the backend guessing.
	PalletIndex byte `json:"pallet_index"`
	CallIndex   byte `json:"call_index"`

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
	family, ct, err := chainsdk.ResolveFamily(req.ChainType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addr, err := chainsdk.AddressFor(family, ct, req.ChainID, seed, wallet.DerivationPath, req.Prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr, "chain_type": ct, "family": string(family)})
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
	family, ct, err := chainsdk.ResolveFamily(req.ChainType, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = ct // chain_type is informational for the sign path
	var sig, pub []byte
	switch family {
	case chainsdk.FamilyUTXO:
		sig, pub, err = chainsdk.BTCMessageSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyCosmos:
		doc := &chainsdk.CosmosSignDoc{AccountNumber: "0", Fee: chainsdk.CosmosFee{Gas: "0"}, Memo: req.Message, Sequence: "0"}
		sig, pub, err = chainsdk.CosmosSign(seed, wallet.DerivationPath, doc)
	case chainsdk.FamilySolana, chainsdk.FamilyAptos, chainsdk.FamilySui, chainsdk.FamilyNear, chainsdk.FamilyAlgorand,
		chainsdk.FamilyNano, chainsdk.FamilyMultiversX, chainsdk.FamilyTezos, chainsdk.FamilyStellar, chainsdk.FamilyTON:
		sig, pub, err = chainsdk.EdSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyCardano:
		sig, pub, err = chainsdk.CardanoSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyWaves:
		sig, pub, err = chainsdk.WavesSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyTron, chainsdk.FamilyVeChain, chainsdk.FamilyRipple, chainsdk.FamilyICP, chainsdk.FamilyZilliqa:
		sig, pub, err = chainsdk.SecpMessageSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyKaspa:
		sig, pub, err = chainsdk.KaspaSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyNervos:
		sig, pub, err = chainsdk.NervosSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyFilecoin:
		sig, pub, err = chainsdk.FilSign(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyStarknet:
		sig, pub, err = chainsdk.StarknetSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyMonero:
		sig, pub, err = chainsdk.MoneroSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyErgo:
		sig, pub, err = chainsdk.ErgoSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyStacks:
		sig, pub, err = chainsdk.StacksSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyNEO:
		sig, pub, err = chainsdk.NEOSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyMina:
		sig, pub, err = chainsdk.EdSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyUnsupported:
		err = chainsdk.ExtraNotFeasible(family, ct)
	case chainsdk.FamilySubstrate:
		sig, pub, err = chainsdk.SubstrateSign(seed, wallet.DerivationPath, []byte(req.Message), req.Prefix)
	case chainsdk.FamilyGraphene, chainsdk.FamilyNEM, chainsdk.FamilySiacoin:
		sig, pub, err = chainsdk.EdSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case chainsdk.FamilyAleo, chainsdk.FamilyHedera, chainsdk.FamilyFlow:
		err = chainsdk.NotFeasible(family)
	default:
		err = fmt.Errorf("family %q message-sign not routable", family)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"signature":  chainsdk.Hex0x(sig),
		"public_key": chainsdk.Hex0x(pub),
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
	family, ct, err := chainsdk.ResolveFamily(req.ChainType, req.ChainID)
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
func nonEvmDispatchSend(ctx context.Context, family chainsdk.Family, ct string, req nonEvmSendReq, seed []byte, path string) (raw, txHash, famStr string, err error) {
	famStr = string(family)
	switch family {
	case chainsdk.FamilyStarknet, chainsdk.FamilyMonero, chainsdk.FamilyArweave, chainsdk.FamilyMina, chainsdk.FamilyUnsupported:
		// These chains need a signed-payload format that depends on node state
		// (Starknet nonce + class hash, Monero ring members, Mina zkApp
		// account, Arweave RSA signing) or have no offline address scheme at
		// all. Broadcast is fail-closed: the backend never returns a
		// plausible-looking transaction it did not really build.
		return "", "", famStr, chainsdk.ExtraNotFeasible(family, ct)
	case chainsdk.FamilyStacks, chainsdk.FamilyNEO, chainsdk.FamilyErgo:
		// Message signing is real for these chains (see the sign dispatch);
		// building a spend transaction additionally needs the chain's own
		// serialization rules (Stacks clarity post-conditions, the NEO VM
		// script build, Ergo box selection), which this backend does not
		// implement. Fail closed rather than emitting an envelope no node
		// would accept.
		return "", "", famStr, errors.New(string(family) + ": transaction building needs the chain's own serializer, which is not implemented here — message signing remains available; fail-closed rather than emitting a transaction no node accepts")
	case chainsdk.FamilyUTXO:
		if len(req.UTXOInputs) == 0 && len(req.BitcoinInputs) > 0 {
			for _, x := range req.BitcoinInputs {
				req.UTXOInputs = append(req.UTXOInputs, chainsdk.UTXOInput{TxID: x.TxID, Vout: x.Vout, ScriptPubKey: x.ScriptPubKey})
			}
			for _, x := range req.BitcoinOutputs {
				req.UTXOOutputs = append(req.UTXOOutputs, chainsdk.UTXOOutput{Address: x.Address, AmountSat: x.AmountSat})
			}
		}
		if ct == "zcash" {
			raw, err := chainsdk.ZECSign(seed, path, req.UTXOInputs, req.UTXOOutputs)
			return raw, "", famStr, err
		}
		raw, err := chainsdk.UTXOSign(seed, path, ct, req.UTXOInputs, req.UTXOOutputs)
		return raw, "", famStr, err

	case chainsdk.FamilyCosmos:
		if req.ChainID == 0 {
			return "", "", "", fmt.Errorf("cosmos send needs {chain_id} for prefix resolution")
		}
		denom := req.Denom
		if denom == "" {
			denom = chainsdk.DefaultDenomOf(req.ChainID, ct)
		}
		resw, err := chainsdk.CosmosExecuteSend(ctx, seed, path, chainsdk.CosmosSendRequest{
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

	case chainsdk.FamilySolana:
		raw, txHash, err := chainsdk.SolanaBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, chainsdk.ParseUintOr(req.Amount), !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyAptos:
		amount := chainsdk.ParseUintOr(req.Amount)
		raw, txHash, err := chainsdk.AptosBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, amount, req.Sequence, req.MaxGas, 100, 0, 0, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilySui:
		amount := chainsdk.ParseUintOr(req.Amount)
		raw, txHash, err := chainsdk.SuiBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, amount, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyNear:
		amt := new(big.Int)
		amt.SetString(req.Amount, 10)
		raw, txHash, err := chainsdk.NearBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, amt, chainsdk.NearParams{Nonce: req.Nonce, BlockHashB58: req.BHash}, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyStellar:
		amount := chainsdk.ParseInt64Or(req.Amount)
		raw, txHash, err := chainsdk.StellarBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), ct, req.To, amount, req.Sequence, 100, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyAlgorand:
		raw, txHash, err := chainsdk.AlgoBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), "", req.To, chainsdk.AlgoBuildRequest{
			GenesisHashB64: req.GenHash,
			Fee:            req.Gas,
			FirstValid:     req.FirstV,
			LastValid:      req.LastV,
			Amount:         chainsdk.ParseUintOr(req.Amount),
		}, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyNano:
		rem, err := hex.DecodeString(req.RemBalHex)
		if err != nil {
			return "", "", famStr, fmt.Errorf("nano remaining_balance_hex malformed: %w", err)
		}
		raw, txHash, err := chainsdk.NanoBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, rem, req.WorkHex, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyMultiversX:
		raw, txHash, err := chainsdk.MultiversXBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, req.Amount, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyTezos:
		raw, txHash, err := chainsdk.TezosBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, chainsdk.ParseUintOr(req.Amount), req.Gas, req.Counter, req.Gas, req.Storage, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyTron:
		raw, txHash, err := chainsdk.TronBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, chainsdk.ParseUintOr(req.Amount), !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyVeChain:
		raw, txHash, err := chainsdk.VETBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, chainsdk.ParseUintOr(req.Amount), req.Gas, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyRipple:
		raw, txHash, err := chainsdk.RippleBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct), req.To, chainsdk.ParseUintOr(req.Amount), req.DestTag, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyCardano:
		raw, txHash, err := chainsdk.CardanoBuildSend(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct),
			req.CardanoInputs, req.CardanoOutputs, req.FeeLovelace, !req.Simulate)
		return raw, txHash, famStr, err

	case chainsdk.FamilyWaves:
		return "", "", "", errors.New("waves tx build requires protobuf tx shape — fail-closed; address derivation + message signing stay complete")

	case chainsdk.FamilyTON:
		return "", "", "", errors.New("ton tx build requires BoC serialization — fail-closed")

	case chainsdk.FamilyICP, chainsdk.FamilyZilliqa, chainsdk.FamilyKaspa, chainsdk.FamilyNervos, chainsdk.FamilyFilecoin:
		return "", "", "", fmt.Errorf("%s tx build is fail-closed in this build (address + message-sign only)", family)

	case chainsdk.FamilyAleo, chainsdk.FamilyHedera, chainsdk.FamilyFlow:
		return "", "", "", chainsdk.NotFeasible(family)

	case chainsdk.FamilySubstrate:
		amount := new(big.Int)
		if _, ok := amount.SetString(req.Amount, 10); !ok || amount.Sign() <= 0 {
			return "", "", "", fmt.Errorf("substrate: amount %q must be a positive integer in plancks", req.Amount)
		}
		raw, txHash, err := chainsdk.SubstrateBuildTransfer(ctx, seed, path, chainsdk.LookupEndpoint(req.ChainID, ct),
			[2]byte{req.PalletIndex, req.CallIndex}, req.To, amount,
			chainsdk.SubstrateChainParams{
				SpecVersion: req.SpecVersion,
				TxVersion:   req.TxVersion,
				GenesisHash: req.GenHash,
				BlockHash:   req.BlockHash,
				Nonce:       req.Nonce,
				Tip:         req.Tip,
				EraPeriod:   req.EraPeriod,
			}, !req.Simulate)
		return raw, txHash, famStr, err
	}
	return "", "", "", fmt.Errorf("family %q not routable for send", family)
}

// ---------- small numeric helpers ----------

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

// ---------- non-EVM wallet creation ----------

// createNonEvmWallet derives and persists a wallet for a non-EVM chain. It is
// reached from handleCreateWallet once the requested chain is known not to be
// EVM. The derivation path comes from the chain registry so the same mnemonic
// always restores the same address, and the path is persisted verbatim so
// /non_evm/* signing and address re-derivation use it later.
func createNonEvmWallet(c *gin.Context, req createWalletReq, chain *ChainConfig) {
	family, ct, err := chainsdk.ResolveFamily(chain.ChainType, chain.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mnemonic := strings.TrimSpace(req.Mnemonic)
	if mnemonic == "" {
		bits := req.EntropyBits
		if bits == 0 {
			bits = 256
		}
		m, err := GenerateMnemonic(bits)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mnemonic"})
			return
		}
		mnemonic = m
	} else if !ValidateMnemonic(mnemonic) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic (BIP-39 checksum failed)"})
		return
	}

	path := strings.TrimSpace(chain.DerivationPath)
	if path == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "chain " + chain.Name + " has no registered derivation path",
		})
		return
	}
	if req.AccountIndex > 0 {
		path = chainsdk.WithAccountIndex(path, req.AccountIndex)
	}

	seed := MnemonicToSeed(mnemonic, "")
	address, err := chainsdk.AddressFor(family, ct, chain.ID, seed, path, 0)
	if err != nil {
		// Arweave, TON, Aleo, Hedera and Flow have no HD address scheme; they
		// report a descriptive error rather than a fabricated address.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if address == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "derivation produced an empty address"})
		return
	}

	encSeed, err := EncryptSeed(seed, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt seed"})
		return
	}

	uid, _ := uuid.Parse(getUserID(c))
	label := req.Label
	if label == "" {
		label = chain.Name + " Wallet"
	}

	w := &WalletRecord{
		UserID:         uid,
		Label:          label,
		ChainID:        chain.ID,
		Address:        address,
		EncryptedSeed:  encSeed,
		DerivationPath: path,
		AccountIndex:   req.AccountIndex,
		IsPrimary:      false,
	}
	if err := store.SaveWallet(c.Request.Context(), w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save wallet"})
		return
	}

	c.JSON(http.StatusCreated, walletResp{
		ID:             w.ID.String(),
		Label:          w.Label,
		ChainID:        w.ChainID,
		Address:        address,
		DerivationPath: w.DerivationPath,
		Mnemonic:       mnemonic,
	})
}

// withAccountIndex replaces the trailing index of a BIP-44 path so a caller can
// request account N. Non-EVM paths are written with a mix of hardened (') and
// plain segments, so only the final element is touched.
