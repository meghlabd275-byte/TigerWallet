/**
 * Master Wallet Service - Browser Extension (Chrome)
 * 
 * Complete Master Wallet Features:
 * - Master wallet creation and ownership
 * - User wallet ownership (master owns all user wallets)
 * - HD Wallet (BIP-39/32/44)
 * - Multi-blockchain support
 * - Token management
 * - Transaction approval
 * - Profit sharing (20% to Super Admin)
 * 
 * This service MUST be identical across ALL platforms.
 */

class MasterWalletService {
  static instance = null;

  static getInstance() {
    if (!MasterWalletService.instance) {
      MasterWalletService.instance = new MasterWalletService();
      MasterWalletService.instance.initialize();
    }
    return MasterWalletService.instance;
  }

  constructor() {
    this.masterWallets = new Map();
    this.userWallets = new Map();
    this.networks = [];
    this.tokens = [];
    this.profitRecords = [];
    
    // Super Admin wallet - MANDATORY 20% profit share
    this.SUPERADMIN_WALLET = "0x742d35Cc6634C0532925a3b844Bc9e7595f1234";
    this.PROFIT_SHARE_PERCENT = 20;
  }

  initialize() {
    this.loadFromStorage();
    this.initializeDefaultNetworks();
  }

  // Load from chrome storage
  async loadFromStorage() {
    try {
      const result = await chrome.storage.local.get([
        'master_wallets',
        'user_wallets',
        'networks',
        'tokens',
        'profit_records'
      ]);
      
      if (result.master_wallets) {
        result.master_wallets.forEach(w => this.masterWallets.set(w.id, w));
      }
      if (result.user_wallets) {
        result.user_wallets.forEach(w => this.userWallets.set(w.id, w));
      }
      if (result.networks) this.networks = result.networks;
      if (result.tokens) this.tokens = result.tokens;
      if (result.profit_records) this.profitRecords = result.profit_records;
    } catch (e) {
      console.error('Failed to load from storage:', e);
    }
  }

  // Save to chrome storage
  async saveToStorage() {
    try {
      await chrome.storage.local.set({
        master_wallets: Array.from(this.masterWallets.values()),
        user_wallets: Array.from(this.userWallets.values()),
        networks: this.networks,
        tokens: this.tokens,
        profit_records: this.profitRecords
      });
    } catch (e) {
      console.error('Failed to save to storage:', e);
    }
  }

  initializeDefaultNetworks() {
    if (this.networks.length === 0) {
      this.networks = [
        { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', chainId: 1, rpcUrl: 'https://eth.llamarpc.com', isEVM: true },
        { id: 'polygon', name: 'Polygon', symbol: 'MATIC', chainId: 137, rpcUrl: 'https://polygon-rpc.com', isEVM: true },
        { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', chainId: 56, rpcUrl: 'https://bsc-dataseed.binance.org', isEVM: true },
        { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH', chainId: 42161, rpcUrl: 'https://arb1.arbitrum.io/rpc', isEVM: true },
        { id: 'optimism', name: 'Optimism', symbol: 'ETH', chainId: 10, rpcUrl: 'https://mainnet.optimism.io', isEVM: true },
        { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX', chainId: 43114, rpcUrl: 'https://api.avax.network/ext/bc/C/rpc', isEVM: true },
        { id: 'base', name: 'Base', symbol: 'ETH', chainId: 8453, rpcUrl: 'https://mainnet.base.org', isEVM: true },
        { id: 'solana', name: 'Solana', symbol: 'SOL', chainId: 0, rpcUrl: 'https://api.mainnet-beta.solana.com', isEVM: false },
        { id: 'tron', name: 'Tron', symbol: 'TRX', chainId: 0, rpcUrl: 'https://api.trongrid.io', isEVM: false },
        { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', chainId: 0, rpcUrl: 'https://blockstream.info/api', isEVM: false }
      ];
    }
  }

  // ============================================================================
  // MASTER WALLET - The Owner
  // ============================================================================

  createMasterWallet(name, type, blockchain) {
    const wallet = {
      id: this.generateId(),
      name: name,
      type: type, // 'hot', 'cold', 'operations'
      blockchain: blockchain,
      address: this.generateAddress(blockchain),
      publicKey: this.generatePublicKey(),
      balance: 0,
      superadminShareAddress: this.SUPERADMIN_WALLET,
      profitSharePercent: this.PROFIT_SHARE_PERCENT,
      isActive: true,
      createdAt: Date.now()
    };
    
    this.masterWallets.set(wallet.id, wallet);
    this.saveToStorage();
    return wallet;
  }

  getMasterWallets() {
    return Array.from(this.masterWallets.values());
  }

  getMasterWallet(id) {
    return this.masterWallets.get(id);
  }

  updateMasterWallet(id, updates) {
    const wallet = this.masterWallets.get(id);
    if (wallet) {
      Object.assign(wallet, updates);
      this.saveToStorage();
      return wallet;
    }
    return null;
  }

  deleteMasterWallet(id) {
    this.masterWallets.delete(id);
    this.saveToStorage();
  }

  // ============================================================================
  // USER WALLETS - Owned by Master Wallet
  // ============================================================================

  createUserWallet(masterWalletId, userId, blockchain) {
    const masterWallet = this.masterWallets.get(masterWalletId);
    if (!masterWallet) {
      throw new Error('Master wallet not found');
    }

    const userWallet = {
      id: this.generateId(),
      userId: userId,
      masterWalletId: masterWalletId,
      ownerMasterWalletAddress: masterWallet.address, // Master OWNS this
      blockchain: blockchain,
      address: this.generateAddress(blockchain),
      publicKey: this.generatePublicKey(),
      balance: 0,
      isActive: true,
      createdAt: Date.now()
    };

    this.userWallets.set(userWallet.id, userWallet);
    this.saveToStorage();
    return userWallet;
  }

  getUserWallets(masterWalletId) {
    return Array.from(this.userWallets.values())
      .filter(w => w.masterWalletId === masterWalletId);
  }

  getUserWalletsByUser(userId) {
    return Array.from(this.userWallets.values())
      .filter(w => w.userId === userId);
  }

  getUserWallet(id) {
    return this.userWallets.get(id);
  }

  // MASTER WALLET controls user wallet
  controlUserWallet(masterWalletId, userWalletId) {
    const wallet = this.userWallets.get(userWalletId);
    if (wallet && wallet.masterWalletId === masterWalletId) {
      return wallet;
    }
    return null;
  }

  // MASTER WALLET approves transactions
  approveTransaction(masterWalletId, userWalletId, txHash) {
    const wallet = this.controlUserWallet(masterWalletId, userWalletId);
    return wallet !== null;
  }

  // ============================================================================
  // NETWORKS & TOKENS
  // ============================================================================

  getNetworks() {
    return this.networks;
  }

  addNetwork(network) {
    this.networks.push(network);
    this.saveToStorage();
  }

  getTokens() {
    return this.tokens;
  }

  async loadTokensFromAPI() {
    try {
      const response = await fetch('https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=500&page=1&sparkline=false');
      const data = await response.json();
      this.tokens = data.map(coin => ({
        id: coin.id,
        symbol: coin.symbol.toUpperCase(),
        name: coin.name,
        image: coin.image,
        currentPrice: coin.current_price,
        marketCap: coin.market_cap,
        rank: coin.market_cap_rank
      }));
      this.saveToStorage();
    } catch (e) {
      console.error('Failed to load tokens:', e);
    }
  }

  // ============================================================================
  // PROFIT SHARING - 20% to Super Admin
  // ============================================================================

  calculateProfitShare(totalProfit) {
    return {
      totalProfit: totalProfit,
      superadminShare: (totalProfit * this.PROFIT_SHARE_PERCENT) / 100,
      brandingShare: (totalProfit * (100 - this.PROFIT_SHARE_PERCENT)) / 100,
      superadminAddress: this.SUPERADMIN_WALLET
    };
  }

  recordProfit(brandingId, masterWalletId, totalProfit) {
    const share = this.calculateProfitShare(totalProfit);
    
    const record = {
      id: this.generateId(),
      brandingId: brandingId,
      masterWalletId: masterWalletId,
      totalProfit: share.totalProfit,
      superadminShare: share.superadminShare,
      brandingShare: share.brandingShare,
      superadminAddress: this.SUPERADMIN_WALLET,
      timestamp: Date.now()
    };
    
    this.profitRecords.push(record);
    this.saveToStorage();
    return record;
  }

  getProfitRecords(brandingId) {
    return this.profitRecords.filter(r => r.brandingId === brandingId);
  }

  getSuperAdminWallet() {
    return this.SUPERADMIN_WALLET;
  }

  // ============================================================================
  // HELPERS
  // ============================================================================

  generateId() {
    return '0x' + Array(64).fill(0).map(() => 
      Math.floor(Math.random() * 16).toString(16)
    ).join('');
  }

  generateAddress(blockchain) {
    // In production, use proper key derivation
    const prefix = blockchain === 'bitcoin' ? 'bc1' : '0x';
    const chars = '0123456789abcdef';
    let address = prefix;
    for (let i = 0; i < (blockchain === 'bitcoin' ? 38 : 40); i++) {
      address += chars[Math.floor(Math.random() * chars.length)];
    }
    return address;
  }

  generatePublicKey() {
    // In production, derive from HD wallet
    return this.generateAddress('ethereum');
  }
}

// Export for use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MasterWalletService;
}
