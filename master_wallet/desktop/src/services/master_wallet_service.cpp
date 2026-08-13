/**
 * MasterWalletService - C++ Implementation
 * Wallet management delegated to the canonical MasterWallet backend on :8450.
 * No client-side key material, no fabricated balances/transactions/signatures:
 * signing and address derivation are performed by the backend (real secp256k1).
 * Operations that cannot be performed client-side throw fail-closed errors.
 */

#include "master_wallet_service.hpp"
#include "api_client.hpp"

#include <algorithm>
#include <chrono>
#include <cstring>
#include <iomanip>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <random>
#include <sstream>

namespace tiger {
namespace master {

namespace {
std::string toHex(const unsigned char* data, size_t len) {
    std::ostringstream ss;
    ss << std::hex << std::setfill('0');
    for (size_t i = 0; i < len; ++i) ss << std::setw(2) << static_cast<int>(data[i]);
    return ss.str();
}

uint64_t nowMs() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
}
} // namespace

MasterWalletService::MasterWalletService() {
    try { loadChainsFromStorage(); } catch (...) {}
}

MasterWalletService::~MasterWalletService() = default;

// ==================== Wallet Management ====================

WalletData MasterWalletService::generateWallet(const std::string& password, ChainID chainId) {
    if (password.empty()) throw std::runtime_error("Wallet creation requires a password");
    const std::string name = "Wallet-" + std::to_string(nowMs());
    auto body = api::buildJsonObject({
        {"name", name},
        {"password", password},
        {"chain_id", std::to_string(chainId)},
    });
    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet", body);
    } catch (const api::APIException& e) {
        throw std::runtime_error(std::string("Wallet creation failed: ") + e.what());
    }

    WalletData wallet;
    wallet.id = api::jsonStringField(resp, "id").value_or("");
    wallet.address = api::jsonStringField(resp, "address").value_or("");
    wallet.publicKey = api::jsonStringField(resp, "public_key")
                          .value_or(api::jsonStringField(resp, "publicKey").value_or(""));
    wallet.createdAt = static_cast<uint64_t>(
        api::jsonNumberField(resp, "created_at").value_or(
            api::jsonNumberField(resp, "createdAt").value_or(0)));
    wallet.isActive = true;
    wallet.supportedChains = {chainId};

    auto mnemonic = api::jsonStringField(resp, "mnemonic");
    if (mnemonic) wallet.encryptedMnemonic = encryptData(*mnemonic, password);

    if (wallet.id.empty() || wallet.address.empty())
        throw std::runtime_error("Backend returned an incomplete wallet object");

    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    wallets_[wallet.id] = wallet;
    return wallet;
}

std::optional<WalletData> MasterWalletService::getWallet(const WalletID& walletId) const {
    {
        std::shared_lock<std::shared_mutex> lock(walletMutex_);
        auto it = wallets_.find(walletId);
        if (it != wallets_.end()) return it->second;
    }
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet/" + walletId);
    } catch (const api::APIException&) { return std::nullopt; }

    WalletData wallet;
    wallet.id = api::jsonStringField(resp, "id").value_or("");
    wallet.address = api::jsonStringField(resp, "address").value_or("");
    wallet.publicKey = api::jsonStringField(resp, "public_key")
                          .value_or(api::jsonStringField(resp, "publicKey").value_or(""));
    wallet.createdAt = static_cast<uint64_t>(api::jsonNumberField(resp, "created_at").value_or(0));
    wallet.isActive = true;
    if (wallet.id.empty()) return std::nullopt;

    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    wallets_[wallet.id] = wallet;
    return wallet;
}

std::vector<WalletData> MasterWalletService::getAllWallets() const {
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet");
    } catch (const api::APIException&) {
        std::shared_lock<std::shared_mutex> lock(walletMutex_);
        std::vector<WalletData> out;
        for (const auto& [id, w] : wallets_) out.push_back(w);
        return out;
    }

    std::vector<WalletData> result;
    auto items = api::jsonArrayOfObjects(resp, "wallets");
    if (items.empty()) items = api::jsonArrayOfObjects(resp, "data");
    for (const auto& obj : items) {
        WalletData w;
        w.id = api::jsonStringField(obj, "id").value_or("");
        w.address = api::jsonStringField(obj, "address").value_or("");
        w.publicKey = api::jsonStringField(obj, "public_key")
                          .value_or(api::jsonStringField(obj, "publicKey").value_or(""));
        w.createdAt = static_cast<uint64_t>(api::jsonNumberField(obj, "created_at").value_or(0));
        w.isActive = true;
        if (!w.id.empty()) result.push_back(w);
    }

    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    for (const auto& w : result) wallets_[w.id] = w;
    return result;
}

bool MasterWalletService::deleteWallet(const WalletID& walletId) {
    try {
        api::backendDelete("/api/v1/master-wallet/" + walletId);
    } catch (const api::APIException&) { return false; }
    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    wallets_.erase(walletId);
    return true;
}

bool MasterWalletService::importWallet(const std::string& mnemonic, const std::string& password) {
    if (mnemonic.empty() || password.empty())
        throw std::runtime_error("Mnemonic and password are required");
    // The canonical backend has no dedicated /import route. Importing an
    // existing mnemonic is done via POST /api/v1/master-wallet (create) with
    // the optional "mnemonic" field set: the backend validates the BIP-39
    // checksum, derives the secp256k1 key, and persists the encrypted seed.
    const std::string name = "Imported-" + std::to_string(nowMs());
    auto body = api::buildJsonObject({
        {"name", name},
        {"mnemonic", mnemonic},
        {"password", password},
        {"chain_id", std::to_string(CHAIN_ETHEREUM)},
    });
    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet", body);
    } catch (const api::APIException& e) {
        throw std::runtime_error(std::string("Wallet import failed: ") + e.what());
    }
    WalletData wallet;
    // The create endpoint returns "wallet_id"; older callers used "id".
    wallet.id = api::jsonStringField(resp, "wallet_id")
                    .value_or(api::jsonStringField(resp, "id").value_or(""));
    wallet.address = api::jsonStringField(resp, "address").value_or("");
    wallet.encryptedMnemonic = encryptData(mnemonic, password);
    wallet.createdAt = nowMs();
    wallet.isActive = true;
    wallet.supportedChains = {CHAIN_ETHEREUM};
    if (wallet.id.empty()) return false;

    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    wallets_[wallet.id] = wallet;
    return true;
}

// ==================== Chain Management ====================

void MasterWalletService::addChain(const ChainConfig& config) {
    std::lock_guard<std::shared_mutex> lock(chainMutex_);
    chains_[config.id] = config;
}

void MasterWalletService::removeChain(ChainID chainId) {
    std::lock_guard<std::shared_mutex> lock(chainMutex_);
    chains_.erase(chainId);
}

std::optional<ChainConfig> MasterWalletService::getChainConfig(ChainID chainId) const {
    {
        std::shared_lock<std::shared_mutex> lock(chainMutex_);
        auto it = chains_.find(chainId);
        if (it != chains_.end()) return it->second;
    }
    const_cast<MasterWalletService*>(this)->loadChainsFromStorage();
    std::shared_lock<std::shared_mutex> lock(chainMutex_);
    auto it = chains_.find(chainId);
    if (it != chains_.end()) return it->second;
    return std::nullopt;
}

std::vector<ChainConfig> MasterWalletService::getAllChains() const {
    {
        std::shared_lock<std::shared_mutex> lock(chainMutex_);
        if (!chains_.empty()) {
            std::vector<ChainConfig> out;
            for (const auto& [id, c] : chains_) out.push_back(c);
            return out;
        }
    }
    const_cast<MasterWalletService*>(this)->loadChainsFromStorage();
    std::shared_lock<std::shared_mutex> lock(chainMutex_);
    std::vector<ChainConfig> out;
    for (const auto& [id, c] : chains_) out.push_back(c);
    return out;
}

// ==================== Token Management ====================

void MasterWalletService::addToken(const TokenConfig& token) {
    std::lock_guard<std::shared_mutex> lock(tokenMutex_);
    tokens_[{token.address, token.chainId}] = token;
}

void MasterWalletService::removeToken(const TokenAddress& token, ChainID chainId) {
    std::lock_guard<std::shared_mutex> lock(tokenMutex_);
    tokens_.erase({token, chainId});
}

std::optional<TokenConfig> MasterWalletService::getToken(const TokenAddress& token, ChainID chainId) const {
    std::shared_lock<std::shared_mutex> lock(tokenMutex_);
    auto it = tokens_.find({token, chainId});
    if (it != tokens_.end()) return it->second;
    return std::nullopt;
}

std::vector<TokenConfig> MasterWalletService::getAllTokens() const {
    std::shared_lock<std::shared_mutex> lock(tokenMutex_);
    std::vector<TokenConfig> out;
    for (const auto& [k, t] : tokens_) out.push_back(t);
    return out;
}

// ==================== Balance Operations ====================

BalanceResult MasterWalletService::getBalance(const WalletID& walletId, ChainID chainId, const TokenAddress& token) {
    BalanceResult result;
    result.symbol = "ETH";
    result.decimals = 18;

    std::string cacheKey = walletId + "_" + std::to_string(chainId) + "_" + token;
    {
        std::shared_lock<std::shared_mutex> lock(cacheMutex_);
        auto it = balanceCache_.find(cacheKey);
        if (it != balanceCache_.end() && nowMs() - it->second.second < cacheTTLMs_) return it->second.first;
    }

    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet/" + walletId + "/balance");
    } catch (const api::APIException& e) {
        result.success = false;
        result.error = e.what();
        return result;
    }

    auto bal = api::jsonStringField(resp, "balance");
    if (!bal) {
        auto native = api::jsonStringField(resp, "native");
        if (native) bal = api::jsonStringField(*native, "balance");
    }
    if (!bal) {
        result.success = false;
        result.error = "Backend balance response missing 'balance' field";
        return result;
    }
    result.balance = *bal;
    auto sym = api::jsonStringField(resp, "symbol");
    if (sym) result.symbol = *sym;
    auto dec = api::jsonNumberField(resp, "decimals");
    if (dec) result.decimals = static_cast<uint8_t>(*dec);
    result.success = true;

    {
        std::lock_guard<std::shared_mutex> lock(cacheMutex_);
        balanceCache_[cacheKey] = {result, nowMs()};
    }
    return result;
}

std::map<std::string, BalanceResult> MasterWalletService::getAllBalances(const WalletID& walletId) {
    std::map<std::string, BalanceResult> results;
    for (const auto& chain : getAllChains())
        results[std::to_string(chain.id)] = getBalance(walletId, chain.id, "");
    return results;
}

// ==================== Transaction Operations ====================

// POST /api/v1/master-wallet/:id/transactions
// Creates a (pending) transaction RECORD on the backend. Distinct from
// signAndBroadcast, which signs+broadcasts a ready-to-send tx via /sign.
TransactionResult MasterWalletService::createTransaction(const TransactionRequest& request) {
    TransactionResult result;
    result.timestamp = nowMs();

    if (request.fromWallet.empty() || request.toAddress.empty()) {
        result.success = false;
        result.error = "Missing from/to";
        return result;
    }

    auto body = api::buildJsonObject({
        {"to", request.toAddress},
        {"value", request.amount.empty() ? "0" : request.amount},
        {"data", request.data},
        {"chain_id", std::to_string(request.chainId)},
    });
    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + request.fromWallet + "/transactions", body);
    } catch (const api::APIException& e) {
        result.success = false;
        result.error = e.what();
        return result;
    }

    // The created record is identified by its id (and optional hash once mined).
    auto txHash = api::jsonStringField(resp, "hash");
    if (!txHash) txHash = api::jsonStringField(resp, "transaction_hash");
    if (!txHash) txHash = api::jsonStringField(resp, "tx_hash");
    if (!txHash) txHash = api::jsonStringField(resp, "id");
    if (!txHash) {
        result.success = false;
        result.error = "Backend create-transaction response missing transaction id";
        return result;
    }
    result.txHash = *txHash;
    auto status = api::jsonStringField(resp, "status");
    result.success = status ? (*status == "pending" || *status == "success" || *status == "confirmed") : true;
    if (!result.success) result.error = status.value_or("unknown status");
    return result;
}

TransactionResult MasterWalletService::signAndBroadcast(const TransactionRequest& request) {
    TransactionResult result;
    result.timestamp = nowMs();

    if (request.fromWallet.empty() || request.toAddress.empty() || request.amount.empty()) {
        result.success = false;
        result.error = "Missing from/to/amount";
        return result;
    }

    auto body = api::buildJsonObject({
        {"to", request.toAddress},
        {"amount", request.amount},
        {"token", request.token.empty() ? "" : request.token},
    });
    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + request.fromWallet + "/sign", body);
    } catch (const api::APIException& e) {
        result.success = false;
        result.error = e.what();
        return result;
    }

    auto txHash = api::jsonStringField(resp, "transaction_hash");
    if (!txHash) txHash = api::jsonStringField(resp, "tx_hash");
    if (!txHash) {
        result.success = false;
        result.error = "Backend sign response missing transaction hash";
        return result;
    }
    result.txHash = *txHash;
    auto status = api::jsonStringField(resp, "status");
    result.success = status ? (*status == "success" || *status == "pending" || *status == "confirmed") : true;
    if (!result.success) result.error = status.value_or("unknown status");
    return result;
}

// POST /api/v1/master-wallet/:id/transactions/:tid/approve
TransactionResult MasterWalletService::approveTransaction(const WalletID& masterId, const std::string& txId) {
    TransactionResult result;
    result.timestamp = nowMs();

    if (masterId.empty() || txId.empty()) {
        result.success = false;
        result.error = "Missing masterId/txId";
        return result;
    }

    auto body = api::buildJsonObject({});
    std::string resp;
    try {
        resp = api::backendPost(
            "/api/v1/master-wallet/" + masterId + "/transactions/" + txId + "/approve", body);
    } catch (const api::APIException& e) {
        result.success = false;
        result.error = e.what();
        return result;
    }

    auto hash = api::jsonStringField(resp, "hash");
    if (!hash) hash = api::jsonStringField(resp, "transaction_hash");
    if (!hash) hash = api::jsonStringField(resp, "tx_hash");
    if (!hash) hash = api::jsonStringField(resp, "id");
    if (hash) result.txHash = *hash;
    auto status = api::jsonStringField(resp, "status");
    result.success = status ? (*status == "approved" || *status == "success" || *status == "pending" || *status == "confirmed") : true;
    if (!result.success) result.error = status.value_or("unknown status");
    return result;
}

// POST /api/v1/master-wallet/:id/transactions/:tid/reject
TransactionResult MasterWalletService::rejectTransaction(const WalletID& masterId, const std::string& txId) {
    TransactionResult result;
    result.timestamp = nowMs();

    if (masterId.empty() || txId.empty()) {
        result.success = false;
        result.error = "Missing masterId/txId";
        return result;
    }

    auto body = api::buildJsonObject({});
    std::string resp;
    try {
        resp = api::backendPost(
            "/api/v1/master-wallet/" + masterId + "/transactions/" + txId + "/reject", body);
    } catch (const api::APIException& e) {
        result.success = false;
        result.error = e.what();
        return result;
    }

    auto hash = api::jsonStringField(resp, "hash");
    if (!hash) hash = api::jsonStringField(resp, "transaction_hash");
    if (!hash) hash = api::jsonStringField(resp, "tx_hash");
    if (!hash) hash = api::jsonStringField(resp, "id");
    if (hash) result.txHash = *hash;
    // A rejected transaction is a successful rejection operation, not a failed tx.
    auto status = api::jsonStringField(resp, "status");
    result.success = status ? (*status == "rejected" || *status == "success") : true;
    if (!result.success) result.error = status.value_or("unknown status");
    return result;
}

std::string MasterWalletService::signMessage(const WalletID&, const std::string&) {
    throw std::runtime_error("Arbitrary message signing is not available: it must be performed by the backend, which does not expose this operation");
}

bool MasterWalletService::verifySignature(const std::string&, const std::string&, const std::string&) {
    throw std::runtime_error("Signature verification is not available client-side; it must be performed by the backend");
}

// ==================== Real Backend Queries ====================

std::vector<TransactionRecord> MasterWalletService::getTransactions(const WalletID& walletId) {
    std::vector<TransactionRecord> out;
    if (walletId.empty()) return out;
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet/" + walletId + "/transactions");
    } catch (const api::APIException&) { return out; }
    auto items = api::jsonArrayOfObjects(resp, "transactions");
    if (items.empty()) items = api::jsonArrayOfObjects(resp, "data");
    for (const auto& obj : items) {
        TransactionRecord r;
        r.hash = api::jsonStringField(obj, "hash").value_or(api::jsonStringField(obj, "transaction_hash").value_or(""));
        r.from = api::jsonStringField(obj, "from").value_or("");
        r.to = api::jsonStringField(obj, "to").value_or("");
        r.amount = api::jsonStringField(obj, "amount").value_or("");
        r.token = api::jsonStringField(obj, "token").value_or("");
        r.status = api::jsonStringField(obj, "status").value_or("");
        r.timestamp = api::jsonStringField(obj, "timestamp").value_or(api::jsonStringField(obj, "time").value_or(""));
        r.blockNumber = api::jsonStringField(obj, "block_number").value_or("");
        if (!r.hash.empty()) out.push_back(r);
    }
    return out;
}

GasEstimate MasterWalletService::getGas(ChainID chainId) {
    GasEstimate g{};
    std::map<std::string, std::string> params = {{"chain_id", std::to_string(chainId)}};
    try {
        auto resp = api::backendGet("/api/v1/gas",
            std::optional<std::map<std::string, std::string>>(params));
        g.gasPrice = api::jsonStringField(resp, "gas_price").value_or("");
        g.maxFee = api::jsonStringField(resp, "max_fee").value_or("");
        g.priorityFee = api::jsonStringField(resp, "priority_fee").value_or("");
        if (g.gasPrice.empty() && g.maxFee.empty() && g.priorityFee.empty()) {
            g.success = false;
            g.error = "Backend gas response missing fields";
        } else {
            g.success = true;
        }
    } catch (const api::APIException& e) {
        g.success = false;
        g.error = e.what();
    }
    return g;
}

PriceQuote MasterWalletService::getPrice(const std::string& coinId) {
    PriceQuote p{};
    if (coinId.empty()) { p.success = false; p.error = "coin_id required"; return p; }
    std::map<std::string, std::string> params = {{"coin_id", coinId}};
    try {
        auto resp = api::backendGet("/api/v1/price",
            std::optional<std::map<std::string, std::string>>(params));
        auto usd = api::jsonNumberField(resp, "usd");
        if (!usd) { p.success = false; p.error = "Backend price response missing usd"; return p; }
        p.usd = *usd;
        p.usd24hChange = api::jsonNumberField(resp, "usd_24h_change").value_or(0.0);
        p.success = true;
    } catch (const api::APIException& e) {
        p.success = false;
        p.error = e.what();
    }
    return p;
}

std::vector<ChainConfig> MasterWalletService::fetchChains() {
    loadChainsFromStorage();
    return getAllChains();
}

// ==================== Sub-wallets ====================

std::string MasterWalletService::getSubWallets(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/sub-wallets");
}

BalanceResult MasterWalletService::getSubWalletBalance(const WalletID& walletId, const std::string& subId) {
    BalanceResult result{};
    try {
        auto resp = api::backendGet("/api/v1/master-wallet/" + walletId + "/sub-wallets/" + subId + "/balance");
        auto bal = api::jsonStringField(resp, "balance");
        if (!bal) { result.success = false; result.error = "missing balance"; return result; }
        result.balance = *bal;
        result.symbol = api::jsonStringField(resp, "symbol").value_or("ETH");
        result.decimals = static_cast<uint8_t>(api::jsonNumberField(resp, "decimals").value_or(18));
        result.success = true;
    } catch (const api::APIException& e) { result.success = false; result.error = e.what(); }
    return result;
}

TransactionResult MasterWalletService::transferSubWallet(const WalletID& walletId, const std::string& subId,
                                                         const std::string& to, const std::string& amount,
                                                         const std::string& password, const std::string& token) {
    TransactionResult r{};
    r.timestamp = nowMs();
    auto body = api::buildJsonObject({{"to", to}, {"amount", amount}, {"password", password},
                                      {"token", token.empty() ? "" : token}});
    try {
        auto resp = api::backendPost("/api/v1/master-wallet/" + walletId + "/sub-wallets/" + subId + "/transfer", body);
        r.txHash = api::jsonStringField(resp, "transaction_hash").value_or(api::jsonStringField(resp, "tx_hash").value_or(""));
        r.success = !r.txHash.empty();
        if (!r.success) r.error = "Backend transfer response missing transaction hash";
    } catch (const api::APIException& e) { r.success = false; r.error = e.what(); }
    return r;
}

// ==================== Policies / Fees / Auto-sign / Users ====================

std::string MasterWalletService::getPolicies(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/policies");
}
std::string MasterWalletService::createPolicy(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/policies", body);
}
std::string MasterWalletService::updatePolicy(const WalletID& walletId, const std::string& policyId, const std::string& body) {
    return api::backendPut("/api/v1/master-wallet/" + walletId + "/policies/" + policyId, body);
}
bool MasterWalletService::deletePolicy(const WalletID& walletId, const std::string& policyId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/policies/" + policyId); return true; }
    catch (const api::APIException&) { return false; }
}

std::string MasterWalletService::getFees(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/fees");
}
std::string MasterWalletService::createFee(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/fees", body);
}
bool MasterWalletService::deleteFee(const WalletID& walletId, const std::string& feeId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/fees/" + feeId); return true; }
    catch (const api::APIException&) { return false; }
}

std::string MasterWalletService::getAutoSignRules(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/auto-sign");
}
std::string MasterWalletService::createAutoSignRule(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/auto-sign", body);
}
bool MasterWalletService::deleteAutoSignRule(const WalletID& walletId, const std::string& ruleId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/auto-sign/" + ruleId); return true; }
    catch (const api::APIException&) { return false; }
}

std::string MasterWalletService::getUsers(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/users");
}
std::string MasterWalletService::createUser(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/users", body);
}
bool MasterWalletService::deleteUser(const WalletID& walletId, const std::string& userId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/users/" + userId); return true; }
    catch (const api::APIException&) { return false; }
}

// ==================== Audit + Analytics ====================

std::string MasterWalletService::getAudit(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/audit");
}
std::string MasterWalletService::getAnalyticsVolume(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/analytics/volume");
}
std::string MasterWalletService::getAnalyticsTransactions(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/analytics/transactions");
}
std::string MasterWalletService::getAnalyticsWallets(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/analytics/wallets");
}

// ==================== Notifications + Webhooks ====================

std::string MasterWalletService::getNotifications(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/notifications");
}
std::string MasterWalletService::createNotification(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/notifications", body);
}
std::string MasterWalletService::getWebhooks(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/webhooks");
}
std::string MasterWalletService::createWebhook(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/webhooks", body);
}
bool MasterWalletService::deleteWebhook(const WalletID& walletId, const std::string& webhookId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/webhooks/" + webhookId); return true; }
    catch (const api::APIException&) { return false; }
}

// ==================== Treasury ====================

std::string MasterWalletService::getTreasury(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/treasury");
}
std::string MasterWalletService::getTreasuryTransactions(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/treasury/transactions");
}
TransactionResult MasterWalletService::treasuryTransfer(const WalletID& walletId, const std::string& to,
                                                         const std::string& amount, const std::string& password) {
    TransactionResult r{};
    r.timestamp = nowMs();
    auto body = api::buildJsonObject({{"to", to}, {"amount", amount}, {"password", password}});
    try {
        auto resp = api::backendPost("/api/v1/master-wallet/" + walletId + "/treasury/transfer", body);
        r.txHash = api::jsonStringField(resp, "transaction_hash").value_or(api::jsonStringField(resp, "tx_hash").value_or(""));
        r.success = !r.txHash.empty();
        if (!r.success) r.error = "Backend treasury transfer response missing transaction hash";
    } catch (const api::APIException& e) { r.success = false; r.error = e.what(); }
    return r;
}
TransactionResult MasterWalletService::treasurySweep(const WalletID& walletId, const std::string& to,
                                                      const std::string& password) {
    TransactionResult r{};
    r.timestamp = nowMs();
    auto body = api::buildJsonObject({{"to", to}, {"password", password}});
    try {
        auto resp = api::backendPost("/api/v1/master-wallet/" + walletId + "/treasury/sweep", body);
        r.txHash = api::jsonStringField(resp, "transaction_hash").value_or(api::jsonStringField(resp, "tx_hash").value_or(""));
        r.success = !r.txHash.empty();
        if (!r.success) r.error = "Backend treasury sweep response missing transaction hash";
    } catch (const api::APIException& e) { r.success = false; r.error = e.what(); }
    return r;
}

// ==================== Multisig ====================

std::string MasterWalletService::getMultisigWallets(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/multisig/wallets");
}
std::string MasterWalletService::createMultisigWallet(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/multisig/wallets", body);
}
std::string MasterWalletService::getMultisigTransactions(const WalletID& walletId, const std::string& multisigId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/multisig/wallets/" + multisigId + "/transactions");
}
std::string MasterWalletService::createMultisigTransaction(const WalletID& walletId, const std::string& multisigId,
                                                           const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/multisig/wallets/" + multisigId + "/transactions", body);
}
std::string MasterWalletService::signMultisigTransaction(const WalletID& walletId, const std::string& txId,
                                                         const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/multisig/transactions/" + txId + "/sign", body);
}
std::string MasterWalletService::executeMultisigTransaction(const WalletID& walletId, const std::string& txId,
                                                            const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/multisig/transactions/" + txId + "/execute", body);
}

// ==================== Public endpoint helpers ====================

std::string MasterWalletService::getTransactionHistory(const std::string& address, ChainID chainId) {
    std::map<std::string, std::string> params = {{"address", address}, {"chain_id", std::to_string(chainId)}};
    return api::backendGet("/api/v1/transactions/history",
        std::optional<std::map<std::string, std::string>>(params));
}

// ==================== HD Wallet Operations ====================

std::string MasterWalletService::deriveAddress(const WalletID&, ChainID, uint32_t) {
    throw std::runtime_error("HD address derivation is not available client-side; it must be performed by the backend");
}

std::string MasterWalletService::derivePublicKey(const WalletID&, ChainID, uint32_t) {
    throw std::runtime_error("HD public key derivation is not available client-side; it must be performed by the backend");
}

// ==================== User Wallet Management ====================

void MasterWalletService::registerUserWallet(const UserID& userId, const WalletID& walletId) {
    try {
        api::backendPost("/api/v1/master-wallet/" + walletId + "/users",
                         api::buildJsonObject({{"user_id", userId}}));
    } catch (const api::APIException&) {}
    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    userWallets_[userId] = walletId;
    walletUsers_[walletId].insert(userId);
}

void MasterWalletService::unregisterUserWallet(const UserID& userId) {
    WalletID walletId;
    {
        std::shared_lock<std::shared_mutex> lock(walletMutex_);
        auto it = userWallets_.find(userId);
        if (it == userWallets_.end()) return;
        walletId = it->second;
    }
    try {
        api::backendDelete("/api/v1/master-wallet/" + walletId + "/users/" + userId);
    } catch (const api::APIException&) {}
    std::lock_guard<std::shared_mutex> lock(walletMutex_);
    userWallets_.erase(userId);
    walletUsers_[walletId].erase(userId);
}

std::optional<WalletID> MasterWalletService::getUserWallet(const UserID& userId) const {
    std::shared_lock<std::shared_mutex> lock(walletMutex_);
    auto it = userWallets_.find(userId);
    if (it != userWallets_.end()) return it->second;
    return std::nullopt;
}

std::vector<UserID> MasterWalletService::getUsersForWallet(const WalletID& walletId) const {
    std::vector<UserID> result;
    try {
        auto resp = api::backendGet("/api/v1/master-wallet/" + walletId + "/users");
        auto items = api::jsonArrayOfObjects(resp, "users");
        if (items.empty()) items = api::jsonArrayOfObjects(resp, "data");
        for (const auto& obj : items) {
            auto id = api::jsonStringField(obj, "id");
            if (!id) id = api::jsonStringField(obj, "user_id");
            if (id) result.push_back(*id);
        }
        return result;
    } catch (const api::APIException&) {
        std::shared_lock<std::shared_mutex> lock(walletMutex_);
        auto it = walletUsers_.find(walletId);
        if (it != walletUsers_.end()) result.assign(it->second.begin(), it->second.end());
        return result;
    }
}

// ==================== Auto-sign Configuration ====================

void MasterWalletService::setAutoSignEnabled(bool enabled) { autoSignEnabled_ = enabled; }
bool MasterWalletService::isAutoSignEnabled() const { return autoSignEnabled_; }
void MasterWalletService::setAutoSignLimit(uint64_t limitInWei) { autoSignLimit_ = limitInWei; }
uint64_t MasterWalletService::getAutoSignLimit() const { return autoSignLimit_; }

// ==================== Cache Management ====================

void MasterWalletService::clearCache() {
    std::lock_guard<std::shared_mutex> lock(cacheMutex_);
    balanceCache_.clear();
}

void MasterWalletService::setCacheTTL(uint64_t ttlMs) { cacheTTLMs_ = ttlMs; }

// ==================== Encryption (AES-256-GCM, local mnemonic protection) ===

std::string MasterWalletService::encryptData(const std::string& data, const std::string& key) {
    if (key.empty()) throw std::runtime_error("Encryption key required");

    unsigned char iv[12];
    if (RAND_bytes(iv, sizeof(iv)) != 1) throw std::runtime_error("RAND_bytes failed");

    unsigned char keyBytes[32];
    EVP_MD_CTX* md = EVP_MD_CTX_new();
    EVP_DigestInit_ex(md, EVP_sha256(), nullptr);
    EVP_DigestUpdate(md, key.data(), key.size());
    unsigned int outLen = 0;
    EVP_DigestFinal_ex(md, keyBytes, &outLen);
    EVP_MD_CTX_free(md);

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) throw std::runtime_error("EVP_CIPHER_CTX_new failed");

    std::string ciphertext(data.size(), '\0');
    std::string tag(16, '\0');
    int len = 0;
    EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, sizeof(iv), nullptr);
    EVP_EncryptInit_ex(ctx, nullptr, nullptr, keyBytes, iv);
    if (!data.empty()) {
        EVP_EncryptUpdate(ctx, reinterpret_cast<unsigned char*>(&ciphertext[0]), &len,
                          reinterpret_cast<const unsigned char*>(data.data()),
                          static_cast<int>(data.size()));
    }
    int finalLen = 0;
    EVP_EncryptFinal_ex(ctx, reinterpret_cast<unsigned char*>(&ciphertext[0]) + len, &finalLen);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, &tag[0]);
    EVP_CIPHER_CTX_free(ctx);
    OPENSSL_cleanse(keyBytes, sizeof(keyBytes));

    std::string result;
    result.append(reinterpret_cast<char*>(iv), sizeof(iv));
    result.append(ciphertext);
    result.append(tag);
    return result;
}

std::string MasterWalletService::decryptData(const std::string& encryptedData, const std::string& key) {
    if (key.empty() || encryptedData.size() < 12 + 16) return "";

    unsigned char keyBytes[32];
    EVP_MD_CTX* md = EVP_MD_CTX_new();
    EVP_DigestInit_ex(md, EVP_sha256(), nullptr);
    EVP_DigestUpdate(md, key.data(), key.size());
    unsigned int outLen = 0;
    EVP_DigestFinal_ex(md, keyBytes, &outLen);
    EVP_MD_CTX_free(md);

    const unsigned char* iv = reinterpret_cast<const unsigned char*>(encryptedData.data());
    size_t tagOffset = encryptedData.size() - 16;
    const unsigned char* tag = reinterpret_cast<const unsigned char*>(encryptedData.data() + tagOffset);
    const unsigned char* ct = reinterpret_cast<const unsigned char*>(encryptedData.data() + 12);
    size_t ctLen = tagOffset - 12;

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, 12, nullptr);
    EVP_DecryptInit_ex(ctx, nullptr, nullptr, keyBytes, iv);

    std::string plaintext(ctLen, '\0');
    int len = 0;
    if (ctLen) {
        EVP_DecryptUpdate(ctx, reinterpret_cast<unsigned char*>(&plaintext[0]), &len, ct,
                          static_cast<int>(ctLen));
    }
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16, const_cast<unsigned char*>(tag));
    int finalLen = 0;
    int ok = EVP_DecryptFinal_ex(ctx, reinterpret_cast<unsigned char*>(&plaintext[0]) + len, &finalLen);
    EVP_CIPHER_CTX_free(ctx);
    OPENSSL_cleanse(keyBytes, sizeof(keyBytes));
    if (ok <= 0) return "";
    plaintext.resize(len + finalLen);
    return plaintext;
}

// ==================== Status ====================

bool MasterWalletService::isHealthy() const { return isHealthy_.load(); }
std::string MasterWalletService::getVersion() const { return version_; }

// ==================== Private Methods ====================

std::string MasterWalletService::generateWalletId() {
    // Wallet ids are issued by the canonical backend on POST /api/v1/master-wallet.
    // Do not fabricate a client-side id that looks like an Ethereum address.
    throw std::runtime_error(
        "Wallet IDs are issued by the backend; client-side generation is not "
        "supported");
}

std::string MasterWalletService::hashPrivateKey(const std::string& privateKey) {
    unsigned char hash[32];
    EVP_MD_CTX* md = EVP_MD_CTX_new();
    EVP_DigestInit_ex(md, EVP_sha256(), nullptr);
    EVP_DigestUpdate(md, privateKey.data(), privateKey.size());
    unsigned int outLen = 0;
    EVP_DigestFinal_ex(md, hash, &outLen);
    EVP_MD_CTX_free(md);
    return std::string(reinterpret_cast<char*>(hash), 32);
}

std::string MasterWalletService::computeAddress(const std::string&) {
    throw std::runtime_error("Address computation is not available client-side; the backend returns addresses for wallets it creates");
}

std::string MasterWalletService::computeMnemonicHash(const std::string& mnemonic) {
    unsigned char hash[32];
    EVP_MD_CTX* md = EVP_MD_CTX_new();
    EVP_DigestInit_ex(md, EVP_sha256(), nullptr);
    EVP_DigestUpdate(md, mnemonic.data(), mnemonic.size());
    unsigned int outLen = 0;
    EVP_DigestFinal_ex(md, hash, &outLen);
    EVP_MD_CTX_free(md);
    return std::string(reinterpret_cast<char*>(hash), 32);
}

bool MasterWalletService::loadWalletsFromStorage() { return true; }
bool MasterWalletService::saveWalletsToStorage() { return true; }

bool MasterWalletService::loadChainsFromStorage() {
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/chains");
    } catch (const api::APIException&) { return false; }
    auto items = api::jsonArrayOfObjects(resp, "chains");
    if (items.empty()) items = api::jsonArrayOfObjects(resp, "data");
    std::lock_guard<std::shared_mutex> lock(chainMutex_);
    chains_.clear();
    for (const auto& obj : items) {
        ChainConfig c{};
        auto idStr = api::jsonStringField(obj, "chain_id");
        if (!idStr) idStr = api::jsonStringField(obj, "id");
        if (!idStr) continue;
        try { c.id = std::stoull(*idStr); } catch (...) { continue; }
        c.name = api::jsonStringField(obj, "name").value_or("");
        c.symbol = api::jsonStringField(obj, "symbol").value_or("");
        c.rpcUrl = api::jsonStringField(obj, "rpc_url").value_or(api::jsonStringField(obj, "rpcUrl").value_or(""));
        c.explorerUrl = api::jsonStringField(obj, "explorer_url").value_or(api::jsonStringField(obj, "explorerUrl").value_or(""));
        auto dec = api::jsonNumberField(obj, "decimals");
        c.decimals = dec ? static_cast<uint8_t>(*dec) : 18;
        c.isEVM = api::jsonBoolField(obj, "is_evm").value_or(true);
        chains_[c.id] = c;
    }
    return !chains_.empty();
}

bool MasterWalletService::loadTokensFromStorage() { return true; }

} // namespace master
} // namespace tiger
