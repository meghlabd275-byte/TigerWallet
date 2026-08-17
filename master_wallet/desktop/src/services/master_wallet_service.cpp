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

// ==================== New MasterWallet backend endpoints ====================

namespace {
// Build a JSON object string that only includes provided (non-nullopt) fields.
// Values are emitted as JSON strings/numbers/booleans via buildJsonObject, which
// passes numbers/booleans through unquoted.
std::string buildOptionalObject(
    const std::vector<std::pair<std::string, std::optional<std::string>>>& strFields,
    const std::vector<std::pair<std::string, std::optional<bool>>>& boolFields) {
    std::vector<std::pair<std::string, std::string>> fields;
    for (const auto& [k, v] : strFields) {
        if (v.has_value()) fields.emplace_back(k, *v);
    }
    for (const auto& [k, v] : boolFields) {
        if (v.has_value()) fields.emplace_back(k, *v ? "true" : "false");
    }
    return api::buildJsonObject(fields);
}

// Parse a JSON array of strings (the value following `"key":`). Returns the
// individual unescaped string elements.
std::vector<std::string> jsonArrayOfStrings(const std::string& json, const std::string& key) {
    std::vector<std::string> out;
    std::string needle = "\"" + key + "\"";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) return out;
    size_t arr = json.find('[', pos);
    if (arr == std::string::npos) return out;
    size_t i = arr + 1;
    while (i < json.size()) {
        while (i < json.size() && (json[i] == ' ' || json[i] == '\t' || json[i] == '\n' ||
               json[i] == '\r' || json[i] == ',')) ++i;
        if (i >= json.size() || json[i] == ']') break;
        if (json[i] != '"') break;
        size_t s = i + 1;
        size_t e = s;
        while (e < json.size() && json[e] != '"') {
            if (json[e] == '\\' && e + 1 < json.size()) ++e;
            ++e;
        }
        if (e >= json.size()) break;
        out.push_back(json.substr(s, e - s));
        i = e + 1;
    }
    return out;
}

// Extract the balanced text of a single JSON object value following `"key":`.
// Returns std::nullopt if the key is absent or its value is not an object.
std::optional<std::string> jsonObjectField(const std::string& json, const std::string& key) {
    std::string needle = "\"" + key + "\"";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) return std::nullopt;
    size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t' || json[i] == '\n' ||
           json[i] == '\r' || json[i] == ':')) ++i;
    if (i >= json.size() || json[i] != '{') return std::nullopt;
    size_t start = i;
    int depth = 0;
    bool inString = false;
    bool escape = false;
    for (; i < json.size(); ++i) {
        char c = json[i];
        if (inString) {
            if (escape) { escape = false; }
            else if (c == '\\') { escape = true; }
            else if (c == '"') { inString = false; }
        } else {
            if (c == '"') { inString = true; }
            else if (c == '{') { ++depth; }
            else if (c == '}') {
                --depth;
                if (depth == 0) { ++i; break; }
            }
        }
    }
    if (depth != 0) return std::nullopt;
    return json.substr(start, i - start);
}
} // namespace

// PUT /api/v1/master-wallet/:id
MasterWalletUpdateResult MasterWalletService::updateMasterWallet(
    const WalletID& masterId,
    const std::optional<std::string>& name,
    const std::optional<bool>& isActive,
    const std::optional<std::string>& dailyLimit,
    const std::optional<std::string>& perTransactionLimit,
    const std::optional<std::string>& metadata) {
    MasterWalletUpdateResult r;
    if (masterId.empty()) {
        r.error = "masterId required";
        return r;
    }
    if (!name && !isActive && !dailyLimit && !perTransactionLimit && !metadata) {
        r.error = "At least one field must be provided to update";
        return r;
    }

    std::string body = buildOptionalObject(
        {{"name", name},
         {"daily_limit", dailyLimit},
         {"per_transaction_limit", perTransactionLimit},
         {"metadata", metadata}},
        {{"is_active", isActive}});

    std::string resp;
    try {
        resp = api::backendPut("/api/v1/master-wallet/" + masterId, body);
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }

    r.id = api::jsonStringField(resp, "id").value_or(masterId);
    auto updated = api::jsonBoolField(resp, "updated");
    r.updated = updated.value_or(true);
    r.success = true;
    return r;
}

// GET /api/v1/master-wallet/:id/transactions/:tid
std::string MasterWalletService::getTransaction(const WalletID& masterId, const std::string& txId) {
    if (masterId.empty() || txId.empty()) {
        throw std::runtime_error("getTransaction: masterId and txId are required");
    }
    return api::backendGet("/api/v1/master-wallet/" + masterId + "/transactions/" + txId);
}

// GET /api/v1/master-wallet/:id/multisig/wallets/:wid
MultisigWalletDetail MasterWalletService::getMultisigWalletDetail(const WalletID& masterId,
                                                                  const std::string& walletId) {
    MultisigWalletDetail d;
    if (masterId.empty() || walletId.empty()) {
        d.error = "masterId and walletId are required";
        return d;
    }
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet/" + masterId + "/multisig/wallets/" + walletId);
    } catch (const api::APIException& e) {
        d.error = e.what();
        return d;
    }

    // The backend wraps the detail in {"multisig_wallet": {...}}; fall back to
    // the top-level object if the wrapper is absent.
    std::string obj = resp;
    auto inner = jsonObjectField(resp, "multisig_wallet");
    if (inner) obj = *inner;

    d.id = api::jsonStringField(obj, "id").value_or("");
    d.name = api::jsonStringField(obj, "name").value_or("");
    d.owners = jsonArrayOfStrings(obj, "owners");
    auto threshold = api::jsonNumberField(obj, "threshold");
    if (threshold) d.threshold = static_cast<uint32_t>(*threshold);
    auto chainId = api::jsonNumberField(obj, "chain_id");
    if (!chainId) chainId = api::jsonNumberField(obj, "chainId");
    if (chainId) d.chainId = static_cast<uint64_t>(*chainId);
    d.address = api::jsonStringField(obj, "address").value_or("");
    d.pendingTransactions = api::jsonArrayOfObjects(obj, "pending_transactions");
    d.success = !d.id.empty();
    if (!d.success) d.error = "Backend multisig wallet response missing id";
    return d;
}

// POST /api/v1/master-wallet/:id/passkey/register
PasskeyRegisterResult MasterWalletService::registerPasskey(
    const WalletID& masterId,
    const std::string& credentialId,
    const std::string& publicKey,
    uint32_t signCount,
    const std::vector<std::string>& transports,
    const std::string& label) {
    PasskeyRegisterResult r;
    r.credentialId = credentialId;
    if (masterId.empty() || credentialId.empty() || publicKey.empty()) {
        r.error = "masterId, credentialId and publicKey are required";
        return r;
    }

    // Build transports JSON array string.
    std::string transportsJson = "[";
    for (size_t i = 0; i < transports.size(); ++i) {
        if (i) transportsJson += ",";
        transportsJson += "\"" + api::jsonEscape(transports[i]) + "\"";
    }
    transportsJson += "]";

    // buildJsonObject quotes string values and passes numeric-looking values
    // through; sign_count is a real uint32, so emit it as a bare number by
    // constructing the body manually to embed the array and number correctly.
    std::ostringstream body;
    body << "{"
         << "\"credential_id\":\"" << api::jsonEscape(credentialId) << "\","
         << "\"public_key\":\"" << api::jsonEscape(publicKey) << "\","
         << "\"sign_count\":" << signCount << ","
         << "\"transports\":" << transportsJson << ","
         << "\"label\":\"" << api::jsonEscape(label) << "\""
         << "}";

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + masterId + "/passkey/register", body.str());
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }

    r.passkeyId = api::jsonStringField(resp, "passkey_id").value_or("");
    r.registered = api::jsonBoolField(resp, "registered").value_or(false);
    r.success = true;
    return r;
}

// GET /api/v1/master-wallet/:id/passkey/credentials
PasskeyListResult MasterWalletService::listPasskeys(const WalletID& masterId) {
    PasskeyListResult r;
    if (masterId.empty()) {
        r.error = "masterId required";
        return r;
    }
    std::string resp;
    try {
        resp = api::backendGet("/api/v1/master-wallet/" + masterId + "/passkey/credentials");
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }
    auto items = api::jsonArrayOfObjects(resp, "passkeys");
    if (items.empty()) items = api::jsonArrayOfObjects(resp, "data");
    for (const auto& obj : items) {
        PasskeyCredentialRow row;
        row.id = api::jsonStringField(obj, "id").value_or("");
        row.credentialId = api::jsonStringField(obj, "credential_id")
                               .value_or(api::jsonStringField(obj, "credentialId").value_or(""));
        auto sc = api::jsonNumberField(obj, "sign_count");
        if (!sc) sc = api::jsonNumberField(obj, "signCount");
        if (sc) row.signCount = static_cast<uint32_t>(*sc);
        row.transports = jsonArrayOfStrings(obj, "transports");
        row.label = api::jsonStringField(obj, "label").value_or("");
        row.createdAt = api::jsonStringField(obj, "created_at")
                            .value_or(api::jsonStringField(obj, "createdAt").value_or(""));
        row.updatedAt = api::jsonStringField(obj, "updated_at")
                            .value_or(api::jsonStringField(obj, "updatedAt").value_or(""));
        if (!row.id.empty() || !row.credentialId.empty()) r.passkeys.push_back(row);
    }
    r.success = true;
    return r;
}

// DELETE /api/v1/master-wallet/:id/passkey/credentials/:credId
bool MasterWalletService::deletePasskey(const WalletID& masterId, const std::string& credId) {
    if (masterId.empty() || credId.empty()) return false;
    try {
        api::backendDelete("/api/v1/master-wallet/" + masterId + "/passkey/credentials/" + credId);
        return true;
    } catch (const api::APIException&) {
        return false;
    }
}

// POST /api/v1/master-wallet/:id/passkey/verify-assertion
PasskeyVerifyResult MasterWalletService::verifyPasskeyAssertion(
    const WalletID& masterId,
    const std::string& credentialId,
    const std::string& authenticatorData,
    const std::string& clientDataJson,
    const std::string& signature) {
    PasskeyVerifyResult r;
    r.credentialId = credentialId;
    if (masterId.empty() || credentialId.empty() || authenticatorData.empty() ||
        clientDataJson.empty() || signature.empty()) {
        r.error = "All assertion fields are required";
        return r;
    }

    auto body = api::buildJsonObject({
        {"credential_id", credentialId},
        {"authenticator_data", authenticatorData},
        {"client_data_json", clientDataJson},
        {"signature", signature},
    });

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + masterId + "/passkey/verify-assertion", body);
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }

    auto verified = api::jsonBoolField(resp, "verified");
    if (!verified) {
        // Backend call succeeded but the response did not carry a verified flag.
        // Fail closed: never assume verification.
        r.success = true;
        r.verified = false;
        r.error = "Backend response missing 'verified' field";
        return r;
    }
    r.verified = *verified;
    r.success = true;
    auto cred = api::jsonStringField(resp, "credential_id");
    if (cred) r.credentialId = *cred;
    return r;
}


// ==================== Sub-wallets ====================

std::string MasterWalletService::getSubWallets(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/sub-wallets");
}

// POST /api/v1/master-wallet/:id/sub-wallets — create a sub-wallet.
// body is a JSON object string ({name, password, chain_id, ...}); returned
// verbatim from the backend (raw JSON). No client-side fabrication.
std::string MasterWalletService::createSubWallet(const WalletID& masterId, const std::string& body) {
    if (masterId.empty()) {
        throw std::runtime_error("createSubWallet: masterId is required");
    }
    if (body.empty()) {
        throw std::runtime_error("createSubWallet: body is required");
    }
    return api::backendPost("/api/v1/master-wallet/" + masterId + "/sub-wallets", body);
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

// ==================== Two-party revenue gate ====================

// POST /api/v1/master-wallet/:id/withdrawal-request
WithdrawalRequestResult MasterWalletService::requestWithdrawal(const WalletID& masterId,
                                                               const std::string& toAddress,
                                                               const std::string& amountWei,
                                                               const std::string& currency,
                                                               ChainID chainId) {
    WithdrawalRequestResult r;
    if (masterId.empty() || toAddress.empty() || amountWei.empty()) {
        r.error = "masterId, toAddress and amountWei are required";
        return r;
    }
    // Build the body manually so optional fields (currency/chain_id) are omitted
    // when not provided, keeping the JSON clean for the backend.
    std::ostringstream body;
    body << "{"
         << "\"to_address\":\"" << api::jsonEscape(toAddress) << "\","
         << "\"amount_wei\":\"" << api::jsonEscape(amountWei) << "\"";
    if (!currency.empty()) {
        body << ",\"currency\":\"" << api::jsonEscape(currency) << "\"";
    }
    if (chainId != 0) {
        body << ",\"chain_id\":" << chainId;
    }
    body << "}";

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + masterId + "/withdrawal-request", body.str());
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }

    auto withdrawalId = api::jsonStringField(resp, "withdrawal_id");
    if (!withdrawalId) withdrawalId = api::jsonStringField(resp, "withdrawalId");
    if (!withdrawalId) {
        r.error = "Backend withdrawal-request response missing withdrawal_id";
        return r;
    }
    r.withdrawalId = *withdrawalId;
    r.status = api::jsonStringField(resp, "status").value_or("");
    r.success = true;
    return r;
}

// POST /api/v1/master-wallet/:id/revenue-payout
RevenuePayoutResult MasterWalletService::revenuePayout(const WalletID& masterId,
                                                       const std::string& to,
                                                       const std::string& amount,
                                                       const std::string& password,
                                                       uint64_t gasLimit,
                                                       const std::string& withdrawalId) {
    RevenuePayoutResult r;
    if (masterId.empty() || to.empty() || amount.empty() || password.empty() || withdrawalId.empty()) {
        r.error = "masterId, to, amount, password and withdrawalId are required";
        return r;
    }
    // gas_limit is optional; omit it unless the caller provided a non-zero value.
    std::ostringstream body;
    body << "{"
         << "\"to\":\"" << api::jsonEscape(to) << "\","
         << "\"amount\":\"" << api::jsonEscape(amount) << "\","
         << "\"password\":\"" << api::jsonEscape(password) << "\","
         << "\"withdrawal_id\":\"" << api::jsonEscape(withdrawalId) << "\"";
    if (gasLimit != 0) {
        body << ",\"gas_limit\":" << gasLimit;
    }
    body << "}";

    std::string resp;
    try {
        resp = api::backendPost("/api/v1/master-wallet/" + masterId + "/revenue-payout", body.str());
    } catch (const api::APIException& e) {
        r.error = e.what();
        return r;
    }

    auto txHash = api::jsonStringField(resp, "transaction_hash");
    if (!txHash) txHash = api::jsonStringField(resp, "tx_hash");
    if (!txHash) {
        r.error = "Backend revenue-payout response missing transaction_hash";
        return r;
    }
    r.transactionHash = *txHash;
    r.status = api::jsonStringField(resp, "status").value_or("");
    r.withdrawalId = api::jsonStringField(resp, "withdrawal_id").value_or("");
    r.from = api::jsonStringField(resp, "from").value_or("");
    auto chainIdNum = api::jsonNumberField(resp, "chain_id");
    if (chainIdNum) r.chainId = static_cast<uint64_t>(*chainIdNum);
    r.success = true;
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

// ==================== UserWallet Management (fetchers) ====================

// ---- EVM chain management ----

std::string MasterWalletService::listUserEVMChains(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/user-chains/evm");
}

std::string MasterWalletService::addUserEVMChain(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/user-chains/evm", body);
}

std::string MasterWalletService::updateUserEVMChain(const WalletID& walletId, const std::string& chainId,
                                                    const std::string& body) {
    return api::backendPut("/api/v1/master-wallet/" + walletId + "/user-chains/evm/" + chainId, body);
}

bool MasterWalletService::removeUserEVMChain(const WalletID& walletId, const std::string& chainId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/user-chains/evm/" + chainId); return true; }
    catch (const api::APIException&) { return false; }
}

// ---- Non-EVM chain management ----

std::string MasterWalletService::listUserNonEVMChains(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/user-chains/nonevm");
}

std::string MasterWalletService::addUserNonEVMChain(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/user-chains/nonevm", body);
}

std::string MasterWalletService::updateUserNonEVMChain(const WalletID& walletId, const std::string& chainId,
                                                       const std::string& body) {
    return api::backendPut("/api/v1/master-wallet/" + walletId + "/user-chains/nonevm/" + chainId, body);
}

bool MasterWalletService::removeUserNonEVMChain(const WalletID& walletId, const std::string& chainId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/user-chains/nonevm/" + chainId); return true; }
    catch (const api::APIException&) { return false; }
}

// ---- Token management ----

std::string MasterWalletService::listUserTokens(const WalletID& walletId,
                                                const std::optional<std::string>& chainId) {
    if (chainId && !chainId->empty()) {
        std::map<std::string, std::string> params = {{"chain_id", *chainId}};
        return api::backendGet("/api/v1/master-wallet/" + walletId + "/user-tokens",
                               std::optional<std::map<std::string, std::string>>(params));
    }
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/user-tokens");
}

std::string MasterWalletService::addUserToken(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/user-tokens", body);
}

std::string MasterWalletService::updateUserToken(const WalletID& walletId, const std::string& tokenId,
                                                 const std::string& body) {
    return api::backendPut("/api/v1/master-wallet/" + walletId + "/user-tokens/" + tokenId, body);
}

bool MasterWalletService::removeUserToken(const WalletID& walletId, const std::string& tokenId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/user-tokens/" + tokenId); return true; }
    catch (const api::APIException&) { return false; }
}

// ---- Address derivation ----

std::string MasterWalletService::deriveUserAddress(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/derive-user-address", body);
}

std::string MasterWalletService::listUserWalletAddresses(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/user-wallet-addresses");
}

// ---- Auto-sign ----

std::string MasterWalletService::autoSignTransaction(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/auto-sign-transaction", body);
}

std::string MasterWalletService::listAutoSignLogs(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/auto-sign-logs");
}

// POST /api/v1/master-wallet/:id/user-wallet-auto-sign — MasterWallet-owner
// auto-sign bridge (policy-based auto-approval of UserWallet txs). body is a
// JSON object string; returned verbatim from the backend (raw JSON).
std::string MasterWalletService::userWalletAutoSign(const WalletID& masterId, const std::string& body) {
    if (masterId.empty()) {
        throw std::runtime_error("userWalletAutoSign: masterId is required");
    }
    if (body.empty()) {
        throw std::runtime_error("userWalletAutoSign: body is required");
    }
    return api::backendPost("/api/v1/master-wallet/" + masterId + "/user-wallet-auto-sign", body);
}

// POST /api/v1/master-wallet/:id/check-auto-sign-policy — policy-only check
// (no signing/broadcast). body is a JSON object string; returned verbatim.
std::string MasterWalletService::checkAutoSignPolicy(const WalletID& masterId, const std::string& body) {
    if (masterId.empty()) {
        throw std::runtime_error("checkAutoSignPolicy: masterId is required");
    }
    if (body.empty()) {
        throw std::runtime_error("checkAutoSignPolicy: body is required");
    }
    return api::backendPost("/api/v1/master-wallet/" + masterId + "/check-auto-sign-policy", body);
}

// ---- Feature flags ----

std::string MasterWalletService::listFeatureFlags(const WalletID& walletId) {
    return api::backendGet("/api/v1/master-wallet/" + walletId + "/feature-flags");
}

std::string MasterWalletService::addFeatureFlag(const WalletID& walletId, const std::string& body) {
    return api::backendPost("/api/v1/master-wallet/" + walletId + "/feature-flags", body);
}

std::string MasterWalletService::updateFeatureFlag(const WalletID& walletId, const std::string& flagId,
                                                   const std::string& body) {
    return api::backendPut("/api/v1/master-wallet/" + walletId + "/feature-flags/" + flagId, body);
}

bool MasterWalletService::removeFeatureFlag(const WalletID& walletId, const std::string& flagId) {
    try { api::backendDelete("/api/v1/master-wallet/" + walletId + "/feature-flags/" + flagId); return true; }
    catch (const api::APIException&) { return false; }
}

// ==================== Public endpoint helpers ====================

std::string MasterWalletService::getTransactionHistory(const std::string& address, ChainID chainId) {
    std::map<std::string, std::string> params = {{"address", address}, {"chain_id", std::to_string(chainId)}};
    return api::backendGet("/api/v1/transactions/history",
        std::optional<std::map<std::string, std::string>>(params));
}

// GET /health — public (no auth) liveness probe. Returns raw backend JSON.
// /health is the only non-/api/v1-prefixed public route (alongside /ws).
std::string MasterWalletService::health() {
    return api::backendGet("/health");
}

// GET /api/v1/health — public (no auth) liveness alias under /api/v1.
// Returns raw backend JSON.
std::string MasterWalletService::apiHealth() {
    return api::backendGet("/api/v1/health");
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
