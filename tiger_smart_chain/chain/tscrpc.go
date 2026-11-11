package chain

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// TSCAPI is the tsc_ JSON-RPC namespace. Every method reads from the live chain
// database of this node, so the values a wallet client reads are exactly the
// values the node has validated.
type TSCAPI struct {
	node *Node
}

// NetworkInfo describes the network this node follows.
type NetworkInfo struct {
	ChainID        string `json:"chainId"`
	Name           string `json:"name"`
	NativeCurrency string `json:"nativeCurrency"`
	NativeName     string `json:"nativeName"`
	Decimals       int    `json:"decimals"`
	GenesisHash    string `json:"genesisHash"`
	Consensus      string `json:"consensus"`
	BlockPeriod    uint64 `json:"blockPeriodSeconds"`
	EpochLength    uint64 `json:"epochLength"`
	GasLimit       uint64 `json:"gasLimit"`
	BaseFeeWei     string `json:"baseFeeWei"`
	UnbondingSecs  int64  `json:"unbondingSeconds"`
	MinStakeWei    string `json:"minValidatorStakeWei"`
	MaxValidators  int    `json:"maxValidators"`
	Treasury       string `json:"treasury"`
}

// Info returns the static network parameters, including the EVM chain metadata
// a wallet needs to add TigerSmartChain as a custom network.
func (api *TSCAPI) Info() (*NetworkInfo, error) {
	cfg := api.node.cfg.Genesis
	baseFee := cfg.BaseFeeWei
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}
	minStake := cfg.MinValidatorStake
	if minStake == nil {
		minStake = big.NewInt(0)
	}
	return &NetworkInfo{
		ChainID:        big.NewInt(cfg.ChainID).String(),
		Name:           cfg.Name,
		NativeCurrency: NativeSymbol,
		NativeName:     NativeName,
		Decimals:       Decimals,
		GenesisHash:    api.node.GenesisHash().Hex(),
		Consensus:      "proof-of-authority (clique)",
		BlockPeriod:    cfg.BlockPeriodSeconds,
		EpochLength:    cfg.EpochLength,
		GasLimit:       cfg.GasLimit,
		BaseFeeWei:     baseFee.String(),
		UnbondingSecs:  cfg.UnbondingSeconds,
		MinStakeWei:    minStake.String(),
		MaxValidators:  cfg.MaxValidators,
		Treasury:       cfg.Treasury,
	}, nil
}

// Status reports the live head, sync progress and mempool state.
func (api *TSCAPI) Status() (*Health, error) {
	h := api.node.Health()
	return &h, nil
}

// ValidatorInfo is the live view of one authority.
type ValidatorInfo struct {
	Signer        string `json:"signer"`
	Name          string `json:"name,omitempty"`
	Payout        string `json:"payout,omitempty"`
	CommissionBps uint32 `json:"commissionBps"`
	SelfStakeWei  string `json:"selfStakeWei"`
	Active        bool   `json:"active"`
	BlocksSigned  uint64 `json:"blocksSigned"`
	LastSignedAt  uint64 `json:"lastSignedBlock"`
	InGenesis     bool   `json:"inGenesis"`
	IsLocalSigner bool   `json:"isLocalSigner"`
}

// Validators returns the authority set at the requested block together with
// the number of blocks each authority has signed in the sampled window.
//
// The block counts are computed from the live chain, not from a cached table,
// so the result always matches what the node has actually validated.
func (api *TSCAPI) Validators(ctx context.Context, blockNr *rpc.BlockNumber) ([]ValidatorInfo, error) {
	bc := api.node.backend.BlockChain()

	to := bc.CurrentBlock().Number.Uint64()
	if blockNr != nil && blockNr.Int64() >= 0 {
		n := blockNr.Int64()
		if uint64(n) > to {
			return nil, fmt.Errorf("tsc: block %d is above head %d", n, to)
		}
		to = uint64(n)
	}

	authority, err := api.cliqueAPI()
	if err != nil {
		return nil, err
	}
	number := rpc.BlockNumber(to)
	snapshot, err := authority.GetSnapshot(&number)
	if err != nil {
		return nil, fmt.Errorf("tsc: load validator snapshot at %d: %w", to, err)
	}

	engine, err := api.consensusEngine()
	if err != nil {
		return nil, err
	}

	counts := map[common.Address]uint64{}
	last := map[common.Address]uint64{}
	from := uint64(0)
	if to > 200_000 {
		from = to - 200_000
	}
	for n := from; n <= to; n++ {
		header := bc.GetHeaderByNumber(n)
		if header == nil {
			continue
		}
		author, err := engine.Author(header)
		if err != nil {
			continue
		}
		counts[author]++
		last[author] = n
	}

	meta := map[common.Address]Validator{}
	for _, v := range api.node.cfg.Genesis.Validators {
		norm, err := NormalizeAddress(v.Signer)
		if err != nil {
			continue
		}
		meta[common.HexToAddress(norm)] = v
	}

	out := make([]ValidatorInfo, 0, len(snapshot.Signers))
	for signer := range snapshot.Signers {
		v := meta[signer]
		selfStake := v.SelfStake
		if selfStake == nil {
			selfStake = big.NewInt(0)
		}
		payout := v.Payout
		if payout == "" {
			payout = signer.Hex()
		}
		out = append(out, ValidatorInfo{
			Signer:        signer.Hex(),
			Name:          v.Name,
			Payout:        payout,
			CommissionBps: v.CommissionBps,
			SelfStakeWei:  selfStake.String(),
			Active:        true,
			BlocksSigned:  counts[signer],
			LastSignedAt:  last[signer],
			InGenesis:     v.Signer != "",
			IsLocalSigner: api.node.signer == signer,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BlocksSigned != out[j].BlocksSigned {
			return out[i].BlocksSigned > out[j].BlocksSigned
		}
		return strings.ToLower(out[i].Signer) < strings.ToLower(out[j].Signer)
	})
	return out, nil
}

// ValidatorSet returns only the authorised signer addresses, which is the
// minimal data a light client needs to verify TigerSmartChain block seals.
func (api *TSCAPI) ValidatorSet(ctx context.Context, blockNr *rpc.BlockNumber) ([]string, error) {
	infos, err := api.Validators(ctx, blockNr)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(infos))
	for _, v := range infos {
		out = append(out, v.Signer)
	}
	return out, nil
}

// AllocationView is a genesis allocation plus its live on-chain balance.
type AllocationView struct {
	Address                string `json:"address"`
	Label                  string `json:"label"`
	GenesisWei             string `json:"genesisWei"`
	CurrentWei             string `json:"currentWei"`
	HasCode                bool   `json:"hasContract"`
	Nonce                  uint64 `json:"nonce"`
	VestingCliffSeconds    int64  `json:"vestingCliffSeconds,omitempty"`
	VestingDurationSeconds int64  `json:"vestingDurationSeconds,omitempty"`
	VestingBeneficiary     string `json:"vestingBeneficiary,omitempty"`
}

// Allocations returns the genesis distribution with the live balances of each
// allocation address, so an auditor can prove the supply is intact.
func (api *TSCAPI) Allocations(ctx context.Context, blockNr *rpc.BlockNumber) ([]AllocationView, error) {
	state, err := api.stateAt(blockNr)
	if err != nil {
		return nil, err
	}
	out := make([]AllocationView, 0, len(api.node.cfg.Genesis.Allocations))
	for _, a := range api.node.cfg.Genesis.Allocations {
		addr := common.HexToAddress(a.Address)
		view := AllocationView{
			Address:    addr.Hex(),
			Label:      a.Label,
			GenesisWei: a.Balance.String(),
			CurrentWei: state.GetBalance(addr).ToBig().String(),
			HasCode:    len(state.GetCode(addr)) > 0,
			Nonce:      state.GetNonce(addr),
		}
		if a.Vesting != nil {
			view.VestingCliffSeconds = a.Vesting.CliffSeconds
			view.VestingDurationSeconds = a.Vesting.DurationSeconds
			view.VestingBeneficiary = a.Vesting.Beneficiary
		}
		out = append(out, view)
	}
	return out, nil
}

// TrackedSupplyWei reports the total TGR held by the platform allocation
// accounts and the validator self stakes at a block.
//
// It is deliberately not called "total supply": accounts created by users also
// hold TGR, and those balances are only discoverable by walking the whole state
// trie. For the invariant that actually matters - the platform never mints
// beyond the configured genesis supply - compare this figure against the
// genesis distribution of the same account set.
func (api *TSCAPI) TrackedSupplyWei(ctx context.Context, blockNr *rpc.BlockNumber) (string, error) {
	state, err := api.stateAt(blockNr)
	if err != nil {
		return "", err
	}
	total := new(big.Int)
	for _, a := range api.node.cfg.Genesis.Allocations {
		total.Add(total, state.GetBalance(common.HexToAddress(a.Address)).ToBig())
	}
	for _, v := range api.node.cfg.Genesis.Validators {
		norm, err := NormalizeAddress(v.Signer)
		if err != nil {
			continue
		}
		total.Add(total, state.GetBalance(common.HexToAddress(norm)).ToBig())
	}
	return total.String(), nil
}

// stateAt resolves the state database of the requested block.
func (api *TSCAPI) stateAt(blockNr *rpc.BlockNumber) (*state.StateDB, error) {
	bc := api.node.backend.BlockChain()
	header := bc.CurrentBlock()
	if blockNr != nil && blockNr.Int64() >= 0 {
		n := uint64(blockNr.Int64())
		if n > header.Number.Uint64() {
			return nil, fmt.Errorf("tsc: block %d is above head %d", n, header.Number.Uint64())
		}
		header = bc.GetHeaderByNumber(n)
	}
	db, err := bc.StateAt(header.Root)
	if err != nil {
		return nil, fmt.Errorf("tsc: read state at %d: %w", header.Number.Uint64(), err)
	}
	return db, nil
}

// ChainStats aggregates the block level metrics an explorer or admin dashboard
// needs, computed from the real chain database.
type ChainStats struct {
	HeadNumber        uint64  `json:"headNumber"`
	HeadHash          string  `json:"headHash"`
	GenesisHash       string  `json:"genesisHash"`
	GenesisTimestamp  uint64  `json:"genesisTimestamp"`
	BlockTimeSeconds  float64 `json:"averageBlockTimeSeconds"`
	TotalBlocks       uint64  `json:"totalBlocks"`
	TotalTransactions uint64  `json:"totalTransactions"`
	GasUsed           uint64  `json:"gasUsedLastBlock"`
	GasLimit          uint64  `json:"gasLimit"`
	BaseFeeWei        string  `json:"baseFeeWei"`
	ValidatorCount    int     `json:"validatorCount"`
	PeerCount         int     `json:"peerCount"`
	PendingTxs        int     `json:"pendingTransactions"`
}

// Stats computes chain level statistics over the most recent window of blocks.
func (api *TSCAPI) Stats(ctx context.Context, window *hexutil.Uint64) (*ChainStats, error) {
	bc := api.node.backend.BlockChain()
	head := bc.CurrentBlock()

	span := uint64(1000)
	if window != nil && uint64(*window) > 0 {
		span = uint64(*window)
	}
	from := uint64(0)
	if head.Number.Uint64() > span {
		from = head.Number.Uint64() - span
	}

	var txs uint64
	for n := from; n <= head.Number.Uint64(); n++ {
		blk := bc.GetBlockByNumber(n)
		if blk == nil {
			continue
		}
		txs += uint64(len(blk.Transactions()))
	}

	pending, _ := api.node.backend.TxPool().Stats()
	stats := &ChainStats{
		HeadNumber:        head.Number.Uint64(),
		HeadHash:          head.Hash().Hex(),
		GenesisHash:       bc.Genesis().Hash().Hex(),
		GenesisTimestamp:  bc.Genesis().Time(),
		TotalBlocks:       head.Number.Uint64() + 1,
		TotalTransactions: txs,
		GasUsed:           head.GasUsed,
		GasLimit:          head.GasLimit,
		BaseFeeWei:        head.BaseFee.String(),
		ValidatorCount:    len(api.node.cfg.Genesis.Validators),
		PeerCount:         api.node.stack.Server().PeerCount(),
		PendingTxs:        pending,
	}

	// Average block time is measured over produced blocks only. Including the
	// genesis block would fold the (arbitrary) gap between the genesis
	// timestamp and the first mined block into the average.
	if head.Number.Uint64() > 1 {
		measureFrom := from
		if measureFrom < 1 {
			measureFrom = 1
		}
		first := bc.GetBlockByNumber(measureFrom)
		span := head.Number.Uint64() - measureFrom
		if first != nil && span > 0 && head.Time > first.Time() {
			stats.BlockTimeSeconds = float64(head.Time-first.Time()) / float64(span)
		}
	}
	if stats.BlockTimeSeconds == 0 {
		stats.BlockTimeSeconds = float64(api.node.cfg.Genesis.BlockPeriodSeconds)
	}
	return stats, nil
}

// BlockSummary is a compact block record for explorer listings.
type BlockSummary struct {
	Number       uint64   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parentHash"`
	Timestamp    uint64   `json:"timestamp"`
	Miner        string   `json:"miner"`
	Validator    string   `json:"validator"`
	GasUsed      uint64   `json:"gasUsed"`
	GasLimit     uint64   `json:"gasLimit"`
	BaseFeeWei   string   `json:"baseFeeWei"`
	TxCount      int      `json:"transactionCount"`
	SizeBytes    uint64   `json:"size"`
	StateRoot    string   `json:"stateRoot"`
	Transactions []string `json:"transactions,omitempty"`
}

// Blocks returns a descending page of blocks starting at from.
func (api *TSCAPI) Blocks(ctx context.Context, from hexutil.Uint64, limit hexutil.Uint64) ([]BlockSummary, error) {
	bc := api.node.backend.BlockChain()
	head := bc.CurrentBlock().Number.Uint64()
	start := uint64(from)
	if start > head {
		return []BlockSummary{}, nil
	}
	count := uint64(limit)
	if count == 0 || count > 100 {
		count = 20
	}

	engine, _ := api.consensusEngine()
	out := make([]BlockSummary, 0, count)
	for n := start; len(out) < int(count); n-- {
		blk := bc.GetBlockByNumber(n)
		if blk == nil {
			if n == 0 {
				break
			}
			continue
		}
		author := blk.Coinbase()
		if engine != nil {
			if a, err := engine.Author(blk.Header()); err == nil {
				author = a
			}
		}
		hashes := make([]string, 0, len(blk.Transactions()))
		for _, tx := range blk.Transactions() {
			hashes = append(hashes, tx.Hash().Hex())
		}
		out = append(out, BlockSummary{
			Number:       blk.NumberU64(),
			Hash:         blk.Hash().Hex(),
			ParentHash:   blk.ParentHash().Hex(),
			Timestamp:    blk.Time(),
			Miner:        blk.Coinbase().Hex(),
			Validator:    author.Hex(),
			GasUsed:      blk.GasUsed(),
			GasLimit:     blk.GasLimit(),
			BaseFeeWei:   blk.BaseFee().String(),
			TxCount:      len(blk.Transactions()),
			SizeBytes:    blk.Size(),
			StateRoot:    blk.Root().Hex(),
			Transactions: hashes,
		})
		if n == 0 {
			break
		}
	}
	return out, nil
}

// TransactionSummary is a decoded transaction for explorer detail pages.
type TransactionSummary struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	From        string `json:"from"`
	To          string `json:"to,omitempty"`
	ValueWei    string `json:"valueWei"`
	Nonce       uint64 `json:"nonce"`
	Gas         uint64 `json:"gas"`
	GasPriceWei string `json:"gasPriceWei"`
	GasUsed     uint64 `json:"gasUsed,omitempty"`
	Status      string `json:"status"`
	Input       string `json:"input"`
	TxType      string `json:"type"`
	ChainID     string `json:"chainId"`
}

// Transaction returns a single transaction with its receipt status.
func (api *TSCAPI) Transaction(ctx context.Context, hash common.Hash) (*TransactionSummary, error) {
	bc := api.node.backend.BlockChain()
	lookup, tx, err := bc.GetTransactionLookup(hash)
	if err != nil {
		return nil, fmt.Errorf("tsc: transaction %s not found: %w", hash.Hex(), err)
	}
	if lookup == nil || tx == nil {
		return nil, fmt.Errorf("tsc: transaction %s not found", hash.Hex())
	}
	from, err := types.Sender(types.LatestSigner(bc.Config()), tx)
	if err != nil {
		return nil, fmt.Errorf("tsc: recover sender for %s: %w", hash.Hex(), err)
	}
	out := &TransactionSummary{
		Hash:        tx.Hash().Hex(),
		BlockNumber: lookup.BlockIndex,
		BlockHash:   lookup.BlockHash.Hex(),
		From:        from.Hex(),
		ValueWei:    tx.Value().String(),
		Nonce:       tx.Nonce(),
		Gas:         tx.Gas(),
		Input:       hexutil.Encode(tx.Data()),
		ChainID:     api.node.ChainID().String(),
		Status:      "pending",
	}
	if to := tx.To(); to != nil {
		out.To = to.Hex()
	}
	if gp := tx.GasPrice(); gp != nil {
		out.GasPriceWei = gp.String()
	}
	out.TxType = txTypeName(tx.Type())
	if receipt := receiptFor(bc, lookup.BlockHash, hash); receipt != nil {
		out.GasUsed = receipt.GasUsed
		out.Status = receiptStatusName(receipt.Status)
	}
	return out, nil
}

// PendingTransactions lists the transactions currently in the mempool.
func (api *TSCAPI) PendingTransactions(ctx context.Context) ([]TransactionSummary, error) {
	pool := api.node.backend.TxPool()
	byAddr := pool.Pending(txpool.PendingFilter{})
	out := make([]TransactionSummary, 0)
	for addr, lazy := range byAddr {
		for _, lt := range lazy {
			tx := lt.Resolve()
			if tx == nil {
				continue
			}
			summary := TransactionSummary{
				Hash:     tx.Hash().Hex(),
				From:     addr.Hex(),
				ValueWei: tx.Value().String(),
				Nonce:    tx.Nonce(),
				Gas:      tx.Gas(),
				Input:    hexutil.Encode(tx.Data()),
				ChainID:  api.node.ChainID().String(),
				Status:   "pending",
				TxType:   txTypeName(tx.Type()),
			}
			if to := tx.To(); to != nil {
				summary.To = to.Hex()
			}
			if gp := tx.GasPrice(); gp != nil {
				summary.GasPriceWei = gp.String()
			}
			out = append(out, summary)
			if len(out) >= 500 {
				return out, nil
			}
		}
	}
	return out, nil
}

// AddressHistory returns the native-coin history of an address by scanning a
// bounded block window. It is a self-hosted replacement for a hosted explorer
// API, so no third-party service is required.
func (api *TSCAPI) AddressHistory(ctx context.Context, addr common.Address, window hexutil.Uint64) ([]TransactionSummary, error) {
	bc := api.node.backend.BlockChain()
	head := bc.CurrentBlock().Number.Uint64()
	span := uint64(window)
	if span == 0 || span > 5_000 {
		span = 5_000
	}
	from := uint64(0)
	if head > span {
		from = head - span
	}

	signer := types.LatestSigner(bc.Config())
	out := make([]TransactionSummary, 0)
	for n := head; ; n-- {
		blk := bc.GetBlockByNumber(n)
		if blk != nil {
			for _, tx := range blk.Transactions() {
				to := tx.To()
				involves := to != nil && *to == addr
				sender, senderErr := types.Sender(signer, tx)
				if !involves && senderErr == nil && sender == addr {
					involves = true
				}
				if !involves {
					continue
				}
				summary := TransactionSummary{
					Hash:        tx.Hash().Hex(),
					BlockNumber: blk.NumberU64(),
					BlockHash:   blk.Hash().Hex(),
					From:        sender.Hex(),
					ValueWei:    tx.Value().String(),
					Nonce:       tx.Nonce(),
					Gas:         tx.Gas(),
					Input:       hexutil.Encode(tx.Data()),
					ChainID:     api.node.ChainID().String(),
					Status:      "success",
					TxType:      txTypeName(tx.Type()),
				}
				if to != nil {
					summary.To = to.Hex()
				}
				if receipt := receiptFor(bc, blk.Hash(), tx.Hash()); receipt != nil {
					summary.GasUsed = receipt.GasUsed
					summary.Status = receiptStatusName(receipt.Status)
				}
				out = append(out, summary)
				if len(out) >= 200 {
					return out, nil
				}
			}
		}
		if n == 0 || n <= from {
			break
		}
	}
	return out, nil
}

// ProposeValidator submits an authority vote to the consensus engine. The
// change only takes effect once a majority of the current authority set has
// signed the same vote, exactly like the underlying proof-of-authority rules.
func (api *TSCAPI) ProposeValidator(ctx context.Context, addr common.Address, authorize bool) (string, error) {
	if api.node.signer == (common.Address{}) {
		return "", fmt.Errorf("tsc: this node holds no validator key and cannot vote")
	}
	authority, err := api.cliqueAPI()
	if err != nil {
		return "", err
	}
	authority.Propose(addr, authorize)
	verb := "discard"
	if authorize {
		verb = "authorize"
	}
	return fmt.Sprintf("vote submitted to %s %s", verb, addr.Hex()), nil
}

// DiscardValidatorVote removes a pending authority vote cast by this node.
func (api *TSCAPI) DiscardValidatorVote(ctx context.Context, addr common.Address) (string, error) {
	authority, err := api.cliqueAPI()
	if err != nil {
		return "", err
	}
	authority.Discard(addr)
	return fmt.Sprintf("pending vote for %s discarded", addr.Hex()), nil
}

// ValidatorVotes lists the votes this node has cast that are not yet effective.
func (api *TSCAPI) ValidatorVotes(ctx context.Context) (map[string]bool, error) {
	authority, err := api.cliqueAPI()
	if err != nil {
		return nil, err
	}
	raw := authority.Proposals()
	out := make(map[string]bool, len(raw))
	for addr, authorize := range raw {
		out[addr.Hex()] = authorize
	}
	return out, nil
}

// EngineStatus reports the live consensus engine state.
func (api *TSCAPI) EngineStatus(ctx context.Context) (map[string]interface{}, error) {
	authority, err := api.cliqueAPI()
	if err != nil {
		return nil, err
	}
	head := rpc.BlockNumber(api.node.backend.BlockChain().CurrentBlock().Number.Uint64())
	snapshot, err := authority.GetSnapshot(&head)
	if err != nil {
		return nil, err
	}
	signers := make([]string, 0, len(snapshot.Signers))
	for s := range snapshot.Signers {
		signers = append(signers, s.Hex())
	}
	sort.Strings(signers)
	return map[string]interface{}{
		"snapshotNumber": snapshot.Number,
		"snapshotHash":   snapshot.Hash.Hex(),
		"signers":        signers,
		"signerCount":    len(signers),
		"localSigner":    api.node.signer.Hex(),
		"mining":         api.node.IsMining(),
		"chainId":        api.node.ChainID().String(),
	}, nil
}

// consensusEngine returns the live consensus engine of this chain.
//
// The eth backend wraps the authority engine in a beacon adapter even for
// pre-merge chains, so the wrapper is unwrapped before the engine is returned.
func (api *TSCAPI) consensusEngine() (consensus.Engine, error) {
	inner := api.node.backend.Engine()
	if inner == nil {
		return nil, fmt.Errorf("tsc: node has no consensus engine")
	}
	if wrapped, ok := inner.(*beacon.Beacon); ok {
		return wrapped.InnerEngine(), nil
	}
	return inner, nil
}

// cliqueAPI returns the authority voting API bound to the live chain.
func (api *TSCAPI) cliqueAPI() (*clique.API, error) {
	engine, err := api.consensusEngine()
	if err != nil {
		return nil, err
	}
	c, ok := engine.(*clique.Clique)
	if !ok {
		return nil, fmt.Errorf("tsc: chain is not running the proof-of-authority engine")
	}
	apis := c.APIs(api.node.backend.BlockChain())
	if len(apis) == 0 {
		return nil, fmt.Errorf("tsc: authority API unavailable")
	}
	authority, ok := apis[0].Service.(*clique.API)
	if !ok {
		return nil, fmt.Errorf("tsc: authority API has an unexpected type")
	}
	return authority, nil
}

// receiptFor finds the receipt of a transaction inside a known block.
func receiptFor(bc *core.BlockChain, blockHash, txHash common.Hash) *types.Receipt {
	for _, r := range bc.GetReceiptsByHash(blockHash) {
		if r.TxHash == txHash {
			return r
		}
	}
	return nil
}

func receiptStatusName(status uint64) string {
	if status == types.ReceiptStatusFailed {
		return "failed"
	}
	return "success"
}

func txTypeName(txType uint8) string {
	switch txType {
	case types.LegacyTxType:
		return "legacy"
	case types.AccessListTxType:
		return "eip2930"
	case types.DynamicFeeTxType:
		return "eip1559"
	case types.BlobTxType:
		return "eip4844"
	default:
		return "unknown"
	}
}
