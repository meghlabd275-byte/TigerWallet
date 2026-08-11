/**
 * TigerWallet Desktop - Multisig Service Implementation
 */

#include "services/multisig_service.h"
#include "services/api_client.h"
#include <iostream>
#include <sstream>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

std::shared_ptr<MultisigService> MultisigService::instance_ = nullptr;

MultisigService::MultisigService() : initialized_(false), baseUrl_("http://localhost:8450") {}

std::shared_ptr<MultisigService> MultisigService::getInstance() {
    if (!instance_) {
        instance_ = std::make_shared<MultisigService>();
    }
    return instance_;
}

void MultisigService::initialize(const std::string& baseUrl) {
    baseUrl_ = baseUrl.empty() ? "http://localhost:8450" : baseUrl;
    // Ensure the global CURL handle is initialized for this process.
    curl_global_init(CURL_GLOBAL_DEFAULT);
    initialized_ = true;
    std::cout << "[MultisigService] Initialized with base URL: " << baseUrl_ << std::endl;
}

void MultisigService::shutdown() {
    initialized_ = false;
}

// Dedicated client (separate from the wallet_api singleton) pointed at the
// multisig service port. Each call uses a fresh client so the base URL is
// always correct regardless of other services' state.
std::string MultisigService::post(const std::string& endpoint, const std::string& body) {
    APIClient client;
    client.initialize(baseUrl_);
    std::string url = client.buildUrl(endpoint, std::nullopt);
    return client.executeRequest(HTTPMethod::POST, url, body);
}

std::string MultisigService::get(const std::string& endpoint) {
    APIClient client;
    client.initialize(baseUrl_);
    std::string url = client.buildUrl(endpoint, std::nullopt);
    return client.executeRequest(HTTPMethod::GET, url, std::nullopt);
}

// ============================================================================
// Wallet lifecycle
// ============================================================================

std::future<MultisigWalletInfo> MultisigService::createWallet(const std::vector<std::string>& owners,
                                                               int threshold) {
    return std::async(std::launch::async, [this, owners, threshold]() -> MultisigWalletInfo {
        MultisigWalletInfo w;
        if (owners.empty() || threshold < 1) return w;

        try {
            std::ostringstream body;
            body << "{\"owners\":[";
            for (size_t i = 0; i < owners.size(); ++i) {
                if (i) body << ",";
                body << "\"" << owners[i] << "\"";
            }
            body << "],\"threshold\":" << threshold << "}";

            std::string resp = post("/api/v1/multisig/wallets", body.str());
            auto id = jsonStringField(resp, "id");
            auto addr = jsonStringField(resp, "address");
            if (id) w.id = *id;
            if (addr) w.address = *addr;
            w.owners = owners;
            w.threshold = threshold;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] createWallet failed: " << e.what() << std::endl;
        }
        return w;
    });
}

std::future<std::vector<MultisigWalletInfo>> MultisigService::listWallets() {
    return std::async(std::launch::async, [this]() -> std::vector<MultisigWalletInfo> {
        std::vector<MultisigWalletInfo> out;
        try {
            std::string resp = get("/api/v1/multisig/wallets");
            // Best-effort parse: the backend returns a JSON array of wallet
            // objects. We extract id + address pairs from the raw string.
            size_t pos = 0;
            while (true) {
                size_t idPos = resp.find("\"id\"", pos);
                if (idPos == std::string::npos) break;
                size_t start = resp.find('"', idPos + 4);
                size_t end = resp.find('"', start + 1);
                std::string id = resp.substr(start + 1, end - start - 1);

                size_t addrPos = resp.find("\"address\"", end);
                size_t aStart = resp.find('"', addrPos + 9);
                size_t aEnd = resp.find('"', aStart + 1);
                std::string addr = resp.substr(aStart + 1, aEnd - aStart - 1);

                MultisigWalletInfo w;
                w.id = id;
                w.address = addr;
                out.push_back(w);
                pos = aEnd;
            }
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] listWallets failed: " << e.what() << std::endl;
        }
        return out;
    });
}

std::future<MultisigWalletInfo> MultisigService::getWallet(const std::string& walletId) {
    return std::async(std::launch::async, [this, walletId]() -> MultisigWalletInfo {
        MultisigWalletInfo w;
        if (walletId.empty()) return w;
        try {
            std::string resp = get("/api/v1/multisig/wallets/" + walletId);
            auto id = jsonStringField(resp, "id");
            auto addr = jsonStringField(resp, "address");
            if (id) w.id = *id;
            if (addr) w.address = *addr;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] getWallet failed: " << e.what() << std::endl;
        }
        return w;
    });
}

// ============================================================================
// Owner management
// ============================================================================

std::future<bool> MultisigService::addOwner(const std::string& walletId, const std::string& ownerAddress) {
    return std::async(std::launch::async, [this, walletId, ownerAddress]() -> bool {
        if (walletId.empty() || ownerAddress.empty()) return false;
        try {
            std::string body = "{\"owner\":\"" + ownerAddress + "\"}";
            std::string resp = post("/api/v1/multisig/wallets/" + walletId + "/owners", body);
            return resp.find("error") == std::string::npos;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] addOwner failed: " << e.what() << std::endl;
            return false;
        }
    });
}

// ============================================================================
// Transaction lifecycle
// ============================================================================

std::future<MultisigTransaction> MultisigService::createTransaction(const std::string& walletId,
                                                                    const std::string& to,
                                                                    const std::string& value,
                                                                    const std::string& data) {
    return std::async(std::launch::async, [this, walletId, to, value, data]() -> MultisigTransaction {
        MultisigTransaction tx;
        tx.wallet_id = walletId;
        tx.to = to;
        tx.value = value;
        tx.data = data;
        if (walletId.empty()) return tx;

        try {
            std::ostringstream body;
            body << "{\"wallet_id\":\"" << walletId << "\","
                 << "\"to\":\"" << to << "\","
                 << "\"value\":\"" << value << "\","
                 << "\"data\":\"" << data << "\"}";
            std::string resp = post("/api/v1/multisig/transactions", body.str());
            auto id = jsonStringField(resp, "id");
            if (id) tx.id = *id;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] createTransaction failed: " << e.what() << std::endl;
        }
        return tx;
    });
}

std::future<MultisigTransaction> MultisigService::getTransaction(const std::string& txId) {
    return std::async(std::launch::async, [this, txId]() -> MultisigTransaction {
        MultisigTransaction tx;
        tx.id = txId;
        if (txId.empty()) return tx;
        try {
            std::string resp = get("/api/v1/multisig/transactions/" + txId);
            auto id = jsonStringField(resp, "id");
            auto to = jsonStringField(resp, "to");
            auto value = jsonStringField(resp, "value");
            if (id) tx.id = *id;
            if (to) tx.to = *to;
            if (value) tx.value = *value;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] getTransaction failed: " << e.what() << std::endl;
        }
        return tx;
    });
}

std::future<bool> MultisigService::signTransaction(const std::string& txId) {
    return std::async(std::launch::async, [this, txId]() -> bool {
        if (txId.empty()) return false;
        try {
            std::string resp = post("/api/v1/multisig/transactions/" + txId + "/sign", "{}");
            return resp.find("error") == std::string::npos;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] signTransaction failed: " << e.what() << std::endl;
            return false;
        }
    });
}

std::future<MultisigTransaction> MultisigService::executeTransaction(const std::string& txId) {
    return std::async(std::launch::async, [this, txId]() -> MultisigTransaction {
        MultisigTransaction tx;
        tx.id = txId;
        if (txId.empty()) return tx;
        try {
            // The backend collects the threshold owner signatures off-chain,
            // assembles the on-chain MultisigWallet.executeTransaction call,
            // signs + broadcasts it via eth_sendRawTransaction, and returns the
            // REAL transaction hash. Honest result: empty hash on failure.
            std::string resp = post("/api/v1/multisig/transactions/" + txId + "/execute", "{}");
            auto hash = jsonStringField(resp, "tx_hash");
            if (!hash) hash = jsonStringField(resp, "txHash");
            if (hash) tx.tx_hash = *hash;
            tx.executed = !tx.tx_hash.empty();
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] executeTransaction failed: " << e.what() << std::endl;
        }
        return tx;
    });
}

std::future<bool> MultisigService::revokeTransaction(const std::string& txId) {
    return std::async(std::launch::async, [this, txId]() -> bool {
        if (txId.empty()) return false;
        try {
            std::string resp = post("/api/v1/multisig/transactions/" + txId + "/revoke", "{}");
            return resp.find("error") == std::string::npos;
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] revokeTransaction failed: " << e.what() << std::endl;
            return false;
        }
    });
}

std::future<std::vector<MultisigTransaction>> MultisigService::pendingTransactions(const std::string& walletId) {
    return std::async(std::launch::async, [this, walletId]() -> std::vector<MultisigTransaction> {
        std::vector<MultisigTransaction> out;
        if (walletId.empty()) return out;
        try {
            std::string resp = get("/api/v1/multisig/wallets/" + walletId + "/transactions");
            // Best-effort parse of the pending-tx array.
            size_t pos = 0;
            while (true) {
                size_t idPos = resp.find("\"id\"", pos);
                if (idPos == std::string::npos) break;
                size_t start = resp.find('"', idPos + 4);
                size_t end = resp.find('"', start + 1);
                MultisigTransaction tx;
                tx.id = resp.substr(start + 1, end - start - 1);
                out.push_back(tx);
                pos = end;
            }
        } catch (const std::exception& e) {
            std::cerr << "[MultisigService] pendingTransactions failed: " << e.what() << std::endl;
        }
        return out;
    });
}

} // namespace wallet
} // namespace tiger
