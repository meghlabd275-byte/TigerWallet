// In-memory pool of transactions awaiting confirmation.
//
// The pool keeps one transaction per (sender, nonce). Adding a transaction at a
// nonce that is already present replaces the earlier one only if the new one
// pays a higher gas price, which is the rule a node applies when a sender
// resubmits. Anything else is rejected rather than silently displacing a
// transaction that may already be propagating.

#ifndef TIGERWALLET_MEMPOOL_HPP
#define TIGERWALLET_MEMPOOL_HPP

#include <cstdint>
#include <map>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

namespace tigerwallet {

struct PooledTransaction {
    std::string hash;
    std::string from;
    uint64_t nonce = 0;
    std::string gas_price_wei;
    uint64_t gas_limit = 0;
    std::string to;
    std::string value_wei;
    std::string data_hex;
    uint64_t added_at = 0;  // seconds since the epoch
};

enum class PoolResult {
    ACCEPTED,
    REPLACED,
    REJECTED_UNDERPRICED,  // same nonce already present at a higher or equal price
    REJECTED_DUPLICATE,    // identical hash already present
};

class TransactionPool {
public:
    // Adds or replaces. Returns what happened.
    PoolResult Add(PooledTransaction tx);

    // Marks a transaction confirmed and drops it from the pool.
    bool Confirm(const std::string& hash);

    // Drops a transaction without confirming it.
    bool Remove(const std::string& hash);

    std::optional<PooledTransaction> Get(const std::string& hash) const;
    std::vector<PooledTransaction> ForSender(const std::string& from) const;
    std::vector<PooledTransaction> All() const;

    size_t Size() const;

    // Drops pending transactions older than `max_age_seconds`.
    size_t PruneOlderThan(uint64_t max_age_seconds, uint64_t now);

private:
    struct Entry {
        PooledTransaction tx;
    };

    mutable std::mutex mutex_;
    std::unordered_map<std::string, Entry> by_hash_;
    // (sender, nonce) -> hash, so a resubmission at the same nonce is found
    // without scanning the whole pool.
    std::map<std::pair<std::string, uint64_t>, std::string> by_sender_nonce_;
};

// Compares two decimal wei strings numerically. Returns true when a > b.
bool DecimalGreater(const std::string& a, const std::string& b);

}  // namespace tigerwallet

#endif  // TIGERWALLET_MEMPOOL_HPP
