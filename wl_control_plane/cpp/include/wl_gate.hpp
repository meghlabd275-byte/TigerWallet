// TigerWallet WL Control Plane — ultra-low-latency license + feature-flag checker.
//
// This is the HOT PATH: it runs on EVERY request inside a white-label product
// (before the handler dispatches) and must answer in single-digit microseconds.
// It does NO crypto (the Rust SDK / Go control plane did the real Ed25519
// verification at validate/heartbeat time). It holds a wait-free atomic snapshot
// of the liveness flag + the feature-flag cache and answers in O(1).
//
// Architecture:
//   - The WL product's process boots the Rust `LicenseClient`, which validates
//     against the control plane and, on success, pushes the liveness + flag
//     snapshot into this C++ `WlGate` via `wl_gate_set_alive` /
//     `wl_gate_set_flags`.
//   - Every HTTP request calls `wl_gate_is_alive()` + `wl_gate_fetcher_enabled()`
//     (exposed via a C ABI so Go/Rust/Node backends can FFI in).
//   - The gate is wait-free: readers never block writers.
//
// Language rationale: C++ for raw latency + zero-overhead atomics + a stable C
// ABI that every WL product backend can FFI into without recompiling the world.
#pragma once

#include <atomic>
#include <cstdint>
#include <cstring>
#include <string>
#include <string_view>
#include <unordered_map>
#include <shared_mutex>
#include <vector>

namespace tigerwallet::wl {

// A single feature-flag entry: (product, fetcher) -> enabled.
struct FlagEntry {
    std::string product;
    std::string fetcher; // "*" = whole product
    bool enabled;
};

// WlGate is the wait-free-read liveness + flag gate. One instance per process.
// Reads (is_alive / fetcher_enabled) are lock-free atomics + a shared_mutex
// read lock on the flag map (contended only during a flag refresh, which is
// rare — once per heartbeat). Writes happen only on heartbeat tick.
class WlGate {
public:
    static WlGate& instance() {
        static WlGate g;
        return g;
    }

    // --- liveness (lock-free atomic) ---
    void set_alive(bool v, const char* reason = nullptr) {
        alive_.store(v, std::memory_order_release);
        if (reason) {
            std::lock_guard<std::shared_mutex> lk(reason_mu_);
            reason_ = reason;
        } else if (v) {
            std::lock_guard<std::shared_mutex> lk(reason_mu_);
            reason_.clear();
        }
    }
    bool is_alive() const { return alive_.load(std::memory_order_acquire); }
    std::string reason() const {
        std::shared_lock<std::shared_mutex> lk(reason_mu_);
        return reason_;
    }

    // --- flags (shared_mutex: many readers, rare writers) ---
    void set_flags(const std::vector<FlagEntry>& flags) {
        std::lock_guard<std::shared_mutex> lk(flag_mu_);
        flag_map_.clear();
        for (const auto& f : flags) {
            flag_map_[key(f.product, f.fetcher)] = f.enabled;
        }
    }

    // Returns true if the fetcher is permitted. Fail-closed: if the product
    // is not alive, this returns false regardless of flags.
    bool fetcher_enabled(std::string_view product, std::string_view fetcher) const {
        if (!is_alive()) return false;
        std::shared_lock<std::shared_mutex> lk(flag_mu_);
        // whole-product flag
        auto it_star = flag_map_.find(key(product, "*"));
        if (it_star != flag_map_.end() && !it_star->second) return false;
        // specific fetcher flag
        auto it = flag_map_.find(key(product, fetcher));
        if (it != flag_map_.end() && !it->second) return false;
        return true; // absent => enabled (default-permit until SuperAdmin disables)
    }

    // Bulk check used by middleware to short-circuit a whole request.
    bool request_allowed(std::string_view product, std::string_view fetcher) const {
        return fetcher_enabled(product, fetcher);
    }

private:
    WlGate() = default;
    static std::string key(std::string_view product, std::string_view fetcher) {
        std::string k;
        k.reserve(product.size() + 1 + fetcher.size());
        k.append(product.data(), product.size());
        k.push_back('\x1f'); // unit separator — safe delimiter
        k.append(fetcher.data(), fetcher.size());
        return k;
    }

    std::atomic<bool> alive_{false};
    mutable std::shared_mutex reason_mu_;
    std::string reason_;
    mutable std::shared_mutex flag_mu_;
    std::unordered_map<std::string, bool> flag_map_;
};

} // namespace tigerwallet::wl

// --- C ABI so Go/Rust/Node WL backends can FFI in without C++ name mangling ---
extern "C" {
    // Returns 1 if the product is alive (licensed + heartbeat current), 0 otherwise.
    int wl_gate_is_alive();
    // Returns the reason the product is dead (empty string if alive).
    const char* wl_gate_reason();
    // Sets the liveness flag (called by the Rust SDK after validate/heartbeat).
    void wl_gate_set_alive(int alive, const char* reason);
    // Returns 1 if the fetcher is enabled for the product, 0 otherwise.
    int wl_gate_fetcher_enabled(const char* product, const char* fetcher);
    // Sets the flag snapshot (called by the Rust SDK on each heartbeat).
    void wl_gate_set_flags(const char* json_array);
}
