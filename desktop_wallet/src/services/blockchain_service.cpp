/**
 * TigerWallet Desktop - Blockchain Service Implementation
 */

#include "services/blockchain_service.h"
#include "models/wallet_models.h"
#include "crypto/bip39.h"
#include "crypto/bip32.h"
#include "crypto/keccak256.h"
#include "crypto/evm_signer.h"
#include "services/keychain_manager.h"
#include <iostream>
#include <sstream>
#include <thread>
#include <chrono>
#include <openssl/sha.h>
#include <openssl/evp.h>
#include <openssl/bio.h>
#include <openssl/buffer.h>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// Static Instance
// ============================================================================

std::shared_ptr<BlockchainService> BlockchainService::instance_ = nullptr;

// ============================================================================
// Constructor/Destructor
// ============================================================================

BlockchainService::BlockchainService() : curl_(nullptr), initialized_(false) {
    chains_ = {
        {"ethereum", Chain{"ethereum", "Ethereum", "ETH", 18, "https://eth.llamarpc.com", "https://etherscan.io", 1, ChainType::EVM, false}},
        {"polygon", Chain{"polygon", "Polygon", "MATIC", 18, "https://polygon-rpc.com", "https://polygonscan.com", 137, ChainType::EVM, false}},
        {"bsc", Chain{"bsc", "BNB Chain", "BNB", 18, "https://bsc-dataseed.binance.org", "https://bscscan.com", 56, ChainType::EVM, false}},
        {"arbitrum", Chain{"arbitrum", "Arbitrum", "ETH", 18, "https://arb1.arbitrum.io/rpc", "https://arbiscan.io", 42161, ChainType::EVM, false}},
        {"optimism", Chain{"optimism", "Optimism", "ETH", 18, "https://mainnet.optimism.io", "https://optimistic.etherscan.io", 10, ChainType::EVM, false}},
        {"avalanche", Chain{"avalanche", "Avalanche", "AVAX", 18, "https://api.avax.network/ext/bc/C/rpc", "https://snowtrace.io", 43114, ChainType::EVM, false}},
        {"solana", Chain{"solana", "Solana", "SOL", 9, "https://api.mainnet-beta.solana.com", "https://solscan.io", 0, ChainType::SOLANA, false}},
        {"bitcoin", Chain{"bitcoin", "Bitcoin", "BTC", 8, "https://blockstream.info/api", "https://blockstream.info", 0, ChainType::BITCOIN, false}},
        {"tron", Chain{"tron", "Tron", "TRX", 6, "https://api.trongrid.io", "https://tronscan.org", 0, ChainType::EVM, false}},
        {"aptos", Chain{"aptos", "Aptos", "APT", 8, "https://api.mainnet.aptoslabs.com/v1", "https://aptoscan.com", 0, ChainType::APTOS, false}},
        {"sui", Chain{"sui", "Sui", "SUI", 9, "https://fullnode.mainnet.sui.io", "https://suiscan.xyz", 0, ChainType::SUI, false}},
        {"ton", Chain{"ton", "Toncoin", "TON", 9, "https://toncenter.com/api/v2", "https://tonscan.org", 0, ChainType::TON, false}},
        {"near", Chain{"near", "NEAR", "NEAR", 24, "https://rpc.mainnet.near.org", "https://explorer.near.org", 0, ChainType::COSMOS, false}},
        {"cosmos", Chain{"cosmos", "Cosmos", "ATOM", 6, "https://cosmos-rpc.polkachu.com", "https://mintscan.io", 0, ChainType::COSMOS, false}},
        {"polkadot", Chain{"polkadot", "Polkadot", "DOT", 10, "https://rpc.polkadot.io", "https://polkadot.subscan.io", 0, ChainType::COSMOS, false}}
    };
}

BlockchainService::~BlockchainService() {
    shutdown();
}

// ============================================================================
// Singleton Access
// ============================================================================

std::shared_ptr<BlockchainService> BlockchainService::getInstance() {
    if (!instance_) {
        instance_ = std::make_shared<BlockchainService>();
    }
    return instance_;
}

// ============================================================================
// Initialization
// ============================================================================

void BlockchainService::initialize() {
    if (initialized_) return;

    curl_ = curl_easy_init();
    initialized_ = true;
    std::cout << "[BlockchainService] Initialized with " << chains_.size() << " chains" << std::endl;
}

void BlockchainService::shutdown() {
    if (curl_) {
        curl_easy_cleanup(curl_);
        curl_ = nullptr;
    }
    initialized_ = false;
}

// ============================================================================
// Chain Management
// ============================================================================

std::vector<Chain> BlockchainService::getSupportedChains() {
    std::vector<Chain> result;
    for (const auto& pair : chains_) {
        result.push_back(pair.second);
    }
    return result;
}

std::optional<Chain> BlockchainService::getChain(const std::string& chainId) {
    auto it = chains_.find(chainId);
    if (it != chains_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::optional<Chain> BlockchainService::getChainBySymbol(const std::string& symbol) {
    for (const auto& pair : chains_) {
        if (pair.second.symbol == symbol || pair.second.name == symbol) {
            return pair.second;
        }
    }
    return std::nullopt;
}

// ============================================================================
// Balance Operations
// ============================================================================

std::future<double> BlockchainService::getBalance(const std::string& address, const std::string& chainId) {
    return std::async(std::launch::async, [this, address, chainId]() -> double {
        auto chain = getChain(chainId);
        if (!chain) {
            throw BlockchainException(BlockchainException::ErrorCode::UnsupportedChain,
                "Unsupported chain: " + chainId);
        }

        switch (chain->type) {
            case ChainType::EVM:
                return evmGetBalance(address, *chain);
            case ChainType::SOLANA:
                return solanaGetBalance(address, *chain);
            case ChainType::BITCOIN:
                return bitcoinGetBalance(address, *chain);
            default:
                return 0.0;
        }
    });
}

std::future<double> BlockchainService::getTokenBalance(const std::string& address, const std::string& tokenAddress, const std::string& chainId) {
    return std::async(std::launch::async, [this, address, tokenAddress, chainId]() -> double {
        auto chain = getChain(chainId);
        if (!chain || chain->type != ChainType::EVM) {
            return 0.0;
        }
        return evmGetTokenBalance(address, tokenAddress, *chain);
    });
}

// ============================================================================
// Transaction Operations
// ============================================================================

std::future<std::string> BlockchainService::sendTransaction(
    const std::string& from,
    const std::string& to,
    const std::string& amount,
    const std::string& chainId,
    const std::optional<std::string>& tokenAddress
) {
    return std::async(std::launch::async, [this, from, to, amount, chainId, tokenAddress]() -> std::string {
        auto chain = getChain(chainId);
        if (!chain) {
            throw BlockchainException(BlockchainException::ErrorCode::UnsupportedChain,
                "Unsupported chain: " + chainId);
        }

        if (chain->type == ChainType::EVM) {
            return evmSendTransaction(from, to, amount, *chain, tokenAddress);
        }

        throw BlockchainException(BlockchainException::ErrorCode::TransactionFailed,
            "Transaction not supported for this chain");
    });
}

std::future<std::optional<Transaction>> BlockchainService::getTransactionReceipt(const std::string& txHash, const std::string& chainId) {
    return std::async(std::launch::async, [this, txHash, chainId]() -> std::optional<Transaction> {
        // Simplified implementation
        return std::nullopt;
    });
}

std::future<std::vector<Transaction>> BlockchainService::getTransactions(const std::string& address, const std::string& chainId, int limit) {
    return std::async(std::launch::async, [this, address, chainId, limit]() -> std::vector<Transaction> {
        // Would call indexer API in production
        return {};
    });
}

// ============================================================================
// Gas Operations
// ============================================================================

std::future<std::string> BlockchainService::getGasPrice(const std::string& chainId) {
    return std::async(std::launch::async, [this, chainId]() -> std::string {
        auto chain = getChain(chainId);
        if (!chain || chain->type != ChainType::EVM) {
            return "0x0";
        }

        JsonRpcRequest request;
        request.method = "eth_gasPrice";
        request.params = {};

        JsonRpcResponse response = sendJsonRpc(chain->rpc_url, request);
        if (response.result) {
            return *response.result;
        }
        return "0x0";
    });
}

std::future<std::string> BlockchainService::estimateGas(
    const std::string& from,
    const std::string& to,
    const std::string& value,
    const std::string& chainId,
    const std::optional<std::string>& data
) {
    return std::async(std::launch::async, [this, from, to, value, chainId, data]() -> std::string {
        auto chain = getChain(chainId);
        if (!chain || chain->type != ChainType::EVM) {
            return "0x5208"; // Default 21000 gas
        }

        JsonRpcRequest request;
        request.method = "eth_estimateGas";

        std::map<std::string, std::string> tx;
        tx["from"] = from;
        tx["to"] = to;
        tx["value"] = value;
        if (data) {
            tx["data"] = *data;
        }

        // Convert to JSON params
        std::ostringstream oss;
        oss << "[" << from << "," << to << "," << value << "]";

        JsonRpcResponse response = sendJsonRpc(chain->rpc_url, request);
        if (response.result) {
            return *response.result;
        }
        return "0x5208";
    });
}

// ============================================================================
// Token Operations
// ============================================================================

std::future<std::vector<Token>> BlockchainService::getTokens(const std::string& address, const std::string& chainId) {
    return std::async(std::launch::async, [this, address, chainId]() -> std::vector<Token> {
        // Simplified - would fetch token list from indexer
        return {};
    });
}

// ============================================================================
// Wallet Operations
// ============================================================================

std::future<Wallet> BlockchainService::createWallet(const Chain& chain, const std::string& name) {
    return std::async(std::launch::async, [this, chain, name]() -> Wallet {
        // Generate a REAL BIP-39 mnemonic (random entropy, proper checksum)
        std::string mnemonic = BIP39::generateMnemonic(256); // 24-word mnemonic

        auto [address, publicKey] = deriveKeyFromMnemonic(mnemonic, chain);

        Wallet wallet;
        wallet.id = generateUUID();
        wallet.name = name.empty() ? chain.name + " Wallet" : name;
        wallet.address = address;
        wallet.public_key = publicKey;
        wallet.chain_id = chain.id;
        wallet.balance = 0.0;
        wallet.balance_usd = 0.0;
        wallet.created_at = std::chrono::system_clock::now();
        wallet.is_backed_up = false;
        wallet.is_hardware = false;

        return wallet;
    });
}

std::future<Wallet> BlockchainService::importWallet(const std::string& mnemonic, const std::string& chainId, const std::string& name) {
    return std::async(std::launch::async, [this, mnemonic, chainId, name]() -> Wallet {
        if (!validateMnemonic(mnemonic)) {
            throw BlockchainException(BlockchainException::ErrorCode::InvalidMnemonic,
                "Invalid mnemonic phrase");
        }

        auto chainOpt = getChain(chainId);
        if (!chainOpt) {
            throw BlockchainException(BlockchainException::ErrorCode::UnsupportedChain,
                "Unsupported chain: " + chainId);
        }

        auto [address, publicKey] = deriveKeyFromMnemonic(mnemonic, *chainOpt);

        Wallet wallet;
        wallet.id = generateUUID();
        wallet.name = name.empty() ? "Imported " + chainOpt->name : name;
        wallet.address = address;
        wallet.public_key = publicKey;
        wallet.chain_id = chainId;
        wallet.balance = 0.0;
        wallet.balance_usd = 0.0;
        wallet.created_at = std::chrono::system_clock::now();
        wallet.is_backed_up = true;
        wallet.is_hardware = false;

        return wallet;
    });
}

std::future<Wallet> BlockchainService::importFromPrivateKey(const std::string& privateKey, const std::string& chainId, const std::string& name) {
    return std::async(std::launch::async, [this, privateKey, chainId, name]() -> Wallet {
        auto chainOpt = getChain(chainId);
        if (!chainOpt) {
            throw BlockchainException(BlockchainException::ErrorCode::UnsupportedChain,
                "Unsupported chain: " + chainId);
        }

        auto [address, publicKey] = deriveKeyFromPrivateKey(privateKey);

        Wallet wallet;
        wallet.id = generateUUID();
        wallet.name = name.empty() ? "Imported " + chainOpt->name : name;
        wallet.address = address;
        wallet.public_key = publicKey;
        wallet.chain_id = chainId;
        wallet.balance = 0.0;
        wallet.balance_usd = 0.0;
        wallet.created_at = std::chrono::system_clock::now();
        wallet.is_backed_up = true;
        wallet.is_hardware = false;

        return wallet;
    });
}

// ============================================================================
// Address Validation
// ============================================================================

bool BlockchainService::isValidAddress(const std::string& address, const std::string& chainId) {
    auto chain = getChain(chainId);
    if (!chain) return false;

    if (chain->type == ChainType::EVM) {
        // Check if it starts with 0x and is 42 characters long
        return address.length() == 42 &&
               address.substr(0, 2) == "0x";
    } else if (chain->type == ChainType::SOLANA) {
        // Base58 encoded, 32-44 characters
        return address.length() >= 32 && address.length() <= 44;
    } else if (chain->type == ChainType::BITCOIN) {
        // Base58 or Bech32
        return address.length() >= 26 && address.length() <= 62;
    }

    return !address.empty();
}

// ============================================================================
// Event Callbacks
// ============================================================================

void BlockchainService::setBalanceUpdateCallback(BalanceUpdateCallback callback) {
    balanceCallback_ = callback;
}

void BlockchainService::setTransactionCallback(TransactionCallback callback) {
    transactionCallback_ = callback;
}

// ============================================================================
// Private: RPC Communication
// ============================================================================

std::string BlockchainService::callRpc(const std::string& url, const std::string& body) {
    if (!curl_) {
        curl_ = curl_easy_init();
    }

    std::string response_string;
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");

    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, body.c_str());
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, +[](char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* str = static_cast<std::string*>(userdata);
        str->append(ptr, size * nmemb);
        return size * nmemb;
    });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response_string);

    CURLcode res = curl_easy_perform(curl_);
    curl_slist_free_all(headers);

    if (res != CURLE_OK) {
        throw BlockchainException(BlockchainException::ErrorCode::NetworkError,
            std::string("RPC call failed: ") + curl_easy_strerror(res));
    }

    return response_string;
}

JsonRpcResponse BlockchainService::sendJsonRpc(const std::string& rpcUrl, const JsonRpcRequest& request) {
    std::ostringstream body;
    body << "{\"jsonrpc\":\"" << request.jsonrpc << "\","
         << "\"method\":\"" << request.method << "\","
         << "\"params\":[";

    for (size_t i = 0; i < request.params.size(); ++i) {
        body << "\"" << request.params[i] << "\"";
        if (i < request.params.size() - 1) body << ",";
    }

    body << "],\"id\":" << request.id << "}";

    std::string response = callRpc(rpcUrl, body.str());

    // Parse response (simplified)
    JsonRpcResponse result;
    result.id = request.id;

    // In production, use proper JSON parsing
    if (response.find("\"error\"") != std::string::npos) {
        result.error = response;
    } else if (response.find("\"result\"") != std::string::npos) {
        size_t result_start = response.find("\"result\":") + 9;
        size_t result_end = response.find(",\"id\"");
        if (result_end == std::string::npos) {
            result_end = response.find("}", result_start);
        }
        if (result_start < result_end) {
            result.result = response.substr(result_start, result_end - result_start);
        }
    }

    return result;
}

// ============================================================================
// Private: EVM Operations
// ============================================================================

double BlockchainService::evmGetBalance(const std::string& address, const Chain& chain) {
    JsonRpcRequest request;
    request.method = "eth_getBalance";
    request.params = {address, "latest"};

    JsonRpcResponse response = sendJsonRpc(chain.rpc_url, request);

    if (response.result) {
        return hexToDouble(*response.result, chain.decimals);
    }
    return 0.0;
}

double BlockchainService::evmGetTokenBalance(const std::string& address, const std::string& tokenAddress, const Chain& chain) {
    std::string method_id = "0x70a08231";
    std::string padded_address = address.substr(2); // Remove 0x
    while (padded_address.length() < 64) padded_address = "0" + padded_address;

    JsonRpcRequest request;
    request.method = "eth_call";
    request.params = {
        "{\"to\":\"" + tokenAddress + "\",\"data\":\"" + method_id + padded_address + "\"}",
        "latest"
    };

    JsonRpcResponse response = sendJsonRpc(chain.rpc_url, request);

    if (response.result) {
        return hexToDouble(*response.result, 18); // Standard ERC20 decimals
    }
    return 0.0;
}

std::string BlockchainService::evmSendTransaction(
    const std::string& from,
    const std::string& to,
    const std::string& amount,
    const Chain& chain,
    const std::optional<std::string>& data
) {
    // No unsigned eth_sendTransaction. Use signAndSendTransaction with the
    // wallet password to sign locally with real ECDSA secp256k1 + EIP-155.
    throw BlockchainException(BlockchainException::ErrorCode::TransactionFailed,
        "Use signAndSendTransaction with wallet password for real signing. "
        "Unsigned eth_sendTransaction is not supported.");
}

std::string BlockchainService::signAndSendTransaction(
    const std::string& walletId,
    const std::string& password,
    const std::string& to,
    const std::string& valueWei,
    const Chain& chain,
    uint64_t gasLimit,
    const std::optional<std::string>& data
) {
    auto keychain = KeychainManager::getInstance();
    auto mnemonicOpt = keychain->loadWalletSeed(walletId, password);
    if (!mnemonicOpt) {
        throw BlockchainException(BlockchainException::ErrorCode::TransactionFailed,
            "Wallet not found or wrong password");
    }
    auto seed = BIP39::mnemonicToSeed(*mnemonicOpt);
    auto master = BIP32::fromSeed(seed);
    auto derivedKey = BIP32::derivePath(master, "m/44'/60'/0'/0/0");
    std::string from = BIP32::evmAddress(derivedKey.private_key);

    JsonRpcRequest nonceReq;
    nonceReq.method = "eth_getTransactionCount";
    nonceReq.params = {from, "pending"};
    auto nonceResp = sendJsonRpc(chain.rpc_url, nonceReq);
    uint64_t nonce = 0;
    if (nonceResp.result) {
        std::string hex = *nonceResp.result;
        if (hex.substr(0, 2) == "0x") hex = hex.substr(2);
        nonce = std::stoull(hex, nullptr, 16);
    }
    JsonRpcRequest gasReq;
    gasReq.method = "eth_gasPrice";
    auto gasResp = sendJsonRpc(chain.rpc_url, gasReq);
    std::string gasPrice = "0x77359400";
    if (gasResp.result) gasPrice = *gasResp.result;
    if (gasLimit == 0) gasLimit = 21000;

    EvmTxParams params;
    params.nonce = nonce;
    params.gasLimit = gasLimit;
    params.gasPriceWei = gasPrice;
    params.toAddress = to;
    params.valueWei = valueWei.empty() ? "0x0" : valueWei;
    params.data = data.value_or("0x");
    params.chainId = std::stoull(chain.id);

    std::string rawTx = EvmSigner::signLegacy(params, derivedKey.private_key);
    JsonRpcRequest sendReq;
    sendReq.method = "eth_sendRawTransaction";
    sendReq.params = {rawTx};
    auto sendResp = sendJsonRpc(chain.rpc_url, sendReq);
    if (sendResp.result) return *sendResp.result;
    throw BlockchainException(BlockchainException::ErrorCode::TransactionFailed,
        "Broadcast failed: " + (sendResp.error ? *sendResp.error : "unknown"));
}

// ============================================================================
// Private: Solana Operations
// ============================================================================

double BlockchainService::solanaGetBalance(const std::string& address, const Chain& chain) {
    JsonRpcRequest request;
    request.method = "getBalance";
    request.params = {address};
    JsonRpcResponse response = sendJsonRpc(chain.rpc_url, request);
    if (response.result) {
        std::string resultStr = *response.result;
        size_t valuePos = resultStr.find("\"value\"");
        if (valuePos != std::string::npos) {
            size_t colon = resultStr.find(":", valuePos);
            size_t end = resultStr.find_first_of(",}", colon);
            std::string valStr = resultStr.substr(colon + 1, end - colon - 1);
            double lamports = std::stod(valStr);
            return lamports / 1e9;
        }
    }
    return 0.0;
}

// ============================================================================
// Private: Bitcoin Operations
// ============================================================================

double BlockchainService::bitcoinGetBalance(const std::string& address, const Chain& chain) {
    std::string url = chain.rpc_url + "/address/" + address;
    try {
        std::string response = callRpc(url, "");
        size_t fundedPos = response.find("\"funded_txo_sum\"");
        size_t spentPos = response.find("\"spent_txo_sum\"");
        if (fundedPos == std::string::npos) return 0.0;
        size_t fundedColon = response.find(":", fundedPos);
        size_t fundedEnd = response.find(",", fundedColon);
        std::string fundedStr = response.substr(fundedColon + 1, fundedEnd - fundedColon - 1);
        double funded = std::stod(fundedStr);
        double spent = 0.0;
        if (spentPos != std::string::npos) {
            size_t spentColon = response.find(":", spentPos);
            size_t spentEnd = response.find(",", spentColon);
            if (spentEnd == std::string::npos) spentEnd = response.find("}", spentColon);
            std::string spentStr = response.substr(spentColon + 1, spentEnd - spentColon - 1);
            spent = std::stod(spentStr);
        }
        return (funded - spent) / 1e8;
    } catch (...) {
        return 0.0;
    }
}

// ============================================================================
// Private: Key Derivation
// ============================================================================

std::pair<std::string, std::string> BlockchainService::deriveKeyFromMnemonic(const std::string& mnemonic, const Chain& chain) {
    // REAL BIP-39 -> BIP-32 -> BIP-44 key derivation (replaces SHA256 hack)
    auto seed = BIP39::mnemonicToSeed(mnemonic);
    auto master = BIP32::fromSeed(seed);
    auto derivedKey = BIP32::derivePath(master, "m/44'/60'/0'/0/0");
    std::string address = BIP32::evmAddress(derivedKey.private_key);
    auto pubKeyBytes = BIP32::compressedPublicKey(derivedKey.private_key);
    return {address, "0x" + Keccak256::hex(pubKeyBytes)};
}

std::pair<std::string, std::string> BlockchainService::deriveKeyFromPrivateKey(const std::string& privateKey) {
    // REAL address derivation from private key via secp256k1 + keccak256
    std::string clean_key = privateKey;
    if (clean_key.find("0x") == 0) clean_key = clean_key.substr(2);
    auto keyBytes = EvmSigner::hexToBytes("0x" + clean_key);
    std::string address = BIP32::evmAddress(keyBytes);
    auto pubBytes = BIP32::compressedPublicKey(keyBytes);
    return {address, "0x" + Keccak256::hex(pubBytes)};
}

// ============================================================================
// Private: Validation
// ============================================================================

bool BlockchainService::validateMnemonic(const std::string& mnemonic) {
    return BIP39::validateMnemonic(mnemonic);
}

// ============================================================================
// Blockchain Exception
// ============================================================================

BlockchainException::BlockchainException(ErrorCode code, const std::string& message)
    : std::runtime_error(message), code_(code) {}

BlockchainException::ErrorCode BlockchainException::getErrorCode() const {
    return code_;
}

} // namespace wallet
} // namespace tiger
