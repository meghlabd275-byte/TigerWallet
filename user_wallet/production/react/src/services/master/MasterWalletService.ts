/**
 * Master Wallet Service - Custom Branding Owner
 * Each custom branding wallet has its own master wallet that controls their users
 */

export type MasterWalletType = 'hot' | 'cold' | 'operations';

// Master Wallet - OWNER of all user wallets for a specific branding
export interface MasterWallet {
  id: string;
  brandingId: string;           // Which branding this master belongs to
  brandingName: string;        // e.g., "TigerWallet", "MyCrypto", etc.
  name: string;
  type: MasterWalletType;
  blockchain: string;
  address: string;             // Master wallet address (signs all transactions)
  publicKey: string;
  balance: number;
  isActive: boolean;
  createdAt: string;
}

// User Wallet - Owned by Master Wallet of that branding
export interface UserWallet {
  id: string;
  userId: string;
  brandingId: string;                    // Which branding this user belongs to
  ownerMasterWalletId: string;            // WHO OWNS THIS WALLET
  ownerMasterWalletAddress: string;       // Master wallet address
  blockchain: string;
  address: string;
  publicKey: string;
  balance: number;
  isActive: boolean;
  createdAt: string;
}

// Custom Branding Platform (e.g., TigerWallet, MyCrypto, etc.)
export interface Branding {
  id: string;
  name: string;                 // Display name
  logo: string;
  primaryColor: string;
  masterWalletId: string;      // The master wallet for this branding
  isActive: boolean;
  createdAt: string;
}

export interface BlockchainNetwork {
  id: string;
  name: string;
  symbol: string;
  chainId: number;
  rpcUrl: string;
  isEVM: boolean;
}

export interface CryptoToken {
  id: string;
  symbol: string;
  name: string;
  image: string;
  currentPrice: number;
  marketCap: number;
  rank: number;
  priceChange24h: number;
}

// Default networks
const DEFAULT_NETWORKS: BlockchainNetwork[] = [
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

class MasterWalletService {
  private brandings: Branding[] = [];
  private masterWallets: MasterWallet[] = [];
  private userWallets: UserWallet[] = [];
  private networks: BlockchainNetwork[] = DEFAULT_NETWORKS;
  private tokens: CryptoToken[] = [];

  constructor() {
    this.loadFromStorage();
    this.loadTokensFromAPI();
  }

  private loadFromStorage() {
    const stored = localStorage.getItem('brandings');
    if (stored) { try { this.brandings = JSON.parse(stored); } catch { this.brandings = []; } }
    
    const storedMaster = localStorage.getItem('master_wallets');
    if (storedMaster) { try { this.masterWallets = JSON.parse(storedMaster); } catch { this.masterWallets = []; } }
    
    const storedUser = localStorage.getItem('user_wallets');
    if (storedUser) { try { this.userWallets = JSON.parse(storedUser); } catch { this.userWallets = []; } }
  }

  private saveToStorage() {
    localStorage.setItem('brandings', JSON.stringify(this.brandings));
    localStorage.setItem('master_wallets', JSON.stringify(this.masterWallets));
    localStorage.setItem('user_wallets', JSON.stringify(this.userWallets));
  }

  private async loadTokensFromAPI() {
    try {
      const response = await fetch('https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=500&page=1&sparkline=false');
      const data = await response.json();
      this.tokens = data.map((coin: any) => ({
        id: coin.id,
        symbol: coin.symbol.toUpperCase(),
        name: coin.name,
        image: coin.image || '',
        currentPrice: coin.current_price || 0,
        marketCap: coin.market_cap || 0,
        rank: coin.market_cap_rank || 0,
        priceChange24h: coin.price_change_24h || 0
      }));
    } catch (e) { this.tokens = []; }
  }

  // ============================================================================
  // BRANDING MANAGEMENT (Custom Wallets like TigerWallet)
  // ============================================================================

  createBranding(name: string, logo: string, primaryColor: string): Branding {
    // Create master wallet for this branding
    const masterWallet = this.createMasterWalletInternal(
      `Master ${name}`, 
      'hot', 
      'ethereum',
      name.toLowerCase().replace(/\s/g, '_')
    );

    const branding: Branding = {
      id: this.generateUUID(),
      name,
      logo,
      primaryColor,
      masterWalletId: masterWallet.id,
      isActive: true,
      createdAt: new Date().toISOString()
    };

    this.brandings.push(branding);
    this.saveToStorage();
    return branding;
  }

  getBrandings(): Branding[] { return this.brandings; }
  getBranding(brandingId: string): Branding | undefined { return this.brandings.find(b => b.id === brandingId); }

  // ============================================================================
  // MASTER WALLET - Owns all user wallets for a specific branding
  // ============================================================================

  private createMasterWalletInternal(name: string, type: MasterWalletType, blockchain: string, brandingId: string): MasterWallet {
    const wallet: MasterWallet = {
      id: this.generateUUID(),
      brandingId,
      brandingName: name,
      type,
      blockchain,
      address: this.generateAddress(blockchain),
      publicKey: this.generatePublicKey(),
      balance: 0,
      isActive: true,
      createdAt: new Date().toISOString()
    };
    this.masterWallets.push(wallet);
    return wallet;
  }

  getMasterWallets(): MasterWallet[] { return this.masterWallets; }
  getMasterWalletsByBranding(brandingId: string): MasterWallet[] { 
    return this.masterWallets.filter(w => w.brandingId === brandingId); 
  }

  // ============================================================================
  // USER WALLETS - Owned by Master Wallet
  // ============================================================================

  createUserWallet(masterWalletId: string, userId: string, blockchain: string): UserWallet {
    const masterWallet = this.masterWallets.find(w => w.id === masterWalletId);
    if (!masterWallet) throw new Error('Master wallet not found');

    const userWallet: UserWallet = {
      id: this.generateUUID(),
      userId,
      brandingId: masterWallet.brandingId,
      ownerMasterWalletId: masterWalletId,
      ownerMasterWalletAddress: masterWallet.address,
      blockchain,
      address: this.generateAddress(blockchain),
      publicKey: this.generatePublicKey(),
      balance: 0,
      isActive: true,
      createdAt: new Date().toISOString()
    };

    this.userWallets.push(userWallet);
    this.saveToStorage();
    return userWallet;
  }

  // Master wallet controls its user wallets
  controlUserWallet(masterWalletId: string, userWalletId: string): UserWallet | undefined {
    return this.userWallets.find(w => w.id === userWalletId && w.ownerMasterWalletId === masterWalletId);
  }

  // Master wallet approves transactions
  approveTransaction(masterWalletId: string, userWalletId: string, txHash: string): boolean {
    return this.controlUserWallet(masterWalletId, userWalletId) !== undefined;
  }

  // Get user wallets for a master wallet
  getUserWallets(masterWalletId: string): UserWallet[] {
    return this.userWallets.filter(w => w.ownerMasterWalletId === masterWalletId);
  }

  // Get user wallets for a branding
  getUserWalletsByBranding(brandingId: string): UserWallet[] {
    return this.userWallets.filter(w => w.brandingId === brandingId);
  }

  // Get user wallets for a user
  getUserWalletsByUser(userId: string, brandingId?: string): UserWallet[] {
    return this.userWallets.filter(w => w.userId === userId && (!brandingId || w.brandingId === brandingId));
  }

  // ============================================================================
  // NETWORKS & TOKENS
  // ============================================================================

  getNetworks(): BlockchainNetwork[] { return this.networks; }
  getTokens(): CryptoToken[] { return this.tokens; }
  addNetwork(network: BlockchainNetwork) { if (!this.networks.find(n => n.id === network.id)) { this.networks.push(network); } }
  addToken(token: CryptoToken) { if (!this.tokens.find(t => t.id === token.id)) { this.tokens.push(token); } }

  // ============================================================================
  // UTILITIES
  // ============================================================================

  private generateUUID(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      const r = Math.random() * 16 | 0;
      return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
    });
  }

  private generateAddress(blockchain: string): string {
    return '0x' + Array(40).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join('');
  }

  private generatePublicKey(): string {
    return '0x' + Array(130).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join('');
  }
}

export const masterWalletService = new MasterWalletService();
export default masterWalletService;
