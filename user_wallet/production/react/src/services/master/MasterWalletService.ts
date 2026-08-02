/**
 * Master Wallet Service - Owner of All User Wallets
 */

export type MasterWalletType = 'hot' | 'cold' | 'operations';

export interface MasterWallet {
  id: string;
  name: string;
  type: MasterWalletType;
  blockchain: string;
  address: string;
  publicKey: string;
  balance: number;
  isActive: boolean;
  createdAt: string;
}

// User wallet OWNED by master wallet
export interface UserWallet {
  id: string;
  userId: string;
  ownerMasterWalletId: string;   // WHO OWNS THIS WALLET
  ownerAddress: string;          // Master wallet address
  blockchain: string;
  address: string;
  publicKey: string;
  balance: number;
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
  private masterWallets: MasterWallet[] = [];
  private userWallets: UserWallet[] = [];
  private networks: BlockchainNetwork[] = DEFAULT_NETWORKS;
  private tokens: CryptoToken[] = [];

  constructor() {
    this.loadFromStorage();
    this.loadTokensFromAPI();
  }

  private loadFromStorage() {
    const stored = localStorage.getItem('master_wallets');
    if (stored) { try { this.masterWallets = JSON.parse(stored); } catch { this.masterWallets = []; } }
    
    const storedUser = localStorage.getItem('user_wallets');
    if (storedUser) { try { this.userWallets = JSON.parse(storedUser); } catch { this.userWallets = []; } }
    
    const storedNetworks = localStorage.getItem('master_networks');
    if (storedNetworks) { try { this.networks = JSON.parse(storedNetworks); } catch { this.networks = DEFAULT_NETWORKS; } }
  }

  private saveToStorage() {
    localStorage.setItem('master_wallets', JSON.stringify(this.masterWallets));
    localStorage.setItem('user_wallets', JSON.stringify(this.userWallets));
    localStorage.setItem('master_networks', JSON.stringify(this.networks));
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
  // MASTER WALLET - The Owner
  // ============================================================================

  createMasterWallet(name: string, type: MasterWalletType, blockchain: string): MasterWallet {
    const wallet: MasterWallet = {
      id: this.generateUUID(),
      name,
      type,
      blockchain,
      address: this.generateAddress(blockchain),
      publicKey: this.generatePublicKey(),
      balance: 0,
      isActive: true,
      createdAt: new Date().toISOString()
    };
    this.masterWallets.push(wallet);
    this.saveToStorage();
    return wallet;
  }

  getMasterWallets(): MasterWallet[] { return this.masterWallets; }

  getMasterWallet(walletId: string): MasterWallet | undefined {
    return this.masterWallets.find(w => w.id === walletId);
  }

  // ============================================================================
  // USER WALLETS - Owned by Master Wallet
  // ============================================================================

  // Master wallet creates/owns user wallets
  createUserWallet(masterWalletId: string, userId: string, blockchain: string): UserWallet {
    const masterWallet = this.masterWallets.find(w => w.id === masterWalletId);
    if (!masterWallet) throw new Error('Master wallet not found');

    const userWallet: UserWallet = {
      id: this.generateUUID(),
      userId,
      ownerMasterWalletId: masterWalletId,   // OWNERSHIP
      ownerAddress: masterWallet.address,    // Master wallet address owns this
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

  // Master wallet can control any user wallet it owns
  controlUserWallet(masterWalletId: string, userWalletId: string): UserWallet | undefined {
    return this.userWallets.find(w => w.id === userWalletId && w.ownerMasterWalletId === masterWalletId);
  }

  // Master wallet approves all transactions from user wallets
  approveTransaction(masterWalletId: string, userWalletId: string, txHash: string): boolean {
    return this.controlUserWallet(masterWalletId, userWalletId) !== undefined;
  }

  // Get all user wallets owned by a master wallet
  getUserWallets(masterWalletId: string): UserWallet[] {
    return this.userWallets.filter(w => w.ownerMasterWalletId === masterWalletId);
  }

  // Get all user wallets for a user
  getUserWalletsByUser(userId: string): UserWallet[] {
    return this.userWallets.filter(w => w.userId === userId);
  }

  // ============================================================================
  // Networks & Tokens
  // ============================================================================

  getNetworks(): BlockchainNetwork[] { return this.networks; }
  getTokens(): CryptoToken[] { return this.tokens; }
  
  addNetwork(network: BlockchainNetwork) {
    if (!this.networks.find(n => n.id === network.id)) {
      this.networks.push(network);
      this.saveToStorage();
    }
  }
  
  addToken(token: CryptoToken) {
    if (!this.tokens.find(t => t.id === token.id)) {
      this.tokens.push(token);
    }
  }

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
