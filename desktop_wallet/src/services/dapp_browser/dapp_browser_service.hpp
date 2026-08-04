/**
 * DApp Browser Service - C++ Desktop Implementation
 * Web3-enabled browser for decentralized applications
 */

#ifndef DAPP_BROWSER_SERVICE_HPP
#define DAPP_BROWSER_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <functional>
#include <optional>

using namespace std;

// DApp categories
enum class DAppCategory { DEFI, NFT, GAMING, SOCIAL, STAKING, BRIDGE, OTHER };

// DApp status
enum class DAppStatus { VERIFIED, PENDING, SUSPENDED };

// DApp info
struct DAppInfo {
    string id;
    string name;
    string url;
    string description;
    string logoUrl;
    string category;
    double rating;
    int users;
    double volume24h;
    DAppStatus status;
    vector<string> chains;
    string contractAddress;
    vector<string> tags;
    uint64_t createdAt;
    
    DAppInfo() : rating(0), users(0), volume24h(0), status(DAppStatus::PENDING), createdAt(0) {}
};

// DApp favorite
struct DAppFavorite {
    string userId;
    string dappId;
    uint64_t addedAt;
};

// Browsing history
struct BrowsingHistory {
    string userId;
    string dappId;
    string url;
    uint64_t visitedAt;
};

// Wallet connection
struct WalletConnection {
    string sessionId;
    string userId;
    string dappId;
    string walletAddress;
    string chain;
    vector<string> permissions;
    uint64_t connectedAt;
    uint64_t expiresAt;
};

// Transaction request
struct TransactionRequest {
    string id;
    string dappId;
    string from;
    string to;
    string value;
    string data;
    string gasLimit;
    string gasPrice;
    string chainId;
    uint64_t createdAt;
};

// Signature request
struct SignatureRequest {
    string id;
    string dappId;
    string address;
    string message;
    string method;
    uint64_t createdAt;
};

// DApp Browser Service
class DAppBrowserService {
private:
    mutex mutex_;
    unordered_map<string, DAppInfo> dapps_;
    unordered_map<string, vector<DAppFavorite>> userFavorites_;
    unordered_map<string, vector<BrowsingHistory>> userHistory_;
    unordered_map<string, WalletConnection> activeConnections_;
    unordered_map<string, TransactionRequest> pendingTransactions_;
    unordered_map<string, SignatureRequest> pendingSignatures_;
    
    // Callbacks
    function<void(const TransactionRequest&)> transactionCallback_;
    function<void(const SignatureRequest&)> signatureCallback_;
    
public:
    DAppBrowserService() {
        initializeDApps();
    }
    
    void initializeDApps() {
        lock_guard<mutex> lock(mutex_);
        
        addDApp("uniswap", "Uniswap", "https://app.uniswap.org", 
                "Decentralized trading protocol", "DeFi", {"ETH", "ARB"}, 4.8, 500000);
        addDApp("curve", "Curve Finance", "https://curve.fi",
                "Stable asset exchange", "DeFi", {"ETH", "BSC"}, 4.7, 300000);
        addDApp("aave", "Aave", "https://app.aave.com",
                "Non-custodial liquidity protocol", "DeFi", {"ETH", "POLY"}, 4.6, 250000);
        addDApp("opensea", "OpenSea", "https://opensea.io",
                "NFT marketplace", "NFT", {"ETH", "BSC"}, 4.9, 1000000);
        addDApp("blur", "Blur", "https://blur.io",
                "NFT marketplace for pro traders", "NFT", {"ETH"}, 4.7, 500000);
        addDApp("lido", "Lido", "https://lido.fi",
                "Liquid staking", "Staking", {"ETH"}, 4.8, 400000);
        addDApp("across", "Across", "https://across.to",
                "Cross-chain bridge", "Bridge", {"ETH", "ARB", "OPT"}, 4.6, 180000);
    }
    
    void addDApp(const string& id, const string& name, const string& url,
                  const string& desc, const string& category,
                  const vector<string>& chains, double rating, int users) {
        DAppInfo dapp;
        dapp.id = id;
        dapp.name = name;
        dapp.url = url;
        dapp.description = desc;
        dapp.category = category;
        dapp.chains = chains;
        dapp.rating = rating;
        dapp.users = users;
        dapp.volume24h = users * 100.0;
        dapp.status = DAppStatus::VERIFIED;
        
        dapps_[id] = dapp;
    }
    
    vector<DAppInfo> getFeaturedDApps() {
        lock_guard<mutex> lock(mutex_);
        vector<DAppInfo> result;
        for (const auto& pair : dapps_) {
            result.push_back(pair.second);
        }
        return result;
    }
    
    vector<DAppInfo> getDAppsByCategory(const string& category) {
        lock_guard<mutex> lock(mutex_);
        vector<DAppInfo> result;
        for (const auto& pair : dapps_) {
            if (pair.second.category == category) {
                result.push_back(pair.second);
            }
        }
        return result;
    }
    
    vector<DAppInfo> searchDApps(const string& query) {
        lock_guard<mutex> lock(mutex_);
        vector<DAppInfo> result;
        string lowerQuery = query;
        transform(lowerQuery.begin(), lowerQuery.end(), lowerQuery.begin(), ::tolower);
        
        for (const auto& pair : dapps_) {
            const auto& dapp = pair.second;
            string nameLower = dapp.name;
            transform(nameLower.begin(), nameLower.end(), nameLower.begin(), ::tolower);
            if (nameLower.find(lowerQuery) != string::npos) {
                result.push_back(dapp);
            }
        }
        return result;
    }
    
    optional<DAppInfo> getDAppDetails(const string& dappId) {
        lock_guard<mutex> lock(mutex_);
        auto it = dapps_.find(dappId);
        if (it != dapps_.end()) {
            return it->second;
        }
        return nullopt;
    }
    
    vector<string> getCategories() {
        lock_guard<mutex> lock(mutex_);
        vector<string> categories;
        for (const auto& pair : dapps_) {
            if (find(categories.begin(), categories.end(), pair.second.category) == categories.end()) {
                categories.push_back(pair.second.category);
            }
        }
        return categories;
    }
    
    bool addFavorite(const string& userId, const string& dappId) {
        lock_guard<mutex> lock(mutex_);
        auto& favs = userFavorites_[userId];
        for (const auto& f : favs) {
            if (f.dappId == dappId) return false;
        }
        DAppFavorite fav = {userId, dappId, (uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count()};
        favs.push_back(fav);
        return true;
    }
    
    bool removeFavorite(const string& userId, const string& dappId) {
        lock_guard<mutex> lock(mutex_);
        auto it = userFavorites_.find(userId);
        if (it == userFavorites_.end()) return false;
        auto& favs = it->second;
        for (auto it2 = favs.begin(); it2 != favs.end(); ++it2) {
            if (it2->dappId == dappId) {
                favs.erase(it2);
                return true;
            }
        }
        return false;
    }
    
    vector<DAppInfo> getUserFavorites(const string& userId) {
        lock_guard<mutex> lock(mutex_);
        vector<DAppInfo> result;
        auto it = userFavorites_.find(userId);
        if (it == userFavorites_.end()) return result;
        for (const auto& fav : it->second) {
            auto dappIt = dapps_.find(fav.dappId);
            if (dappIt != dapps_.end()) {
                result.push_back(dappIt->second);
            }
        }
        return result;
    }
    
    bool addToHistory(const string& userId, const string& dappId, const string& url) {
        lock_guard<mutex> lock(mutex_);
        BrowsingHistory hist;
        hist.userId = userId;
        hist.dappId = dappId;
        hist.url = url;
        hist.visitedAt = (uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count();
        userHistory_[userId].push_back(hist);
        return true;
    }
    
    vector<BrowsingHistory> getHistory(const string& userId, int limit = 50) {
        lock_guard<mutex> lock(mutex_);
        auto it = userHistory_.find(userId);
        if (it == userHistory_.end()) return {};
        const auto& history = it->second;
        int start = max(0, (int)history.size() - limit);
        return vector<BrowsingHistory>(history.begin() + start, history.end());
    }
    
    bool clearHistory(const string& userId) {
        lock_guard<mutex> lock(mutex_);
        auto it = userHistory_.find(userId);
        if (it != userHistory_.end()) {
            it->second.clear();
            return true;
        }
        return false;
    }
    
    WalletConnection connectWallet(const string& userId, const string& dappId,
                                  const string& walletAddress, const string& chain,
                                  const vector<string>& permissions) {
        lock_guard<mutex> lock(mutex_);
        WalletConnection conn;
        conn.sessionId = "SESSION-" + to_string((uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count());
        conn.userId = userId;
        conn.dappId = dappId;
        conn.walletAddress = walletAddress;
        conn.chain = chain;
        conn.permissions = permissions;
        conn.connectedAt = (uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count();
        conn.expiresAt = conn.connectedAt + 24 * 60 * 60 * 1000;
        activeConnections_[conn.sessionId] = conn;
        return conn;
    }
    
    bool disconnectWallet(const string& sessionId) {
        lock_guard<mutex> lock(mutex_);
        auto it = activeConnections_.find(sessionId);
        if (it != activeConnections_.end()) {
            activeConnections_.erase(it);
            return true;
        }
        return false;
    }
    
    optional<WalletConnection> getConnection(const string& sessionId) {
        lock_guard<mutex> lock(mutex_);
        auto it = activeConnections_.find(sessionId);
        if (it != activeConnections_.end()) {
            return it->second;
        }
        return nullopt;
    }
    
    string requestTransaction(const string& dappId, const string& from,
                            const string& to, const string& value,
                            const string& data, const string& chainId) {
        lock_guard<mutex> lock(mutex_);
        TransactionRequest req;
        req.id = "TX-" + to_string((uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count());
        req.dappId = dappId;
        req.from = from;
        req.to = to;
        req.value = value;
        req.data = data;
        req.chainId = chainId;
        req.createdAt = (uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count();
        pendingTransactions_[req.id] = req;
        if (transactionCallback_) transactionCallback_(req);
        return req.id;
    }
    
    string requestSignature(const string& dappId, const string& address,
                          const string& message, const string& method) {
        lock_guard<mutex> lock(mutex_);
        SignatureRequest req;
        req.id = "SIG-" + to_string((uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count());
        req.dappId = dappId;
        req.address = address;
        req.message = message;
        req.method = method;
        req.createdAt = (uint64_t)chrono::duration_cast<chrono::milliseconds>(
            chrono::system_clock::now().time_since_epoch()).count();
        pendingSignatures_[req.id] = req;
        if (signatureCallback_) signatureCallback_(req);
        return req.id;
    }
    
    void setTransactionCallback(function<void(const TransactionRequest&)> cb) {
        transactionCallback_ = cb;
    }
    
    void setSignatureCallback(function<void(const SignatureRequest&)> cb) {
        signatureCallback_ = cb;
    }
};

#endif
