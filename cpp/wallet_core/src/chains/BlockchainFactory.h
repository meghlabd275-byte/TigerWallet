/**
 * Blockchain Factory - Creates and manages blockchain-specific implementations
 * Supports 130+ blockchains with proper address derivation and transaction signing
 */

#ifndef TWCORE_BLOCKCHAIN_FACTORY_H
#define TWCORE_BLOCKCHAIN_FACTORY_H

#include <memory>
#include <string>
#include <vector>
#include <unordered_map>
#include "TigerWalletCore.h"

namespace TigerWalletCore {

// Forward declarations
class ChainBase;
class BitcoinChain;
class EthereumChain;
class CosmosChain;
class SolanaChain;
class PolkadotChain;

/**
 * Base class for all blockchain implementations
 */
class ChainBase {
public:
    virtual ~ChainBase() = default;
    
    virtual TWCoreCoinType getCoinType() const = 0;
    virtual std::string getName() const = 0;
    virtual std::string getSymbol() const = 0;
    virtual uint8_t getDecimals() const = 0;
    virtual uint64_t getChainId() const = 0;
    
    virtual std::string deriveAddress(const std::vector<uint8_t>& publicKey) const = 0;
    virtual bool validateAddress(const std::string& address) const = 0;
    virtual std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const = 0;
    
    virtual std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const = 0;
    
    virtual std::string formatAmount(const std::string& rawAmount) const = 0;
    virtual std::string parseAmount(const std::string& formattedAmount) const = 0;
};

/**
 * Bitcoin blockchain implementation
 */
class BitcoinChain : public ChainBase {
public:
    enum class AddressType {
        Legacy,      // P2PKH
        SegWit,      // P2WPKH
        NestedSegWit, // P2SH
        Taproot      // P2TR
    };
    
    BitcoinChain(AddressType addrType = AddressType::SegWit) 
        : addressType_(addrType), network_(0) {}
    
    void setNetwork(TWCoreNetwork network) { network_ = network; }
    void setAddressType(AddressType type) { addressType_ = type; }
    
    TWCoreCoinType getCoinType() const override { return TWCORE_COIN_BITCOIN; }
    std::string getName() const override { return "Bitcoin"; }
    std::string getSymbol() const override { return "BTC"; }
    uint8_t getDecimals() const override { return 8; }
    uint64_t getChainId() const override { return 0; }
    
    std::string deriveAddress(const std::vector<uint8_t>& publicKey) const override;
    bool validateAddress(const std::string& address) const override;
    std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const override;
    
    std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const override {
        return "m/44'/0'/" + std::to_string(account) + "'/" + 
               std::to_string(change) + "/" + std::to_string(index);
    }
    
    std::string formatAmount(const std::string& rawAmount) const override;
    std::string parseAmount(const std::string& formattedAmount) const override;
    
private:
    AddressType addressType_;
    uint32_t network_;
    
    std::vector<uint8_t> hash160(const std::vector<uint8_t>& data) const;
    std::vector<uint8_t> doubleSha256(const std::vector<uint8_t>& data) const;
    std::string base58Encode(const std::vector<uint8_t>& data) const;
    std::string bech32Encode(const std::string& hrp, const std::vector<uint8_t>& data) const;
};

/**
 * Ethereum and EVM-compatible chains
 */
class EthereumChain : public ChainBase {
public:
    EthereumChain(uint64_t chainId = 1, const std::string& name = "Ethereum")
        : chainId_(chainId), name_(name) {
        // Set defaults for Ethereum
        if (chainId == 1) {
            symbol_ = "ETH";
            decimals_ = 18;
        }
    }
    
    void setChainId(uint64_t id) { chainId_ = id; }
    void setName(const std::string& name) { name_ = name; }
    void setSymbol(const std::string& symbol) { symbol_ = symbol; }
    void setDecimals(uint8_t decimals) { decimals_ = decimals; }
    
    TWCoreCoinType getCoinType() const override { return TWCORE_COIN_ETHEREUM; }
    std::string getName() const override { return name_; }
    std::string getSymbol() const override { return symbol_; }
    uint8_t getDecimals() const override { return decimals_; }
    uint64_t getChainId() const override { return chainId_; }
    
    std::string deriveAddress(const std::vector<uint8_t>& publicKey) const override;
    bool validateAddress(const std::string& address) const override;
    std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const override;
    
    std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const override {
        return "m/44'/60'/" + std::to_string(account) + "'/" + 
               std::to_string(change) + "/" + std::to_string(index);
    }
    
    std::string formatAmount(const std::string& rawAmount) const override;
    std::string parseAmount(const std::string& formattedAmount) const override;
    
    // EIP-1559 support
    std::vector<uint8_t> encodeEIP1559Transaction(
        uint64_t chainId,
        uint64_t nonce,
        uint64_t gasLimit,
        const std::string& maxFeePerGas,
        const std::string& maxPriorityFeePerGas,
        const std::string& to,
        const std::string& value,
        const std::vector<uint8_t>& data) const;
    
    // Message signing (EIP-191)
    std::vector<uint8_t> signMessage(
        const std::vector<uint8_t>& privateKey,
        const std::string& message) const;
    
    // Typed data signing (EIP-712)
    std::vector<uint8_t> signTypedData(
        const std::vector<uint8_t>& privateKey,
        const std::string& domainSeparator,
        const std::string& messageHash) const;
    
private:
    uint64_t chainId_;
    std::string name_;
    std::string symbol_;
    uint8_t decimals_;
    
    std::vector<uint8_t> keccak256(const std::vector<uint8_t>& data) const;
};

/**
 * Cosmos SDK chains
 */
class CosmosChain : public ChainBase {
public:
    CosmosChain(TWCoreCoinType coinType, const std::string& name, const std::string& prefix)
        : coinType_(coinType), name_(name), prefix_(prefix) {}
    
    TWCoreCoinType getCoinType() const override { return coinType_; }
    std::string getName() const override { return name_; }
    std::string getSymbol() const override { return symbol_; }
    uint8_t getDecimals() const override { return decimals_; }
    uint64_t getChainId() const override { return chainId_; }
    
    std::string deriveAddress(const std::vector<uint8_t>& publicKey) const override;
    bool validateAddress(const std::string& address) const override;
    std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const override;
    
    std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const override {
        return "m/44'/" + std::to_string(coinType_) + "'/" + 
               std::to_string(account) + "'/" + 
               std::to_string(change) + "/" + std::to_string(index);
    }
    
    std::string formatAmount(const std::string& rawAmount) const override;
    std::string parseAmount(const std::string& formattedAmount) const override;
    
    // Cosmos-specific transaction building
    std::vector<uint8_t> buildSendMessage(
        const std::string& fromAddress,
        const std::string& toAddress,
        const std::string& amount,
        const std::string& denom) const;
    
    std::vector<uint8_t> buildDelegateMessage(
        const std::string& delegatorAddress,
        const std::string& validatorAddress,
        const std::string& amount) const;
    
    std::vector<uint8_t> buildUndelegateMessage(
        const std::string& delegatorAddress,
        const std::string& validatorAddress,
        const std::string& amount) const;
    
    std::vector<uint8_t> buildRedelegateMessage(
        const std::string& delegatorAddress,
        const std::string& fromValidator,
        const std::string& toValidator,
        const std::string& amount) const;
    
    // Sign transaction with Amino or Protobuf
    std::vector<uint8_t> signAmino(
        const std::vector<uint8_t>& privateKey,
        const std::string& jsonTx,
        uint64_t accountNumber,
        uint64_t sequence,
        const std::string& chainId,
        const std::string& memo) const;
    
    std::vector<uint8_t> signProtobuf(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txBody,
        const std::vector<uint8_t>& authInfo,
        const std::vector<uint8_t>& signDoc) const;
    
private:
    TWCoreCoinType coinType_;
    std::string name_;
    std::string symbol_ = "ATOM";
    uint8_t decimals_ = 6;
    uint64_t chainId_ = 0;
    std::string prefix_ = "cosmos";
    
    std::vector<uint8_t> sha256(const std::vector<uint8_t>& data) const;
};

/**
 * Solana blockchain
 */
class SolanaChain : public ChainBase {
public:
    SolanaChain() 
        : coinType_(TWCORE_COIN_SOLANA), name_("Solana"), symbol_("SOL"), decimals_(9) {}
    
    TWCoreCoinType getCoinType() const override { return coinType_; }
    std::string getName() const override { return name_; }
    std::string getSymbol() const override { return symbol_; }
    uint8_t getDecimals() const override { return decimals_; }
    uint64_t getChainId() const override { return 101; }
    
    std::string deriveAddress(const std::vector<uint8_t>& publicKey) const override;
    bool validateAddress(const std::string& address) const override;
    std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const override;
    
    std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const override {
        return "m/44'/501'/" + std::to_string(account) + "'/" + 
               std::to_string(change) + "/" + std::to_string(index);
    }
    
    std::string formatAmount(const std::string& rawAmount) const override;
    std::string parseAmount(const std::string& formattedAmount) const override;
    
    // Solana-specific instructions
    std::vector<uint8_t> createTransferInstruction(
        const std::string& from,
        const std::string& to,
        uint64_t lamports) const;
    
    std::vector<uint8_t> createTransferCheckedInstruction(
        const std::string& from,
        const std::string& to,
        const std::string& mint,
        uint64_t amount,
        uint8_t decimals) const;
    
    std::vector<uint8_t> createAssociatedTokenAccountInstruction(
        const std::string& fundingAccount,
        const std::string& walletAccount,
        const std::string& tokenMint) const;
    
    // Versioned transactions
    std::vector<uint8_t> createVersionedTransaction(
        const std::vector<uint8_t>& message,
        const std::vector<std::vector<uint8_t>>& signatures) const;
    
    // Serialize transaction
    std::string serializeTransaction(
        const std::vector<uint8_t>& signedTx) const;
    
private:
    TWCoreCoinType coinType_;
    std::string name_;
    std::string symbol_;
    uint8_t decimals_;
    
    std::vector<uint8_t> ed25519Sign(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& message) const;
};

/**
 * Polkadot/Substrate chains
 */
class PolkadotChain : public ChainBase {
public:
    PolkadotChain(TWCoreCoinType coinType = TWCORE_COIN_POLKADOT, const std::string& name = "Polkadot")
        : coinType_(coinType), name_(name) {
        if (coinType == TWCORE_COIN_POLKADOT) {
            symbol_ = "DOT";
            decimals_ = 10;
            chainId_ = 0;
            prefix_ = "1";
        } else if (coinType == TWCORE_COIN_KUSAMA) {
            symbol_ = "KSM";
            decimals_ = 12;
            chainId_ = 2;
            prefix_ = "2";
        }
    }
    
    TWCoreCoinType getCoinType() const override { return coinType_; }
    std::string getName() const override { return name_; }
    std::string getSymbol() const override { return symbol_; }
    uint8_t getDecimals() const override { return decimals_; }
    uint64_t getChainId() const override { return chainId_; }
    
    std::string deriveAddress(const std::vector<uint8_t>& publicKey) const override;
    bool validateAddress(const std::string& address) const override;
    std::vector<uint8_t> signTransaction(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& txData) const override;
    
    std::string getDerivationPath(uint32_t account, uint32_t change, uint32_t index) const override {
        return "//" + std::to_string(account) + "//" + 
               std::to_string(change) + "//" + std::to_string(index);
    }
    
    std::string formatAmount(const std::string& rawAmount) const override;
    std::string parseAmount(const std::string& formattedAmount) const override;
    
    // Substrate-specific
    std::vector<uint8_t> signPayload(
        const std::vector<uint8_t>& privateKey,
        const std::vector<uint8_t>& payload) const;
    
    std::string encodeAddress(const std::vector<uint8_t>& publicKey) const;
    
private:
    TWCoreCoinType coinType_;
    std::string name_;
    std::string symbol_ = "DOT";
    uint8_t decimals_ = 10;
    uint64_t chainId_ = 0;
    std::string prefix_ = "1";
    
    std::vector<uint8_t> blake2b512(const std::vector<uint8_t>& data) const;
};

/**
 * Factory for creating blockchain implementations
 */
class ChainFactory {
public:
    static ChainFactory& instance();
    
    // Register custom chain
    void registerChain(uint32_t coinType, std::unique_ptr<ChainBase> chain);
    
    // Get chain by coin type
    ChainBase* getChain(uint32_t coinType);
    
    // Create default chains
    void initializeDefaultChains();
    
    // Get all supported coin types
    std::vector<uint32_t> getSupportedCoinTypes();
    
    // Chain info helpers
    std::string getChainName(uint32_t coinType);
    std::string getChainSymbol(uint32_t coinType);
    uint64_t getChainId(uint32_t coinType);
    uint8_t getChainDecimals(uint32_t coinType);

private:
    ChainFactory() = default;
    std::unordered_map<uint32_t, std::unique_ptr<ChainBase>> chains_;
};

// Predefined chain configurations
struct ChainConfig {
    uint32_t coinType;
    uint64_t chainId;
    std::string name;
    std::string symbol;
    std::string prefix;
    uint8_t decimals;
    std::string rpcUrl;
    std::string explorerUrl;
    std::string derivationPath;
    bool isEVM;
};

// Get all supported chain configurations
inline std::vector<ChainConfig> getAllChainConfigs() {
    return {
        // Bitcoin Family
        {0, 0, "Bitcoin", "BTC", "", 8, "https://blockstream.info/api", "https://mempool.space", "m/44'/0'/0'/0/0", false},
        {145, 0, "Bitcoin Cash", "BCH", "bitcoincash", 8, "https://bch.loping.net", "https://blockchair.com", "m/44'/145'/0'/0/0", false},
        {2, 0, "Litecoin", "LTC", "ltc", 8, "https://litecoin.losab.io", "https://blockchair.com", "m/44'/2'/0'/0/0", false},
        {3, 0, "Dogecoin", "DOGE", "dogecoin", 8, "https://dogecoin.losab.io", "https://blockchair.com", "m/44'/3'/0'/0/0", false},
        {5, 0, "Dash", "DASH", "dash", 8, "https://dash.losab.io", "https://blockchair.com", "m/44'/5'/0'/0/0", false},
        {133, 0, "Zcash", "ZEC", "zec", 8, "https://zcash.losab.io", "https://blockchair.com", "m/44'/133'/0'/0/0", false},
        
        // Ethereum Family (EVM)
        {60, 1, "Ethereum", "ETH", "0x", 18, "https://eth.llamarpc.com", "https://etherscan.io", "m/44'/60'/0'/0/0", true},
        {60, 5, "Ethereum Goerli", "ETH", "0x", 18, "https://goerli.infura.io/v3/", "https://goerli.etherscan.io", "m/44'/60'/0'/0/0", true},
        {60, 11155111, "Ethereum Sepolia", "ETH", "0x", 18, "https://sepolia.infura.io/v3/", "https://sepolia.etherscan.io", "m/44'/60'/0'/0/0", true},
        
        // EVM Chains
        {56, 56, "BNB Smart Chain", "BNB", "0x", 18, "https://bsc-dataseed1.binance.org", "https://bscscan.com", "m/44'/60'/0'/0/0", true},
        {56, 97, "BNB Testnet", "BNB", "0x", 18, "https://data-seed-prebsc-1-s1.bnb.org:8545", "https://testnet.bscscan.com", "m/44'/60'/0'/0/0", true},
        {137, 137, "Polygon", "MATIC", "0x", 18, "https://polygon-rpc.com", "https://polygonscan.com", "m/44'/60'/0'/0/0", true},
        {137, 80001, "Polygon Mumbai", "MATIC", "0x", 18, "https://rpc-mumbai.maticvigil.com", "https://mumbai.polygonscan.com", "m/44'/60'/0'/0/0", true},
        {42161, 42161, "Arbitrum One", "ETH", "0x", 18, "https://arb1.arbitrum.io/rpc", "https://arbiscan.io", "m/44'/60'/0'/0/0", true},
        {10, 10, "Optimism", "ETH", "0x", 18, "https://mainnet.optimism.io", "https://optimistic.etherscan.io", "m/44'/60'/0'/0/0", true},
        {43114, 43114, "Avalanche C-Chain", "AVAX", "0x", 18, "https://api.avax.network/ext/bc/C/rpc", "https://snowtrace.io", "m/44'/60'/0'/0/0", true},
        {43114, 43113, "Avalanche Fuji", "AVAX", "0x", 18, "https://api.avax-test.network/ext/bc/C/rpc", "https://testnet.snowtrace.io", "m/44'/60'/0'/0/0", true},
        {8453, 8453, "Base", "ETH", "0x", 18, "https://mainnet.base.org", "https://basescan.org", "m/44'/60'/0'/0/0", true},
        {25, 25, "Cronos", "CRO", "0x", 18, "https://evm.cronos.org", "https://cronoscan.com", "m/44'/60'/0'/0/0", true},
        {42220, 42220, "Celo", "CELO", "0x", 18, "https://forno.celo.org", "https://explorer.celo.org", "m/44'/60'/0'/0/0", true},
        {8217, 8217, "Klaytn", "KLAY", "0x", 18, "https://klaytn.fandom.finance", "https://scope.klaytn.com", "m/44'/60'/0'/0/0", true},
        {1284, 1284, "Moonbeam", "GLMR", "0x", 18, "https://rpc.api.moonbeam.network", "https://moonbeam.moonscan.io", "m/44'/60'/0'/0/0", true},
        {1285, 1285, "Moonriver", "MOVR", "0x", 18, "https://rpc.api.moonriver.moonbeam.network", "https://moonriver.moonscan.io", "m/44'/60'/0'/0/0", true},
        {4002, 4002, "Fantom", "FTM", "0x", 18, "https://rpc.fantom.network", "https://ftmscan.com", "m/44'/60'/0'/0/0", true},
        {1023, 1023, "Harmony", "ONE", "0x", 18, "https://api.harmony.one", "https://explorer.harmony.one", "m/44'/60'/0'/0/0", true},
        {324, 324, "zkSync Era", "ETH", "0x", 18, "https://mainnet.era.zksync.io", "https://explorer.zksync.io", "m/44'/60'/0'/0/0", true},
        {59144, 59144, "Linea", "ETH", "0x", 18, "https://rpc.linea.build", "https://lineascan.build", "m/44'/60'/0'/0/0", true},
        {534352, 534352, "Scroll", "ETH", "0x", 18, "https://scroll.io/rpc", "https://scrollscan.com", "m/44'/60'/0'/0/0", true},
        
        // Solana
        {501, 101, "Solana", "SOL", "", 9, "https://api.mainnet-beta.solana.com", "https://solscan.io", "m/44'/501'/0'/0'", false},
        {501, 103, "Solana Devnet", "SOL", "", 9, "https://api.devnet.solana.com", "https://solscan.io", "m/44'/501'/0'/0'", false},
        
        // Cosmos Family
        {118, 1, "Cosmos Hub", "ATOM", "cosmos", 6, "https://cosmos-rpc.polkachu.com", "https://mintscan.io/cosmos", "m/44'/118'/0'/0/0", false},
        {459, 1, "Kava", "KAVA", "kava", 6, "https://rpc-kava.securechain.info", "https://mintscan.io/kava", "m/44'/459'/0'/0/0", false},
        {529, 1, "Secret Network", "SCRT", "secret", 6, "https://rpc.secret.expert", "https://mintscan.io/secret", "m/44'/529'/0'/0/0", false},
        {511, 1, "Juno", "JUNO", "juno", 6, "https://juno.lcd.stakingresort.com", "https://mintscan.io/juno", "m/44'/511'/0'/0/0", false},
        {464, 1, "Osmosis", "OSMO", "osmo", 6, "https://osmosis-rpc.polkachu.com", "https://mintscan.io/osmosis", "m/44'/464'/0'/0/0", false},
        {9001, 1, "Evmos", "EVMOS", "evmos", 18, "https://evmos-rpc.polkachu.com", "https://mintscan.io/evmos", "m/44'/9001'/0'/0/0", true},
        
        // TRON
        {195, 7281265, "TRON", "TRX", "T", 6, "https://api.trongrid.io", "https://tronscan.org", "m/44'/195'/0'/0/0", false},
        
        // Algorand
        {283, 0, "Algorand", "ALGO", "", 6, "https://mainnet-api.algorand.network", "https://algoexplorer.io", "m/44'/283'/0'/0/0", false},
        
        // NEAR
        {397, 0, "NEAR", "NEAR", "", 24, "https://rpc.mainnet.near.org", "https://explorer.near.org", "m/44'/397'/0'/0", false},
        
        // Aptos
        {637, 1, "Aptos", "APT", "0x", 8, "https://fullnode.mainnet.aptoslabs.com", "https://aptoscan.com", "m/44'/637'/0'/0'/0", false},
        
        // Sui
        {784, 1, "Sui", "SUI", "0x", 9, "https://fullnode.mainnet.sui.io", "https://suiexplorer.com", "m/44'/784'/0'/0'/0", false},
        
        // TON
        {607, 0, "TON", "TON", "0:", 9, "https://toncenter.com/api/v2/", "https://tonscan.org", "m/44'/607'/0'/0/0", false},
        
        // Polkadot
        {354, 0, "Polkadot", "DOT", "", 10, "https://rpc.polkadot.io", "https://polkadot.subscan.io", "//0//0", false},
        {434, 0, "Kusama", "KSM", "", 12, "https://rpc.kusama.network", "https://kusama.subscan.io", "//0//0", false},
        
        // Cardano
        {1815, 1, "Cardano", "ADA", "", 6, "https://cardano-mainnet.blockfrost.io", "https://cardanoscan.io", "m/44'/1815'/0'/0/0", false},
        
        // Stellar
        {148, 0, "Stellar", "XLM", "G", 7, "https://horizon.stellar.org", "https://stellar.expert", "m/44'/148'/0'/0/0", false},
        
        // Ripple
        {144, 0, "XRP", "XRP", "r", 6, "https://xrplcluster.org", "https://xrpscan.com", "m/44'/144'/0'/0/0", false},
        
        // Tezos
        {1729, 1, "Tezos", "XTZ", "tz1", 6, "https://mainnet.api.tez.ie", "https://tzstats.com", "m/44'/1729'/0'/0/0", false},
        
        // VeChain
        {818, 1, "VeChain", "VET", "0x", 18, "https://mainnet.infurenet.com", "https://vechainstats.com", "m/44'/818'/0'/0/0", false},
        
        // ICON
        {74, 1, "ICON", "ICX", "hx", 18, "https://api.icon.community", "https://tracker.icon.community", "m/44'/74'/0'/0/0", false},
        
        // Qtum
        {2301, 1, "Qtum", "QTUM", "0x", 8, "https://qtum-rpc.firecore.co", "https://qtum.info", "m/44'/2301'/0'/0/0", false},
        
        // Firo
        {157, 1, "Firo", "FIRO", "", 8, "https://api.firo.org", "https://explorer.firo.org", "m/44'/157'/0'/0/0", false},
        
        // Decred
        {42, 1, "Decred", "DCR", "Ds", 8, "https://dcrdata.decred.org", "https://explorer.dcrdata.org", "m/44'/42'/0'/0/0", false},
        
        // Kaspa
        {111111, 0, "Kaspa", "KAS", "kaspa:", 8, "https://kaspad.kaspa.org", "https://explorer.kaspa.org", "m/44'/111111'/0'/0/0", false},
        
        // Neo
        {888, 1, "Neo", "NEO", "N", 8, "https://rpc1.nelify.org", "https://neotube.io", "m/44'/888'/0'/0/0", false},
        
        // Zilliqa
        {119, 1, "Zilliqa", "ZIL", "zil1", 12, "https://api.zilliqa.com", "https://viewblock.io/zilliqa", "m/44'/119'/0'/0/0", false},
        
        // Thorchain
        {931, 1, "THORChain", "RUNE", "thor1", 8, "https://rpc.thorchain.info", "https://viewblock.io/thorchain", "m/44'/931'/0'/0/0", false},
        
        // Celestia
        {118, 1, "Celestia", "TIA", "celestia", 6, "https://celestia-rpc.polkachu.com", "https://mintscan.io/celestia", "m/44'/118'/0'/0/0", false},
        
        // Injective
        {118, 1, "Injective", "INJ", "inj", 18, "https://injective-rpc.polkachu.com", "https://mintscan.io/injective", "m/44'/118'/0'/0/0", false},
        
        // Sei
        {118, 1, "Sei", "SEI", "sei1", 6, "https://sei-rpc.polkachu.com", "https://mintscan.io/sei", "m/44'/118'/0'/0/0", false},
        
        // Stride
        {118, 1, "Stride", "STRD", "stride1", 6, "https://stride-rpc.polkachu.com", "https://mintscan.io/stride", "m/44'/118'/0'/0/0", false},
        
        // dYdX
        {118, 1, "dYdX", "DYDX", "dydx1", 18, "https://dydx-rpc.polkachu.com", "https://mintscan.io/dydx", "m/44'/118'/0'/0/0", false},
    };
}

} // namespace TigerWalletCore

#endif // TWCORE_BLOCKCHAIN_FACTORY_H
