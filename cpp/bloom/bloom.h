/**
 * TigerWallet Bloom Filter
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Probabilistic membership testing
 * - Space-efficient
 * - False positive detection
 */

#ifndef TIGER_BLOOM_H
#define TIGER_BLOOM_H

#include <bitset>
#include <vector>
#include <string>
#include <functional>

namespace tiger {

class BloomFilter {
private:
    std::vector<bool> bits_;
    size_t size_;
    size_t num_hashes_;
    std::hash<std::string> hasher_;

public:
    BloomFilter(size_t size, size_t num_hashes)
        : bits_(size, false), size_(size), num_hashes_(num_hashes) {}

    void add(const std::string& item) {
        for (size_t i = 0; i < num_hashes_; ++i) {
            size_t index = hash(item, i);
            bits_[index] = true;
        }
    }

    bool contains(const std::string& item) const {
        for (size_t i = 0; i < num_hashes_; ++i) {
            size_t index = hash(item, i);
            if (!bits_[index]) {
                return false;
            }
        }
        return true;
    }

    double false_positive_rate() const {
        size_t set_bits = 0;
        for (bool bit : bits_) {
            if (bit) set_bits++;
        }
        return std::pow((double)set_bits / size_, num_hashes_);
    }

private:
    size_t hash(const std::string& item, size_t seed) const {
        size_t h = hasher_(item);
        return (h + seed * 2654435761) % size_;
    }
};

} // namespace tiger

#endif
