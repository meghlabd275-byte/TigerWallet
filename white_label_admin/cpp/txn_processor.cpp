/**
 * TigerWallet Ultra-Low Latency Transaction Processor
 * Implementation
 */

#include "txn_processor.hpp"
#include <algorithm>
#include <cstring>
#include <sstream>
#include <iomanip>

namespace tigerwallet {
namespace txn {

// Address implementation
void Address::fromHex(const std::string& hex) {
    std::string cleanHex = hex;
    if (cleanHex.substr(0, 2) == "0x") {
        cleanHex = cleanHex.substr(2);
    }
    
    std::fill(data.begin(), data.end(), 0);
    
    size_t len = std::min(cleanHex.size() / 2, size_t(32));
    for (size_t i = 0; i < len; i++) {
        std::string byte_str = cleanHex.substr(i * 2, 2);
        data[31 - i] = static_cast<uint8_t>(std::stoi(byte_str, nullptr, 16));
    }
}

std::string Address::toHex() const {
    std::ostringstream oss;
    oss << "0x";
    for (int i = 31; i >= 0; i--) {
        oss << std::hex << std::setfill('0') << std::setw(2) << static_cast<int>(data[i]);
    }
    return oss.str();
}

bool Address::isZero() const {
    return std::all_of(data.begin(), data.end(), [](uint8_t b) { return b == 0; });
}

// TxHash implementation
void TxHash::fromHex(const std::string& hex) {
    std::string cleanHex = hex;
    if (cleanHex.substr(0, 2) == "0x") {
        cleanHex = cleanHex.substr(2);
    }
    
    std::fill(data.begin(), data.end(), 0);
    
    size_t len = std::min(cleanHex.size() / 2, size_t(32));
    for (size_t i = 0; i < len; i++) {
        std::string byte_str = cleanHex.substr(i * 2, 2);
        data[31 - i] = static_cast<uint8_t>(std::stoi(byte_str, nullptr, 16));
    }
}

std::string TxHash::toHex() const {
    std::ostringstream oss;
    oss << "0x";
    for (int i = 31; i >= 0; i--) {
        oss << std::hex << std::setfill('0') << std::setw(2) << static_cast<int>(data[i]);
    }
    return oss.str();
}

size_t TxHash::hash::operator()(const TxHash& h) const {
    size_t result = 0;
    for (size_t i = 0; i < 4; i++) {
        result ^= std::hash<uint64_t>{}(*(uint64_t*)(&h.data[i * 8])) + 0x9e3779b9 + (result << 6) + (result >> 2);
    }
    return result;
}

// uint256_t implementation
uint256_t uint256_t::operator+(const uint256_t& other) const {
    uint256_t result;
    __uint128_t carry = 0;
    for (int i = 0; i < 4; i++) {
        __uint128_t sum = (__uint128_t)words_[i] + other.words_[i] + carry;
        result.words_[i] = (uint64_t)sum;
        carry = sum >> 64;
    }
    return result;
}

uint256_t uint256_t::operator-(const uint256_t& other) const {
    uint256_t result;
    __int128_t borrow = 0;
    for (int i = 0; i < 4; i++) {
        __int128_t diff = (__int128_t)words_[i] - other.words_[i] - borrow;
        if (diff < 0) {
            diff += (__int128_t)1 << 64;
            borrow = 1;
        } else {
            borrow = 0;
        }
        result.words_[i] = (uint64_t)diff;
    }
    return result;
}

uint256_t uint256_t::operator*(const uint256_t& other) const {
    uint256_t result;
    for (int i = 0; i < 4; i++) {
        __uint128_t carry = 0;
        for (int j = 0; j < 4 - i; j++) {
            __uint128_t prod = (__uint128_t)words_[i + j] * other.words_[j] + result.words_[i + j] + carry;
            result.words_[i + j] = (uint64_t)prod;
            carry = prod >> 64;
        }
    }
    return result;
}

bool uint256_t::operator==(const uint256_t& other) const {
    return words_ == other.words_;
}

bool uint256_t::operator!=(const uint256_t& other) const {
    return words_ != other.words_;
}

bool uint256_t::operator<(const uint256_t& other) const {
    for (int i = 3; i >= 0; i--) {
        if (words_[i] < other.words_[i]) return true;
        if (words_[i] > other.words_[i]) return false;
    }
    return false;
}

std::string uint256_t::toString() const {
    if (words_[3] == 0 && words_[2] == 0 && words_[1] == 0) {
        return std::to_string(words_[0]);
    }
    std::ostringstream oss;
    oss << "0x" << std::hex << words_[3] << std::setfill('0') << std::setw(16) << words_[2]
        << std::setw(16) << words_[1] << std::setw(16) << words_[0];
    return oss.str();
}

// TransactionPool implementation
TransactionPool::TransactionPool() {
    pending_.reserve(MAX_POOL_SIZE);
}

TransactionPool::~TransactionPool() {
    clear();
}

bool TransactionPool::add(const Transaction& txn) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (pending_.size() >= MAX_POOL_SIZE) {
        return false;
    }
    
    if (index_.find(txn.hash) != index_.end()) {
        return false;
    }
    
    PendingTransaction pt;
    pt.txn = txn;
    pt.priority = txn.gas_price;
    pt.timestamp = txn.created_at.nanoseconds;
    
    size_t idx = pending_.size();
    pending_.push_back(pt);
    index_[txn.hash] = idx;
    
    return true;
}

bool TransactionPool::remove(const TxHash& hash) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = index_.find(hash);
    if (it == index_.end()) {
        return false;
    }
    
    size_t idx = it->second;
    size_t last = pending_.size() - 1;
    
    if (idx != last) {
        pending_[idx] = pending_[last];
        index_[pending_[idx].txn.hash] = idx;
    }
    
    pending_.pop_back();
    index_.erase(hash);
    
    return true;
}

std::optional<Transaction> TransactionPool::getNext() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (pending_.empty()) {
        return std::nullopt;
    }
    
    // Find highest gas price transaction
    size_t best_idx = 0;
    uint64_t best_priority = pending_[0].priority;
    
    for (size_t i = 1; i < pending_.size(); i++) {
        uint64_t p = pending_[i].priority;
        if (p > best_priority) {
            best_priority = p;
            best_idx = i;
        }
    }
    
    return pending_[best_idx].txn;
}

std::optional<Transaction> TransactionPool::get(const TxHash& hash) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = index_.find(hash);
    if (it == index_.end()) {
        return std::nullopt;
    }
    
    return pending_[it->second].txn;
}

void TransactionPool::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    pending_.clear();
    index_.clear();
}

std::vector<Transaction> TransactionPool::getByAddress(const Address& addr) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Transaction> result;
    for (const auto& pt : pending_) {
        if (pt.txn.from.data == addr.data || pt.txn.to.data == addr.data) {
            result.push_back(pt.txn);
        }
    }
    
    return result;
}

// EVMSignatureVerifier implementation
SignatureResult EVMSignatureVerifier::verify(const Transaction& txn, const std::vector<uint8_t>& signature) {
    // Simplified - in production would use proper ECDSA
    if (signature.size() < 65) {
        return {false, Address(), "Invalid signature length"};
    }
    
    // Verify using message hash
    std::vector<uint8_t> message;
    message.insert(message.end(), txn.from.data.begin(), txn.from.data.end());
    message.insert(message.end(), txn.to.data.begin(), txn.to.data.end());
    message.insert(message.end(), (uint8_t*)&txn.amount, (uint8_t*)&txn.amount + sizeof(txn.amount));
    
    Address signer = recoverSigner(message, signature);
    
    return {true, signer, ""};
}

bool EVMSignatureVerifier::batchVerify(const std::vector<Transaction>& txns, std::vector<uint8_t>& results) {
    results.resize(txns.size(), 0);
    
    for (size_t i = 0; i < txns.size(); i++) {
        results[i] = 1; // Simplified
    }
    
    return true;
}

Address EVMSignatureVerifier::recoverSigner(const std::vector<uint8_t>& msg, const std::vector<uint8_t>& sig) {
    // Simplified - in production use proper secp256k1 recovery
    Address addr;
    if (sig.size() >= 32) {
        for (size_t i = 0; i < 32 && i < msg.size(); i++) {
            addr.data[i] = msg[i] ^ sig[i % 32];
        }
    }
    return addr;
}

// TransactionProcessor implementation
TransactionProcessor::TransactionProcessor(const ProcessorConfig& config)
    : config_(config), running_(false), processed_count_(0), failed_count_(0) {
    
    pool_ = std::make_unique<TransactionPool>();
    verifier_ = std::make_unique<EVMSignatureVerifier>();
    
    stats_.total_processed = 0;
    stats_.total_failed = 0;
    stats_.total_gas_used = 0;
    stats_.avg_latency_ns = 0;
    stats_.max_latency_ns = 0;
    stats_.min_latency_ns = UINT64_MAX;
}

TransactionProcessor::~TransactionProcessor() {
    stop();
}

void TransactionProcessor::start() {
    running_ = true;
    
    for (uint32_t i = 0; i < config_.num_workers; i++) {
        workers_.emplace_back(&TransactionProcessor::workerLoop, this);
    }
}

void TransactionProcessor::stop() {
    running_ = false;
    
    for (auto& worker : workers_) {
        if (worker.joinable()) {
            worker.join();
        }
    }
}

ProcessingResult TransactionProcessor::submit(const Transaction& txn) {
    if (!validateTransaction(txn)) {
        return {false, TransactionStatus::FAILED, "Invalid transaction", HighPrecisionTimestamp::now(), 0};
    }
    
    if (!pool_->add(txn)) {
        return {false, TransactionStatus::FAILED, "Pool full", HighPrecisionTimestamp::now(), 0};
    }
    
    return processTransaction(const_cast<Transaction&>(txn));
}

std::vector<ProcessingResult> TransactionProcessor::submitBatch(const std::vector<Transaction>& txns) {
    std::vector<ProcessingResult> results;
    results.reserve(txns.size());
    
    for (const auto& txn : txns) {
        results.push_back(submit(txn));
    }
    
    return results;
}

std::optional<Transaction> TransactionProcessor::getTransaction(const TxHash& hash) {
    return pool_->get(hash);
}

TransactionProcessor::ProcessorStats TransactionProcessor::getStats() const {
    return {
        stats_.total_processed.load(),
        stats_.total_failed.load(),
        stats_.total_gas_used.load(),
        stats_.avg_latency_ns.load(),
        stats_.max_latency_ns.load(),
        stats_.min_latency_ns.load(),
        pool_->size()
    };
}

bool TransactionProcessor::isHealthy() const {
    return running_ && pool_->size() < pool_->capacity() * 0.9;
}

void TransactionProcessor::workerLoop() {
    while (running_) {
        auto txn_opt = pool_->getNext();
        
        if (!txn_opt) {
            std::this_thread::sleep_for(std::chrono::microseconds(100));
            continue;
        }
        
        auto result = processTransaction(*txn_opt);
        
        if (result.success) {
            processed_count_++;
        } else {
            failed_count_++;
        }
    }
}

ProcessingResult TransactionProcessor::processTransaction(Transaction& txn) {
    auto start = HighPrecisionTimestamp::now();
    
    // Process transaction
    txn.status = TransactionStatus::CONFIRMED;
    txn.processed_at = HighPrecisionTimestamp::now();
    txn.processed = true;
    
    auto end = HighPrecisionTimestamp::now();
    uint64_t latency = end.nanoseconds - start.nanoseconds;
    
    // Update statistics
    stats_.total_processed++;
    stats_.total_gas_used += txn.gas_used;
    
    uint64_t current_avg = stats_.avg_latency_ns.load();
    uint64_t new_avg = (current_avg * (stats_.total_processed.load() - 1) + latency) / stats_.total_processed.load();
    stats_.avg_latency_ns = new_avg;
    
    uint64_t current_max = stats_.max_latency_ns.load();
    if (latency > current_max) {
        stats_.max_latency_ns = latency;
    }
    
    uint64_t current_min = stats_.min_latency_ns.load();
    if (latency < current_min) {
        stats_.min_latency_ns = latency;
    }
    
    return {true, txn.status, "", end, txn.gas_used};
}

bool TransactionProcessor::validateTransaction(const Transaction& txn) {
    if (txn.from.isZero()) {
        return false;
    }
    
    if (txn.gas_price < config_.min_gas_price) {
        return false;
    }
    
    if (txn.gas_price > config_.max_gas_price) {
        return false;
    }
    
    return true;
}

bool TransactionProcessor::deduplicateTransaction(const Transaction& txn) {
    if (!config_.enable_deduplication) {
        return true;
    }
    
    return pool_->get(txn.hash).has_value() == false;
}

// TransactionEventEmitter implementation
void TransactionEventEmitter::onTransactionConfirmed(EventCallback cb) {
    std::lock_guard<std::mutex> lock(mutex_);
    confirmed_.push_back(cb);
}

void TransactionEventEmitter::onTransactionFailed(EventCallback cb) {
    std::lock_guard<std::mutex> lock(mutex_);
    failed_.push_back(cb);
}

void TransactionEventEmitter::onTransactionFlagged(EventCallback cb) {
    std::lock_guard<std::mutex> lock(mutex_);
    flagged_.push_back(cb);
}

void TransactionEventEmitter::emitConfirmed(const Transaction& txn) {
    std::lock_guard<std::mutex> lock(mutex_);
    for (const auto& cb : confirmed_) {
        cb(txn);
    }
}

void TransactionEventEmitter::emitFailed(const Transaction& txn) {
    std::lock_guard<std::mutex> lock(mutex_);
    for (const auto& cb : failed_) {
        cb(txn);
    }
}

void TransactionEventEmitter::emitFlagged(const Transaction& txn) {
    std::lock_guard<std::mutex> lock(mutex_);
    for (const auto& cb : flagged_) {
        cb(txn);
    }
}

// RateLimiter implementation
RateLimiter::RateLimiter(uint64_t max_rps) : max_requests_per_second_(max_rps), current_count_(0) {
    window_start_ = std::chrono::steady_clock::now();
}

bool RateLimiter::allow() {
    auto now = std::chrono::steady_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(now - window_start_).count();
    
    if (elapsed >= 1) {
        reset();
        window_start_ = now;
    }
    
    uint64_t current = current_count_.fetch_add(1);
    return current < max_requests_per_second_;
}

void RateLimiter::reset() {
    current_count_ = 0;
}

uint64_t RateLimiter::currentCount() const {
    return current_count_.load();
}

// BlockBuilder implementation
BlockBuilder::BlockBuilder(uint64_t block_number, uint64_t gas_limit)
    : gas_limit_(gas_limit), gas_used_(0), block_number_(block_number) {}

bool BlockBuilder::addTransaction(const Transaction& txn) {
    if (gas_used_ + txn.gas_limit > gas_limit_) {
        return false;
    }
    
    transactions_.push_back(txn);
    gas_used_ += txn.gas_limit;
    return true;
}

void BlockBuilder::removeTransaction(size_t index) {
    if (index >= transactions_.size()) {
        return;
    }
    
    gas_used_ -= transactions_[index].gas_limit;
    transactions_.erase(transactions_.begin() + index);
}

std::vector<Transaction> BlockBuilder::build() {
    // Sort by gas price for optimal block composition
    std::sort(transactions_.begin(), transactions_.end(), [](const Transaction& a, const Transaction& b) {
        return a.gas_price > b.gas_price;
    });
    
    return transactions_;
}

// MempoolMonitor implementation
void MempoolMonitor::addTransaction(const Transaction& txn) {
    std::lock_guard<std::mutex> lock(mutex_);
    transactions_[txn.hash] = txn;
}

void MempoolMonitor::removeTransaction(const TxHash& hash) {
    std::lock_guard<std::mutex> lock(mutex_);
    transactions_.erase(hash);
}

std::vector<Transaction> MempoolMonitor::getHighValueTransactions(uint64_t threshold) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Transaction> result;
    for (const auto& [hash, txn] : transactions_) {
        if (txn.amount.low64() >= threshold) {
            result.push_back(txn);
        }
    }
    
    return result;
}

std::vector<Transaction> MempoolMonitor::getByAddress(const Address& addr) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<Transaction> result;
    for (const auto& [hash, txn] : transactions_) {
        if (txn.from.data == addr.data || txn.to.data == addr.data) {
            result.push_back(txn);
        }
    }
    
    return result;
}

size_t MempoolMonitor::size() const {
    return transactions_.size();
}

} // namespace txn
} // namespace tigerwallet
