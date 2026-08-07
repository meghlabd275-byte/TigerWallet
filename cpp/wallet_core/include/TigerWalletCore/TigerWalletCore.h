/**
 * TigerWalletCore - High-Performance Cross-Platform Wallet Library
 * 
 * Copyright (c) 2024 TigerWallet
 * All rights reserved.
 * 
 * This is the main header file for TigerWalletCore - a comprehensive,
 * production-grade wallet library supporting 130+ blockchains.
 * 
 * Features:
 * - 130+ blockchain support (Bitcoin, Ethereum, Solana, Cosmos, Polkadot, etc.)
 * - Advanced cryptography (secp256k1, ed25519, sr25519, Schnorr signatures)
 * - Hardware wallet support (Trezor, Ledger)
 * - Multi-party computation (MPC)
 * - Account abstraction (EIP-4337)
 * - Lightning Network support
 * - Ultra-low latency design
 * 
 * @author TigerWallet Team
 * @version 2.0.0
 */

#ifndef TIGER_WALLET_CORE_H
#define TIGER_WALLET_CORE_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// ============================================================================
// Version Information
// ============================================================================

#define TWCORE_VERSION_MAJOR 2
#define TWCORE_VERSION_MINOR 0
#define TWCORE_VERSION_PATCH 0
#define TWCORE_VERSION_STRING "2.0.0"

// ============================================================================
// Error Codes
// ============================================================================

typedef enum {
    TWCORE_SUCCESS = 0,
    TWCORE_ERROR_INVALID_PARAMETER = -1,
    TWCORE_ERROR_INVALID_ADDRESS = -2,
    TWCORE_ERROR_INVALID_MNEMONIC = -3,
    TWCORE_ERROR_INVALID_KEY = -4,
    TWCORE_ERROR_INVALID_SIGNATURE = -5,
    TWCORE_ERROR_INVALID_TRANSACTION = -6,
    TWCORE_ERROR_INSUFFICIENT_BALANCE = -7,
    TWCORE_ERROR_INSUFFICIENT_GAS = -8,
    TWCORE_ERROR_INVALID_CHAIN = -9,
    TWCORE_ERROR_NOT_SUPPORTED = -10,
    TWCORE_ERROR_HARDWARE_NOT_CONNECTED = -11,
    TWCORE_ERROR_HARDWARE_PIN_REQUIRED = -12,
    TWCORE_ERROR_HARDWARE_PIN_INVALID = -13,
    TWCORE_ERROR_HARDWARE_ACTION_CANCELLED = -14,
    TWCORE_ERROR_CRYPTO_ERROR = -15,
    TWCORE_ERROR_ENCODING_ERROR = -16,
    TWCORE_ERROR_DECODING_ERROR = -17,
    TWCORE_ERROR_NETWORK_ERROR = -18,
    TWCORE_ERROR_TIMEOUT = -19,
    TWCORE_ERROR_OUT_OF_MEMORY = -20,
    TWCORE_ERROR_BUFFER_TOO_SMALL = -21,
    TWCORE_ERROR_INVALID_DERIVATION_PATH = -22,
    TWCORE_ERROR_ACCOUNT_ABSTRACTION = -23,
    TWCORE_ERROR_MULTISIG = -24,
    TWCORE_ERROR_MPC = -25,
    TWCORE_ERROR_UNKNOWN = -999
} TWCoreError;

// ============================================================================
// Blockchain Types (Coin Types from SLIP-0044)
// ============================================================================

typedef enum {
    // Bitcoin Family
    TWCORE_COIN_BITCOIN = 0,
    TWCORE_COIN_BITCOIN_CASH = 145,
    TWCORE_COIN_LITECOIN = 2,
    TWCORE_COIN_DOGECOIN = 3,
    TWCORE_COIN_DASH = 5,
    TWCORE_COIN_ZCASH = 133,
    TWCORE_COIN_LITECOIN = 2,
    TWCORE_COIN_RIEGEL = 276,
    TWCORE_COIN_DIGIBYTE = 20,
    TWCORE_COIN_MONACOIN = 22,
    TWCORE_COIN_VIPSTARCOIN = 26,
    TWCORE_COIN_REDDCOIN = 4,
    TWCORE_COIN_SYNC = 296,
    TWCORE_COIN_ETHEREUM_CLASSIC = 61,
    TWCORE_COIN_EXPANSE = 40,
    TWCORE_COIN_CALLISTO = 820,
    TWCORE_COIN_TOMOCHAIN = 889,
    TWCORE_COIN_GOCHAIN = 6060,
    TWCORE_COIN_EOSIO = 194,
    TWCORE_COIN_TRON = 195,
    TWCORE_COIN_BINANCE = 714,
    TWCORE_COIN_POLKADOT = 354,
    TWCORE_COIN_KUSAMA = 434,
    TWCORE_COIN_ACALA = 787,
    TWCORE_COIN_ASTAR = 810,
    TWCORE_COIN_MOONBEAM = 1284,
    TWCORE_COIN_MOONRIVER = 1285,
    
    // Ethereum Family
    TWCORE_COIN_ETHEREUM = 60,
    TWCORE_COIN_POLYGON = 966,
    TWCORE_COIN_BNB_SMART_CHAIN = 20000714,
    TWCORE_COIN_ARBITRUM = 11021,
    TWCORE_COIN_OPTIMISM = 10,
    TWCORE_COIN_CELO = 52752,
    TWCORE_COIN_AVALANCHE_C = 9000,
    TWCORE_COIN_FANTOM = 4000,
    TWCORE_COIN_HARMONY = 1023,
    TWCORE_COIN_CRONOS = 25,
    TWCORE_COIN_KLAYTN = 8217,
    TWCORE_COIN_OASIS = 47452,
    TWCORE_COIN_GATHER = 1908,
    TWCORE_COIN_REDFIRE = 170010,
    TWCORE_COIN_SMARTCASH = 2241,
    TWCORE_COIN_DEX = 10000714,
    TWCORE_COIN_DEXIT = 10000740,
    TWCORE_COIN_PALM = 11297108109,
    TWCORE_COIN_SECRET = 529,
    TWCORE_COIN_THORCHAIN = 931,
    TWCORE_COIN_BIFROST = 4294967293ULL,
    TWCORE_COIN_AURORA = 1313161556,
    TWCORE_COIN_CRV = 10000061,
    TWCORE_COIN_CELO_DOLLAR = 0,
    TWCORE_COIN_NEAR = 397,
    
    // Cosmos Family
    TWCORE_COIN_COSMOS = 118,
    TWCORE_COIN_OSMOSIS = 464,
    TWCORE_COIN_TERRA = 330,
    TWCORE_COIN_TERRA_V2 = 0,
    TWCORE_COIN_JUNO = 511,
    TWCORE_COIN_STARGAZE = 118,
    TWCORE_COIN_CRESCENT = 118,
    TWCORE_COIN_INJECTIVE = 118,
    TWCORE_COIN_CRONOS_POS = 60,
    TWCORE_COIN_EVMOS = 9001,
    TWCORE_COIN_OKC = 996,
    TWCORE_COIN_CELO_ORGANUT = 52752,
    TWCORE_COIN_REDDGLOBE = 6000,
    TWCORE_COIN_GIA = 29000,
    TWCORE_COIN_PROVENANCE = 5050,
    TWCORE_COIN_CHIHUAHUA = 118,
    TWCORE_COIN_KAVA = 459,
    TWCORE_COIN_NEUTRON = 118,
    TWCORE_COIN_QUICK = 118,
    TWCORE_COIN_TICKER = 10000000,
    TWCORE_COIN_CELESTIA = 118,
    
    // Solana Family
    TWCORE_COIN_SOLANA = 501,
    TWCORE_COIN_POOL = 501,
    
    // Algorand Family
    TWCORE_COIN_ALGORAND = 283,
    
    // NEAR Family
    TWCORE_COIN_NEAR = 397,
    
    // Aptos Family
    TWCORE_COIN_APTOS = 637,
    
    // Sui Family
    TWCORE_COIN_SUI = 784,
    
    // Ton Family
    TWCORE_COIN_TON = 607,
    
    // Stellar Family
    TWCORE_COIN_STELLAR = 148,
    TWCORE_COIN_XLM = 148,
    
    // Ripple Family
    TWCORE_COIN_RIPPLE = 144,
    TWCORE_COIN_XRP = 144,
    
    // Tezos Family
    TWCORE_COIN_TEZOS = 1729,
    TWCORE_COIN_XTZ = 1729,
    
    // Cardano Family
    TWCORE_COIN_CARDANO = 1815,
    TWCORE_COIN_ADA = 1815,
    
    // VeChain Family
    TWCORE_COIN_VECHAIN = 818,
    TWCORE_COIN_VET = 818,
    
    // ICON Family
    TWCORE_COIN_ICON = 74,
    TWCORE_COIN_ICX = 74,
    
    // Harmony
    TWCORE_COIN_HARMONY = 1023,
    
    // Qtum
    TWCORE_COIN_QTUM = 2301,
    
    // VeThor
    TWCORE_COIN_VTHOR = 10000714,
    
    // Polygon
    TWCORE_COIN_POLYGON = 966,
    
    // Base
    TWCORE_COIN_BASE = 8453,
    
    // zkSync Era
    TWCORE_COIN_ZKSYNC_ERA = 324,
    
    // Linea
    TWCORE_COIN_LINEA = 59144,
    
    // Scroll
    TWCORE_COIN_SCROLL = 534352,
    
    // Mona
    TWCORE_COIN_MONA = 22,
    
    // Firo
    TWCORE_COIN_FIRO = 157,
    
    // Decred
    TWCORE_COIN_DECRED = 42,
    
    // Groestlcoin
    TWCORE_COIN_GROESTLCOIN = 17,
    
    // Diamond
    TWCORE_COIN_DIAMOND = 0x1f5b3,
    
    // Particl
    TWCORE_COIN_PARTICL = 44,
    
    // Bread
    TWCORE_COIN_BREAD = 0x1f5b4,
    
    // DeepOnion
    TWCORE_COIN_DEEPONION = 305,
    
    // XINFIN
    TWCORE_COIN_XINFIN = 550,
    
    // AION
    TWCORE_COIN_AION = 425,
    
    // Nimiq
    TWCORE_COIN_NIMIQ = 242,
    
    // Ellaism
    TWCORE_COIN_ELLAISM = 163,
    
    // ETHEREUM_ZERO
    TWCORE_COIN_ETHEREUM_ZERO = 10000061,
    
    // Mixin
    TWCORE_COIN_MIXIN = 500,
    
    // Polymath
    TWCORE_COIN_POLYMATH = 2917,
    
    // Woocommerce
    TWCORE_COIN_WOOCOMMERCE = 0x1f5b5,
    
    // Berith
    TWCORE_COIN_BERITH = 12051,
    
    // Ankr
    TWCORE_COIN_ANKR = 10000714,
    
    // AlveyChain
    TWCORE_COIN_ALVEYCHAIN = 2910,
    
    // Radicle
    TWCORE_COIN_RADICLE = 0x1f5b6,
    
    // Adcoin
    TWCORE_COIN_ADCOIN = 259,
    
    // Tausch
    TWCORE_COIN_TAUSCH = 0x1f5b7,
    
    // GoChain
    TWCORE_COIN_GOCHAIN = 6060,
    
    // Filecoin
    TWCORE_COIN_FILECOIN = 461,
    
    // Livepeer
    TWCORE_COIN_LIVEPER = 46041,
    
    // Flow
    TWCORE_COIN_FLOW = 539,
    
    // Mask
    TWCORE_COIN_MASK = 10000714,
    
    // SuperRare
    TWCORE_COIN_SUPERRARE = 10000714,
    TWCORE_COIN_RUNE = 10000714,
    TWCORE_COIN_KASPA = 111111,
    TWCORE_COIN_NEO = 888,
    TWCORE_COIN_NEO3 = 888,
    TWCORE_COIN_COMDEX = 10000714,
    TWCORE_COIN_STAKECUBE = 10000714,
    TWCORE_COIN_ZIL = 119,
    TWCORE_COIN_IRIS = 566,
    TWCORE_COIN_RACEFI = 10000714,
    TWCORE_COIN_THETA = 500,
    TWCORE_COIN_SCRT = 529,
    TWCORE_COIN_LUNA = 330,
    TWCORE_COIN_LUNA_V2 = 0,
    TWCORE_COIN_BTT = 10000714,
    TWCORE_COIN_TT = 10000714,
    TWCORE_COIN_BSC = 20000714,
    TWCORE_COIN_BEP2 = 714,
    TWCORE_COIN_FIO = 235111,
    TWCORE_COIN_INFINITUS = 10000714,
    TWCORE_COIN_KARDIA = 10000714,
    TWCORE_COIN_METER = 18000,
    TWCORE_COIN_NGL = 10000714,
    TWCORE_COIN_XDC = 10000714,
    TWCORE_COIN_SXP = 10000714,
    TWCORE_COIN_SPECTRE = 10000714,
    TWCORE_COIN_DVPN = 10000714,
    TWCORE_COIN_ARDR = 273,
    TWCORE_COIN_NXT = 78,
    TWCORE_COIN_WAVES = 5741564,
    TWCORE_COIN_AUGUR = 558,
    TWCORE_COIN_DAI = 10000714,
    TWCORE_COIN_USDT = 10000714,
    TWCORE_COIN_USDC = 10000714,
    TWCORE_COIN_MIM = 10000714,
    TWCORE_COIN_1INCH = 10000714,
    TWCORE_COIN_CRVRENBTC = 10000714,
    TWCORE_COIN_RENBTC = 10000714,
    TWCORE_COIN_RENBCH = 10000714,
    TWCORE_COIN_RENZEC = 10000714,
    TWCORE_COIN_RENFIL = 10000714,
    TWCORE_COIN_OBITS = 10000714,
    TWCORE_COIN_ENJINCOIN = 121,
    TWCORE_COIN_CRYPTO_COM = 10000714,
    TWCORE_COIN_CRO = 10000714,
    TWCORE_COIN_CRV = 10000714,
    TWCORE_COIN_SNX = 10000714,
    TWCORE_COIN_YFI = 10000714,
    TWCORE_COIN_COMP = 10000714,
    TWCORE_COIN_BAL = 10000714,
    TWCORE_COIN_UNI = 10000714,
    TWCORE_COIN_SUSHI = 10000714,
    TWCORE_COIN_FETCH = 10000714,
    TWCORE_COIN_BAND = 10000714,
    TWCORE_COIN_SAND = 10000714,
    TWCORE_COIN_MAN = 10000714,
    TWCORE_COIN_ALICE = 10000714,
    TWCORE_COIN_BOR = 10000714,
    TWCORE_COIN_SXP = 10000714,
    TWCORE_COIN_ATRI = 10000714,
    TWCORE_COIN_MTR = 10000714,
    TWCORE_COIN_WMT = 10000714,
    TWCORE_COIN_PLG = 10000714,
    TWCORE_COIN_EGLD = 508,
    TWCORE_COIN_ZRX = 10000714,
    TWCORE_COIN_AMLT = 10000714,
    TWCORE_COIN_GMT = 10000714,
    TWCORE_COIN_BTTOLD = 10000714,
    TWCORE_COIN_WEMIX = 10000714,
    TWCORE_COIN_APT = 637,
    TWCORE_COIN_ARB = 11021,
    TWCORE_COIN_OP = 10,
    TWCORE_COIN_ZK = 324,
    
    // Default
    TWCORE_COIN_UNKNOWN = 0xFFFFFFFF
} TWCoreCoinType;

// ============================================================================
// Transaction Types
// ============================================================================

typedef enum {
    TWCORE_TX_TYPE_LEGACY = 0,
    TWCORE_TX_TYPE_EIP2930 = 1,
    TWCORE_TX_TYPE_EIP1559 = 2,
    TWCORE_TX_TYPE_EIP4844 = 3,
    TWCORE_TX_TYPE_BITCOIN_UTXO = 10,
    TWCORE_TX_TYPE_COSMOS = 20,
    TWCORE_TX_TYPE_SOLANA = 30,
    TWCORE_TX_TYPE_NEAR = 40,
    TWCORE_TX_TYPE_APTOS = 50,
    TWCORE_TX_TYPE_SUI = 60,
    TWCORE_TX_TYPE_TON = 70
} TWCoreTxType;

// ============================================================================
// Address Types
// ============================================================================

typedef enum {
    TWCORE_ADDRESS_TYPE_LEGACY = 0,
    TWCORE_ADDRESS_TYPE_P2SH = 1,
    TWCORE_ADDRESS_TYPE_P2WPKH = 2,
    TWCORE_ADDRESS_TYPE_P2WSH = 3,
    TWCORE_ADDRESS_TYPE_P2TR = 4,
    TWCORE_ADDRESS_TYPE_EOA = 10,
    TWCORE_ADDRESS_TYPE_CONTRACT = 11,
    TWCORE_ADDRESS_TYPE_ERC4337 = 12
} TWCoreAddressType;

// ============================================================================
// Key Types
// ============================================================================

typedef enum {
    TWCORE_KEY_TYPE_SECP256K1 = 0,
    TWCORE_KEY_TYPE_ED25519 = 1,
    TWCORE_KEY_TYPE_SR25519 = 2,
    TWCORE_KEY_TYPE_BLS12_381 = 3,
    TWCORE_KEY_TYPE_RSA = 4,
    TWCORE_KEY_TYPE_STARKNET = 5
} TWCoreKeyType;

// ============================================================================
// Wallet Types
// ============================================================================

typedef enum {
    TWCORE_WALLET_TYPE_EOA = 0,
    TWCORE_WALLET_TYPE_SMART_CONTRACT = 1,
    TWCORE_WALLET_TYPE_MULTISIG = 2,
    TWCORE_WALLET_TYPE_MPC = 3,
    TWCORE_WALLET_TYPE_HARDWARE = 4,
    TWCORE_WALLET_TYPE_MULTI_SIG = 5,
    TWCORE_WALLET_TYPE_SOCIAL_RECOVERY = 6
} TWCoreWalletType;

// ============================================================================
// Network Types
// ============================================================================

typedef enum {
    TWCORE_NETWORK_MAINNET = 0,
    TWCORE_NETWORK_TESTNET = 1,
    TWCORE_NETWORK_DEVNET = 2,
    TWCORE_NETWORK_BETANET = 3,
    TWCORE_NETWORK_REGTEST = 4
} TWCoreNetwork;

// ============================================================================
// Opaque Structures
// ============================================================================

typedef struct TWPrivateKey TWPrivateKey;
typedef struct TWPublicKey TWPublicKey;
typedef struct TWAddress TWAddress;
typedef struct TWMnemonic TWMnemonic;
typedef struct TWWallet TWWallet;
typedef struct TWTransaction TWTransaction;
typedef struct TWBitcoinTransaction TWBitcoinTransaction;
typedef struct TWEVMTransaction TWEVMTransaction;
typedef struct TWCosmosTransaction TWCosmosTransaction;
typedef struct TWSolanaTransaction TWSolanaTransaction;
typedef struct TWAccountAbstraction TWAccountAbstraction;
typedef struct TWMultisig TWMultisig;
typedef struct TWMPC TWMPC;
typedef struct TWBitcoinUTXO TWBitcoinUTXO;
typedef struct TWAccount TWAccount;
typedef struct TWToken TWToken;
typedef struct TWChainInfo TWChainInfo;

// ============================================================================
// Mnemonic Functions
// ============================================================================

/**
 * Generate a new mnemonic phrase
 * @param wordCount Number of words (12, 15, 18, 21, or 24)
 * @return Newly allocated mnemonic string (must be freed with tw_mnemonic_delete)
 */
TWMnemonic* tw_mnemonic_generate(int wordCount);

/**
 * Create mnemonic from existing phrase
 * @param phrase Mnemonic phrase string
 * @return Newly allocated mnemonic or NULL on error
 */
TWMnemonic* tw_mnemonic_from_phrase(const char* phrase);

/**
 * Validate a mnemonic phrase
 * @param phrase Mnemonic phrase to validate
 * @return true if valid, false otherwise
 */
bool tw_mnemonic_is_valid(const char* phrase);

/**
 * Convert mnemonic to seed
 * @param mnemonic Mnemonic to convert
 * @param passphrase Optional passphrase (can be empty string)
 * @param output Output buffer (64 bytes)
 * @param outputSize Size of output buffer
 * @return TWCORE_SUCCESS on success, error code on failure
 */
TWCoreError tw_mnemonic_to_seed(const TWMnemonic* mnemonic, const char* passphrase, 
                                  uint8_t* output, size_t outputSize);

/**
 * Get mnemonic word list language
 * @param mnemonic Mnemonic to query
 * @return Language code string (en, zh, ja, ko, etc.)
 */
const char* tw_mnemonic_get_language(const TWMnemonic* mnemonic);

/**
 * Free mnemonic memory
 * @param mnemonic Mnemonic to free
 */
void tw_mnemonic_delete(TWMnemonic* mnemonic);

// ============================================================================
// Private Key Functions
// ============================================================================

/**
 * Create private key from seed
 * @param seed Seed bytes
 * @param seedLen Length of seed
 * @param keyType Type of key (secp256k1, ed25519, etc.)
 * @return Newly allocated private key or NULL on error
 */
TWPrivateKey* tw_private_key_from_seed(const uint8_t* seed, size_t seedLen, TWCoreKeyType keyType);

/**
 * Create private key from hex string
 * @param hex Hex string (with or without 0x prefix)
 * @return Newly allocated private key or NULL on error
 */
TWPrivateKey* tw_private_key_from_hex(const char* hex);

/**
 * Create private key from WIF
 * @param wif WIF string
 * @return Newly allocated private key or NULL on error
 */
TWPrivateKey* tw_private_key_from_wif(const char* wif);

/**
 * Get public key from private key
 * @param privateKey Private key
 * @param compressed Whether to return compressed public key
 * @return Newly allocated public key or NULL on error
 */
TWPublicKey* tw_private_key_get_public_key(const TWPrivateKey* privateKey, bool compressed);

/**
 * Get private key as hex string
 * @param privateKey Private key
 * @return Hex string (must be freed)
 */
char* tw_private_key_to_hex(const TWPrivateKey* privateKey);

/**
 * Sign data with private key
 * @param privateKey Private key to sign with
 * @param data Data to sign
 * @param dataLen Length of data
 * @param signature Output signature buffer
 * @param signatureSize Size of signature buffer
 * @param sigType Type of signature (ECDSA, Schnorr, etc.)
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_private_key_sign(const TWPrivateKey* privateKey,
                                const uint8_t* data, size_t dataLen,
                                uint8_t* signature, size_t* signatureSize,
                                TWCoreKeyType sigType);

/**
 * Sign hash with private key (for Bitcoin)
 * @param privateKey Private key
 * @param hash Hash to sign (32 bytes)
 * @param signature Output signature buffer
 * @param signatureSize Size of signature buffer
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_private_key_sign_hash(const TWPrivateKey* privateKey,
                                     const uint8_t* hash,
                                     uint8_t* signature, size_t* signatureSize);

/**
 * Free private key memory
 * @param privateKey Private key to free
 */
void tw_private_key_delete(TWPrivateKey* privateKey);

// ============================================================================
// Public Key Functions
// ============================================================================

/**
 * Create public key from bytes
 * @param bytes Public key bytes
 * @param bytesLen Length of bytes
 * @param keyType Type of key
 * @return Newly allocated public key or NULL
 */
TWPublicKey* tw_public_key_from_bytes(const uint8_t* bytes, size_t bytesLen, TWCoreKeyType keyType);

/**
 * Get public key as hex string
 * @param publicKey Public key
 * @param compressed Whether to return compressed format
 * @return Hex string (must be freed)
 */
char* tw_public_key_to_hex(const TWPublicKey* publicKey, bool compressed);

/**
 * Verify signature with public key
 * @param publicKey Public key
 * @param data Data that was signed
 * @param dataLen Length of data
 * @param signature Signature bytes
 * @param signatureLen Length of signature
 * @param keyType Type of key/signature
 * @return true if signature is valid
 */
bool tw_public_key_verify(const TWPublicKey* publicKey,
                          const uint8_t* data, size_t dataLen,
                          const uint8_t* signature, size_t signatureLen,
                          TWCoreKeyType keyType);

/**
 * Get key ID (hash of public key)
 * @param publicKey Public key
 * @return Key ID string (must be freed)
 */
char* tw_public_key_get_key_id(const TWPublicKey* publicKey);

/**
 * Free public key memory
 * @param publicKey Public key to free
 */
void tw_public_key_delete(TWPublicKey* publicKey);

// ============================================================================
// Address Functions
// ============================================================================

/**
 * Create address from public key
 * @param publicKey Public key
 * @param coinType Blockchain type
 * @return Newly allocated address or NULL
 */
TWAddress* tw_address_from_public_key(const TWPublicKey* publicKey, TWCoreCoinType coinType);

/**
 * Create address from private key
 * @param privateKey Private key
 * @param coinType Blockchain type
 * @return Newly allocated address or NULL
 */
TWAddress* tw_address_from_private_key(const TWPrivateKey* privateKey, TWCoreCoinType coinType);

/**
 * Create address from hex string
 * @param addressString Address string
 * @param coinType Blockchain type
 * @return Newly allocated address or NULL
 */
TWAddress* tw_address_from_string(const char* addressString, TWCoreCoinType coinType);

/**
 * Get address as string
 * @param address Address
 * @return Address string (must be freed)
 */
char* tw_address_to_string(const TWAddress* address);

/**
 * Validate address format
 * @param addressString Address string to validate
 * @param coinType Blockchain type
 * @return true if valid
 */
bool tw_address_is_valid(const char* addressString, TWCoreCoinType coinType);

/**
 * Get address from derivation path
 * @param seed Seed bytes
 * @param seedLen Length of seed
 * @param coinType Blockchain type
 * @param derivationPath BIP-44 derivation path
 * @return Newly allocated address or NULL
 */
TWAddress* tw_address_from_derivation_path(const uint8_t* seed, size_t seedLen,
                                            TWCoreCoinType coinType, const char* derivationPath);

/**
 * Free address memory
 * @param address Address to free
 */
void tw_address_delete(TWAddress* address);

// ============================================================================
// Wallet Functions
// ============================================================================

/**
 * Create new HD wallet
 * @param mnemonic Mnemonic phrase
 * @return Newly allocated wallet or NULL
 */
TWWallet* tw_wallet_create(const TWMnemonic* mnemonic);

/**
 * Create wallet from seed
 * @param seed Seed bytes
 * @param seedLen Length of seed
 * @return Newly allocated wallet or NULL
 */
TWWallet* tw_wallet_create_from_seed(const uint8_t* seed, size_t seedLen);

/**
 * Get wallet address for coin
 * @param wallet Wallet
 * @param coinType Blockchain type
 * @return Address string (must be freed)
 */
char* tw_wallet_get_address(const TWWallet* wallet, TWCoreCoinType coinType);

/**
 * Get wallet address for coin with derivation index
 * @param wallet Wallet
 * @param coinType Blockchain type
 * @param derivationIndex Derivation index
 * @return Address string (must be freed)
 */
char* tw_wallet_get_address_at_index(const TWWallet* wallet, TWCoreCoinType coinType, uint32_t derivationIndex);

/**
 * Get private key for coin
 * @param wallet Wallet
 * @param coinType Blockchain type
 * @return Private key (must be freed)
 */
TWPrivateKey* tw_wallet_get_private_key(const TWWallet* wallet, TWCoreCoinType coinType);

/**
 * Get private key for coin at index
 * @param wallet Wallet
 * @param coinType Blockchain type
 * @param derivationIndex Derivation index
 * @return Private key (must be freed)
 */
TWPrivateKey* tw_wallet_get_private_key_at_index(const TWWallet* wallet, TWCoreCoinType coinType, uint32_t derivationIndex);

/**
 * Get all addresses from wallet
 * @param wallet Wallet
 * @param addresses Output array of address strings (must be freed each and array)
 * @param addressesCount Output count
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_wallet_get_all_addresses(const TWWallet* wallet, 
                                         char*** addresses, size_t* addressesCount);

/**
 * Free wallet memory
 * @param wallet Wallet to free
 */
void tw_wallet_delete(TWWallet* wallet);

// ============================================================================
// Bitcoin Transaction Functions
// ============================================================================

/**
 * Create new Bitcoin transaction
 * @return Newly allocated transaction
 */
TWBitcoinTransaction* tw_bitcoin_transaction_create(void);

/**
 * Add UTXO input to transaction
 * @param tx Transaction
 * @param txId Previous transaction ID (32 bytes)
 * @param vout Output index
 * @param amount Satoshi amount
 * @param script Script bytes
 * @param scriptLen Script length
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_bitcoin_transaction_add_input(TWBitcoinTransaction* tx,
                                             const uint8_t* txId, uint32_t vout,
                                             uint64_t amount,
                                             const uint8_t* script, size_t scriptLen);

/**
 * Add output to transaction
 * @param tx Transaction
 * @param address Destination address
 * @param amount Satoshi amount
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_bitcoin_transaction_add_output(TWBitcoinTransaction* tx,
                                               const char* address, uint64_t amount);

/**
 * Sign Bitcoin transaction
 * @param tx Transaction to sign
 * @param privateKey Private key
 * @param utxos UTXO array for signing
 * @param utxoCount Number of UTXOs
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_bitcoin_transaction_sign(TWBitcoinTransaction* tx,
                                          const TWPrivateKey* privateKey,
                                          const TWBitcoinUTXO* utxos, size_t utxoCount);

/**
 * Get transaction as hex
 * @param tx Transaction
 * @return Hex string (must be freed)
 */
char* tw_bitcoin_transaction_to_hex(const TWBitcoinTransaction* tx);

/**
 * Get transaction fee
 * @param tx Transaction
 * @return Fee in satoshi
 */
uint64_t tw_bitcoin_transaction_get_fee(const TWBitcoinTransaction* tx);

/**
 * Get transaction ID
 * @param tx Transaction
 * @return Transaction ID string (must be freed)
 */
char* tw_bitcoin_transaction_get_id(const TWBitcoinTransaction* tx);

/**
 * Free Bitcoin transaction
 * @param tx Transaction to free
 */
void tw_bitcoin_transaction_delete(TWBitcoinTransaction* tx);

// ============================================================================
// Bitcoin UTXO Functions
// ============================================================================

/**
 * Create UTXO from parameters
 * @param txId Transaction ID
 * @param vout Output index
 * @param amount Amount in satoshi
 * @param script Script bytes
 * @param scriptLen Script length
 * @param confirmations Number of confirmations
 * @return Newly allocated UTXO
 */
TWBitcoinUTXO* tw_bitcoin_utxo_create(const uint8_t* txId, uint32_t vout,
                                        uint64_t amount,
                                        const uint8_t* script, size_t scriptLen,
                                        uint32_t confirmations);

/**
 * Free UTXO
 * @param utxo UTXO to free
 */
void tw_bitcoin_utxo_delete(TWBitcoinUTXO* utxo);

// ============================================================================
// EVM Transaction Functions
// ============================================================================

/**
 * Create new EVM transaction
 * @param chainId Chain ID
 * @param txType Transaction type
 * @return Newly allocated transaction
 */
TWEVMTransaction* tw_evm_transaction_create(uint64_t chainId, TWCoreTxType txType);

/**
 * Set transaction parameters
 * @param tx Transaction
 * @param from Sender address
 * @param to Recipient address (can be NULL for contract creation)
 * @param value Value in wei
 * @param data Transaction data
 * @param dataLen Data length
 * @param gasLimit Gas limit
 * @param maxFeePerGas Max fee per gas (EIP-1559)
 * @param maxPriorityFeePerGas Max priority fee (EIP-1559)
 * @param gasPrice Gas price (legacy)
 * @param nonce Nonce
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_evm_transaction_set_params(TWEVMTransaction* tx,
                                            const char* from,
                                            const char* to,
                                            const char* value,
                                            const uint8_t* data, size_t dataLen,
                                            uint64_t gasLimit,
                                            const char* maxFeePerGas,
                                            const char* maxPriorityFeePerGas,
                                            const char* gasPrice,
                                            uint64_t nonce);

/**
 * Sign EVM transaction
 * @param tx Transaction to sign
 * @param privateKey Private key
 * @return Signed transaction RLP encoded (must be freed)
 */
char* tw_evm_transaction_sign(const TWEVMTransaction* tx, const TWPrivateKey* privateKey);

/**
 * Encode transaction to RLP
 * @param tx Transaction
 * @return RLP encoded bytes (must be freed)
 */
uint8_t* tw_evm_transaction_encode(const TWEVMTransaction* tx, size_t* outputLen);

/**
 * Get transaction hash
 * @param tx Transaction
 * @return Hash string (must be freed)
 */
char* tw_evm_transaction_get_hash(const TWEVMTransaction* tx);

/**
 * Get sender address
 * @param tx Transaction
 * @return Sender address (must be freed)
 */
char* tw_evm_transaction_get_sender(const TWEVMTransaction* tx);

/**
 * Estimate gas for transaction
 * @param tx Transaction
 * @return Estimated gas (0 on error)
 */
uint64_t tw_evm_transaction_estimate_gas(const TWEVMTransaction* tx);

/**
 * Free EVM transaction
 * @param tx Transaction to free
 */
void tw_evm_transaction_delete(TWEVMTransaction* tx);

// ============================================================================
// ERC-20 Token Functions
// ============================================================================

/**
 * Create ERC-20 transfer data
 * @param to Recipient address
 * @param amount Token amount
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc20_transfer_data(const char* to, const char* amount, size_t* outputLen);

/**
 * Create ERC-20 approve data
 * @param spender Spender address
 * @param amount Approval amount
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc20_approve_data(const char* spender, const char* amount, size_t* outputLen);

/**
 * Create ERC-20 transferFrom data
 * @param from From address
 * @param to To address
 * @param amount Amount
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc20_transfer_from_data(const char* from, const char* to, const char* amount, size_t* outputLen);

// ============================================================================
// ERC-721 (NFT) Functions
// ============================================================================

/**
 * Create ERC-721 safeTransferFrom data
 * @param from From address
 * @param to To address
 * @param tokenId Token ID
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc721_safe_transfer_from_data(const char* from, const char* to, const char* tokenId, size_t* outputLen);

/**
 * Create ERC-721 setApprovalForAll data
 * @param operator Operator address
 * @param approved Approval status
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc721_set_approval_for_all_data(const char* operator, bool approved, size_t* outputLen);

/**
 * Create ERC-721 safeTransferFrom with data
 * @param from From address
 * @param to To address
 * @param tokenId Token ID
 * @param data Additional data
 * @param dataLen Data length
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc721_safe_transfer_from_with_data(const char* from, const char* to, 
                                                  const char* tokenId,
                                                  const uint8_t* data, size_t dataLen,
                                                  size_t* outputLen);

// ============================================================================
// ERC-1155 (Multi-Token) Functions
// ============================================================================

/**
 * Create ERC-1155 safeTransferFrom data
 * @param from From address
 * @param to To address
 * @param tokenId Token ID
 * @param amount Amount
 * @param data Additional data
 * @param dataLen Data length
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc1155_safe_transfer_from_data(const char* from, const char* to,
                                              const char* tokenId, const char* amount,
                                              const uint8_t* data, size_t dataLen,
                                              size_t* outputLen);

/**
 * Create ERC-1155 safeBatchTransferFrom data
 * @param from From address
 * @param to To address
 * @param tokenIds Array of token IDs
 * @param amounts Array of amounts
 * @param count Number of tokens
 * @param data Additional data
 * @param dataLen Data length
 * @return Transaction data (must be freed)
 */
uint8_t* tw_erc1155_safe_batch_transfer_from_data(const char* from, const char* to,
                                                    const char** tokenIds, const char** amounts,
                                                    size_t count,
                                                    const uint8_t* data, size_t dataLen,
                                                    size_t* outputLen);

// ============================================================================
// Cosmos Transaction Functions
// ============================================================================

/**
 * Create new Cosmos transaction
 * @param coinType Cosmos chain type
 * @return Newly allocated transaction
 */
TWCosmosTransaction* tw_cosmos_transaction_create(TWCoreCoinType coinType);

/**
 * Add message to Cosmos transaction
 * @param tx Transaction
 * @param type Message type (send, delegate, etc.)
 * @param jsonData JSON-encoded message data
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_cosmos_transaction_add_message(TWCosmosTransaction* tx,
                                                const char* type,
                                                const char* jsonData);

/**
 * Sign Cosmos transaction
 * @param tx Transaction to sign
 * @param privateKey Private key
 * @param accountNumber Account number
 * @param sequence Sequence number
 * @param chainId Chain ID
 * @return Signed transaction (must be freed)
 */
char* tw_cosmos_transaction_sign(const TWCosmosTransaction* tx,
                                   const TWPrivateKey* privateKey,
                                   uint64_t accountNumber,
                                   uint64_t sequence,
                                   const char* chainId);

/**
 * Get transaction hash
 * @param tx Transaction
 * @return Hash string (must be freed)
 */
char* tw_cosmos_transaction_get_hash(const TWCosmosTransaction* tx);

/**
 * Free Cosmos transaction
 * @param tx Transaction to free
 */
void tw_cosmos_transaction_delete(TWCosmosTransaction* tx);

// ============================================================================
// Solana Transaction Functions
// ============================================================================

/**
 * Create new Solana transaction
 * @return Newly allocated transaction
 */
TWSolanaTransaction* tw_solana_transaction_create(void);

/**
 * Add instruction to Solana transaction
 * @param tx Transaction
 * @param programId Program ID
 * @param accounts Array of account metas
 * @param accountCount Number of accounts
 * @param data Instruction data
 * @param dataLen Data length
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_solana_transaction_add_instruction(TWSolanaTransaction* tx,
                                                    const char* programId,
                                                    const char** accounts, size_t accountCount,
                                                    const uint8_t* data, size_t dataLen);

/**
 * Add transfer instruction
 * @param tx Transaction
 * @param from From address
 * @param to To address
 * @param lamports Amount in lamports
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_solana_transaction_add_transfer(TWSolanaTransaction* tx,
                                                 const char* from,
                                                 const char* to,
                                                 uint64_t lamports);

/**
 * Sign Solana transaction
 * @param tx Transaction to sign
 * @param privateKey Private key
 * @return Signed transaction (must be freed)
 */
char* tw_solana_transaction_sign(const TWSolanaTransaction* tx, const TWPrivateKey* privateKey);

/**
 * Get transaction hash
 * @param tx Transaction
 * @return Hash string (must be freed)
 */
char* tw_solana_transaction_get_hash(const TWSolanaTransaction* tx);

/**
 * Free Solana transaction
 * @param tx Transaction to free
 */
void tw_solana_transaction_delete(TWSolanaTransaction* tx);

// ============================================================================
// Account Abstraction (EIP-4337) Functions
// ============================================================================

/**
 * Create account abstraction entry point
 * @param chainId Chain ID
 * @return Account abstraction handle
 */
TWAccountAbstraction* tw_account_abstraction_create(uint64_t chainId);

/**
 * Create user operation
 * @param aa Account abstraction
 * @param sender Sender address
 * @param nonce Nonce
 * @param initCode Initialization code
 * @param callData Call data
 * @param callGasLimit Gas limit for call
 * @param verificationGasLimit Gas limit for verification
 * @param preVerificationGas Pre-verification gas
 * @param maxFeePerGas Max fee per gas
 * @param maxPriorityFeePerGas Max priority fee
 * @param paymaster Paymaster address (optional)
 * @param signature User signature
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_account_abstraction_create_user_op(TWAccountAbstraction* aa,
                                                    const char* sender,
                                                    uint64_t nonce,
                                                    const uint8_t* initCode, size_t initCodeLen,
                                                    const uint8_t* callData, size_t callDataLen,
                                                    uint64_t callGasLimit,
                                                    uint64_t verificationGasLimit,
                                                    uint64_t preVerificationGas,
                                                    const char* maxFeePerGas,
                                                    const char* maxPriorityFeePerGas,
                                                    const char* paymaster,
                                                    const uint8_t* signature, size_t signatureLen);

/**
 * Sign user operation
 * @param aa Account abstraction
 * @param privateKey Private key
 * @return Signed operation data (must be freed)
 */
char* tw_account_abstraction_sign_user_op(const TWAccountAbstraction* aa, const TWPrivateKey* privateKey);

/**
 * Get entry point address
 * @param aa Account abstraction
 * @return Entry point address (must be freed)
 */
char* tw_account_abstraction_get_entry_point(const TWAccountAbstraction* aa);

/**
 * Get counterfactual address
 * @param aa Account abstraction
 * @param owner Owner address
 * @param salt Salt value
 * @return Counterfactual address (must be freed)
 */
char* tw_account_abstraction_get_counterfactual_address(const TWAccountAbstraction* aa,
                                                        const char* owner,
                                                        uint32_t salt);

/**
 * Free account abstraction
 * @param aa Account abstraction to free
 */
void tw_account_abstraction_delete(TWAccountAbstraction* aa);

// ============================================================================
// Multisig Functions
// ============================================================================

/**
 * Create multisig wallet
 * @param threshold Required signatures
 * @param signers Array of signer addresses
 * @param signerCount Number of signers
 * @return Multisig handle
 */
TWMultisig* tw_multisig_create(uint8_t threshold, const char** signers, size_t signerCount);

/**
 * Create multisig transaction
 * @param multisig Multisig handle
 * @param to Destination address
 * @param value Amount
 * @param data Transaction data
 * @param dataLen Data length
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_multisig_create_transaction(TWMultisig* multisig,
                                             const char* to,
                                             const char* value,
                                             const uint8_t* data, size_t dataLen);

/**
 * Add signature to multisig
 * @param multisig Multisig handle
 * @param signer Signer address
 * @param signature Signature bytes
 * @param signatureLen Signature length
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_multisig_add_signature(TWMultisig* multisig,
                                        const char* signer,
                                        const uint8_t* signature, size_t signatureLen);

/**
 * Get multisig address
 * @param multisig Multisig handle
 * @return Multisig address (must be freed)
 */
char* tw_multisig_get_address(const TWMultisig* multisig);

/**
 * Is transaction fully signed
 * @param multisig Multisig handle
 * @return true if threshold met
 */
bool tw_multisig_is_complete(const TWMultisig* multisig);

/**
 * Get combined signature
 * @param multisig Multisig handle
 * @param output Output buffer
 * @param outputSize Buffer size
 * @param actualSize Actual signature size
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_multisig_get_signature(const TWMultisig* multisig,
                                        uint8_t* output, size_t outputSize,
                                        size_t* actualSize);

/**
 * Free multisig
 * @param multisig Multisig to free
 */
void tw_multisig_delete(TWMultisig* multisig);

// ============================================================================
// MPC Functions
// ============================================================================

/**
 * Create MPC session
 * @param threshold Required shares
 * @param totalShares Total number of shares
 * @return MPC handle
 */
TWMPC* tw_mpc_create(uint8_t threshold, uint8_t totalShares);

/**
 * Generate key share
 * @param mpc MPC handle
 * @param shareIndex Share index
 * @param seed Random seed
 * @param seedLen Seed length
 * @return Generated share (must be freed)
 */
char* tw_mpc_generate_share(TWMPC* mpc, uint8_t shareIndex,
                              const uint8_t* seed, size_t seedLen);

/**
 * Combine shares to reconstruct key
 * @param mpc MPC handle
 * @param shares Array of share strings
 * @param shareCount Number of shares
 * @return Reconstructed private key (must be freed)
 */
TWPrivateKey* tw_mpc_combine_shares(TWMPC* mpc,
                                      const char** shares, size_t shareCount);

/**
 * Sign with MPC
 * @param mpc MPC handle
 * @param shares Array of partial signatures
 * @param shareCount Number of shares
 * @param data Data to sign
 * @param dataLen Data length
 * @param signature Output signature
 * @param signatureSize Signature buffer size
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_mpc_sign(TWMPC* mpc,
                         const char** shares, size_t shareCount,
                         const uint8_t* data, size_t dataLen,
                         uint8_t* signature, size_t* signatureSize);

/**
 * Free MPC
 * @param mpc MPC to free
 */
void tw_mpc_delete(TWMPC* mpc);

// ============================================================================
// Chain Info Functions
// ============================================================================

/**
 * Get chain info
 * @param coinType Blockchain type
 * @return Chain info (must be freed)
 */
TWChainInfo* tw_chain_info_get(TWCoreCoinType coinType);

/**
 * Get all supported chains
 * @param count Output chain count
 * @return Array of chain info (must be freed each and array)
 */
TWChainInfo** tw_chain_info_get_all(size_t* count);

/**
 * Get chain ID
 * @param info Chain info
 * @return Chain ID
 */
uint64_t tw_chain_info_get_id(const TWChainInfo* info);

/**
 * Get chain name
 * @param info Chain info
 * @return Chain name (must be freed)
 */
char* tw_chain_info_get_name(const TWChainInfo* info);

/**
 * Get chain symbol
 * @param info Chain info
 * @return Chain symbol (must be freed)
 */
char* tw_chain_info_get_symbol(const TWChainInfo* info);

/**
 * Get chain decimals
 * @param info Chain info
 * @return Chain decimals
 */
uint8_t tw_chain_info_get_decimals(const TWChainInfo* info);

/**
 * Get chain RPC URL
 * @param info Chain info
 * @return RPC URL (must be freed)
 */
char* tw_chain_info_get_rpc_url(const TWChainInfo* info);

/**
 * Get chain explorer URL
 * @param info Chain info
 * @return Explorer URL (must be freed)
 */
char* tw_chain_info_get_explorer_url(const TWChainInfo* info);

/**
 * Free chain info
 * @param info Chain info to free
 */
void tw_chain_info_delete(TWChainInfo* info);

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Get version string
 * @return Version string (do not free)
 */
const char* tw_core_get_version(void);

/**
 * Initialize the library
 * @return TWCORE_SUCCESS on success
 */
TWCoreError tw_core_initialize(void);

/**
 * Shutdown the library
 */
void tw_core_shutdown(void);

/**
 * Get last error message
 * @return Error message string (do not free)
 */
const char* tw_core_get_last_error(void);

/**
 * Securely clear memory
 * @param ptr Memory pointer
 * @param size Size to clear
 */
void tw_secure_clear(void* ptr, size_t size);

/**
 * Securely allocate memory
 * @param size Size to allocate
 * @return Pointer to memory or NULL
 */
void* tw_secure_malloc(size_t size);

/**
 * Free secure memory
 * @param ptr Memory pointer
 * @param size Size to free
 */
void tw_secure_free(void* ptr, size_t size);

#ifdef __cplusplus
}
#endif

#endif // TIGER_WALLET_CORE_H
