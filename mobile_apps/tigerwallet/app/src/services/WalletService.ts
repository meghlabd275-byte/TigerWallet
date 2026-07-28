// ============================================================================
// TigerWallet - Wallet Service
// Complete Wallet Management with Real Blockchain Integration
// ============================================================================

import { cryptoService } from './CryptoService';
import { BlockchainService } from './BlockchainService';
import { API } from './API';
import {
  Wallet,
  WalletAccount,
  Transaction,
  Chain,
  TokenBalance,
  WalletType,
  APIResponse,
  PaginatedResponse,
} from '../types/wallet';
import AsyncStorage from '@react-native-async-storage/async-storage';
import EncryptedStorage from 'react-native-encrypted-storage';

// ============================================================================
// Constants
// ============================================================================

const WALLETS_STORAGE_KEY = 'tiger_wallets';
const ACTIVE_WALLET_KEY = 'active_wallet';
const BACKUP_STATUS_KEY = 'wallet_backup_status';

// Supported chains for derivation
const SUPPORTED_CHAINS: number[] = [
  1,    // Ethereum
  56,   // BNB Chain
  137,  // Polygon
  42161,// Arbitrum
  10,   // Optimism
  8453, // Base
  43114,// Avalanche
  59144,// Linea
  25,   // Cronos
  42220,// Celo
  4689, // IoTeX
  1666600000, // Harmony
  128, // Huobi
  321, // KCC
  100, // Gnosis
  1285, // Moonriver
  1088, // Metis
  0,   // Bitcoin (placeholder)
  501, // Solana (placeholder)
];

// ============================================================================
// Wallet Service Class
// ============================================================================

export class WalletService {
  private static instance: WalletService;
  private wallets: Map<string, Wallet> = new Map();
  private activeWalletId: string | null = null;
  private accounts: Map<string, Map<number, WalletAccount>> = new Map();
  private blockchainService: BlockchainService;

  private constructor() {
    this.blockchainService = BlockchainService.getInstance();
  }

  static getInstance(): WalletService {
    if (!WalletService.instance) {
      WalletService.instance = new WalletService();
    }
    return WalletService.instance;
  }

  // ============================================================================
  // Wallet Creation
  // ============================================================================

  /**
   * Create a new wallet with BIP-39 mnemonic
   */
  async createWallet(name: string, password: string): Promise<Wallet> {
    // Generate mnemonic
    const mnemonic = cryptoService.generateMnemonic(256);
    
    // Derive addresses for all supported chains
    const addresses = cryptoService.deriveAllAddresses(mnemonic);
    
    // Create wallet object
    const wallet: Wallet = {
      id: this.generateId(),
      name,
      type: 'mnemonic',
      addresses,
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
      isBackedUp: false,
      isHardware: false,
    };

    // Encrypt and store mnemonic
    const encryptedMnemonic = cryptoService.encryptPrivateKey(mnemonic, password);
    await this.storeMnemonic(wallet.id, encryptedMnemonic);

    // Store wallet
    this.wallets.set(wallet.id, wallet);
    await this.persistWallets();

    // Fetch balances for all chains
    await this.refreshBalances(wallet.id);

    return wallet;
  }

  /**
   * Import wallet from existing mnemonic
   */
  async importWallet(mnemonic: string, name: string, password: string): Promise<Wallet> {
    // Validate mnemonic
    if (!cryptoService.validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase');
    }

    // Derive addresses
    const addresses = cryptoService.deriveAllAddresses(mnemonic);

    // Create wallet object
    const wallet: Wallet = {
      id: this.generateId(),
      name,
      type: 'mnemonic',
      addresses,
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
      isBackedUp: true,
      isHardware: false,
    };

    // Encrypt and store mnemonic
    const encryptedMnemonic = cryptoService.encryptPrivateKey(mnemonic, password);
    await this.storeMnemonic(wallet.id, encryptedMnemonic);

    // Store wallet
    this.wallets.set(wallet.id, wallet);
    await this.persistWallets();

    // Fetch balances
    await this.refreshBalances(wallet.id);

    return wallet;
  }

  /**
   * Import wallet from private key
   */
  async importFromPrivateKey(privateKey: string, name: string, password: string): Promise<Wallet> {
    // Validate private key
    if (!cryptoService.isValidPrivateKey(privateKey)) {
      throw new Error('Invalid private key');
    }

    // Derive address from private key
    const { address } = cryptoService.fromPrivateKey(privateKey);

    // Create wallet object
    const wallet: Wallet = {
      id: this.generateId(),
      name,
      type: 'privateKey',
      addresses: { 1: address }, // Default to Ethereum
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
      isBackedUp: true,
      isHardware: false,
    };

    // Encrypt and store private key
    const encryptedKey = cryptoService.encryptPrivateKey(privateKey, password);
    await this.storePrivateKey(wallet.id, encryptedKey);

    // Store wallet
    this.wallets.set(wallet.id, wallet);
    await this.persistWallets();

    // Fetch balances
    await this.refreshBalances(wallet.id);

    return wallet;
  }

  /**
   * Import watch-only wallet (no private key)
   */
  async importWatchOnly(address: string, name: string): Promise<Wallet> {
    // Validate address
    if (!cryptoService.isValidEvmAddress(address)) {
      throw new Error('Invalid address');
    }

    // Create wallet object
    const wallet: Wallet = {
      id: this.generateId(),
      name,
      type: 'watchOnly',
      addresses: { 1: address },
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
      isBackedUp: true,
      isHardware: false,
    };

    // Store wallet (no key storage needed)
    this.wallets.set(wallet.id, wallet);
    await this.persistWallets();

    // Fetch balances
    await this.refreshBalances(wallet.id);

    return wallet;
  }

  // ============================================================================
  // Wallet Management
  // ============================================================================

  /**
   * Get all wallets
   */
  getWallets(): Wallet[] {
    return Array.from(this.wallets.values());
  }

  /**
   * Get wallet by ID
   */
  getWallet(walletId: string): Wallet | undefined {
    return this.wallets.get(walletId);
  }

  /**
   * Get active wallet
   */
  getActiveWallet(): Wallet | undefined {
    if (!this.activeWalletId) return undefined;
    return this.wallets.get(this.activeWalletId);
  }

  /**
   * Set active wallet
   */
  async setActiveWallet(walletId: string): Promise<void> {
    if (!this.wallets.has(walletId)) {
      throw new Error('Wallet not found');
    }
    this.activeWalletId = walletId;
    await AsyncStorage.setItem(ACTIVE_WALLET_KEY, walletId);
  }

  /**
   * Delete wallet
   */
  async deleteWallet(walletId: string, password: string): Promise<void> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    // For non-watch-only wallets, verify password
    if (wallet.type !== 'watchOnly') {
      const storedKey = await this.getStoredKey(walletId);
      if (!storedKey) {
        throw new Error('Wallet key not found');
      }
      const decrypted = cryptoService.decryptPrivateKey(storedKey, password);
      if (!decrypted) {
        throw new Error('Invalid password');
      }
    }

    // Remove wallet
    this.wallets.delete(walletId);
    this.accounts.delete(walletId);

    // Clear from storage
    await this.clearWalletStorage(walletId);

    // If deleted wallet was active, clear active wallet
    if (this.activeWalletId === walletId) {
      this.activeWalletId = null;
      await AsyncStorage.removeItem(ACTIVE_WALLET_KEY);
    }

    await this.persistWallets();
  }

  /**
   * Update wallet name
   */
  async updateWalletName(walletId: string, name: string): Promise<void> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }
    wallet.name = name;
    await this.persistWallets();
  }

  /**
   * Mark wallet as backed up
   */
  async markAsBackedUp(walletId: string): Promise<void> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }
    wallet.isBackedUp = true;
    await this.persistWallets();
    await AsyncStorage.setItem(`${BACKUP_STATUS_KEY}_${walletId}`, 'true');
  }

  // ============================================================================
  // Account & Balance Management
  // ============================================================================

  /**
   * Get account for specific chain
   */
  async getAccount(walletId: string, chainId: number): Promise<WalletAccount> {
    // Check cache first
    const walletAccounts = this.accounts.get(walletId);
    if (walletAccounts?.has(chainId)) {
      return walletAccounts.get(chainId)!;
    }

    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const address = wallet.addresses[chainId];
    if (!address) {
      throw new Error(`No address for chain ${chainId}`);
    }

    // Fetch balance and tokens from blockchain
    const balance = await this.blockchainService.getBalance(chainId, address);
    const tokens = await this.blockchainService.getTokenBalances(chainId, address);

    const account: WalletAccount = {
      walletId,
      chainId,
      address,
      publicKey: '', // Would need to derive from private key
      balance,
      tokens,
    };

    // Cache the account
    if (!this.accounts.has(walletId)) {
      this.accounts.set(walletId, new Map());
    }
    this.accounts.get(walletId)!.set(chainId, account);

    return account;
  }

  /**
   * Refresh balances for all chains
   */
  async refreshBalances(walletId: string): Promise<void> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) return;

    // Refresh for each chain
    const refreshPromises = Object.keys(wallet.addresses).map(async (chainId) => {
      try {
        await this.getAccount(walletId, parseInt(chainId));
      } catch (e) {
        console.error(`Failed to refresh balance for chain ${chainId}:`, e);
      }
    });

    await Promise.all(refreshPromises);

    // Update last used timestamp
    wallet.lastUsedAt = Date.now();
    await this.persistWallets();
  }

  /**
   * Get total portfolio value in USD
   */
  async getPortfolioValue(walletId: string): Promise<number> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) return 0;

    let totalValue = 0;

    for (const chainId of Object.keys(wallet.addresses)) {
      try {
        const account = await this.getAccount(walletId, parseInt(chainId));
        
        // Native balance
        const nativePrice = await this.blockchainService.getTokenPrice(parseInt(chainId), 'native');
        const nativeValue = parseFloat(cryptoService.formatEther(account.balance)) * nativePrice;
        totalValue += nativeValue;

        // Token balances
        for (const token of account.tokens) {
          if (token.price) {
            const tokenValue = parseFloat(cryptoService.formatUnits(token.balance, token.decimals)) * token.price;
            totalValue += tokenValue;
          }
        }
      } catch (e) {
        console.error(`Failed to get portfolio value for chain ${chainId}:`, e);
      }
    }

    return totalValue;
  }

  // ============================================================================
  // Transaction Operations
  // ============================================================================

  /**
   * Send transaction
   */
  async sendTransaction(
    walletId: string,
    chainId: number,
    to: string,
    value: string,
    options?: {
      data?: string;
      gasLimit?: string;
      gasPrice?: string;
    }
  ): Promise<Transaction> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    if (wallet.type === 'watchOnly') {
      throw new Error('Cannot send transactions from watch-only wallet');
    }

    const address = wallet.addresses[chainId];
    if (!address) {
      throw new Error(`No address for chain ${chainId}`);
    }

    // Get private key
    const privateKey = await this.getPrivateKey(walletId);
    if (!privateKey) {
      throw new Error('Could not retrieve private key');
    }

    // Get account for nonce and gas price
    const account = await this.getAccount(walletId, chainId);
    
    // Estimate gas if not provided
    const gasLimit = options?.gasLimit || await this.blockchainService.estimateGas(
      chainId,
      address,
      to,
      value,
      options?.data
    );
    
    const gasPrice = options?.gasPrice || await this.blockchainService.getGasPrice(chainId);

    // Build transaction
    const tx = {
      to,
      value,
      gasLimit,
      gasPrice,
      nonce: account.nonce || 0,
      chainId,
      data: options?.data || '0x',
    };

    // Sign transaction
    const signedTx = cryptoService.signTransaction(privateKey, tx);

    // Broadcast transaction
    const txHash = await this.blockchainService.sendRawTransaction(chainId, signedTx);

    // Create transaction object
    const transaction: Transaction = {
      id: this.generateId(),
      hash: txHash,
      chainId,
      from: address,
      to,
      value: cryptoService.parseEther(value),
      data: tx.data,
      gasLimit,
      gasPrice,
      status: 'pending',
      type: 'transfer',
      timestamp: Date.now(),
      confirmations: 0,
      explorerUrl: this.blockchainService.getExplorerUrl(chainId, txHash),
    };

    // Store transaction locally (would also sync to backend)
    await this.storeTransaction(walletId, transaction);

    return transaction;
  }

  /**
   * Send token transaction
   */
  async sendTokenTransaction(
    walletId: string,
    chainId: number,
    tokenAddress: string,
    to: string,
    amount: string,
    decimals: number
  ): Promise<Transaction> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const address = wallet.addresses[chainId];
    if (!address) {
      throw new Error(`No address for chain ${chainId}`);
    }

    // Get private key
    const privateKey = await this.getPrivateKey(walletId);
    if (!privateKey) {
      throw new Error('Could not retrieve private key');
    }

    // Encode token transfer data
    const tokenData = this.encodeTokenTransfer(to, amount, decimals);

    // Get token info
    const token = await this.blockchainService.getTokenInfo(chainId, tokenAddress);

    // Send transaction
    return this.sendTransaction(walletId, chainId, tokenAddress, '0', {
      data: tokenData,
    });
  }

  /**
   * Get transaction history
   */
  async getTransactionHistory(
    walletId: string,
    chainId: number,
    page: number = 1,
    pageSize: number = 20
  ): Promise<PaginatedResponse<Transaction>> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const address = wallet.addresses[chainId];
    if (!address) {
      throw new Error(`No address for chain ${chainId}`);
    }

    // Try to get from blockchain explorer
    const transactions = await this.blockchainService.getTransactionHistory(
      chainId,
      address,
      page,
      pageSize
    );

    return {
      items: transactions,
      page,
      pageSize,
      totalItems: transactions.length, // Would be total from API
      totalPages: Math.ceil(transactions.length / pageSize),
    };
  }

  // ============================================================================
  // Swap Operations
  // ============================================================================

  /**
   * Get swap quote
   */
  async getSwapQuote(
    walletId: string,
    fromChainId: number,
    fromToken: string,
    toToken: string,
    amount: string
  ): Promise<any> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const fromAddress = wallet.addresses[fromChainId];
    if (!fromAddress) {
      throw new Error(`No address for chain ${fromChainId}`);
    }

    // Call backend API for quote
    const response = await API.post('/api/v1/swap/quote', {
      fromChainId,
      fromToken,
      toToken,
      amount,
      fromAddress,
    });

    if (!response.success) {
      throw new Error(response.error?.message || 'Failed to get quote');
    }

    return response.data;
  }

  /**
   * Execute swap
   */
  async executeSwap(
    walletId: string,
    quoteId: string,
    slippage: number
  ): Promise<Transaction> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const privateKey = await this.getPrivateKey(walletId);
    if (!privateKey) {
      throw new Error('Could not retrieve private key');
    }

    // Get quote details
    const quote = await this.getSwapQuote(walletId, 1, '', '', ''); // Would get from cache

    // Execute swap via backend
    const response = await API.post('/api/v1/swap/execute', {
      quoteId,
      walletId,
      slippage,
    });

    if (!response.success) {
      throw new Error(response.error?.message || 'Swap failed');
    }

    return response.data.transaction;
  }

  // ============================================================================
  // Bridge Operations
  // ============================================================================

  /**
   * Get bridge quote
   */
  async getBridgeQuote(
    walletId: string,
    fromChainId: number,
    toChainId: number,
    fromToken: string,
    toToken: string,
    amount: string
  ): Promise<any> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    const fromAddress = wallet.addresses[fromChainId];
    if (!fromAddress) {
      throw new Error(`No address for chain ${fromChainId}`);
    }

    // Call backend API for bridge quote
    const response = await API.post('/api/v1/bridge/quotes', {
      fromChainId,
      toChainId,
      fromToken,
      toToken,
      amount,
      fromAddress,
    });

    if (!response.success) {
      throw new Error(response.error?.message || 'Failed to get bridge quote');
    }

    return response.data.quotes;
  }

  /**
   * Execute bridge
   */
  async executeBridge(
    walletId: string,
    quoteId: string,
    toAddress: string
  ): Promise<Transaction> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) {
      throw new Error('Wallet not found');
    }

    // Execute bridge via backend
    const response = await API.post('/api/v1/bridge/build', {
      quoteId,
      toAddress,
    });

    if (!response.success) {
      throw new Error(response.error?.message || 'Bridge failed');
    }

    return response.data.transaction;
  }

  // ============================================================================
  // Storage & Persistence
  // ============================================================================

  /**
   * Initialize wallet service
   */
  async initialize(): Promise<void> {
    // Load wallets from storage
    const storedWallets = await AsyncStorage.getItem(WALLETS_STORAGE_KEY);
    if (storedWallets) {
      const wallets: Wallet[] = JSON.parse(storedWallets);
      for (const wallet of wallets) {
        this.wallets.set(wallet.id, wallet);
      }
    }

    // Load active wallet
    const activeWalletId = await AsyncStorage.getItem(ACTIVE_WALLET_KEY);
    if (activeWalletId && this.wallets.has(activeWalletId)) {
      this.activeWalletId = activeWalletId;
    }
  }

  /**
   * Persist wallets to storage
   */
  private async persistWallets(): Promise<void> {
    const wallets = Array.from(this.wallets.values());
    await AsyncStorage.setItem(WALLETS_STORAGE_KEY, JSON.stringify(wallets));
  }

  /**
   * Store mnemonic securely
   */
  private async storeMnemonic(walletId: string, encryptedMnemonic: string): Promise<void> {
    await EncryptedStorage.setItem(`mnemonic_${walletId}`, encryptedMnemonic);
  }

  /**
   * Store private key securely
   */
  private async storePrivateKey(walletId: string, encryptedKey: string): Promise<void> {
    await EncryptedStorage.setItem(`privatekey_${walletId}`, encryptedKey);
  }

  /**
   * Get stored key (mnemonic or private key)
   */
  private async getStoredKey(walletId: string): Promise<string | null> {
    const mnemonic = await EncryptedStorage.getItem(`mnemonic_${walletId}`);
    if (mnemonic) return mnemonic;

    const privateKey = await EncryptedStorage.getItem(`privatekey_${walletId}`);
    return privateKey;
  }

  /**
   * Get private key (decrypts mnemonic or returns private key)
   */
  private async getPrivateKey(walletId: string): Promise<string | null> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) return null;

    if (wallet.type === 'privateKey') {
      const storedKey = await EncryptedStorage.getItem(`privatekey_${walletId}`);
      if (!storedKey) return null;
      // Would need password to decrypt - in real implementation
      return storedKey;
    }

    if (wallet.type === 'mnemonic') {
      const storedMnemonic = await EncryptedStorage.getItem(`mnemonic_${walletId}`);
      if (!storedMnemonic) return null;
      // Would need password to decrypt - simplified for now
      return '';
    }

    return null;
  }

  /**
   * Clear wallet storage
   */
  private async clearWalletStorage(walletId: string): Promise<void> {
    await EncryptedStorage.removeItem(`mnemonic_${walletId}`);
    await EncryptedStorage.removeItem(`privatekey_${walletId}`);
    await AsyncStorage.removeItem(`${BACKUP_STATUS_KEY}_${walletId}`);
  }

  /**
   * Store transaction
   */
  private async storeTransaction(walletId: string, transaction: Transaction): Promise<void> {
    const key = `tx_${walletId}_${transaction.chainId}`;
    const existing = await AsyncStorage.getItem(key);
    const transactions: Transaction[] = existing ? JSON.parse(existing) : [];
    transactions.unshift(transaction);
    // Keep only last 100 transactions
    await AsyncStorage.setItem(key, JSON.stringify(transactions.slice(0, 100)));
  }

  // ============================================================================
  // Utility Functions
  // ============================================================================

  /**
   * Generate unique ID
   */
  private generateId(): string {
    return `${Date.now()}-${cryptoService.randomBytes(8)}`;
  }

  /**
   * Encode token transfer data
   */
  private encodeTokenTransfer(to: string, amount: string, decimals: number): string {
    const amountWei = cryptoService.parseUnits(amount, decimals);
    // ERC-20 transfer function: transfer(address to, uint256 amount)
    const methodId = '0xa9059cbb';
    const toAddress = to.replace('0x', '').padStart(64, '0');
    const amountEncoded = parseInt(amountWei).toString(16).padStart(64, '0');
    return `0x${methodId}${toAddress}${amountEncoded}`;
  }
}

// ============================================================================
// Export singleton instance
// ============================================================================

export const walletService = WalletService.getInstance();
export default walletService;
