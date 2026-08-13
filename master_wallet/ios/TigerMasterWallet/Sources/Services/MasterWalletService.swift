/**
 * MasterWalletService - iOS Implementation
 * Wallet management facade over the canonical MasterWallet Go backend (:8450).
 *
 * The backend is the ONLY key holder / signer. This client NEVER generates
 * mnemonics, derives keys, signs transactions, or fabricates balances / tx
 * hashes locally. Balance is fetched from GET /balance; sending/signing is
 * delegated to POST /sign (real secp256k1 + broadcast server-side).
 *
 * NOTE: `web3j` is a Java library and has no place in Swift. All wallet
 * operations go through the real REST backend.
 */

import Foundation
import Security
import CryptoKit

public class MasterWalletService {

    // MARK: - Singleton (also constructable with a custom API client)
    public static let shared = MasterWalletService(apiService: MasterAPIService())

    // MARK: - Properties
    private let apiService: MasterAPIService

    // Supported EVM chain ids (metadata only; the backend is source of truth
    // for balances, gas, and signing).
    public static let CHAIN_ETHEREUM = 1
    public static let CHAIN_BSC = 56
    public static let CHAIN_POLYGON = 137
    public static let CHAIN_ARBITRUM = 42161
    public static let CHAIN_OPTIMISM = 10
    public static let CHAIN_AVALANCHE = 43114

    // MARK: - Initialization
    public init(apiService: MasterAPIService) {
        self.apiService = apiService
    }

    // MARK: - Wallet Creation (backend performs real BIP-39 + BIP-32 derivation)

    /// Create a new master wallet via the backend. The backend generates the
    /// real BIP-39 mnemonic, derives the HD key, and returns the address. The
    /// mnemonic is returned by the backend exactly once and is never stored or
    /// regenerated on-device.
    public func createWallet(name: String, password: String, chainId: Int,
                             completion: @escaping (WalletResult) -> Void) {
        Task {
            do {
                let wallet = try await apiService.createMasterWallet(name: name, password: password, chainId: chainId)
                completion(WalletResult(
                    success: true,
                    walletId: wallet.id,
                    address: wallet.address,
                    mnemonic: nil
                ))
            } catch {
                completion(WalletResult(success: false, error: error.localizedDescription))
            }
        }
    }

    /// Importing a wallet from a mnemonic is a key-management operation that the
    /// canonical backend does not expose; the backend is the sole key holder.
    /// Fail closed rather than deriving keys with a client-side fake.
    public func importWallet(mnemonic: String, password: String,
                             completion: @escaping (WalletResult) -> Void) {
        completion(WalletResult(
            success: false,
            error: "Importing a wallet from mnemonic is not supported: the backend is the sole key holder."
        ))
    }

    // MARK: - Balance (real backend: GET /api/v1/master-wallet/:id/balance)

    /// Fetch the real native + token balances from the backend.
    public func getBalance(walletId: String, chainId: Int,
                           completion: @escaping (BalanceResult) -> Void) {
        Task {
            do {
                let resp = try await apiService.getBalance(walletId: walletId, chainId: chainId)
                completion(BalanceResult(
                    success: true,
                    balance: resp.native.usdValue,
                    nativeBalance: resp.native.balance,
                    symbol: resp.native.symbol,
                    decimals: resp.native.decimals,
                    tokens: resp.tokens.map { TokenBalanceInfo(contract: $0.contract, symbol: $0.symbol, balance: $0.balance, decimals: $0.decimals, usdValue: $0.usdValue) },
                    error: nil
                ))
            } catch {
                completion(BalanceResult(success: false, error: error.localizedDescription))
            }
        }
    }

    // MARK: - Send / Sign (real backend: POST /api/v1/master-wallet/:id/sign)

    /// Send a transaction by delegating signing + broadcast to the backend.
    /// Returns the real on-chain transaction hash; never fabricated client-side.
    public func sendTransaction(walletId: String, chainId: Int, toAddress: String,
                                amount: String, password: String, token: String? = nil,
                                completion: @escaping (TransactionResult) -> Void) {
        Task {
            do {
                let resp = try await apiService.sign(walletId: walletId, to: toAddress,
                                                     amount: amount, password: password, token: token)
                completion(TransactionResult(
                    success: true,
                    txHash: resp.transactionHash,
                    status: resp.status,
                    to: toAddress,
                    amount: amount,
                    error: nil
                ))
            } catch {
                completion(TransactionResult(success: false, error: error.localizedDescription))
            }
        }
    }

    // MARK: - Wallet Listing

    /// List all master wallets for the authenticated user.
    public func getAllWallets(completion: @escaping ([MasterWallet]?, String?) -> Void) {
        Task {
            do {
                let resp = try await apiService.listMasterWallets()
                completion(resp.wallets, nil)
            } catch {
                completion(nil, error.localizedDescription)
            }
        }
    }

    /// Delete a master wallet via the backend.
    public func deleteWallet(walletId: String, completion: @escaping (Bool, String?) -> Void) {
        Task {
            do {
                try await apiService.deleteMasterWallet(id: walletId)
                completion(true, nil)
            } catch {
                completion(false, error.localizedDescription)
            }
        }
    }

    // MARK: - Sub Wallets

    public func getSubWallets(masterWalletId: String,
                              completion: @escaping ([SubWallet]?, String?) -> Void) {
        Task {
            do {
                let wallets = try await apiService.getSubWallets(masterWalletId: masterWalletId)
                completion(wallets, nil)
            } catch {
                completion(nil, error.localizedDescription)
            }
        }
    }
}

// MARK: - Result Structures

public struct WalletResult {
    public let success: Bool
    public let walletId: String?
    public let address: String?
    public let mnemonic: String?
    public let error: String?
}

public struct BalanceResult {
    public let success: Bool
    public let balance: Double          // native USD value
    public let nativeBalance: String    // native balance in wei units (string)
    public let symbol: String
    public let decimals: Int
    public let tokens: [TokenBalanceInfo]
    public let error: String?

    public init(success: Bool, balance: Double = 0, nativeBalance: String = "0",
                symbol: String = "", decimals: Int = 18, tokens: [TokenBalanceInfo] = [],
                error: String? = nil) {
        self.success = success
        self.balance = balance
        self.nativeBalance = nativeBalance
        self.symbol = symbol
        self.decimals = decimals
        self.tokens = tokens
        self.error = error
    }
}

public struct TokenBalanceInfo {
    public let contract: String
    public let symbol: String
    public let balance: String
    public let decimals: Int
    public let usdValue: Double
}

public struct TransactionResult {
    public let success: Bool
    public let txHash: String?
    public let status: String?
    public let to: String?
    public let amount: String?
    public let error: String?

    public init(success: Bool, txHash: String? = nil, status: String? = nil,
                to: String? = nil, amount: String? = nil, error: String? = nil) {
        self.success = success
        self.txHash = txHash
        self.status = status
        self.to = to
        self.amount = amount
        self.error = error
    }
}
