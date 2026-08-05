// TigerWallet Admin - Type Definitions
// Complete type definitions for all admin features

// User Types
export interface User {
  id: string;
  email: string;
  username: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  status: 'active' | 'suspended' | 'banned';
  kycStatus: 'none' | 'pending' | 'level1' | 'level2' | 'level3' | 'approved' | 'rejected';
  kycLevel: number;
  totalVolume: string;
  createdAt: string;
  updatedAt: string;
  lastLogin?: string;
  ipAddress?: string;
  country?: string;
  verified: boolean;
  suspended: boolean;
  suspendReason?: string;
  walletCount: number;
  twoFactorEnabled: boolean;
  referrerId?: string;
  tags?: string[];
}

// Transaction Types
export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  tokenSymbol: string;
  chain: string;
  chainId: number;
  status: 'pending' | 'confirmed' | 'failed' | 'flagged' | 'cancelled';
  type: 'transfer' | 'swap' | 'stake' | 'unstake' | 'bridge' | 'mint' | 'burn' | 'approve';
  timestamp: string;
  blockNumber?: number;
  gasUsed?: string;
  gasPrice?: string;
  nonce?: number;
  flagReason?: string;
  metadata?: Record<string, unknown>;
}

// KYC Types
export interface KycRecord {
  id: string;
  userId: string;
  userEmail: string;
  userName: string;
  status: 'pending' | 'approved' | 'rejected' | 'under_review';
  submittedAt: string;
  reviewedAt?: string;
  reviewerId?: string;
  documentType: 'passport' | 'driver_license' | 'national_id' | 'utility_bill' | 'bank_statement';
  documentFront?: string;
  documentBack?: string;
  documentUrl?: string;
  selfieUrl?: string;
  rejectionReason?: string;
  notes?: string;
  riskScore?: number;
}

// Token Types
export interface Token {
  id: string;
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  totalSupply: string;
  isListed: boolean;
  isPaused: boolean;
  marketCap?: string;
  volume24h?: string;
  price?: string;
  priceChange24h?: string;
  chain: string;
  chainId: number;
  logoUrl?: string;
  websiteUrl?: string;
  description?: string;
  listingDate?: string;
  verified: boolean;
  trustScore?: number;
}

// Trading Pair Types
export interface TradingPair {
  id: string;
  baseToken: string;
  quoteToken: string;
  baseSymbol: string;
  quoteSymbol: string;
  price: string;
  priceChange24h: string;
  volume24h: string;
  liquidity: string;
  isActive: boolean;
  minTradeAmount: string;
  maxTradeAmount: string;
  makerFee: string;
  takerFee: string;
}

// Blockchain Types
export interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chainId: number;
  rpcUrl: string;
  explorerUrl: string;
  isActive: boolean;
  isTestnet: boolean;
  nativeToken: string;
  avgBlockTime: number;
  gasToken: string;
}

// Fee Configuration Types
export interface FeeConfig {
  tradingFee: string;
  withdrawalFee: string;
  withdrawalFeeMin: string;
  depositFee: string;
  networkFee: string;
  makerFee: string;
  takerFee: string;
  flatFee: string;
  feeDiscounts: FeeDiscount[];
}

export interface FeeDiscount {
  volume: string;
  discount: string;
}

// System Types
export interface SystemService {
  name: string;
  status: 'running' | 'stopped' | 'error' | 'degraded';
  uptime: string;
  latency: string;
  lastCheck: string;
  cpu?: number;
  memory?: number;
  requestsPerSecond?: number;
  errorRate?: string;
}

export interface SystemMetrics {
  totalUsers: number;
  activeUsers24h: number;
  totalTransactions: number;
  transactionVolume24h: string;
  totalVolume: string;
  revenue24h: string;
  gasSpent24h: string;
  uptime: string;
  apiLatency: string;
}

// Analytics Types
export interface Analytics {
  totalUsers: number;
  totalVolume: string;
  dailyTransactions: number;
  activeUsers: number;
  revenue: string;
  growth: string;
  userGrowth: number[];
  volumeHistory: number[];
  transactionHistory: number[];
}

export interface ChartData {
  labels: string[];
  values: number[];
}

// White Label Types
export interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  logoUrl: string;
  primaryColor: string;
  secondaryColor: string;
  status: 'pending' | 'active' | 'suspended' | 'rejected';
  createdAt: string;
  approvedAt?: string;
  feeStructure: FeeConfig;
  apiKey?: string;
  apiSecret?: string;
  ownerEmail: string;
  ownerName: string;
}

// Admin Types
export interface Admin {
  id: string;
  email: string;
  username: string;
  role: 'super_admin' | 'admin' | 'moderator' | 'support';
  status: 'active' | 'suspended';
  createdAt: string;
  lastLogin?: string;
  permissions: string[];
  twoFactorEnabled: boolean;
}

// Audit Log Types
export interface AuditLog {
  id: string;
  adminId: string;
  adminEmail: string;
  action: string;
  resource: string;
  resourceId?: string;
  details: Record<string, unknown>;
  ipAddress: string;
  userAgent: string;
  timestamp: string;
}

// Withdrawal Types
export interface Withdrawal {
  id: string;
  userId: string;
  userEmail: string;
  amount: string;
  token: string;
  chain: string;
  toAddress: string;
  status: 'pending' | 'approved' | 'rejected' | 'processing' | 'completed' | 'failed';
  fee: string;
  txHash?: string;
  requestedAt: string;
  processedAt?: string;
  processedBy?: string;
  rejectionReason?: string;
}

// Notification Types
export interface Notification {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'warning' | 'error' | 'success';
  read: boolean;
  createdAt: string;
  link?: string;
}

// Pagination
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// API Response
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// Theme
export type Theme = 'light' | 'dark' | 'system';

// Dashboard Stats
export interface DashboardStats {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  volume24h: string;
  revenue24h: string;
  pendingWithdrawals: number;
  pendingKyc: number;
}
