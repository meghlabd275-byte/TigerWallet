/**
 * Master Wallet Service - Profit Sharing with TigerWallet Superadmin
 * Each custom branding must share 20% profit with TigerWallet Superadmin
 */

export type MasterWalletType = 'hot' | 'cold' | 'operations';

// TigerWallet Superadmin - Receives 20% profit from all custom brandings.
// The address MUST be provided via configuration (env / backend), never hardcoded.
const TIGERWALLET_SUPERADMIN_ADDRESS =
  (import.meta?.env?.VITE_TIGERWALLET_SUPERADMIN_ADDRESS as string | undefined) ||
  (typeof process !== 'undefined' && (process.env?.REACT_APP_TIGERWALLET_SUPERADMIN_ADDRESS as string | undefined)) ||
  '';
const PROFIT_SHARE_PERCENT = 20; // 20% goes to TigerWallet

export interface MasterWallet {
  id: string;
  brandingId: string;
  brandingName: string;
  name: string;
  type: MasterWalletType;
  blockchain: string;
  address: string;
  publicKey: string;
  balance: number;
  superadminShareAddress: string;  // Must be TigerWallet superadmin
  profitSharePercent: number;      // 20% mandatory
  isActive: boolean;
  createdAt: string;
}

export interface UserWallet {
  id: string;
  userId: string;
  brandingId: string;
  ownerMasterWalletId: string;
  ownerMasterWalletAddress: string;
  blockchain: string;
  address: string;
  publicKey: string;
  balance: number;
  isActive: boolean;
  createdAt: string;
}

export interface Branding {
  id: string;
  name: string;
  logo: string;
  primaryColor: string;
  masterWalletId: string;
  superadminShareAddress: string;   // TigerWallet superadmin address
  profitSharePercent: number;      // 20% mandatory
  isActive: boolean;
  createdAt: string;
}

export interface ProfitRecord {
  id: string;
  brandingId: string;
  masterWalletId: string;
  totalProfit: number;
  superadminShare: number;    // 20% to TigerWallet
  brandingShare: number;      // 80% to branding
  superadminAddress: string;
  timestamp: string;
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

export class MasterWalletService {
  private brandings: Branding[] = [];
  private masterWallets: MasterWallet[] = [];
  private userWallets: UserWallet[] = [];
  private profitRecords: ProfitRecord[] = [];
  private networks: BlockchainNetwork[] = DEFAULT_NETWORKS;
  private tokens: CryptoToken[] = [];
  
  // TigerWallet Superadmin - MANDATORY
  readonly SUPERADMIN_ADDRESS: string = TIGERWALLET_SUPERADMIN_ADDRESS;
  readonly MANDATORY_SHARE_PERCENT: number = PROFIT_SHARE_PERCENT;

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
    
    const storedProfit = localStorage.getItem('profit_records');
    if (storedProfit) { try { this.profitRecords = JSON.parse(storedProfit); } catch { this.profitRecords = []; } }
  }

  private saveToStorage() {
    localStorage.setItem('brandings', JSON.stringify(this.brandings));
    localStorage.setItem('master_wallets', JSON.stringify(this.masterWallets));
    localStorage.setItem('user_wallets', JSON.stringify(this.userWallets));
    localStorage.setItem('profit_records', JSON.stringify(this.profitRecords));
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
  // PROFIT SHARING - MANDATORY 20% TO TIGERWALLET SUPERADMIN
  // ============================================================================

  /**
   * Calculate profit share - MANDATORY 20% to TigerWallet Superadmin
   * Without this sharing, the custom branding wallet will NOT work.
   * Throws if no superadmin address is configured (fail-closed).
   */
  calculateProfitShare(totalProfit: number): { superadminShare: number, brandingShare: number } {
    if (!this.SUPERADMIN_ADDRESS) {
      throw new Error('Profit sharing requires a configured TigerWallet superadmin address (VITE_TIGERWALLET_SUPERADMIN_ADDRESS).');
    }
    const superadminShare = (totalProfit * this.MANDATORY_SHARE_PERCENT) / 100;
    const brandingShare = totalProfit - superadminShare;
    return { superadminShare, brandingShare };
  }

  /**
   * Record profit sharing - MUST be called for every transaction
   */
  recordProfit(brandingId: string, masterWalletId: string, totalProfit: number): ProfitRecord {
    const { superadminShare, brandingShare } = this.calculateProfitShare(totalProfit);
    
    const record: ProfitRecord = {
      id: this.generateUUID(),
      brandingId,
      masterWalletId,
      totalProfit,
      superadminShare,
      brandingShare,
      superadminAddress: this.SUPERADMIN_ADDRESS,
      timestamp: new Date().toISOString()
    };
    
    this.profitRecords.push(record);
    this.saveToStorage();
    
    // Verify superadmin got their share
    console.log(`[Profit Sharing] Branding ${brandingId}: $${superadminShare} (20%) → TigerWallet Superadmin ${this.SUPERADMIN_ADDRESS}`);
    
    return record;
  }

  /**
   * Get total profit shared with TigerWallet Superadmin
   */
  getTotalSuperadminProfit(brandingId?: string): number {
    return this.profitRecords
      .filter(r => !brandingId || r.brandingId === brandingId)
      .reduce((sum, r) => sum + r.superadminShare, 0);
  }

  /**
   * Get profit records
   */
  getProfitRecords(brandingId?: string): ProfitRecord[] {
    return brandingId ? this.profitRecords.filter(r => r.brandingId === brandingId) : this.profitRecords;
  }

  /**
   * Verify superadmin address is set correctly
   */
  verifySuperadminSetup(): boolean {
    return this.SUPERADMIN_ADDRESS !== "" && this.MANDATORY_SHARE_PERCENT === 20;
  }

  // ============================================================================
  // BRANDING MANAGEMENT - With Mandatory Profit Sharing
  // ============================================================================

  createBranding(name: string, logo: string, primaryColor: string): Branding {
    // Create master wallet with MANDATORY superadmin share
    const masterWallet = this.createMasterWalletInternal(
      `Master ${name}`,
      'hot',
      'ethereum',
      name.toLowerCase().replace(/\s/g, '_'),
      name
    );

    const branding: Branding = {
      id: this.generateUUID(),
      name,
      logo,
      primaryColor,
      masterWalletId: masterWallet.id,
      superadminShareAddress: this.SUPERADMIN_ADDRESS,  // MANDATORY
      profitSharePercent: this.MANDATORY_SHARE_PERCENT, // 20% MANDATORY
      isActive: true,
      createdAt: new Date().toISOString()
    };

    this.brandings.push(branding);
    this.saveToStorage();
    
    console.log(`[Branding Created] ${name} - 20% profit sharing to ${this.SUPERADMIN_ADDRESS} is MANDATORY`);
    
    return branding;
  }

  getBrandings(): Branding[] { return this.brandings; }
  getBranding(brandingId: string): Branding | undefined { return this.brandings.find(b => b.id === brandingId); }

  // Verify branding has proper superadmin setup
  isBrandingValid(brandingId: string): boolean {
    const branding = this.getBranding(brandingId);
    if (!branding) return false;
    return branding.superadminShareAddress === this.SUPERADMIN_ADDRESS && 
           branding.profitSharePercent === this.MANDATORY_SHARE_PERCENT;
  }

  // ============================================================================
  // MASTER WALLET - With Superadmin Enforcement
  // ============================================================================

  private createMasterWalletInternal(name: string, type: MasterWalletType, blockchain: string, brandingId: string, brandingName: string): MasterWallet {
    // A real master/branding wallet address is derived by the canonical
    // wallet-api backend (/wallets) from a BIP-39 seed; this client must NOT
    // fabricate one. Callers should POST /wallets on the backend and store the
    // returned address/label here.
    throw new Error(
      `Master wallet creation for "${name}" requires a real address from the canonical wallet-api backend (POST /api/v1/wallets); client-side address fabrication is disabled`
    );
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

    // Verify branding has valid superadmin setup
    if (!this.isBrandingValid(masterWallet.brandingId)) {
      throw new Error('Branding must have valid 20% profit sharing to TigerWallet Superadmin');
    }

    // A real user-wallet address is derived by the canonical wallet-api
    // backend (POST /api/v1/wallets). This client never fabricates one —
    // fail honestly so callers wire the backend.
    throw new Error(
      `User wallet creation for "${userId}" requires a real address from the canonical wallet-api backend (POST /api/v1/wallets); client-side address fabrication is disabled`
    );
  }

  controlUserWallet(masterWalletId: string, userWalletId: string): UserWallet | undefined {
    return this.userWallets.find(w => w.id === userWalletId && w.ownerMasterWalletId === masterWalletId);
  }

  getUserWallets(masterWalletId: string): UserWallet[] {
    return this.userWallets.filter(w => w.ownerMasterWalletId === masterWalletId);
  }

  getUserWalletsByBranding(brandingId: string): UserWallet[] {
    return this.userWallets.filter(w => w.brandingId === brandingId);
  }

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
    // Use the Web Crypto API for a real CSPRNG-backed UUID (not Math.random).
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return crypto.randomUUID();
    }
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  }

  private generateAddress(_blockchain: string): string {
    // Address derivation is performed by the canonical wallet-api backend
    // (/wallets) from a real BIP-39 seed. This client never fabricates an
    // address — fail honestly so callers wire the real backend.
    throw new Error(
      'Address derivation is performed by the canonical wallet-api backend (/wallets); never fabricated client-side'
    );
  }

  private generatePublicKey(): string {
    throw new Error(
      'Public-key derivation is performed by the canonical wallet-api backend; never fabricated client-side'
    );
  }
}

export const masterWalletService = new MasterWalletService();
export default masterWalletService;
