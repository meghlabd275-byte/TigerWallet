/**
 * TigerWallet API Client Library
 * 
 * Complete TypeScript API client for connecting frontend to all backend services.
 * Provides type-safe access to all TigerWallet services.
 */

import axios, { AxiosInstance } from 'axios';

// ============================================================================
// Types
// ============================================================================

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Wallet {
  id: string;
  address: string;
  chainType: string;
  chainId: number;
  type: 'user' | 'master' | 'white_label';
  name: string;
  createdAt: number;
}

export interface WalletBalance {
  available: string;
  total: string;
  usdValue: number;
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  value: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
}

export interface StakingPool {
  id: string;
  name: string;
  chainId: number;
  apy: string;
  totalStaked: string;
  status: 'active' | 'paused';
}

export interface EarnProduct {
  id: string;
  name: string;
  productType: 'fixed' | 'flexible';
  apy: string;
  minDeposit: string;
  status: 'active' | 'paused';
}

export interface NFTCollection {
  id: string;
  name: string;
  contractAddress: string;
  floorPrice: string;
  totalSupply: number;
}

export interface PerpetualMarket {
  id: string;
  pair: string;
  markPrice: string;
  maxLeverage: string;
  status: 'active' | 'paused';
}

export interface AddressBookEntry {
  id: string;
  name: string;
  address: string;
  chain: string;
  symbol: string;
  notes?: string;
  isFavorite?: boolean;
  lastUsed?: number;
}

export interface Proposal {
  id: string;
  title: string;
  description: string;
  proposer: string;
  proposerAvatar?: string;
  status: 'active' | 'pending' | 'passed' | 'failed' | 'executed' | 'queued';
  type: 'governance' | 'treasury' | 'parameter' | 'partnership';
  forVotes: number;
  againstVotes: number;
  totalVotes: number;
  quorumRequired: number;
  startTime: number;
  endTime: number;
  createdAt: number;
  discussionUrl?: string;
  ipfsHash?: string;
  executionData?: string;
  forPercentage: number;
  againstPercentage: number;
  quorumPercentage: number;
  myVote?: 'for' | 'against' | 'abstain';
  myVotingPower?: number;
}

export interface Delegate {
  address: string;
  name?: string;
  votingPower: number;
  delegatedPower: number;
  proposalsCreated: number;
  votesCast: number;
  since: number;
}

export interface LaunchpadProject {
  id: string;
  name: string;
  symbol: string;
  description: string;
  logo?: string;
  status: 'upcoming' | 'live' | 'completed' | 'cancelled';
  chain: string;
  chainIcon?: string;
  tokenPrice: number;
  totalRaise: number;
  hardCap: number;
  softCap: number;
  startTime: number;
  endTime: number;
  minAllocation: number;
  maxAllocation: number;
  participants: number;
  totalRaised: number;
  tokenSymbol?: string;
  saleTokenAddress?: string;
  acceptedTokens: string[];
  vestingPercent?: number;
  vestingCliff?: number;
  vestingPeriod?: number;
  website?: string;
  twitter?: string;
  telegram?: string;
  whitepaper?: string;
  kycRequired?: boolean;
  auditStatus?: 'completed' | 'in-progress' | 'pending';
}

export interface PoolToken {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  logoURI?: string;
  priceUSD?: number;
  chainId: number;
  isPopular?: boolean;
  isNative?: boolean;
  isStable?: boolean;
}

export interface LiquidityPool {
  id: string;
  address: string;
  token0: PoolToken;
  token1: PoolToken;
  dex: string;
  dexName: string;
  feeTier: number;
  tvlUSD: number;
  volume24h: number;
  volume7d: number;
  apr: number;
  token0Reserve: string;
  token1Reserve: string;
  liquidity: number;
  price0: number;
  price1: number;
}

export interface LiquidityPosition {
  id: string;
  pool: LiquidityPool;
  token0Amount: string;
  token1Amount: string;
  liquidityTokenBalance: string;
  totalLiquidity: number;
  poolShare: number;
  feesEarned0: string;
  feesEarned1: string;
  rangeLow?: number;
  rangeHigh?: number;
  isActive: boolean;
}

export interface PortfolioAsset {
  symbol: string;
  name: string;
  balance: number;
  value: number;
  change24h: number;
  icon?: string;
  address?: string;
}

export interface PortfolioPosition {
  type: 'liquidity' | 'farming' | 'staking';
  protocol: string;
  pair: string;
  value: number;
  apr: number;
  pnl: number;
  icon?: string;
}

export interface Portfolio {
  assets: PortfolioAsset[];
  positions: PortfolioPosition[];
  transactions: Transaction[];
}

// Bot Platform
export interface BotUser {
  id: string;
  address: string;
  name: string;
  role: 'admin' | 'operator' | 'client';
  email?: string;
  createdAt: number;
  isActive: boolean;
}

export interface BotConfig {
  minInvestment: number;
  maxInvestment: number;
  targetApy: number;
  riskLevel: number;
  maxDailyLoss: number;
  stopLoss: number;
}

export interface Bot {
  id: string;
  name: string;
  type: 'market_maker' | 'arbitrage' | 'sniper' | 'liquidity' | 'front_run' | 'mev' | 'flash_loan' | 'cross_chain' | 'perp_hedge';
  status: 'running' | 'paused' | 'stopped';
  owner: string;
  profit: number;
  volume: number;
  trades: number;
  winRate: number;
  createdAt: number;
  lastActive: number;
  config: BotConfig;
}

export interface BotTransaction {
  id: string;
  botId: string;
  exchange: string;
  type: 'buy' | 'sell';
  amount: number;
  price: number;
  profit: number;
  status: 'success' | 'failed';
  timestamp: number;
}

// Fiat ramp
export interface FiatProvider {
  id: string;
  name: string;
  logo?: string;
  supportedMethods: string[];
  supportedFiat: string[];
  supportedCrypto: string[];
  minAmount: number;
  maxAmount: number;
  fees: number;
  processingTime: string;
  available: boolean;
}

export interface FiatRate {
  symbol: string;
  price: number;
  currency: string;
  change24h?: number;
}

export interface FiatOrder {
  id: string;
  provider: string;
  type: 'buy' | 'sell';
  fiatAmount: number;
  cryptoAmount: number;
  cryptoCurrency: string;
  fiatCurrency: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  createdAt: number;
  expiresAt?: number;
  paymentMethod: string;
  redirectUrl?: string;
}

// Social recovery
export interface Guardian {
  address: string;
  name: string;
  confirmed: boolean;
  addedAt: number;
}

export interface RecoveryRequest {
  id: string;
  newOwner: string;
  guardians: Guardian[];
  confirmations: number;
  threshold: number;
  status: 'pending' | 'confirmed' | 'completed' | 'cancelled';
  createdAt: number;
}

// Insurance fund
export interface InsuranceStats {
  totalPool: number;
  reserves: number;
  coverage: number;
  claimsPaid: number;
  apy: number;
  members: number;
}

export interface InsurancePosition {
  id: string;
  pool: string;
  amount: number;
  coverage: number;
  premium: number;
  startTime: number;
  expiryTime: number;
  status: 'active' | 'expired' | 'claimed';
}

export interface InsuranceClaim {
  id: string;
  pool: string;
  amount: number;
  reason: string;
  status: 'pending' | 'approved' | 'paid' | 'rejected';
  date: number;
}

export interface InsuranceProduct {
  name: string;
  coverage: string;
  premium: string;
  desc: string;
}

// IEO / IDO
export interface IEOProject {
  id: string;
  name: string;
  symbol: string;
  description: string;
  price: number;
  hardCap: number;
  softCap: number;
  raised: number;
  participants: number;
  status: 'upcoming' | 'sale' | 'ended';
  startTime: number;
  endTime: number;
  minBuy: number;
  maxBuy: number;
  chain: string;
  logo?: string;
  tokenAllocation: number;
  listingPrice: number;
}

// Leaderboard / copy trading
export interface LeaderboardTrader {
  id: string;
  address: string;
  ensName?: string;
  avatar?: string;
  isVerified: boolean;
  isCopiable: boolean;
  totalPnL: number;
  totalTrades: number;
  winRate: number;
  avgHoldingTime?: string;
  followers: number;
  following: number;
  totalVolume: number;
  profitFactor?: number;
  sharpeRatio?: number;
  maxDrawdown?: number;
  tradingPair: string;
  monthlyReturn: number;
  lastTradeTime: number;
  isFollowing: boolean;
}

export interface LeaderboardEntry {
  rank: number;
  trader: LeaderboardTrader;
  monthlyReturn: number;
  totalPnL: number;
}

export interface CopySignal {
  id: string;
  trader: string;
  tokenA: string;
  tokenB: string;
  action: 'BUY' | 'SELL';
  amount: string;
  price: string;
  timestamp: number;
  status: 'active' | 'closed';
  pnl?: number;
}

export interface CopyPosition {
  id: string;
  signal: CopySignal;
  amount: string;
  entryPrice: string;
  currentPrice: string;
  pnl: number;
  pnlPercent: number;
  status: 'open' | 'closed';
  openedAt: number;
  closedAt?: number;
}

// Security scanner
export interface ScanResult {
  id: string;
  address: string;
  risk: 'safe' | 'warning' | 'danger';
  issues: string[];
  scannedAt: number;
}

// ============================================================================
// API Client
// ============================================================================

class TigerWalletAPI {
  private client: AxiosInstance;
  private static instance: TigerWalletAPI;

  private constructor(baseURL: string = '/api/v1') {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: { 'Content-Type': 'application/json' },
    });

    this.client.interceptors.request.use((config) => {
      // Prefer a bot-platform token when present (the bot dashboard sets it),
      // otherwise fall back to the wallet auth token.
      const token = localStorage.getItem('bot_auth_token') || localStorage.getItem('auth_token');
      if (token) config.headers.Authorization = `Bearer ${token}`;
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token');
          localStorage.removeItem('bot_auth_token');
        }
        return Promise.reject(error);
      }
    );
  }

  public static getInstance(): TigerWalletAPI {
    if (!TigerWalletAPI.instance) {
      TigerWalletAPI.instance = new TigerWalletAPI();
    }
    return TigerWalletAPI.instance;
  }

  setBaseURL(url: string): void {
    this.client.defaults.baseURL = url;
  }

  setAuthToken(token: string): void {
    localStorage.setItem('auth_token', token);
  }

  // Wallet
  // Both create and mnemonic-import are served by the wallet-api backend's
  // POST /api/v1/wallets endpoint, which derives the secp256k1 key + EVM
  // address and stores the encrypted seed. Import is simply create-with-a-
  // mnemonic; omit `mnemonic` to generate a fresh wallet, include it to import.
  async createWallet(params: {
    password: string;
    label?: string;
    chainId?: number;
    mnemonic?: string;
    accountIndex?: number;
    entropyBits?: number;
  }): Promise<APIResponse<Wallet>> {
    return (await this.client.post('/wallet/create', params)).data;
  }

  async importWallet(params: {
    mnemonic: string;
    password: string;
    label?: string;
    chainId?: number;
    accountIndex?: number;
  }): Promise<APIResponse<Wallet>> {
    return (await this.client.post('/wallet/import', params)).data;
  }

  async getWallets(): Promise<APIResponse<Wallet[]>> {
    return (await this.client.get('/wallet/list')).data;
  }

  async getWalletBalance(address: string, chainId: number): Promise<APIResponse<WalletBalance>> {
    return (await this.client.get('/balance', { params: { address, chain_id: chainId } })).data;
  }

  async sendTransaction(walletId: string, to: string, value: string, chainId: number): Promise<APIResponse<Transaction>> {
    return (await this.client.post('/wallet/send', { walletId, to, value, chainId })).data;
  }

  async getTransactions(walletId?: string, params?: { chainId?: number; status?: string; type?: string }): Promise<APIResponse<Transaction[]>> {
    const query = { ...(params || {}) };
    if (walletId) (query as any).walletId = walletId;
    return (await this.client.get('/wallet/transactions', { params: query })).data;
  }

  // Staking
  async getStakingPools(params?: { chainId?: number }): Promise<APIResponse<StakingPool[]>> {
    return (await this.client.get('/staking/pools', { params })).data;
  }

  async stake(poolId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post('/staking/stake', { poolId, amount })).data;
  }

  async unstake(stakeId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/staking/unstake', { stakeId })).data;
  }

  async claimStakingRewards(stakeId: string): Promise<APIResponse<string>> {
    return (await this.client.post('/staking/claim', { stakeId })).data;
  }

  // Earn
  async getEarnProducts(params?: { chainId?: number }): Promise<APIResponse<EarnProduct[]>> {
    return (await this.client.get('/earn/products', { params })).data;
  }

  async deposit(productId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post('/earn/deposit', { productId, amount })).data;
  }

  async withdraw(depositId: string): Promise<APIResponse<string>> {
    return (await this.client.post('/earn/withdraw', { depositId })).data;
  }

  // NFT
  async getNFTCollections(): Promise<APIResponse<NFTCollection[]>> {
    return (await this.client.get('/nft/collections')).data;
  }

  async getNFTItems(collectionId: string): Promise<APIResponse<any[]>> {
    return (await this.client.get(`/nft/collections/${collectionId}/nfts`)).data;
  }

  async createListing(itemId: string, price: string): Promise<APIResponse<any>> {
    return (await this.client.post('/nft/list', { itemId, price })).data;
  }

  async buyNFT(listingId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/nft/buy', { listingId })).data;
  }

  // Perpetual
  async getPerpetualMarkets(): Promise<APIResponse<PerpetualMarket[]>> {
    return (await this.client.get('/perpetual/markets')).data;
  }

  async openPosition(marketId: string, side: string, size: string, leverage: string): Promise<APIResponse<any>> {
    return (await this.client.post('/perpetual/open', { marketId, side, size, leverage }));
  }

  async closePosition(positionId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/perpetual/close', { positionId })).data;
  }

  // Copy Trading
  async getTraders(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/copy-trading/traders')).data;
  }

  // Token Deployer
  async createTokenDeployment(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/token/create', config)).data;
  }

  // Multisig
  async createMultisigWallet(name: string, owners: string[], requiredSigs: number): Promise<APIResponse<any>> {
    return (await this.client.post('/multisig/create', { name, owners, requiredSigs })).data;
  }

  async signTransaction(txId: string, signature: string): Promise<APIResponse<any>> {
    return (await this.client.post('/multisig/sign', { txId, signature })).data;
  }

  // Airdrop
  async getAirdropCampaigns(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/airdrop/campaigns')).data;
  }

  async claimAirdrop(campaignId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/airdrop/claim', { campaignId })).data;
  }

  // Coupon
  async validateCoupon(code: string): Promise<APIResponse<any>> {
    return (await this.client.get('/coupon/validate', { params: { code } })).data;
  }

  // Red Packets
  async createRedPacket(totalAmount: string, quantity: number, claimType: string): Promise<APIResponse<any>> {
    return (await this.client.post('/red-packets/create', { totalAmount, quantity, claimType })).data;
  }

  async claimRedPacket(packetId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/red-packets/claim', { packetId })).data;
  }

  // Auth
  async login(email: string, password: string): Promise<APIResponse<{ token: string }>> {
    return (await this.client.post('/auth/login', { email, password })).data;
  }

  async register(email: string, password: string, name: string): Promise<APIResponse<any>> {
    return (await this.client.post('/auth/register', { email, password, name })).data;
  }

  // Portfolio
  async getPortfolio(walletId?: string): Promise<APIResponse<Portfolio>> {
    const params = walletId ? { walletId } : undefined;
    return (await this.client.get('/portfolio', { params })).data;
  }

  // Address Book
  async getAddressBook(): Promise<APIResponse<AddressBookEntry[]>> {
    return (await this.client.get('/address-book/contacts')).data;
  }

  async addAddress(entry: Omit<AddressBookEntry, 'id' | 'lastUsed'>): Promise<APIResponse<AddressBookEntry>> {
    return (await this.client.post('/address-book/contacts', entry)).data;
  }

  async updateAddress(id: string, entry: Partial<AddressBookEntry>): Promise<APIResponse<AddressBookEntry>> {
    return (await this.client.put(`/address-book/contacts/${id}`, entry)).data;
  }

  async deleteAddress(id: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/address-book/contacts/${id}`)).data;
  }

  // Governance
  async getProposals(status?: string): Promise<APIResponse<Proposal[]>> {
    const params = status ? { status } : undefined;
    return (await this.client.get('/governance/proposals', { params })).data;
  }

  async getProposal(id: string): Promise<APIResponse<Proposal>> {
    return (await this.client.get(`/governance/proposals/${id}`)).data;
  }

  async createProposal(proposal: { title: string; description: string; type: Proposal['type']; executionData?: string }): Promise<APIResponse<Proposal>> {
    return (await this.client.post('/governance/proposals', proposal)).data;
  }

  async voteOnProposal(proposalId: string, vote: 'for' | 'against' | 'abstain', reason?: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/governance/proposals/${proposalId}/vote`, { vote, reason })).data;
  }

  async executeProposal(proposalId: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/governance/proposals/${proposalId}/execute`)).data;
  }

  async getGovernanceDelegates(): Promise<APIResponse<Delegate[]>> {
    return (await this.client.get('/governance/delegates')).data;
  }

  async getVotingPower(): Promise<APIResponse<number>> {
    return (await this.client.get('/governance/voting-power')).data;
  }

  async delegateVotes(delegateAddress: string): Promise<APIResponse<any>> {
    return (await this.client.post('/governance/delegate', { delegateAddress })).data;
  }

  // Launchpad
  async getLaunchpadProjects(status?: string): Promise<APIResponse<LaunchpadProject[]>> {
    const params = status ? { status } : undefined;
    return (await this.client.get('/launchpad/projects', { params })).data;
  }

  async getProjectDetails(projectId: string): Promise<APIResponse<LaunchpadProject>> {
    return (await this.client.get(`/launchpad/projects/${projectId}`)).data;
  }

  async participateInSale(projectId: string, amount: string, token: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/launchpad/projects/${projectId}/participate`, { amount, token })).data;
  }

  async claimLaunchpadTokens(projectId: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/launchpad/projects/${projectId}/claim`)).data;
  }

  // Liquidity Pools
  async getLiquidityPools(params?: { chainId?: number; dex?: string }): Promise<APIResponse<LiquidityPool[]>> {
    return (await this.client.get('/pool/list', { params })).data;
  }

  async getPoolPositions(walletId?: string): Promise<APIResponse<LiquidityPosition[]>> {
    const params = walletId ? { walletId } : undefined;
    return (await this.client.get('/pool/positions', { params })).data;
  }

  async addLiquidity(poolId: string, amount0: string, amount1: string): Promise<APIResponse<any>> {
    return (await this.client.post('/pool/add-liquidity', { poolId, amount0, amount1 })).data;
  }

  async removeLiquidity(positionId: string, liquidity?: string): Promise<APIResponse<any>> {
    return (await this.client.post('/pool/remove-liquidity', { positionId, liquidity })).data;
  }

  async createLiquidityPool(config: {
    token0: string;
    token1: string;
    feeTier: number;
    amount0: string;
    amount1: string;
    chainId?: number;
    priceRangeLow?: number;
    priceRangeHigh?: number;
  }): Promise<APIResponse<LiquidityPool>> {
    return (await this.client.post('/pool/create', config)).data;
  }

  // Analytics
  async getAnalytics(params?: { range?: string }): Promise<APIResponse<any>> {
    return (await this.client.get('/analytics', { params })).data;
  }

  async getAnalyticsRevenue(params?: { range?: string }): Promise<APIResponse<any>> {
    return (await this.client.get('/analytics/revenue', { params })).data;
  }

  async getTransactionStats(params?: { range?: string }): Promise<APIResponse<any>> {
    return (await this.client.get('/analytics/transactions', { params })).data;
  }

  // Admin Fees
  async getFeeConfigs(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/fees')).data;
  }

  async createFeeConfig(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/admin/fees', config)).data;
  }

  async updateFeeConfig(feeId: string, config: any): Promise<APIResponse<any>> {
    return (await this.client.put(`/admin/fees/${feeId}`, config)).data;
  }

  async deleteFeeConfig(feeId: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/admin/fees/${feeId}`)).data;
  }

  async getFeeTransactions(params?: { type?: string; status?: string }): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/fees/transactions', { params })).data;
  }

  async getFeeRevenueStats(params?: { range?: string }): Promise<APIResponse<any>> {
    return (await this.client.get('/admin/fees/revenue', { params })).data;
  }

  // Admin Wallets
  async getAdminWallets(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/wallets')).data;
  }

  async createAdminWallet(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/admin/wallets', config)).data;
  }

  async updateAdminWallet(walletId: string, config: any): Promise<APIResponse<any>> {
    return (await this.client.put(`/admin/wallets/${walletId}`, config)).data;
  }

  async deleteAdminWallet(walletId: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/admin/wallets/${walletId}`)).data;
  }

  async getAdminTransactions(walletId?: string): Promise<APIResponse<any[]>> {
    const params = walletId ? { walletId } : undefined;
    return (await this.client.get('/admin/wallets/transactions', { params })).data;
  }

  // Terminal (Advanced Trading)
  async getOrderbook(symbol: string): Promise<APIResponse<any>> {
    return (await this.client.get(`/terminal/orderbook/${symbol}`)).data;
  }

  async getAdvancedPositions(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/terminal/positions')).data;
  }

  async closeAdvancedPosition(positionId: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/terminal/positions/${positionId}/close`)).data;
  }

  async getAdvancedOrders(): Promise<APIResponse<any>> {
    return (await this.client.get('/terminal/orders')).data;
  }

  async placeAdvancedOrder(order: any): Promise<APIResponse<any>> {
    return (await this.client.post('/terminal/orders', order)).data;
  }

  async cancelAdvancedOrder(orderId: string): Promise<APIResponse<any>> {
    return (await this.client.delete(`/terminal/orders/${orderId}`)).data;
  }

  // Chain Admin
  async getChains(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/chains')).data;
  }

  async addChain(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/admin/chains', config)).data;
  }

  async updateChainConfig(chainId: string, config: any): Promise<APIResponse<any>> {
    return (await this.client.put(`/admin/chains/${chainId}`, config)).data;
  }

  async getValidators(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/chains/validators')).data;
  }

  async addValidator(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/admin/chains/validators', config)).data;
  }

  async getBridges(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/chains/bridges')).data;
  }

  async addBridge(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/admin/chains/bridges', config)).data;
  }

  async getTokenDeployments(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/admin/chains/token-deployments')).data;
  }

  async getChainMetrics(): Promise<APIResponse<Record<string, any>>> {
    return (await this.client.get('/admin/chains/metrics')).data;
  }

  // TWAP / DCA
  async getTwapOrders(): Promise<APIResponse<any>> {
    return (await this.client.get('/twap')).data;
  }

  async createTwapOrder(order: any): Promise<APIResponse<any>> {
    return (await this.client.post('/twap', order)).data;
  }

  async updateTwapOrder(orderId: string, config: any): Promise<APIResponse<any>> {
    return (await this.client.put(`/twap/${orderId}`, config)).data;
  }

  async cancelTwapOrder(orderId: string): Promise<APIResponse<any>> {
    return (await this.client.delete(`/twap/${orderId}`)).data;
  }

  // Token Price (shared by fiat ramps)
  async getTokenPrice(symbol: string, currency: string = 'USD'): Promise<APIResponse<FiatRate>> {
    return (await this.client.get('/price', { params: { symbol, currency } })).data;
  }

  // KYC (fiat ramps)
  async getKycStatus(): Promise<APIResponse<any>> {
    return (await this.client.get('/kyc/status')).data;
  }

  // Bot Platform
  async getBotUsers(): Promise<APIResponse<BotUser[]>> {
    return (await this.client.get('/bots/users')).data;
  }

  async getBots(): Promise<APIResponse<Bot[]>> {
    return (await this.client.get('/bots/instances')).data;
  }

  async getBotTransactions(botId?: string): Promise<APIResponse<BotTransaction[]>> {
    const params = botId ? { botId } : undefined;
    return (await this.client.get('/bots/transactions', { params })).data;
  }

  async createBot(config: {
    name: string;
    type: Bot['type'];
    minInvestment: number;
    maxInvestment: number;
    targetApy: number;
    riskLevel: number;
  }): Promise<APIResponse<Bot>> {
    return (await this.client.post('/bots/create', config)).data;
  }

  async startBot(botId: string): Promise<APIResponse<Bot>> {
    return (await this.client.post(`/bots/${botId}/start`)).data;
  }

  async stopBot(botId: string): Promise<APIResponse<Bot>> {
    return (await this.client.post(`/bots/${botId}/stop`)).data;
  }

  async pauseBot(botId: string): Promise<APIResponse<Bot>> {
    return (await this.client.post(`/bots/${botId}/pause`)).data;
  }

  async deleteBot(botId: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/bots/${botId}`)).data;
  }

  async getCurrentBotUser(): Promise<APIResponse<BotUser>> {
    return (await this.client.get('/bots/me')).data;
  }

  async createBotUser(user: { name: string; email?: string; role: BotUser['role']; address: string }): Promise<APIResponse<BotUser>> {
    return (await this.client.post('/bots/users', user)).data;
  }

  async deleteBotUser(userId: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/bots/users/${userId}`)).data;
  }

  // Bot-platform auth (separate JWT from the wallet_api; stored under
  // 'bot_auth_token' so it does not clobber the wallet 'auth_token').
  async loginBotPlatform(username: string, password: string): Promise<{ token: string; user_id: string; role: string }> {
    const res = await axios.post('/api/v1/bot-platform/auth/login', { username, password });
    const data = res.data;
    if (data?.token) {
      localStorage.setItem('bot_auth_token', data.token);
      this.client.defaults.headers.Authorization = `Bearer ${data.token}`;
    }
    return data;
  }

  async setBotPlatformToken(token: string | null): Promise<void> {
    if (token) {
      localStorage.setItem('bot_auth_token', token);
      this.client.defaults.headers.Authorization = `Bearer ${token}`;
    } else {
      localStorage.removeItem('bot_auth_token');
      delete this.client.defaults.headers.Authorization;
    }
  }

  // Fiat Ramp
  async getFiatRates(params?: { currency?: string; crypto?: string }): Promise<APIResponse<FiatRate[]>> {
    return (await this.client.get('/fiat/rates', { params })).data;
  }

  async getFiatProviders(params?: { fiat?: string; crypto?: string }): Promise<APIResponse<FiatProvider[]>> {
    return (await this.client.get('/fiat/providers', { params })).data;
  }

  async createFiatOrder(order: {
    providerId: string;
    type: 'buy' | 'sell';
    fiatAmount: number;
    cryptoCurrency: string;
    fiatCurrency: string;
    paymentMethod: string;
    email?: string;
    walletAddress?: string;
  }): Promise<APIResponse<FiatOrder>> {
    return (await this.client.post('/fiat/orders', order)).data;
  }

  // Social Recovery
  async getGuardians(): Promise<APIResponse<Guardian[]>> {
    return (await this.client.get('/social-recovery/guardians')).data;
  }

  async addGuardian(guardian: { address: string; name: string }): Promise<APIResponse<Guardian>> {
    return (await this.client.post('/social-recovery/guardians', guardian)).data;
  }

  async removeGuardian(address: string): Promise<APIResponse<void>> {
    return (await this.client.delete(`/social-recovery/guardians/${address}`)).data;
  }

  async initiateRecovery(newOwner: string): Promise<APIResponse<RecoveryRequest>> {
    return (await this.client.post('/social-recovery/initiate', { newOwner })).data;
  }

  async confirmRecovery(recoveryId: string, guardianAddress: string): Promise<APIResponse<RecoveryRequest>> {
    return (await this.client.post(`/social-recovery/${recoveryId}/confirm`, { guardianAddress })).data;
  }

  async cancelRecovery(recoveryId: string): Promise<APIResponse<void>> {
    return (await this.client.post(`/social-recovery/${recoveryId}/cancel`)).data;
  }

  // Insurance Fund
  async getInsuranceStats(): Promise<APIResponse<InsuranceStats>> {
    return (await this.client.get('/insurance/stats')).data;
  }

  async getInsuranceProducts(): Promise<APIResponse<InsuranceProduct[]>> {
    return (await this.client.get('/insurance/products')).data;
  }

  async getInsurancePositions(): Promise<APIResponse<InsurancePosition[]>> {
    return (await this.client.get('/insurance/positions')).data;
  }

  async getInsuranceClaims(): Promise<APIResponse<InsuranceClaim[]>> {
    return (await this.client.get('/insurance/claims')).data;
  }

  async buyInsuranceCoverage(pool: string, coverageAmount: string): Promise<APIResponse<InsurancePosition>> {
    return (await this.client.post('/insurance/coverage', { pool, coverageAmount })).data;
  }

  async fileInsuranceClaim(claim: { pool: string; amount: string; reason: string }): Promise<APIResponse<InsuranceClaim>> {
    return (await this.client.post('/insurance/claims', claim)).data;
  }

  // IEO / IDO
  async getIEOProjects(status?: string): Promise<APIResponse<IEOProject[]>> {
    const params = status ? { status } : undefined;
    return (await this.client.get('/ieo/projects', { params })).data;
  }

  async participateInIEO(projectId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/ieo/projects/${projectId}`, { amount })).data;
  }

  async claimIEOTokens(projectId: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/ieo/projects/${projectId}/claim`)).data;
  }

  // Leaderboard
  async getLeaderboard(params?: { pair?: string; limit?: number }): Promise<APIResponse<LeaderboardEntry[]>> {
    return (await this.client.get('/leaderboard', { params })).data;
  }

  async followTrader(traderId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/copy-trading/follow', { traderId })).data;
  }

  async unfollowTrader(traderId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/copy-trading/stop-all', { traderId })).data;
  }

  async copyTrader(traderId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post('/copy-trading/follow', { traderId, allocation: amount })).data;
  }

  // Copy Trading
  async getCopyTraders(): Promise<APIResponse<LeaderboardTrader[]>> {
    return (await this.client.get('/copy-trading/traders')).data;
  }

  async getCopySignals(): Promise<APIResponse<CopySignal[]>> {
    return (await this.client.get('/copy-trading/signals')).data;
  }

  async getCopyPositions(): Promise<APIResponse<CopyPosition[]>> {
    return (await this.client.get('/copy-trading/copiers')).data;
  }

  async startCopying(traderId: string, amount: string): Promise<APIResponse<CopyPosition>> {
    return (await this.client.post('/copy-trading/follow', { traderId, allocation: amount })).data;
  }

  async stopCopying(positionId: string): Promise<APIResponse<any>> {
    return (await this.client.post(`/copy-trading/copiers/${positionId}/stop`)).data;
  }

  // Security Scanner
  async scanAddress(address: string): Promise<APIResponse<ScanResult>> {
    return (await this.client.post('/security/scan', { address })).data;
  }
}

export const api = TigerWalletAPI.getInstance();
export default api;
