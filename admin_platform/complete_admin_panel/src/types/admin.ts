// ============================================================================
// TigerWallet Complete Admin Types - Production Ready
// ============================================================================

// ==================== USER TYPES ====================
export interface User {
  id: string;
  email: string;
  username: string;
  phone?: string;
  kycStatus: KYCStatus;
  kycLevel: number;
  status: UserStatus;
  roles: UserRole[];
  walletAddresses: WalletAddress[];
  createdAt: Date;
  updatedAt: Date;
  lastLoginAt?: Date;
 TwoFactorEnabled: boolean;
  referralCode?: string;
  referredBy?: string;
}

export type UserStatus = 'active' | 'suspended' | 'banned' | 'pending';
export type KYCStatus = 'none' | 'pending' | 'level1' | 'level2' | 'level3' | 'rejected';
export type UserRole = 'user' | 'trader' | 'broker' | 'institutional' | 'white_label' | 'admin' | 'super_admin';

export interface WalletAddress {
  id: string;
  userId: string;
  chain: string;
  address: string;
  type: 'evm' | 'bitcoin' | 'solana' | 'cosmos' | 'other';
  label?: string;
  isPrimary: boolean;
  createdAt: Date;
}

// ==================== ADMIN TYPES ====================
export interface Admin {
  id: string;
  email: string;
  username: string;
  role: AdminRole;
  permissions: Permission[];
  status: AdminStatus;
  createdBy: string;
  createdAt: Date;
  lastLoginAt?: Date;
  twoFactorEnabled: boolean;
  ipWhitelist?: string[];
}

export type AdminRole = 'super_admin' | 'admin' | 'support' | 'analyst' | 'moderator';
export type AdminStatus = 'active' | 'suspended' | 'inactive';
export type Permission = 
  | 'users.read' | 'users.write' | 'users.delete'
  | 'admins.read' | 'admins.write' | 'admins.delete'
  | 'kyc.read' | 'kyc.write'
  | 'wallets.read' | 'wallets.write'
  | 'transactions.read' | 'transactions.write'
  | 'pairs.read' | 'pairs.write' | 'pairs.delete'
  | 'liquidity.read' | 'liquidity.write' | 'liquidity.delete'
  | 'fees.read' | 'fees.write'
  | 'whitelabel.read' | 'whitelabel.write' | 'whitelabel.delete'
  | 'withdrawals.read' | 'withdrawals.write' | 'withdrawals.approve' | 'withdrawals.reject'
  | 'api.read' | 'api.write' | 'api.delete'
  | 'analytics.read'
  | 'settings.read' | 'settings.write'
  | 'nft.read' | 'nft.write'
  | 'tokens.read' | 'tokens.write' | 'tokens.delete'
  | 'multisend.read' | 'multisend.write';

export interface AdminActivity {
  id: string;
  adminId: string;
  action: string;
  resource: string;
  resourceId?: string;
  details: Record<string, any>;
  ipAddress: string;
  userAgent: string;
  timestamp: Date;
}

// ==================== WHITE LABEL TYPES ====================
export interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  subdomain?: string;
  status: WhiteLabelStatus;
  plan: WhiteLabelPlan;
  features: WhiteLabelFeatures;
  branding: WhiteLabelBranding;
  customDomain?: string;
  sslEnabled: boolean;
  createdBy: string;
  createdAt: Date;
  updatedAt: Date;
  expiresAt?: Date;
  maxUsers: number;
  currentUsers: number;
  apiKeys: ApiKey[];
  permissions: WhiteLabelPermission[];
}

export type WhiteLabelStatus = 'active' | 'suspended' | 'pending' | 'expired' | 'cancelled';
export type WhiteLabelPlan = 'starter' | 'professional' | 'enterprise' | 'custom';

export interface WhiteLabelFeatures {
  wallet: boolean;
  exchange: boolean;
  defi: boolean;
  nft: boolean;
  staking: boolean;
  launchpad: boolean;
  liquidity: boolean;
  api: boolean;
  customBranding: boolean;
  customDomain: boolean;
  whiteLabel: boolean;
  institutional: boolean;
  broker: boolean;
}

export interface WhiteLabelBranding {
  logoUrl?: string;
  faviconUrl?: string;
  primaryColor: string;
  secondaryColor: string;
  accentColor: string;
  backgroundColor: string;
  textColor: string;
  fontFamily?: string;
  termsUrl?: string;
  privacyUrl?: string;
  supportEmail?: string;
  supportUrl?: string;
}

export interface WhiteLabelPermission {
  userId: string;
  permissions: Permission[];
  createdAt: Date;
  expiresAt?: Date;
}

// ==================== TRADING PAIRS ====================
export interface TradingPair {
  id: string;
  baseAsset: string;
  quoteAsset: string;
  symbol: string;
  pair: string;
  status: PairStatus;
  type: PairType;
  pricePrecision: number;
  quantityPrecision: number;
  minQuantity: string;
  maxQuantity: string;
  minPrice: string;
  maxPrice: string;
  makerFee: string;
  takerFee: string;
  liquiditySource?: string;
  createdAt: Date;
  updatedAt: Date;
}

export type PairStatus = 'active' | 'suspended' | 'halted' | 'delisted';
export type PairType = 'spot' | 'futures' | 'margin' | 'dex';

export interface LiquidityPool {
  id: string;
  pairId: string;
  provider: string;
  liquidity: string;
  share: string;
  createdAt: Date;
}

// ==================== TOKEN MANAGEMENT ====================
export interface Token {
  id: string;
  symbol: string;
  name: string;
  address: string;
  decimals: number;
  chain: string;
  type: TokenType;
  totalSupply: string;
  circulatingSupply?: string;
  status: TokenStatus;
  price?: string;
  priceChange24h?: string;
  marketCap?: string;
  volume24h?: string;
  logoUrl?: string;
  websiteUrl?: string;
  explorerUrl?: string;
  createdAt: Date;
  updatedAt: Date;
}

export type TokenType = 'native' | 'erc20' | 'trc20' | 'spl' | 'bep20' | 'other';
export type TokenStatus = 'active' | 'inactive' | 'suspended' | 'delisted';

// ==================== TRANSACTIONS ====================
export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  value: string;
  token?: string;
  chain: string;
  status: TransactionStatus;
  type: TransactionType;
  fee: string;
  feeCurrency: string;
  blockNumber?: number;
  timestamp: Date;
  confirmedAt?: Date;
  userId?: string;
  walletId?: string;
  metadata?: Record<string, any>;
}

export type TransactionStatus = 'pending' | 'confirming' | 'confirmed' | 'failed' | 'rejected';
export type TransactionType = 'deposit' | 'withdrawal' | 'transfer' | 'swap' | 'trade' | 'fee' | 'reward' | 'internal';

// ==================== WITHDRAWALS ====================
export interface Withdrawal {
  id: string;
  userId: string;
  walletAddress: string;
  chain: string;
  token: string;
  amount: string;
  fee: string;
  total: string;
  status: WithdrawalStatus;
  type: WithdrawalType;
  txHash?: string;
  approvedBy?: string;
  rejectedBy?: string;
  rejectionReason?: string;
  createdAt: Date;
  processedAt?: Date;
}

export type WithdrawalStatus = 'pending' | 'approved' | 'processing' | 'completed' | 'rejected' | 'cancelled';
export type WithdrawalType = 'user' | 'master' | 'white_label' | 'fee';

// ==================== KYC ====================
export interface KYCSubmission {
  id: string;
  userId: string;
  level: number;
  status: KYCStatus;
  documents: KYCDocument[];
  verifiedBy?: string;
  verifiedAt?: Date;
  rejectionReason?: string;
  submittedAt: Date;
}

export interface KYCDocument {
  type: 'id_front' | 'id_back' | 'passport' | 'drivers_license' | 'utility_bill' | 'bank_statement' | 'selfie';
  url: string;
  status: 'pending' | 'verified' | 'rejected';
  verifiedAt?: Date;
}

// ==================== FEES ====================
export interface FeeStructure {
  id: string;
  type: FeeType;
  asset: string;
  network?: string;
  fixedFee?: string;
  percentageFee?: string;
  minFee?: string;
  maxFee?: string;
  tier?: string;
  status: 'active' | 'inactive';
}

export type FeeType = 'withdrawal' | 'deposit' | 'swap' | 'trade' | 'transfer' | 'network';

// ==================== API KEYS ====================
export interface ApiKey {
  id: string;
  userId: string;
  key: string;
  secret: string;
  name: string;
  permissions: string[];
  ipWhitelist?: string[];
  rateLimit?: number;
  status: 'active' | 'suspended' | 'revoked';
  lastUsedAt?: Date;
  expiresAt?: Date;
  createdAt: Date;
}

// ==================== ANALYTICS ====================
export interface Analytics {
  totalUsers: number;
  activeUsers24h: number;
  activeUsers7d: number;
  totalVolume24h: number;
  totalVolume7d: number;
  totalVolume30d: number;
  tradingVolume24h: number;
  withdrawalVolume24h: number;
  depositVolume24h: number;
  newUsers24h: number;
  newUsers7d: number;
  kycPending: number;
  withdrawalsPending: number;
  pairsActive: number;
  tokensActive: number;
}

// ==================== NOTIFICATIONS ====================
export interface Notification {
  id: string;
  userId?: string;
  adminId?: string;
  type: NotificationType;
  title: string;
  message: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'unread' | 'read' | 'archived';
  actionUrl?: string;
  createdAt: Date;
  readAt?: Date;
}

export type NotificationType = 'system' | 'security' | 'transaction' | 'kyc' | 'withdrawal' | 'deposit' | 'alert';

// ==================== MARKET MAKER ====================
export interface MarketMakerBot {
  id: string;
  name: string;
  userId: string;
  pairId: string;
  status: 'active' | 'paused' | 'stopped';
  strategy: MarketMakerStrategy;
  params: MarketMakerParams;
  profit: string;
  volume: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface MarketMakerStrategy {
  type: 'arbritrage' | 'liquidty_provision' | 'price_stabilization';
  minSpread: string;
  maxSpread: string;
  orderSize: string;
  maxOrders: number;
}

export interface MarketMakerParams {
  minOrderValue: string;
  maxOrderValue: string;
  priceRefreshInterval: number;
  slippageTolerance: string;
}

// ==================== NFT ====================
export interface NFT {
  id: string;
  tokenId: string;
  contractAddress: string;
  chain: string;
  name: string;
  description?: string;
  imageUrl?: string;
  animationUrl?: string;
  attributes?: NFTAttribute[];
  owner?: string;
  status: 'active' | 'suspended' | 'delisted';
  price?: string;
  currency?: string;
  createdAt: Date;
}

export interface NFTAttribute {
  trait_type: string;
  value: string | number;
}

// ==================== MULTISIG ====================
export interface MultisigWallet {
  id: string;
  name: string;
  owners: string[];
  threshold: number;
  chain: string;
  address: string;
  balance: string;
  status: 'active' | 'suspended';
  createdAt: Date;
}

export interface MultisigTransaction {
  id: string;
  walletId: string;
  to: string;
  value: string;
  data?: string;
  status: 'pending' | 'approved' | 'executed' | 'rejected';
  confirmations: number;
  requiredConfirmations: number;
  createdAt: Date;
  executedAt?: Date;
}

// ==================== LISTING REQUEST ====================
export interface ListingRequest {
  id: string;
  tokenId: string;
  requestedBy: string;
  status: 'pending' | 'approved' | 'rejected';
  tier: 'tier1' | 'tier2' | 'tier3';
  listingFee: string;
  notes?: string;
  reviewedBy?: string;
  reviewedAt?: Date;
  createdAt: Date;
}

// ==================== BROKER ====================
export interface Broker {
  id: string;
  name: string;
  email: string;
  status: 'active' | 'suspended' | 'inactive';
  commission: string;
  volume: string;
  clients: number;
  createdAt: Date;
  updatedAt: Date;
}

// ==================== INSTITUTIONAL ====================
export interface InstitutionalClient {
  id: string;
  name: string;
  type: 'hedge_fund' | 'family_office' | 'corporate' | 'government' | 'other';
  status: 'active' | 'pending' | 'suspended';
  kycStatus: KYCStatus;
  accountManager?: string;
  limits: InstitutionalLimits;
  volume: string;
  createdAt: Date;
}

export interface InstitutionalLimits {
  dailyWithdrawalLimit: string;
  monthlyWithdrawalLimit: string;
  singleTransactionLimit: string;
  kycRequired: boolean;
}

// ==================== BLOCKCHAIN ====================
export interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chainId?: number;
  type: 'evm' | 'bitcoin' | 'cosmos' | 'solana' | 'other';
  rpcUrl: string;
  explorerUrl: string;
  explorerApiUrl?: string;
  status: 'active' | 'inactive' | 'maintenance';
  nativeToken?: string;
  decimals: number;
  gasToken?: string;
  isTestnet: boolean;
  createdAt: Date;
}

// ==================== AUDIT LOG ====================
export interface AuditLog {
  id: string;
  userId?: string;
  adminId?: string;
  action: string;
  resource: string;
  resourceId?: string;
  oldValue?: Record<string, any>;
  newValue?: Record<string, any>;
  ipAddress: string;
  userAgent: string;
  timestamp: Date;
}
