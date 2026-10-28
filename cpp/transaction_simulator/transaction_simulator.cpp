/**
 * TigerWallet - Transaction Simulator Implementation
 * High-Performance C++ with Ultra-Low Latency
 */

#include "transaction_simulator.h"

#include "rpc.hpp"
#include "uint256.hpp"

// Shared, self-built primitives. A transaction hash is keccak256 of the RLP
// encoding of the transaction, so both are needed to produce one.
#include "keccak256.hpp"
#include "rlp.hpp"
#include "uint256.hpp"

#include <algorithm>
#include <cstring>

namespace TigerWallet {

namespace {

// Parses a 0x-prefixed hexadecimal quantity. A value wider than 256 bits is
// rejected rather than wrapped.
bool parseHexQuantity(const std::string& text, tigerwallet::math::U256& out) {
    size_t start = 0;
    if (text.size() >= 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')) {
        start = 2;
    }

    std::string digits;
    digits.reserve(text.size() - start);
    for (size_t i = start; i < text.size(); ++i) {
        const char ch = text[i];
        if (ch == '_') continue;
        const bool is_hex = (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') ||
                            (ch >= 'A' && ch <= 'F');
        if (!is_hex) {
            return false;
        }
        digits.push_back(ch);
    }

    if (digits.empty()) {
        return false;
    }
    if (digits.size() > 64) {
        return false;
    }

    out = tigerwallet::math::U256();
    for (char ch : digits) {
        uint64_t nibble;
        if (ch >= '0' && ch <= '9') {
            nibble = static_cast<uint64_t>(ch - '0');
        } else if (ch >= 'a' && ch <= 'f') {
            nibble = static_cast<uint64_t>(ch - 'a' + 10);
        } else {
            nibble = static_cast<uint64_t>(ch - 'A' + 10);
        }
        // Shift left one nibble and add.
        uint64_t carry = nibble;
        for (size_t w = 0; w < 4; ++w) {
            const uint64_t next = out.word[w] >> 60;
            out.word[w] = (out.word[w] << 4) | carry;
            carry = next;
        }
        if (carry != 0) {
            return false;
        }
    }
    return true;
}

// Renders a 256-bit value as a decimal string, for the human-readable diff.
std::string toDecimalString(const tigerwallet::math::U256& value) {
    return value.toDecimal();
}

// The simulator carries amounts as four words; the arithmetic type owns the
// operations. These two move between them.
tigerwallet::math::U256 toU256(const uint256_t& value) {
    return tigerwallet::math::U256::fromWords(value[0], value[1], value[2], value[3]);
}

uint256_t fromU256(const tigerwallet::math::U256& value) {
    return uint256_t{value.word[0], value.word[1], value.word[2], value.word[3]};
}

} // namespace

// Address implementation
std::string Address::toString() const {
    std::stringstream ss;
    ss << "0x";
    for (auto byte : bytes) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)byte;
    }
    return ss.str();
}

Address Address::fromString(const std::string& hex) {
    Address addr;
    std::string cleanHex = hex;
    if (cleanHex.substr(0, 2) == "0x") {
        cleanHex = cleanHex.substr(2);
    }

    if (cleanHex.length() != 40) {
        return addr;
    }

    for (size_t i = 0; i < 20; i++) {
        std::string byteStr = cleanHex.substr(i * 2, 2);
        addr.bytes[i] = static_cast<uint8_t>(std::stoi(byteStr, nullptr, 16));
    }

    addr.checksum = addr.toString();
    return addr;
}

bool Address::isZero() const {
    return std::all_of(bytes.begin(), bytes.end(), [](uint8_t b) { return b == 0; });
}

// TransactionSimulator implementation
TransactionSimulator::TransactionSimulator() : chain_id_(1) {
    initialize();
}

TransactionSimulator::TransactionSimulator(uint64_t chain_id)
    : chain_id_(chain_id == 0 ? 1 : chain_id) {
    initialize();
}

void TransactionSimulator::initialize() {
    // Initialize with default block
    currentBlock_.number = 0;
    currentBlock_.timestamp = static_cast<uint64_t>(std::time(nullptr));
    currentBlock_.gasLimit = 30000000;
    currentBlock_.baseFeePerGas = 10000000000ULL; // 10 gwei

    // Initialize default tokens
    Token eth;
    eth.address = "0x0000000000000000000000000000000000000000";
    eth.symbol = "ETH";
    eth.name = "Ethereum";
    eth.decimals = 18;
    eth.isNative = true;
    tokens_[eth.address] = eth;

    Token usdt;
    usdt.address = "0xdAC17F958D2ee523a2206206994597C13D831ec7";
    usdt.symbol = "USDT";
    usdt.name = "Tether USD";
    usdt.decimals = 6;
    tokens_[usdt.address] = usdt;

    Token usdc;
    usdc.address = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48";
    usdc.symbol = "USDC";
    usdc.name = "USD Coin";
    usdc.decimals = 6;
    tokens_[usdc.address] = usdc;
}

TransactionSimulator::~TransactionSimulator() {}

void TransactionSimulator::setRpcClient(std::shared_ptr<tigerwallet::RpcClient> rpc) {
    std::unique_lock lock(mutex_);
    rpc_ = std::move(rpc);
}

SimulationResult TransactionSimulator::simulateTransaction(const Transaction& tx) {
    std::unique_lock lock(mutex_);
    return simulateTransactionLocked(tx);
}

SimulationResult TransactionSimulator::simulateTransactionLocked(const Transaction& tx) {
    auto startTime = std::chrono::high_resolution_clock::now();

    SimulationResult result;


    // Validate transaction
    if (!validateTransactionLocked(tx)) {
        result.success = false;
        result.error = "Transaction validation failed";
        metrics_.failedSimulations++;
        return result;
    }

    // Check MEV
    MEVAnalysis mevAnalysis = analyzeMEVLocked(tx);
    result.mevType = mevAnalysis.type;
    result.mevRisk = mevAnalysis.riskScore;

    if (mevAnalysis.riskScore > 0.7) {
        result.warnings.push_back("High MEV risk detected: " + mevAnalysis.description);
    }

    // Execute simulation
    Transaction txCopy = tx;
    executeTransactionLocked(txCopy, result);

    auto endTime = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration<double, std::milli>(endTime - startTime);
    result.executionTimeMs = duration.count();

    metrics_.totalSimulations++;
    if (result.success) {
        metrics_.successfulSimulations++;
    } else {
        metrics_.failedSimulations++;
    }

    return result;
}

SimulationResult TransactionSimulator::simulateBundle(const std::vector<Transaction>& txs) {
    SimulationResult result;

    if (txs.empty()) {
        result.success = false;
        result.error = "Empty bundle";
        return result;
    }

    // Check if this is a Flashbots bundle
    bool isFlashbots = isFlashbotsBundle(txs);

    if (isFlashbots) {
        // Flashbots bundles get priority
        for (const auto& tx : txs) {
            auto simResult = simulateTransaction(tx);
            if (!simResult.success) {
                result.success = false;
                result.error = "Bundle simulation failed: " + simResult.error;
                return result;
            }
        }
        result.warnings.push_back("Flashbots bundle detected - MEV protected");
    } else {
        // Regular bundle - check for sandwich attacks
        for (size_t i = 0; i < txs.size() - 1; i++) {
            auto mevAnalysis = detectSandwichAttack(txs[i], std::vector<Transaction>(txs.begin() + i + 1, txs.end()));
            if (mevAnalysis.type != MEVType::NONE) {
                result.warnings.push_back("Potential sandwich attack detected");
                result.mevType = MEVType::SANDFWICH_BOT;
            }
        }

        for (const auto& tx : txs) {
            auto simResult = simulateTransaction(tx);
            if (!simResult.success) {
                result.success = false;
                result.error = simResult.error;
                return result;
            }
        }
    }

    result.success = true;
    return result;
}

SimulationResult TransactionSimulator::simulateBlock(const Block& block) {
    SimulationResult result;

    Block simBlock = block;
    uint64_t totalGas = 0;

    for (auto& tx : simBlock.transactions) {
        auto txResult = simulateTransaction(tx);

        if (!txResult.success) {
            result.success = false;
            result.error = "Block simulation failed at tx " + tx.hash + ": " + txResult.error;
            return result;
        }

        totalGas += txResult.gasUsed[0];
    }

    result.success = true;
    result.gasUsed = {totalGas, 0, 0, 0};
    metrics_.blocksProcessed++;

    return result;
}

MEVAnalysis TransactionSimulator::analyzeMEV(const Transaction& tx) {
    std::shared_lock lock(mutex_);
    return analyzeMEVLocked(tx);
}

MEVAnalysis TransactionSimulator::analyzeMEVLocked(const Transaction& tx) {
    MEVAnalysis analysis;

    // Check if transaction is a swap
    bool isSwap = false;
    if (tx.data.size() >= 4) {
        // Common DEX function selectors
        std::vector<uint8_t> swapSelector = {0x7f, 0x36, 0x2f, 0x0a}; // swapExactETHForTokens
        std::vector<uint8_t> swapSelector2 = {0x88, 0x56, 0x8c, 0x51}; // swapExactTokensForETH

        if (std::equal(swapSelector.begin(), swapSelector.end(), tx.data.begin()) ||
            std::equal(swapSelector2.begin(), swapSelector2.end(), tx.data.begin())) {
            isSwap = true;
        }
    }

    if (isSwap) {
        // Check mempool for potential sandwich
        auto relatedTxs = findRelatedMempoolTransactions(tx);

        if (!relatedTxs.empty()) {
            // Check for front-run
            bool hasFrontRun = false;
            bool hasBackRun = false;

            for (const auto& memTx : relatedTxs) {
                if (memTx.gasPrice > tx.gasPrice * 1.1) {
                    hasFrontRun = true;
                }
                if (memTx.gasPrice < tx.gasPrice * 0.9) {
                    hasBackRun = true;
                }
            }

            if (hasFrontRun && hasBackRun) {
                analysis.type = MEVType::SANDFWICH_BOT;
                analysis.riskScore = 0.95;
                analysis.description = "High probability sandwich attack detected";
                analysis.relatedTransactions = relatedTxs;
                analysis.botProbability = 0.85;
                metrics_.mevDetections++;
            } else if (hasFrontRun) {
                analysis.type = MEVType::FRONTRUN_BOT;
                analysis.riskScore = 0.75;
                analysis.description = "Potential front-run detected";
                analysis.botProbability = 0.70;
            }
        }
    }

    // Calculate general MEV risk
    analysis.riskScore = calculateMEVRisk(tx);

    return analysis;
}

MEVAnalysis TransactionSimulator::detectSandwichAttack(const Transaction& tx, const std::vector<Transaction>& mempool) {
    MEVAnalysis analysis;

    // Find transactions that could be part of a sandwich
    for (const auto& memTx : mempool) {
        // Check if same token pair
        if (tx.to.toString() == memTx.to.toString() &&
            tx.data.size() > 0 && memTx.data.size() > 0 &&
            tx.data.size() == memTx.data.size()) {

            // High gas price difference indicates front-running
            if (memTx.gasPrice > tx.gasPrice * 1.2) {
                analysis.type = MEVType::FRONTRUN_BOT;
                analysis.riskScore = 0.8;
                analysis.description = "Front-run transaction detected in mempool";
                analysis.relatedTransactions.push_back(memTx);
            }
        }
    }

    return analysis;
}

bool TransactionSimulator::isProtectedTransaction(const Transaction& tx) {
    // Check for Flashbots protection markers
    if (tx.data.size() >= 4) {
        // Flashbots bundle indicator
        std::vector<uint8_t> fbIndicator = {0x00, 0x00, 0x00, 0x0c}; // Simulation type indicator
        if (std::equal(fbIndicator.begin(), fbIndicator.end(), tx.data.begin())) {
            return true;
        }
    }

    // Check for EIP-1559 - inherently more protected
    if (tx.type == TransactionType::EIP1559) {
        return true;
    }

    return false;
}

GasEstimate TransactionSimulator::estimateGas(const Transaction& tx) {
    GasEstimate estimate;

    const uint64_t intrinsic = calculateIntrinsicGas(tx);

    // The node measures the gas the call actually needs. The local intrinsic
    // cost is the floor when no node is reachable.
    uint64_t measured = 0;
    if (rpc_ != nullptr) {
        std::string tx_object = "{\"from\":\"" + tx.from.toString() + "\"";
        if (!tx.to.isZero()) {
            tx_object += ",\"to\":\"" + tx.to.toString() + "\"";
        }
        tx_object += ",\"value\":\"" + tigerwallet::math::toHex(toU256(tx.value)) + "\"";
        if (!tx.data.empty()) {
            tx_object += ",\"data\":\"0x" + tigerwallet::rlp::Encoder::toHex(tx.data) + "\"";
        }
        tx_object += "}";

        const auto result = rpc_->EstimateGas(tx_object);
        if (result.ok) {
            measured = result.value;
        }
    }

    const uint64_t base = (measured > intrinsic) ? measured : intrinsic;

    // A twenty percent margin covers a state change between the estimate and
    // the send.
    estimate.gasLimit = static_cast<uint64_t>(base + base / 5);

    // The node's current gas price is preferred over a local guess.
    estimate.gasPrice = calculateOptimalGasPrice(tx);
    if (rpc_ != nullptr) {
        const auto price = rpc_->GetGasPrice();
        if (price.ok) {
            tigerwallet::math::U256 parsed;
            if (parseHexQuantity(price.value, parsed)) {
                estimate.gasPrice = parsed.toUint64();
            }
        }
    }

    estimate.estimatedCost = estimate.gasLimit * estimate.gasPrice;

    // A measurement from the node is far more reliable than the floor.
    estimate.confidence = measured > 0 ? 0.95 : 0.5;

    estimate.factors.push_back("Intrinsic gas: " + std::to_string(intrinsic));
    if (measured > 0) {
        estimate.factors.push_back("Node estimate: " + std::to_string(measured));
    } else {
        estimate.factors.push_back("No node estimate available");
    }
    estimate.factors.push_back("Gas price: " + std::to_string(estimate.gasPrice));

    return estimate;
}

GasEstimate TransactionSimulator::estimateGasEIP1559(const Transaction& tx) {
    GasEstimate estimate = estimateGas(tx);

    // The base fee is the one the node reports; a locally held block is only a
    // fallback.
    uint64_t base_fee = currentBlock_.baseFeePerGas;
    if (rpc_ != nullptr) {
        const auto block =
            rpc_->CallRaw("eth_getBlockByNumber", "[\"latest\",false]");
        if (block.ok) {
            const std::string key = "\"baseFeePerGas\":\"";
            const size_t at = block.value.find(key);
            if (at != std::string::npos) {
                const size_t from = at + key.size();
                const size_t to = block.value.find('"', from);
                if (to != std::string::npos) {
                    tigerwallet::math::U256 parsed;
                    if (parseHexQuantity(block.value.substr(from, to - from), parsed)) {
                        base_fee = parsed.toUint64();
                    }
                }
            }
        }
    }

    const uint64_t priority =
        tx.maxPriorityFeePerGas > 0 ? tx.maxPriorityFeePerGas : 1000000000ULL;

    estimate.maxPriorityFeePerGas = priority;
    estimate.maxFeePerGas = base_fee * 2 + priority;
    estimate.gasPrice = std::min(estimate.maxFeePerGas, base_fee + priority);
    estimate.estimatedCost = estimate.gasLimit * estimate.gasPrice;

    estimate.confidence = 0.95;

    return estimate;
}

SwapSimulation TransactionSimulator::simulateSwap(
    const Address& router,
    const std::string& fromToken,
    const std::string& toToken,
    const uint256_t& amountIn,
    double slippageTolerance
) {
    SwapSimulation sim;

    sim.routerAddress = router;
    sim.fromToken = fromToken;
    sim.toToken = toToken;
    sim.amountIn = amountIn;

    // Find optimal path
    sim.path = {fromToken, toToken};
    if (fromToken != "0x0000000000000000000000000000000000000000" &&
        toToken != "0x0000000000000000000000000000000000000000") {
        // Check for intermediate path through stablecoins
        sim.path = {fromToken, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", toToken}; // USDC
    }

    // A quote needs the pool's reserves. This used to hardcode reserveIn = 1000
    // and reserveOut = 500, so every swap on every chain against every pair
    // returned the same number, and the price impact was computed against a
    // reserve of 1000 tokens that does not exist.
    //
    // Reserves come from the pool contract. Until this simulator is given a way
    // to read them, it reports that it cannot quote rather than inventing a
    // figure: expectedAmountOut and minimumAmountOut stay zero, priceImpact
    // stays zero, and quoteAvailable says why.
    sim.gasEstimate = 150000 + (sim.path.size() - 1) * 50000;
    sim.expectedAmountOut = uint256_t{0, 0, 0, 0};
    sim.minimumAmountOut = uint256_t{0, 0, 0, 0};
    sim.priceImpact = 0.0;
    sim.quoteAvailable = false;
    sim.quoteUnavailableReason =
        "Pool reserves are not available; a swap quote cannot be computed "
        "without them.";

    // The slippage tolerance is still validated, so a caller passing an
    // impossible value learns about it now rather than at execution time.
    if (slippageTolerance < 0.0 || slippageTolerance > 100.0) {
        sim.quoteUnavailableReason =
            "Slippage tolerance must be between 0 and 100 percent.";
    }

    return sim;
}

void TransactionSimulator::addToMempool(const Transaction& tx) {
    std::unique_lock lock(mutex_);

    if (mempool_.size() >= SimConfig::MEMPOOL_SIZE) {
        // Remove lowest gas price transaction
        auto lowest = std::min_element(mempool_.begin(), mempool_.end(),
            [](const auto& a, const auto& b) {
                return a.second.gasPrice < b.second.gasPrice;
            });
        if (lowest != mempool_.end()) {
            mempool_.erase(lowest);
        }
    }

    mempool_[tx.hash] = tx;
    metrics_.mempoolSize = mempool_.size();
}

void TransactionSimulator::removeFromMempool(const std::string& txHash) {
    std::unique_lock lock(mutex_);
    mempool_.erase(txHash);
    metrics_.mempoolSize = mempool_.size();
}

std::vector<Transaction> TransactionSimulator::getMempool() const {
    std::shared_lock lock(mutex_);

    std::vector<Transaction> txs;
    for (const auto& pair : mempool_) {
        txs.push_back(pair.second);
    }

    return txs;
}

std::vector<Transaction> TransactionSimulator::getPendingTransactions(const Address& from) const {
    std::shared_lock lock(mutex_);

    std::vector<Transaction> txs;
    for (const auto& pair : mempool_) {
        if (pair.second.from.toString() == from.toString()) {
            txs.push_back(pair.second);
        }
    }

    return txs;
}

void TransactionSimulator::setCurrentBlock(const Block& block) {
    std::unique_lock lock(mutex_);
    currentBlock_ = block;
}

Block TransactionSimulator::getCurrentBlock() const {
    std::shared_lock lock(mutex_);
    return currentBlock_;
}

Block TransactionSimulator::getBlock(uint64_t number) const {
    std::shared_lock lock(mutex_);

    auto it = blocks_.find(std::to_string(number));
    if (it != blocks_.end()) {
        return it->second;
    }

    return Block();
}

void TransactionSimulator::setAccountState(const AccountState& state) {
    std::unique_lock lock(mutex_);
    accountStates_[state.address] = state;
}

AccountState TransactionSimulator::getAccountState(const Address& address) const {
    std::shared_lock lock(mutex_);
    return getAccountStateLocked(address);
}

bool TransactionSimulator::fetchAccountStateLocked(const Address& address,
                                                   AccountState& out) const {
    if (rpc_ == nullptr) {
        return false;
    }

    const std::string hex_address = address.toString();

    // The balance arrives as a 0x-prefixed quantity. It is kept as a
    // 256-bit value so a large holder is not truncated to 64 bits.
    const auto balance = rpc_->GetBalance(hex_address);
    if (!balance.ok) {
        return false;
    }
    tigerwallet::math::U256 balance_value;
    if (!parseHexQuantity(balance.value, balance_value)) {
        return false;
    }
    out.balance = fromU256(balance_value);

    const auto nonce = rpc_->GetTransactionCount(hex_address);
    if (!nonce.ok) {
        return false;
    }
    out.nonce = nonce.value;

    // The nonce the node reports is the count of transactions already sent, so
    // the next one the account may use is one higher.
    return true;
}

uint256_t TransactionSimulator::spendLimitLocked(const Transaction& tx) const {
    // EIP-1559 caps the fee at maxFeePerGas; a legacy transaction uses
    // gasPrice. The value is added on top either way.
    const uint64_t fee_cap =
        (tx.maxFeePerGas > 0) ? tx.maxFeePerGas : tx.gasPrice;
    const tigerwallet::math::U256 fee = tigerwallet::math::U256::fromUint64(tx.gasLimit) *
                           tigerwallet::math::U256::fromUint64(fee_cap);

    return fromU256(fee + toU256(tx.value));
}

AccountState TransactionSimulator::getAccountStateLocked(const Address& address) const {
    // A state the caller set explicitly wins, which is what a test or an
    // override wants.
    auto it = accountStates_.find(address);
    if (it != accountStates_.end()) {
        return it->second;
    }

    // Otherwise the live state is read from the node.
    AccountState live;
    if (fetchAccountStateLocked(address, live)) {
        return live;
    }

    // With no node and no override there is nothing to report, so the caller
    // sees an empty state and its checks fail closed.
    return AccountState();
}

void TransactionSimulator::updateAccountState(const Address& address, const AccountState& state) {
    std::unique_lock lock(mutex_);
    updateAccountStateLocked(address, state);
}

void TransactionSimulator::updateAccountStateLocked(const Address& address, const AccountState& state) {
    accountStates_[address] = state;
}

void TransactionSimulator::addToken(const Token& token) {
    std::unique_lock lock(mutex_);
    tokens_[token.address] = token;
}

Token TransactionSimulator::getToken(const std::string& address) const {
    std::shared_lock lock(mutex_);

    auto it = tokens_.find(address);
    if (it != tokens_.end()) {
        return it->second;
    }

    return Token();
}

std::vector<Token> TransactionSimulator::getAllTokens() const {
    std::shared_lock lock(mutex_);

    std::vector<Token> tokens;
    for (const auto& pair : tokens_) {
        tokens.push_back(pair.second);
    }

    return tokens;
}

void TransactionSimulator::setSimulationCallback(SimulationCallback callback) {
    simulationCallback_ = callback;
}

TransactionSimulator::PerformanceMetrics TransactionSimulator::getMetrics() const {
    std::shared_lock lock(mutex_);
    return metrics_;
}

void TransactionSimulator::resetMetrics() {
    std::unique_lock lock(mutex_);
    metrics_ = PerformanceMetrics();
}

// Private methods
bool TransactionSimulator::validateTransaction(const Transaction& tx) {
    std::shared_lock lock(mutex_);
    return validateTransactionLocked(tx);
}

bool TransactionSimulator::validateTransactionLocked(const Transaction& tx) {
    if (!tx.isValid) {
        return false;
    }

    if (tx.from.isZero()) {
        return false;
    }

    if (tx.gasLimit < 21000) {
        return false;
    }

    // The bound above compares the gas *limit* against MAX_GAS_PRICE, which is
    // a price ceiling, so it let a limit of up to 1000 gwei through. The
    // meaningful bound is the block gas limit.
    if (tx.gasLimit > currentBlock_.gasLimit && currentBlock_.gasLimit > 0) {
        return false;
    }

    if (!checkGasLimit(tx)) {
        return false;
    }

    if (!checkNonce(tx)) {
        return false;
    }

    // A transaction that carries a signature must have a valid one. A
    // transaction with no signature at all is still simulatable, which is what
    // a user does before signing, but a malformed signature is rejected rather
    // than ignored.
    if (!tx.signature.empty() && !checkSignature(tx)) {
        return false;
    }

    return true;
}

void TransactionSimulator::executeTransaction(Transaction& tx, SimulationResult& result) {
    std::unique_lock lock(mutex_);
    executeTransactionLocked(tx, result);
}

void TransactionSimulator::executeTransactionLocked(Transaction& tx, SimulationResult& result) {
    // A simulation has to run the transaction on a node to observe what it
    // does. Without one there is nothing to report, so the result is a failure
    // rather than an optimistic success.
    if (rpc_ == nullptr) {
        result.success = false;
        result.error = "no node is configured for simulation";
        tx.status = TransactionStatus::FAILED;
        tx.simulationError = result.error;
        return;
    }

    // The node runs the transaction against its current state and reports the
    // gas it consumed, or the reason it reverted.
    std::string tx_object = "{\"from\":\"" + tx.from.toString() + "\"";
    if (!tx.to.isZero()) {
        tx_object += ",\"to\":\"" + tx.to.toString() + "\"";
    }
    tx_object += ",\"value\":\"" + tigerwallet::math::toHex(toU256(tx.value)) + "\"";
    if (!tx.data.empty()) {
        tx_object += ",\"data\":\"0x" + tigerwallet::rlp::Encoder::toHex(tx.data) + "\"";
    }
    tx_object += "}";

    const auto estimate = rpc_->EstimateGas(tx_object);
    if (!estimate.ok) {
        // A revert is a real outcome, not a transport problem, and the node
        // explains it. It is reported as a failed simulation.
        result.success = false;
        result.error = estimate.error.message.empty()
                           ? "the node rejected the transaction"
                           : estimate.error.message;
        tx.status = TransactionStatus::FAILED;
        tx.simulationError = result.error;
        return;
    }

    result.success = true;
    result.gasUsed = {estimate.value, 0, 0, 0};

    // The cost is the gas the node measured, at the price the transaction
    // declares.
    const uint64_t price = (tx.maxFeePerGas > 0) ? tx.maxFeePerGas : tx.gasPrice;
    result.gasPrice = price;

    const tigerwallet::math::U256 gas_cost =
        tigerwallet::math::U256::fromUint64(estimate.value) * tigerwallet::math::U256::fromUint64(price);

    // The observable effect: the sender pays the value and the gas, and the
    // recipient receives the value.
    if (!toU256(tx.value).isZero()) {
        result.valueOut = tx.value;
        result.affectedAddresses.push_back(tx.from);
        result.affectedAddresses.push_back(tx.to);
        result.stateChanges[tx.to.toString()] = "+" + toDecimalString(toU256(tx.value));
    }

    const tigerwallet::math::U256 debit = gas_cost + toU256(tx.value);
    result.stateChanges[tx.from.toString()] = "-" + toDecimalString(debit);

    tx.gasUsed = result.gasUsed;
    tx.status = TransactionStatus::SIMULATED;
}

std::string TransactionSimulator::signingHash(const Transaction& tx) const {
    using tigerwallet::crypto::Keccak256;
    using tigerwallet::rlp::Encoder;

    // The signing payload is the RLP list of the transaction's fields with no
    // signature. For a legacy transaction the fields are nonce, gasPrice,
    // gasLimit, to, value, data, chainId, 0, 0. The trailing zeros are the
    // EIP-155 replay-protection placeholders; omitting them produces a hash
    // that no EIP-155 node accepts.
    std::vector<tigerwallet::rlp::Bytes> fields;
    fields.push_back(Encoder::encodeUint(tx.nonce));
    fields.push_back(Encoder::encodeUint(tx.gasPrice));
    fields.push_back(Encoder::encodeUint(tx.gasLimit));
    // A contract creation has no recipient, which RLP encodes as the empty
    // string rather than as twenty zero bytes.
    if (tx.to.isZero()) {
        fields.push_back(Encoder::encodeBytes(nullptr, 0));
    } else {
        fields.push_back(Encoder::encodeBytes(tx.to.bytes.data(), 20));
    }
    fields.push_back(Encoder::encodeUint256(tx.value.data()));
    fields.push_back(Encoder::encodeBytes(tx.data));
    fields.push_back(Encoder::encodeUint(tx.chainId));
    fields.push_back(Encoder::encodeUint(0));
    fields.push_back(Encoder::encodeUint(0));

    const tigerwallet::rlp::Bytes encoded = Encoder::encodeList(fields);
    return "0x" + Keccak256::toHex(Keccak256::hash(encoded));
}

std::string TransactionSimulator::computeTransactionHash(
    const Transaction& tx
) const {
    using tigerwallet::crypto::Keccak256;
    using tigerwallet::rlp::Encoder;

    // The identity hash covers the same fields plus the signature components.
    // With no signature the two hashes coincide, which is the case for a
    // pending transaction that has not been signed yet.
    std::vector<tigerwallet::rlp::Bytes> fields;
    fields.push_back(Encoder::encodeUint(tx.nonce));
    fields.push_back(Encoder::encodeUint(tx.gasPrice));
    fields.push_back(Encoder::encodeUint(tx.gasLimit));
    if (tx.to.isZero()) {
        fields.push_back(Encoder::encodeBytes(nullptr, 0));
    } else {
        fields.push_back(Encoder::encodeBytes(tx.to.bytes.data(), 20));
    }
    fields.push_back(Encoder::encodeUint256(tx.value.data()));
    fields.push_back(Encoder::encodeBytes(tx.data));

    if (tx.signature.size() >= 65) {
        const uint8_t v = tx.signature[64];
        fields.push_back(Encoder::encodeBytes(tx.signature.data(), 32));
        fields.push_back(Encoder::encodeBytes(tx.signature.data() + 32, 32));
        fields.push_back(Encoder::encodeUint(v));
    } else {
        fields.push_back(Encoder::encodeUint(tx.chainId));
        fields.push_back(Encoder::encodeUint(0));
        fields.push_back(Encoder::encodeUint(0));
    }

    const tigerwallet::rlp::Bytes encoded = Encoder::encodeList(fields);
    return "0x" + Keccak256::toHex(Keccak256::hash(encoded));
}

bool TransactionSimulator::checkSignature(const Transaction& tx) {
    // This used to be `tx.signature.size() >= 64`, which accepts any 64 bytes
    // at all, a string of zeros included. validateTransaction did not call it
    // either, so a transaction with no valid signature was simulated and
    // reported as successful.
    //
    // A signature is r, s and v: 32 + 32 + 1 bytes. r and s must be non-zero
    // and below the secp256k1 group order n, because a value at or above n is
    // not a valid scalar and permits a malleable counterpart signature.
    if (tx.signature.size() != 65) {
        return false;
    }

    // The secp256k1 group order.
    static const uint8_t kOrder[32] = {
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
        0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b,
        0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41
    };

    auto isZero = [](const uint8_t* p) {
        for (int i = 0; i < 32; ++i) {
            if (p[i] != 0) return false;
        }
        return true;
    };
    auto lessThanOrder = [&](const uint8_t* p) {
        for (int i = 0; i < 32; ++i) {
            if (p[i] < kOrder[i]) return true;
            if (p[i] > kOrder[i]) return false;
        }
        return false;  // Equal to the order is not less than it.
    };

    const uint8_t* r = tx.signature.data();
    const uint8_t* s = tx.signature.data() + 32;
    const uint8_t v = tx.signature[64];

    if (isZero(r) || isZero(s)) return false;
    if (!lessThanOrder(r) || !lessThanOrder(s)) return false;

    // v is the recovery id, either 0 or 1, offset by 27 or by the chain id for
    // EIP-155 replay protection.
    if (v < 27) return false;

    return true;
}

bool TransactionSimulator::checkNonce(const Transaction& tx) {
    AccountState state = getAccountStateLocked(tx.from);

    // A transaction is only valid at the account's next nonce. A higher one
    // would be held as a gap, and a lower one is a replay.
    return tx.nonce == state.nonce;
}

bool TransactionSimulator::checkBalance(const Transaction& tx) {
    AccountState state = getAccountStateLocked(tx.from);

    // The comparison is 256-bit: a 64-bit multiply would overflow on a large
    // gas limit and report a balance as sufficient when it is not.
    const uint256_t required = spendLimitLocked(tx);
    return toU256(state.balance) >= toU256(required);
}

bool TransactionSimulator::checkGasLimit(const Transaction& tx) {
    return tx.gasLimit >= 21000 && tx.gasLimit <= currentBlock_.gasLimit;
}

double TransactionSimulator::calculateMEVRisk(const Transaction& tx) {
    double risk = 0.0;

    // High value transactions are more attractive for MEV
    if (tx.value[0] > 1000000000000000000ULL) { // > 1 ETH
        risk += 0.3;
    }

    // Swap transactions have higher MEV exposure
    if (tx.data.size() >= 4) {
        risk += 0.2;
    }

    // Gas price indicates urgency - could be MEV
    if (tx.gasPrice > currentBlock_.baseFeePerGas * 2) {
        risk += 0.2;
    }

    return std::min(risk, 1.0);
}

std::vector<Transaction> TransactionSimulator::findRelatedMempoolTransactions(const Transaction& tx) {
    // Reads mempool_ and is called from analyzeMEVLocked, which holds mutex_.
    std::vector<Transaction> related;

    for (const auto& pair : mempool_) {
        const auto& memTx = pair.second;

        // Same token pair (simplified check)
        if (memTx.to.toString() == tx.to.toString()) {
            related.push_back(memTx);
        }
    }

    return related;
}

bool TransactionSimulator::isFlashbotsBundle(const std::vector<Transaction>& txs) {
    // Check for Flashbots bundle indicators
    if (txs.empty()) return false;

    // First tx with specific markers indicates Flashbots
    const auto& first = txs[0];

    // Check for bundle hash or specific data patterns
    if (first.data.size() >= 4) {
        // Check for simulation type indicator
        if (first.data[0] == 0x00 && first.data[1] == 0x00 &&
            first.data[2] == 0x00 && first.data[3] == 0x0c) {
            return true;
        }
    }

    return false;
}

uint64_t TransactionSimulator::calculateIntrinsicGas(const Transaction& tx) {
    uint64_t gas = 21000; // Base transaction cost

    // Contract creation
    if (tx.isContractCreation) {
        gas += 32000;
    }

    // Data cost
    for (auto byte : tx.data) {
        if (byte == 0) {
            gas += 4;
        } else {
            gas += 16;
        }
    }

    return gas;
}

uint64_t TransactionSimulator::calculateOptimalGasPrice(const Transaction& tx) {
    uint64_t baseFee = currentBlock_.baseFeePerGas;

    // EIP-1559
    if (tx.type == TransactionType::EIP1559) {
        uint64_t priorityFee = tx.maxPriorityFeePerGas;
        return std::min(baseFee * 2 + priorityFee, tx.maxFeePerGas);
    }

    // Legacy
    return std::max(baseFee, tx.gasPrice);
}

uint64_t TransactionSimulator::calculateBaseFee(uint64_t parentBaseFee, uint64_t parentGasUsed, uint64_t gasLimit) {
    double targetGas = gasLimit / 2.0;
    double gasDelta = static_cast<double>(parentGasUsed) - targetGas;
    double feeDelta = parentBaseFee * gasDelta / targetGas;

    uint64_t newBaseFee = parentBaseFee + static_cast<uint64_t>(feeDelta);

    // Bounds
    newBaseFee = std::max(newBaseFee, SimConfig::MIN_GAS_PRICE);
    newBaseFee = std::min(newBaseFee, SimConfig::MAX_GAS_PRICE);

    return newBaseFee;
}

uint256_t TransactionSimulator::getAmountOut(uint256_t amountIn, uint256_t reserveIn, uint256_t reserveOut) {
    // Constant product with the 0.3% fee:
    //     amountOut = amountIn * 997 * reserveOut
    //                 -----------------------------
    //                 reserveIn * 1000 + amountIn * 997
    //
    // The previous version did this in 64-bit words and used only word[0] of
    // each value, so `amountIn[0] * reserveOut[0]` wrapped silently for any
    // realistic reserve. With reserves of 1e18 and 5e17 it produced
    // 437910009842106368 where the correct answer is 498, a wrong result with
    // nothing to indicate that anything had failed.
    //
    // The 512-bit intermediate below cannot wrap before the division brings
    // the result back into 256 bits.
    using tigerwallet::math::U256;
    using tigerwallet::math::mulDiv;
    using tigerwallet::math::divideFull;
    using tigerwallet::math::multiplyFull;

    const U256 in = U256::fromWords(amountIn[0], amountIn[1], amountIn[2], amountIn[3]);
    const U256 rIn = U256::fromWords(reserveIn[0], reserveIn[1], reserveIn[2], reserveIn[3]);
    const U256 rOut = U256::fromWords(reserveOut[0], reserveOut[1], reserveOut[2], reserveOut[3]);

    // A pool with no reserves has no price.
    if (rIn.isZero() || rOut.isZero()) {
        return uint256_t{0, 0, 0, 0};
    }

    const U256 amountInWithFee = mulDiv(in, U256(997), U256(1000));
    const U256 numerator = mulDiv(amountInWithFee, rOut, U256(1));
    const U256 denominator = rIn + amountInWithFee;
    const U256 out = divideFull(multiplyFull(numerator, U256(1)), denominator);

    uint256_t result;
    for (int i = 0; i < 4; ++i) {
        result[i] = out.word[i];
    }
    return result;
}
std::vector<Address> TransactionSimulator::findOptimalPath(const std::string& from, const std::string& to) {
    std::vector<Address> path;

    // Direct path
    path.push_back(Address::fromString(from));
    path.push_back(Address::fromString(to));

    return path;
}

} // namespace TigerWallet

// Main entry point for testing
#ifdef TEST_TRANSACTION_SIMULATOR
int main() {
    TigerWallet::TransactionSimulator simulator;

    // Create test transaction
    TigerWallet::Transaction tx;
    tx.from = TigerWallet::Address::fromString("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E");
    tx.to = TigerWallet::Address::fromString("0xdAC17F958D2ee523a2206206994597C13D831ec7");
    tx.value = {1000000000000000000ULL, 0, 0, 0}; // 1 ETH
    tx.gasLimit = 21000;
    tx.gasPrice = 20000000000ULL; // 20 gwei
    tx.nonce = 0;
    tx.chainId = 1;

    // The transaction hash is keccak256 of the RLP encoding, not a substring
    // of the addresses. The old line produced the same 16 characters for every
    // transaction between a given pair.
    tx.hash = simulator.computeTransactionHash(tx);

    // Simulate
    auto result = simulator.simulateTransaction(tx);

    std::cout << "Simulation success: " << (result.success ? "true" : "false") << std::endl;
    std::cout << "Gas used: " << result.gasUsed[0] << std::endl;
    std::cout << "Execution time: " << result.executionTimeMs << "ms" << std::endl;

    // MEV analysis
    auto mev = simulator.analyzeMEV(tx);
    std::cout << "MEV risk: " << mev.riskScore << std::endl;

    // Gas estimate
    auto gasEst = simulator.estimateGas(tx);
    std::cout << "Estimated gas: " << gasEst.gasLimit << std::endl;
    std::cout << "Estimated cost: " << gasEst.estimatedCost << " wei" << std::endl;

    // Get metrics
    auto metrics = simulator.getMetrics();
    std::cout << "Total simulations: " << metrics.totalSimulations << std::endl;
    std::cout << "Mempool size: " << metrics.mempoolSize << std::endl;

    return 0;
}
#endif
