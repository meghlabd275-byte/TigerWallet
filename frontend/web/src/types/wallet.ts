// Wallet Types
export interface Wallet {
  id: string;
  userId: string;
  type: 'user' | 'master' | 'whitelabel';
  address: string;
  blockchainId: string;
  publicKey: string;
  encryptedPrivateKey?: string;
  createdAt: string;
  updatedAt: string;
  isActive: boolean;
  label?: string;
}

export interface UserWallet extends Wallet {
  type: 'user';
  seedPhrase?: string; // Encrypted
  derivationPath: string;
}

export interface MasterWallet extends Wallet {
  type: 'master';
  permissions: MasterPermissions;
}

export interface MasterPermissions {
  canAutoSign: boolean;
  canAirdrop: boolean;
  canClaim: boolean;
  canAdjustFees: boolean;
  maxTransactionLimit: bigint;
}

export interface WalletBalance {
  walletId: string;
  tokenId: string;
  balance: string;
  balanceUSD: number;
  frozenBalance: string;
  availableBalance: string;
  lastUpdated: string;
}

export interface WalletState {
  isConnected: boolean;
  currentWallet: Wallet | null;
  wallets: Wallet[];
  balances: Map<string, WalletBalance>;
  isLoading: boolean;
  error: string | null;
}

// Transaction Types
export type TransactionStatus = 'pending' | 'confirmed' | 'failed' | 'cancelled';
export type TransactionType = 'send' | 'receive' | 'swap' | 'stake' | 'unstake' | 'bridge' | 'approve' | 'contract' | 'nft_transfer' | 'claim';

export interface Transaction {
  id: string;
  walletId: string;
  blockchainId: string;
  type: TransactionType;
  status: TransactionStatus;
  from: string;
  to: string;
  tokenSymbol: string;
  tokenAddress?: string;
  amount: string;
  amountUSD: number;
  fee: string;
  feeUSD: number;
  gasPrice?: string;
  gasUsed?: string;
  nonce?: number;
  hash: string;
  blockNumber?: number;
  timestamp: string;
  metadata?: Record<string, unknown>;
  error?: string;
}

export interface TransactionRequest {
  blockchainId: string;
  to: string;
  tokenSymbol: string;
  tokenAddress?: string;
  amount: string;
  maxFeePerGas?: string;
  maxPriorityFeePerGas?: string;
  data?: string;
  nonce?: number;
}

export interface TransactionResponse {
  transaction: Transaction;
  estimatedGas: string;
  estimatedFee: string;
  estimatedTime: number;
}

// Swap Types
export interface SwapQuote {
  id: string;
  fromToken: string;
  toToken: string;
  fromAmount: string;
  toAmount: string;
  toAmountUSD: number;
  priceImpact: number;
  guaranteedPrice: string;
  route: SwapRoute[];
  allowanceTarget: string;
  txData: string;
  validityPeriod: number;
  gasEstimate: string;
}

export interface SwapRoute {
  protocol: string;
  fromToken: string;
  toToken: string;
  poolAddress: string;
  poolFee: number;
}

export interface SwapRequest {
  fromToken: string;
  toToken: string;
  fromAmount: string;
  toAddress?: string;
  slippageTolerance: number;
  gasPrice?: string;
}

export interface SwapResponse {
  quote: SwapQuote;
  transaction: Transaction;
}

// Perpetual Trading Types
export interface PerpetualPosition {
  id: string;
  walletId: string;
  symbol: string;
  side: 'long' | 'short';
  size: string;
  entryPrice: string;
  markPrice: string;
  liquidationPrice: string;
  margin: string;
  marginRatio: number;
  unrealizedPnL: string;
  realizedPnL: string;
  fundingPayment: string;
  leverage: number;
  openedAt: string;
  updatedAt: string;
}

export interface PerpetualOrder {
  id: string;
  walletId: string;
  symbol: string;
  side: 'long' | 'short';
  orderType: 'market' | 'limit' | 'stop_market' | 'stop_limit';
  size: string;
  price?: string;
  triggerPrice?: string;
  margin: string;
  leverage: number;
  status: 'pending' | 'filled' | 'partially_filled' | 'cancelled' | 'expired';
  filledSize: string;
  filledPrice?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PerpetualMarket {
  symbol: string;
  displayName: string;
  indexPrice: string;
  markPrice: string;
  lastPrice: string;
  change24h: number;
  changePercent24h: number;
  high24h: string;
  low24h: string;
  volume24h: string;
  openInterest: string;
  fundingRate: string;
  nextFundingTime: string;
  maxLeverage: number;
  minMargin: string;
  liquidationFee: string;
}

// Copy Trading Types
export interface CopyTrade {
  id: string;
  followerId: string;
  traderId: string;
  symbol: string;
  side: 'buy' | 'sell';
  size: string;
  entryPrice: string;
  exitPrice?: string;
  pnl: string;
  pnlPercent: number;
  status: 'open' | 'closed';
  openedAt: string;
  closedAt?: string;
}

export interface CopyTrader {
  id: string;
  address: string;
  name: string;
  avatar?: string;
  totalTrades: number;
  winRate: number;
  profitFactor: number;
  avgHoldingTime: number;
  aum: string;
  followersCount: number;
  performance: {
    daily: number;
    weekly: number;
    monthly: number;
    allTime: number;
  };
  isVerified: boolean;
}

export interface CopyTradeSettings {
  followerId: string;
  traderId: string;
  allocation: string;
  maxSlippage: number;
  stopLoss?: string;
  takeProfit?: string;
  autoClose: boolean;
}

// Staking Types
export interface StakingPosition {
  id: string;
  walletId: string;
  blockchainId: string;
  tokenSymbol: string;
  validatorAddress: string;
  amount: string;
  rewardAmount: string;
  rewardClaimed: string;
  apy: number;
  lockedUntil?: string;
  unbondingStart?: string;
  unbondingEnd?: string;
  status: 'active' | 'unbonding' | 'withdrawn';
  createdAt: string;
  updatedAt: string;
}

export interface StakingPool {
  id: string;
  blockchainId: string;
  tokenSymbol: string;
  name: string;
  description: string;
  minStake: string;
  maxStake: string;
  apy: number;
  lockPeriod: number;
  isActive: boolean;
}

// NFT Types
export interface NFT {
  id: string;
  walletId: string;
  blockchainId: string;
  collectionAddress: string;
  tokenId: string;
  name: string;
  description?: string;
  imageUrl: string;
  animationUrl?: string;
  attributes?: NFTAttribute[];
  owner: string;
  standard: 'ERC721' | 'ERC1155' | 'SPL' | 'other';
  metadataUrl?: string;
  isListed: boolean;
  listingPrice?: string;
  marketplace?: string;
}

export interface NFTAttribute {
  trait_type: string;
  value: string | number;
  display_type?: string;
}

export interface NFTCollection {
  address: string;
  blockchainId: string;
  name: string;
  symbol: string;
  description?: string;
  imageUrl: string;
  bannerUrl?: string;
  floorPrice: string;
  floorPriceUSD: number;
  totalSupply: string;
  holderCount: string;
  transactionCount: string;
}

// API Response Types
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
  meta?: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface PaginatedResponse<T> {
  items: T[];
  page: number;
  limit: number;
  total: number;
  hasMore: boolean;
}

// WebSocket Event Types
export interface WSMessage<T = unknown> {
  type: 'price_update' | 'balance_update' | 'transaction_update' | 'order_update' | 'position_update' | 'notification';
  payload: T;
  timestamp: string;
}

export interface PriceUpdate {
  symbol: string;
  price: string;
  change24h: number;
  changePercent24h: number;
  volume24h: string;
  timestamp: string;
}

export interface BalanceUpdate {
  walletId: string;
  tokenSymbol: string;
  newBalance: string;
  delta: string;
  timestamp: string;
}
