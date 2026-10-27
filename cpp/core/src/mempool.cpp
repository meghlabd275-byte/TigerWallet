// In-memory transaction pool. See include/mempool.hpp.

#include "mempool.hpp"

#include <algorithm>
#include <cstddef>

namespace tigerwallet {
namespace {

// Strip leading zeros so equal magnitudes compare equal regardless of padding.
std::string NormaliseDecimal(const std::string& value) {
    size_t start = 0;
    while (start + 1 < value.size() && value[start] == '0') ++start;
    std::string out = value.substr(start);
    if (out.empty()) out = "0";
    return out;
}

}  // namespace

bool DecimalGreater(const std::string& a, const std::string& b) {
    const std::string lhs = NormaliseDecimal(a);
    const std::string rhs = NormaliseDecimal(b);
    if (lhs.size() != rhs.size()) return lhs.size() > rhs.size();
    return lhs > rhs;
}

PoolResult TransactionPool::Add(PooledTransaction tx) {
    std::lock_guard<std::mutex> lock(mutex_);

    if (by_hash_.count(tx.hash) != 0) {
        return PoolResult::REJECTED_DUPLICATE;
    }

    const auto key = std::make_pair(tx.from, tx.nonce);
    const auto existing = by_sender_nonce_.find(key);
    if (existing != by_sender_nonce_.end()) {
        const auto prior = by_hash_.find(existing->second);
        // Only a strictly higher gas price justifies displacing a transaction
        // that may already be in a node's mempool.
        if (prior != by_hash_.end() &&
            !DecimalGreater(tx.gas_price_wei, prior->second.tx.gas_price_wei)) {
            return PoolResult::REJECTED_UNDERPRICED;
        }
        if (prior != by_hash_.end()) {
            by_hash_.erase(prior);
        }
        by_sender_nonce_.erase(existing);

        const std::string hash = tx.hash;
        by_hash_.emplace(hash, Entry{std::move(tx)});
        by_sender_nonce_[key] = hash;
        return PoolResult::REPLACED;
    }

    const std::string hash = tx.hash;
    by_hash_.emplace(hash, Entry{std::move(tx)});
    by_sender_nonce_[key] = hash;
    return PoolResult::ACCEPTED;
}

bool TransactionPool::Confirm(const std::string& hash) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = by_hash_.find(hash);
    if (it == by_hash_.end()) return false;
    by_sender_nonce_.erase(std::make_pair(it->second.tx.from, it->second.tx.nonce));
    by_hash_.erase(it);
    return true;
}

bool TransactionPool::Remove(const std::string& hash) {
    return Confirm(hash);
}

std::optional<PooledTransaction> TransactionPool::Get(const std::string& hash) const {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = by_hash_.find(hash);
    if (it == by_hash_.end()) return std::nullopt;
    return it->second.tx;
}

std::vector<PooledTransaction> TransactionPool::ForSender(const std::string& from) const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<PooledTransaction> out;
    for (const auto& kv : by_hash_) {
        if (kv.second.tx.from == from) out.push_back(kv.second.tx);
    }
    std::sort(out.begin(), out.end(),
              [](const PooledTransaction& a, const PooledTransaction& b) {
                  return a.nonce < b.nonce;
              });
    return out;
}

std::vector<PooledTransaction> TransactionPool::All() const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<PooledTransaction> out;
    out.reserve(by_hash_.size());
    for (const auto& kv : by_hash_) out.push_back(kv.second.tx);
    return out;
}

size_t TransactionPool::Size() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return by_hash_.size();
}

size_t TransactionPool::PruneOlderThan(uint64_t max_age_seconds, uint64_t now) {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t removed = 0;
    for (auto it = by_hash_.begin(); it != by_hash_.end();) {
        const uint64_t age = now > it->second.tx.added_at ? now - it->second.tx.added_at : 0;
        if (age > max_age_seconds) {
            by_sender_nonce_.erase(
                std::make_pair(it->second.tx.from, it->second.tx.nonce));
            it = by_hash_.erase(it);
            ++removed;
        } else {
            ++it;
        }
    }
    return removed;
}

}  // namespace tigerwallet
