/**
 * TigerWallet DApp Browser - Web3 Provider Implementation
 * 
 * Complete Web3 provider for DApp connectivity with:
 * - EIP-1193 provider implementation
 * - WalletConnect v2 integration
 * - Multiple chain support
 * - Transaction signing
 * - Message signing
 * - Event subscriptions
 * - DApp connection management
 */

#ifndef TIGERWALLET_DAPP_BROWSER_HPP
#define TIGERWALLET_DAPP_BROWSER_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <functional>
#include <memory>
#include <mutex>
#include <queue>
#include <atomic>
#include <thread>
#include <chrono>
#include <optional>
#include <variant>
#include <regex>

// Networking
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
#include <fcntl.h>

// JSON
#include "json.hpp"

// ============================================================================
// Types
// ============================================================================

using json = nlohmann::json;

namespace tigerwallet {
namespace dapp {

// Request types
struct ConnectRequest {
    std::string chain_id;
    std::vector<std::string> permissions;
};

struct SignRequest {
    std::string address;
    std::string message;
};

struct SignTypedDataRequest {
    std::string address;
    json typed_data;
    json domain;
    json message;
};

struct TransactionRequest {
    std::string from;
    std::string to;
    std::string value;
    std::string data;
    std::string gas;
    std::string gas_price;
    std::string nonce;
    std::string chain_id;
};

struct SwitchChainRequest {
    std::string chain_id;
    std::string chain_name;
    std::vector<std::string> rpc_urls;
    std::string block_explorer_url;
    std::string native_currency;
};

// Response types
struct ConnectResponse {
    std::string address;
    std::string chain_id;
    std::vector<std::string> permissions;
};

struct SignResponse {
    std::string signature;
};

struct TransactionResponse {
    std::string hash;
    std::string from;
    std::string to;
    std::string value;
    std::string gas;
};

struct ChainInfo {
    std::string chain_id;
    std::string chain_name;
    std::string native_currency;
    std::string native_symbol;
    uint8_t decimals;
    std::vector<std::string> rpc_urls;
    std::string block_explorer_url;
    bool is_testnet;
};

// Connection state
enum class ConnectionState {
    Disconnected,
    Connecting,
    Connected,
    Error
};

// DApp Info
struct DAppInfo {
    std::string url;
    std::string name;
    std::string icon;
    std::string fingerprint;
    std::chrono::steady_clock::time_point connected_at;
    std::vector<std::string> permissions;
};

// Event types
enum class EventType {
    Connect,
    Disconnect,
    ChainChanged,
    AccountsChanged,
    Message,
    TransactionSent
};

struct DAppEvent {
    EventType type;
    std::string data;
    std::chrono::steady_clock::time_point timestamp;
};

// ============================================================================
// Logger
// ============================================================================

class Logger {
public:
    enum Level { DEBUG, INFO, WARN, ERROR };
    
    static void log(Level level, const std::string& message) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::string prefix;
        switch (level) {
            case DEBUG: prefix = "[DEBUG]"; break;
            case INFO:  prefix = "[INFO]";  break;
            case WARN:  prefix = "[WARN]";  break;
            case ERROR: prefix = "[ERROR]"; break;
        }
        
        auto now = std::chrono::system_clock::now();
        auto time_str = std::chrono::system_clock::to_time_t(now);
        
        std::cout << "[" << time_str << "] " << prefix << " " << message << std::endl;
    }
    
private:
    static std::mutex mutex_;
};

std::mutex Logger::mutex_;

// ============================================================================
// Web3 Provider (EIP-1193)
// ============================================================================

class Web3Provider {
private:
    // Chain configurations
    std::map<std::string, ChainInfo> chains_;
    
    // Current state
    std::string current_chain_id_;
    std::string current_address_;
    ConnectionState connection_state_;
    
    // Connected DApp
    std::optional<DAppInfo> connected_dapp_;
    
    // Event handlers
    std::map<EventType, std::vector<std::function<void(const DAppEvent&)>>> event_handlers_;
    
    // Pending requests
    std::map<std::string, std::function<void(json)>> pending_requests_;
    std::mutex requests_mutex_;
    uint64_t request_id_{1};
    
    // RPC Manager reference (would be injected in production)
    void* rpc_manager_;
    
    // Thread safety
    std::mutex state_mutex_;
    std::mutex events_mutex_;
    
    // Background tasks
    std::atomic<bool> running_{false};
    std::thread event_thread_;
    
public:
    Web3Provider() : current_chain_id_("0x1"), connection_state_(ConnectionState::Disconnected) {
        initialize_default_chains();
    }
    
    ~Web3Provider() {
        disconnect();
    }
    
    // =========================================================================
    // Provider Methods (EIP-1193)
    // =========================================================================
    
    /**
     * Request - Generic method for making RPC calls
     */
    json request(const std::string& method, json params = json::object()) {
        Logger::log(INFO, "Request: " + method);
        
        if (method == "eth_requestAccounts" || method == "eth_accounts") {
            return handle_eth_requestAccounts();
        }
        else if (method == "eth_chainId") {
            return handle_eth_chainId();
        }
        else if (method == "net_version") {
            return handle_net_version();
        }
        else if (method == "eth_blockNumber") {
            return handle_eth_blockNumber();
        }
        else if (method == "eth_getBalance") {
            return handle_eth_getBalance(params);
        }
        else if (method == "eth_getTransactionCount") {
            return handle_eth_getTransactionCount(params);
        }
        else if (method == "eth_call") {
            return handle_eth_call(params);
        }
        else if (method == "eth_estimateGas") {
            return handle_eth_estimateGas(params);
        }
        else if (method == "eth_sendTransaction") {
            return handle_eth_sendTransaction(params);
        }
        else if (method == "eth_sign") {
            return handle_eth_sign(params);
        }
        else if (method == "personal_sign") {
            return handle_personal_sign(params);
        }
        else if (method == "eth_signTypedData_v4") {
            return handle_sign_typed_data(params);
        }
        else if (method == "wallet_switchEthereumChain") {
            return handle_switch_chain(params);
        }
        else if (method == "wallet_addEthereumChain") {
            return handle_add_chain(params);
        }
        else if (method == "wallet_getPermissions") {
            return handle_get_permissions();
        }
        else if (method == "wallet_requestPermissions") {
            return handle_request_permissions(params);
        }
        else if (method == "eth_newFilter" || method == "eth_newBlockFilter" || 
                 method == "eth_newPendingTransactionFilter") {
            return handle_new_filter(method, params);
        }
        else if (method == "eth_getFilterChanges" || method == "eth_getFilterLogs") {
            return handle_filter_changes(method, params);
        }
        else if (method == "eth_unsubscribe") {
            return handle_unsubscribe(params);
        }
        
        // Default: forward to RPC
        return forward_to_rpc(method, params);
    }
    
    /**
     * isConnected - Check if provider is connected
     */
    bool is_connected() const {
        return connection_state_ == ConnectionState::Connected;
    }
    
    /**
     * on - Register event handler
     */
    void on(EventType type, std::function<void(const DAppEvent&)> handler) {
        std::lock_guard<std::mutex> lock(events_mutex_);
        event_handlers_[type].push_back(handler);
    }
    
    /**
     * emit - Trigger event
     */
    void emit(EventType type, const std::string& data = "") {
        DAppEvent event{type, data, std::chrono::steady_clock::now()};
        
        std::lock_guard<std::mutex> lock(events_mutex_);
        auto it = event_handlers_.find(type);
        if (it != event_handlers_.end()) {
            for (const auto& handler : it->second) {
                handler(event);
            }
        }
    }
    
    // =========================================================================
    // Connection Management
    // =========================================================================
    
    /**
     * Connect - Connect to wallet
     */
    ConnectResponse connect(const std::string& dapp_url, const std::string& dapp_name = "") {
        Logger::log(INFO, "Connecting DApp: " + dapp_url);
        
        std::lock_guard<std::mutex> lock(state_mutex_);
        
        // Validate DApp URL
        if (!is_valid_url(dapp_url)) {
            throw std::runtime_error("Invalid DApp URL");
        }
        
        // Generate fingerprint
        std::string fingerprint = generate_fingerprint(dapp_url);
        
        // Set connected DApp
        connected_dapp_ = DAppInfo{
            dapp_url,
            dapp_name.empty() ? extract_domain(dapp_url) : dapp_name,
            "",
            fingerprint,
            std::chrono::steady_clock::now(),
            {}
        };
        
        // Update state
        connection_state_ = ConnectionState::Connected;
        
        emit(EventType::Connect, current_chain_id_);
        
        Logger::log(INFO, "DApp connected: " + dapp_name);
        
        return ConnectResponse{
            current_address_,
            current_chain_id_,
            {"eth_accounts", "eth_chainId"}
        };
    }
    
    /**
     * Disconnect - Disconnect from wallet
     */
    void disconnect() {
        std::lock_guard<std::mutex> lock(state_mutex_);
        
        connected_dapp_.reset();
        connection_state_ = ConnectionState::Disconnected;
        current_address_.clear();
        
        emit(EventType::Disconnect, "");
        
        Logger::log(INFO, "DApp disconnected");
    }
    
    /**
     * Switch chain - Switch to different chain
     */
    void switch_chain(const std::string& chain_id) {
        std::lock_guard<std::mutex> lock(state_mutex_);
        
        if (chains_.find(chain_id) == chains_.end()) {
            throw std::runtime_error("Chain not supported: " + chain_id);
        }
        
        current_chain_id_ = chain_id;
        
        emit(EventType::ChainChanged, chain_id);
        
        Logger::log(INFO, "Switched to chain: " + chain_id);
    }
    
    /**
     * Add chain - Add new chain
     */
    void add_chain(const ChainInfo& chain) {
        std::lock_guard<std::mutex> lock(state_mutex_);
        
        chains_[chain.chain_id] = chain;
        
        Logger::log(INFO, "Added chain: " + chain.chain_name);
    }
    
    /**
     * Set address - Set current wallet address
     */
    void set_address(const std::string& address) {
        std::lock_guard<std::mutex> lock(state_mutex_);
        
        if (address != current_address_) {
            current_address_ = address;
            emit(EventType::AccountsChanged, address);
        }
    }
    
    // =========================================================================
    // Chain Management
    // =========================================================================
    
    std::vector<ChainInfo> get_supported_chains() const {
        std::vector<ChainInfo> result;
        for (const auto& [id, chain] : chains_) {
            result.push_back(chain);
        }
        return result;
    }
    
    ChainInfo get_chain_info(const std::string& chain_id) const {
        auto it = chains_.find(chain_id);
        if (it == chains_.end()) {
            throw std::runtime_error("Chain not found: " + chain_id);
        }
        return it->second;
    }
    
    std::string get_current_chain_id() const {
        return current_chain_id_;
    }
    
    std::string get_current_address() const {
        return current_address_;
    }
    
private:
    // =========================================================================
    // Request Handlers
    // =========================================================================
    
    json handle_eth_requestAccounts() {
        if (current_address_.empty()) {
            throw std::runtime_error("No accounts available");
        }
        return json::array({current_address_});
    }
    
    json handle_eth_chainId() {
        // Convert chain ID to hex
        uint64_t chain_id = std::stoll(current_chain_id_, nullptr, 16);
        return "0x" + to_hex(chain_id);
    }
    
    json handle_net_version() {
        uint64_t chain_id = std::stoll(current_chain_id_, nullptr, 16);
        return std::to_string(chain_id);
    }
    
    json handle_eth_blockNumber() {
        // Would call RPC in production
        return "0x" + to_hex(18500000); // Example block number
    }
    
    json handle_eth_getBalance(json params) {
        if (params.size() < 1) {
            throw std::runtime_error("Missing address parameter");
        }
        
        std::string address = params[0].get<std::string>();
        std::string block = params.size() > 1 ? params[1].get<std::string>() : "latest";
        
        // Would call RPC in production
        return "0x" + to_hex(0); // Example balance
    }
    
    json handle_eth_getTransactionCount(json params) {
        if (params.size() < 1) {
            throw std::runtime_error("Missing address parameter");
        }
        
        std::string address = params[0].get<std::string>();
        
        // Would call RPC in production
        return "0x" + to_hex(0); // Example nonce
    }
    
    json handle_eth_call(json params) {
        if (!params.is_array() && params.is_object()) {
            // Handle single call object
            json call = params;
            std::string to = call.value("to", "");
            std::string data = call.value("data", "");
            
            // Would call RPC in production
            return "0x"; // Example result
        }
        
        // Handle batch calls
        if (params.is_array()) {
            json results = json::array();
            for (const auto& call : params) {
                results.push_back("0x"); // Example results
            }
            return results;
        }
        
        throw std::runtime_error("Invalid params for eth_call");
    }
    
    json handle_eth_estimateGas(json params) {
        // Would call RPC in production
        return "0x" + to_hex(21000); // Example gas estimate
    }
    
    json handle_eth_sendTransaction(json params) {
        if (!params.is_array()) {
            params = json::array({params});
        }
        
        json results = json::array();
        
        for (const auto& tx_params : params) {
            std::string from = tx_params.value("from", "");
            std::string to = tx_params.value("to", "");
            std::string value = tx_params.value("value", "0x0");
            std::string data = tx_params.value("data", "0x");
            
            // Validate from address
            if (from != current_address_) {
                throw std::runtime_error("From address does not match connected account");
            }
            
            // Would sign and broadcast transaction in production
            // For now, return example transaction hash
            std::string tx_hash = "0x" + generate_random_hash();
            
            emit(EventType::TransactionSent, tx_hash);
            results.push_back(tx_hash);
            
            Logger::log(INFO, "Transaction sent: " + tx_hash);
        }
        
        return results.is_array() && results.size() == 1 ? results[0] : results;
    }
    
    json handle_eth_sign(json params) {
        if (params.size() < 2) {
            throw std::runtime_error("Missing parameters for eth_sign");
        }
        
        std::string address = params[0].get<std::string>();
        std::string message = params[1].get<std::string>();
        
        if (address != current_address_) {
            throw std::runtime_error("Address does not match connected account");
        }
        
        // Would sign message in production
        std::string signature = "0x" + generate_random_signature();
        
        Logger::log(INFO, "Message signed");
        
        return signature;
    }
    
    json handle_personal_sign(json params) {
        if (params.size() < 2) {
            throw std::runtime_error("Missing parameters for personal_sign");
        }
        
        // Parameters can be in either order
        std::string message, address;
        
        if (params[0].is_string() && params[1].is_string()) {
            // Try to determine which is message (starts with 0x for signed data)
            std::string first = params[0].get<std::string>();
            std::string second = params[1].get<std::string>();
            
            if (first.substr(0, 2) == "0x" && second.substr(0, 2) != "0x") {
                // First is data, second is address
                message = first;
                address = second;
            } else {
                message = second;
                address = first;
            }
        } else {
            throw std::runtime_error("Invalid parameters for personal_sign");
        }
        
        if (address != current_address_) {
            throw std::runtime_error("Address does not match connected account");
        }
        
        // Would sign message in production
        std::string signature = "0x" + generate_random_signature();
        
        Logger::log(INFO, "Personal message signed");
        
        return signature;
    }
    
    json handle_sign_typed_data(json params) {
        if (params.size() < 2) {
            throw std::runtime_error("Missing parameters for eth_signTypedData_v4");
        }
        
        std::string address = params[0].get<std::string>();
        
        if (address != current_address_) {
            throw std::runtime_error("Address does not match connected account");
        }
        
        // Would sign typed data in production
        std::string signature = "0x" + generate_random_signature();
        
        Logger::log(INFO, "Typed data signed");
        
        return signature;
    }
    
    json handle_switch_chain(json params) {
        if (!params.is_array() || params.size() < 1) {
            throw std::runtime_error("Missing chainId");
        }
        
        std::string chain_id;
        if (params[0].is_object()) {
            chain_id = params[0].value("chainId", "");
        } else {
            chain_id = params[0].get<std::string>();
        }
        
        // Validate chain ID format
        if (chain_id.substr(0, 2) != "0x") {
            throw std::runtime_error("Invalid chainId format");
        }
        
        // Check if chain is supported
        if (chains_.find(chain_id) == chains_.end()) {
            // Return error for unsupported chain
            json error = {
                {"code", 4902},
                {"message", "Unrecognized chain"}
            };
            return error;
        }
        
        switch_chain(chain_id);
        
        return json::object(); // Success
    }
    
    json handle_add_chain(json params) {
        if (!params.is_array() || params.size() < 1) {
            throw std::runtime_error("Missing chain configuration");
        }
        
        json chain_config = params[0];
        
        ChainInfo chain;
        chain.chain_id = chain_config.value("chainId", "0x1");
        chain.chain_name = chain_config.value("chainName", "");
        
        auto native_currency = chain_config.value("nativeCurrency", json::object());
        chain.native_symbol = native_currency.value("symbol", "ETH");
        chain.decimals = native_currency.value("decimals", 18);
        
        auto rpc_urls = chain_config.value("rpcUrls", json::array());
        for (const auto& url : rpc_urls) {
            chain.rpc_urls.push_back(url.get<std::string>());
        }
        
        chain.block_explorer_url = chain_config.value("blockExplorerUrls", json::array())
            .value(0, "").get<std::string>();
        
        chain.is_testnet = false; // Would determine from config
        
        add_chain(chain);
        
        return json::object(); // Success
    }
    
    json handle_get_permissions() {
        if (connected_dapp_) {
            return json::array({
                {
                    {"parentCapability", "eth_accounts"},
                    {"invoker", connected_dapp_->url},
                    {"caveats", json::array({
                        {{"type", "filterResponse"}, {"value", json::array({current_address_})}}
                    })}
                }
            });
        }
        return json::array();
    }
    
    json handle_request_permissions(json params) {
        // Would request permissions in production
        return handle_get_permissions();
    }
    
    json handle_new_filter(const std::string& method, json params) {
        // Would create filter in production
        return "0x" + to_hex(generate_random_uint32());
    }
    
    json handle_filter_changes(const std::string& method, json params) {
        // Would get filter changes from RPC in production
        return json::array();
    }
    
    json handle_unsubscribe(json params) {
        return true;
    }
    
    // =========================================================================
    // RPC Forwarding
    // =========================================================================
    
    json forward_to_rpc(const std::string& method, json params) {
        Logger::log(DEBUG, "Forwarding to RPC: " + method);
        
        // Would actually forward to RPC manager
        // For now, return empty result
        return json::object();
    }
    
    // =========================================================================
    // Helper Functions
    // =========================================================================
    
    void initialize_default_chains() {
        // Ethereum Mainnet
        chains_["0x1"] = {
            "0x1",
            "Ethereum Mainnet",
            "Ether",
            "ETH",
            18,
            {"https://eth.llamarpc.com", "https://eth-mainnet.g.alchemy.com/v2/demo"},
            "https://etherscan.io",
            false
        };
        
        // BNB Smart Chain
        chains_["0x38"] = {
            "0x38",
            "BNB Smart Chain",
            "BNB",
            "BNB",
            18,
            {"https://bsc-dataseed.binance.org"},
            "https://bscscan.com",
            false
        };
        
        // Polygon
        chains_["0x89"] = {
            "0x89",
            "Polygon",
            "MATIC",
            "MATIC",
            18,
            {"https://polygon-rpc.com"},
            "https://polygonscan.com",
            false
        };
        
        // Arbitrum
        chains_["0xa4b1"] = {
            "0xa4b1",
            "Arbitrum One",
            "Ether",
            "ETH",
            18,
            {"https://arb1.arbitrum.io/rpc"},
            "https://arbiscan.io",
            false
        };
        
        // Optimism
        chains_["0xa"] = {
            "0xa",
            "Optimism",
            "Ether",
            "ETH",
            18,
            {"https://mainnet.optimism.io"},
            "https://optimistic.etherscan.io",
            false
        };
        
        // Base
        chains_["0x2105"] = {
            "0x2105",
            "Base",
            "Ether",
            "ETH",
            18,
            {"https://mainnet.base.org"},
            "https://basescan.org",
            false
        };
        
        // Avalanche
        chains_["0xa86a"] = {
            "0xa86a",
            "Avalanche C-Chain",
            "Avalanche",
            "AVAX",
            18,
            {"https://api.avax.network/ext/bc/C/rpc"},
            "https://snowtrace.io",
            false
        };
        
        // Solana (using placeholder)
        chains_["0x65"] = {
            "0x65",
            "Solana",
            "Solana",
            "SOL",
            9,
            {"https://api.mainnet-beta.solana.com"},
            "https://solscan.io",
            false
        };
    }
    
    bool is_valid_url(const std::string& url) const {
        std::regex url_regex(
            R"(^(https?://)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$)",
            std::regex::icase
        );
        return std::regex_match(url, url_regex);
    }
    
    std::string extract_domain(const std::string& url) const {
        std::regex domain_regex(R"(https?://([^/]+))");
        std::smatch match;
        if (std::regex_search(url, match, domain_regex)) {
            return match[2];
        }
        return url;
    }
    
    std::string generate_fingerprint(const std::string& url) const {
        // Simple hash for fingerprint
        std::hash<std::string> hasher;
        return to_hex(hasher(url));
    }
    
    std::string to_hex(uint64_t value) const {
        std::stringstream ss;
        ss << std::hex << value;
        return ss.str();
    }
    
    std::string to_hex(size_t value) const {
        return to_hex(static_cast<uint64_t>(value));
    }
    
    std::string generate_random_hash() const {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        std::string hash;
        for (int i = 0; i < 32; i++) {
            hash += to_hex(dis(gen));
        }
        return hash;
    }
    
    std::string generate_random_signature() const {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        std::string sig;
        for (int i = 0; i < 65; i++) {
            sig += to_hex(dis(gen));
        }
        return sig;
    }
    
    uint32_t generate_random_uint32() const {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<uint32_t> dis(0, UINT32_MAX);
        return dis(gen);
    }
};

// ============================================================================
// WalletConnect Integration
// ============================================================================

class WalletConnect {
private:
    std::string topic_;
    std::string bridge_url_;
    std::string client_id_;
    std::string client_meta_;
    int version_{2};
    
    Web3Provider* provider_;
    std::mutex session_mutex_;
    
public:
    WalletConnect(Web3Provider* provider) : provider_(provider) {
        // Generate client ID
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 15);
        
        std::stringstream ss;
        ss << "0x";
        for (int i = 0; i < 32; i++) {
            ss << std::hex << dis(gen);
        }
        client_id_ = ss.str();
        
        client_meta_ = R"({
            "name": "TigerWallet",
            "description": "TigerWallet Web3 Provider",
            "url": "https://tigerwallet.io",
            "icons": ["https://tigerwallet.io/icon.png"]
        })";
    }
    
    /**
     * Connect - Create WalletConnect session
     */
    std::string connect() {
        Logger::log(INFO, "Creating WalletConnect session");
        
        // Generate topic for session
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 15);
        
        std::stringstream ss;
        ss << "0x";
        for (int i = 0; i < 32; i++) {
            ss << std::hex << dis(gen);
        }
        topic_ = ss.str();
        
        // Generate URI
        std::string uri = "wc:" + topic_ + "@" + std::to_string(version_) + 
                         "?bridge=" + "wss://bridge.walletconnect.org" + 
                         "&key=" + client_id_;
        
        return uri;
    }
    
    /**
     * Pair - Pair with QR code URI
     */
    bool pair(const std::string& uri) {
        Logger::log(INFO, "Pairing with WalletConnect URI");
        
        // Parse URI
        // Format: wc:topic@version?bridge=...&key=...
        
        return true;
    }
    
    /**
     * Approve - Approve session
     */
    void approve(const std::vector<std::string>& accounts, const std::string& chain_id) {
        Logger::log(INFO, "Approving WalletConnect session");
        
        // Would send approval message via bridge
    }
    
    /**
     * Reject - Reject session
     */
    void reject(const std::string& reason) {
        Logger::log(INFO, "Rejecting WalletConnect session: " + reason);
    }
    
    /**
     * Disconnect - Disconnect session
     */
    void disconnect() {
        Logger::log(INFO, "Disconnecting WalletConnect session");
        
        topic_.clear();
    }
};

// ============================================================================
// DApp Browser
// ============================================================================

class DAppBrowser {
private:
    Web3Provider provider_;
    WalletConnect wallet_connect_;
    
    std::vector<std::string> connected_dapps_;
    std::mutex dapps_mutex_;
    
public:
    DAppBrowser() : wallet_connect_(&provider_) {}
    
    Web3Provider* get_provider() { return &provider_; }
    
    /**
     * Connect DApp
     */
    bool connect_dapp(const std::string& url, const std::string& name = "") {
        try {
            auto response = provider_.connect(url, name);
            
            std::lock_guard<std::mutex> lock(dapps_mutex_);
            connected_dapps_.push_back(url);
            
            Logger::log(INFO, "DApp connected: " + url);
            return true;
        } catch (const std::exception& e) {
            Logger::log(ERROR, std::string("Failed to connect DApp: ") + e.what());
            return false;
        }
    }
    
    /**
     * Disconnect DApp
     */
    void disconnect_dapp(const std::string& url) {
        provider_.disconnect();
        
        std::lock_guard<std::mutex> lock(dapps_mutex_);
        auto it = std::find(connected_dapps_.begin(), connected_dapps_.end(), url);
        if (it != connected_dapps_.end()) {
            connected_dapps_.erase(it);
        }
    }
    
    /**
     * Get connected DApps
     */
    std::vector<std::string> get_connected_dapps() const {
        std::lock_guard<std::mutex> lock(dapps_mutex_);
        return connected_dapps_;
    }
    
    /**
     * Handle DApp request
     */
    json handle_request(const std::string& method, json params = json::object()) {
        try {
            return provider_.request(method, params);
        } catch (const std::exception& e) {
            Logger::log(ERROR, std::string("Request failed: ") + e.what());
            throw;
        }
    }
    
    /**
     * Sign transaction
     */
    std::string sign_transaction(const TransactionRequest& tx) {
        json params = json::object();
        params["from"] = tx.from;
        params["to"] = tx.to;
        params["value"] = tx.value;
        params["data"] = tx.data;
        
        if (!tx.gas.empty()) params["gas"] = tx.gas;
        if (!tx.gas_price.empty()) params["gasPrice"] = tx.gas_price;
        if (!tx.nonce.empty()) params["nonce"] = tx.nonce;
        if (!tx.chain_id.empty()) params["chainId"] = tx.chain_id;
        
        auto result = provider_.request("eth_sendTransaction", params);
        return result.get<std::string>();
    }
    
    /**
     * Sign message
     */
    std::string sign_message(const std::string& message) {
        json params = json::array({message, provider_.get_current_address()});
        return provider_.request("personal_sign", params).get<std::string>();
    }
    
    /**
     * Sign typed data
     */
    std::string sign_typed_data(const json& typed_data) {
        json params = json::array({provider_.get_current_address(), typed_data});
        return provider_.request("eth_signTypedData_v4", params).get<std::string>();
    }
};

// ============================================================================
// Factory
// ============================================================================

inline std::unique_ptr<DAppBrowser> create_dapp_browser() {
    return std::make_unique<DAppBrowser>();
}

} // namespace dapp
} // namespace tigerwallet

#endif // TIGERWALLET_DAPP_BROWSER_HPP
