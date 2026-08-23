// ============================================================================
// TigerWallet - Core Type Definitions
// High-Performance Multi-Chain Wallet System
// ============================================================================

import { BigNumber } from 'ethers';

// ============================================================================
// Chain & Network Types
// ============================================================================

export interface Chain {
  id: number;
  name: string;
  symbol: string;
  decimals: number;
  color: string;
  explorer: string;
  rpc: string;
  chainType: 'evm' | 'solana' | 'bitcoin' | 'cosmos' | 'aptos' | 'ton' | 'near' | 'polkadot';
  logoUrl: string;
  coinGeckoId?: string;
  isTestnet: boolean;
  nativeCurrency: NativeCurrency;
  contracts: ContractInfo[];
}

export interface NativeCurrency {
  name: string;
  symbol: string;
  decimals: number;
  address: string;
}

export interface ContractInfo {
  address: string;
  type: 'erc20' | 'erc721' | 'erc1155' | 'native';
  name?: string;
  symbol?: string;
  decimals?: number;
  logoUrl?: string;
}

export interface RPCEndpoint {
  url: string;
  chainId: number;
  name: string;
  weight: number;
  latency?: number;
}

// ============================================================================
// Wallet Types
// ============================================================================

export interface Wallet {
  id: string;
  name: string;
  type: WalletType;
  addresses: Record<number, string>; // chainId -> address
  createdAt: number;
  lastUsedAt: number;
  isBackedUp: boolean;
  isHardware: boolean;
  hardwareType?: 'ledger' | 'trezor';
}

export type WalletType = 
  | 'mnemonic' 
  | 'privateKey' 
  | 'watchOnly' 
  | 'hardware' 
  | 'multisig'
  | 'contract';

export interface WalletAccount {
  walletId: string;
  chainId: number;
  address: string;
  publicKey: string;
  balance: string;
  tokens: TokenBalance[];
  nonce?: number;
}

export interface TokenBalance {
  contractAddress: string;
  chainId: number;
  name: string;
  symbol: string;
  decimals: number;
  balance: string;
  balanceUSD: number;
  logoUrl?: string;
  price?: number;
  isNative: boolean;
}

// ============================================================================
// Transaction Types
// ============================================================================

export interface Transaction {
  id: string;
  hash: string;
  chainId: number;
  from: string;
  to: string;
  value: string;
  data: string;
  gasLimit: string;
  gasPrice: string;
  gasUsed?: string;
  nonce: number;
  status: TransactionStatus;
  type: TransactionType;
  token?: TokenTransfer;
  timestamp: number;
  blockNumber?: number;
  confirmations: number;
  explorerUrl: string;
}

export type TransactionStatus = 
  | 'pending' 
  | 'confirmed' 
  | 'failed' 
  | 'cancelled';

export type TransactionType = 
  | 'transfer' 
  | 'approve' 
  | 'swap' 
  | 'contract' 
  | 'stake' 
  | 'unstake' 
  | 'claim'
  | 'bridge';

export interface TokenTransfer {
  from: string;
  to: string;
  contractAddress: string;
  tokenId?: string;
  amount: string;
  decimals: number;
  symbol: string;
  logoUrl?: string;
}

// ============================================================================
// Signing Types
// ============================================================================

export interface SignRequest {
  id: string;
  walletId: string;
  chainId: number;
  type: SignRequestType;
  data: string;
  message?: string;
  metadata?: Record<string, unknown>;
  timestamp: number;
  expiresAt: number;
  status: SignRequestStatus;
}

export type SignRequestType = 
  | 'transaction' 
  | 'message' 
  | 'typedData' 
  | 'eip712';

export type SignRequestStatus = 
  | 'pending' 
  | 'approved' 
  | 'rejected' 
  | 'expired';

// ============================================================================
// Swap Types
// ============================================================================

export interface SwapQuote {
  id: string;
  fromToken: TokenInfo;
  toToken: TokenInfo;
  fromAmount: string;
  toAmount: string;
  toAmountUSD: number;
  priceImpact: number;
  gasEstimate: string;
  gasEstimateUSD: number;
  route: SwapRoute[];
  dex: string;
  dexLogo: string;
  expiresAt: number;
}

export interface TokenInfo {
  chainId: number;
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  logoUrl?: string;
}

export interface SwapRoute {
  fromToken: string;
  toToken: string;
  pool: string;
  dex: string;
  fee: number;
}

export interface SwapRequest {
  quoteId: string;
  fromAddress: string;
  toAddress: string;
  slippage: number;
}

// ============================================================================
// Bridge Types
// ============================================================================

export interface BridgeQuote {
  id: string;
  fromChain: Chain;
  toChain: Chain;
  fromToken: TokenInfo;
  toToken: TokenInfo;
  fromAmount: string;
  toAmount: string;
  toAmountUSD: number;
  gasEstimate: string;
  gasEstimateUSD: number;
  bridge: string;
  bridgeLogo: string;
  duration: string;
  steps: BridgeStep[];
  expiresAt: number;
}

export interface BridgeStep {
  type: 'swap' | 'bridge' | 'destination';
  chainId: number;
  token: string;
  amount: string;
  action: string;
  tool: string;
  toolLogo: string;
  contract?: string;
}

export interface BridgeRequest {
  quoteId: string;
  toAddress: string;
  fromAddress: string;
}

// ============================================================================
// Staking Types
// ============================================================================

export interface StakePosition {
  id: string;
  chainId: number;
  token: TokenInfo;
  amount: string;
  rewards: string;
  rewardsUSD: number;
  apy: number;
  lockPeriod: number;
  startTime: number;
  endTime?: number;
  status: 'active' | 'unlocking' | 'completed';
}

export interface StakingPool {
  chainId: number;
  token: TokenInfo;
  totalStaked: string;
  totalStakedUSD: number;
  apy: number;
  minStake: string;
  maxStake?: string;
  lockPeriod: number;
  rewardsToken: TokenInfo;
}

// ============================================================================
// NFT Types
// ============================================================================

export interface NFT {
  id: string;
  chainId: number;
  contractAddress: string;
  tokenId: string;
  name: string;
  description?: string;
  imageUrl: string;
  animationUrl?: string;
  attributes?: NFTAttribute[];
  owner: string;
  collection: NFTCollection;
}

export interface NFTAttribute {
  trait_type: string;
  value: string | number;
  display_type?: string;
}

export interface NFTCollection {
  address: string;
  name: string;
  symbol?: string;
  description?: string;
  imageUrl: string;
  floorPrice?: number;
  totalSupply: number;
}

// ============================================================================
// DApp Connection Types
// ============================================================================

export interface DAppConnection {
  id: string;
  walletId: string;
  chainId: number;
  dappUrl: string;
  dappName: string;
  dappIcon?: string;
  permissions: DAppPermission[];
  sessionId: string;
  createdAt: number;
  lastConnectedAt: number;
}

export type DAppPermission = 
  | 'eth_accounts' 
  | 'eth_chainId' 
  | 'eth_sendTransaction'
  | 'personal_sign'
  | 'eth_signTypedData';
// ============================================================================
// Analytics Types
// ============================================================================

export interface PortfolioData {
  totalValueUSD: number;
  change24h: number;
  change24hPercent: number;
  assets: PortfolioAsset[];
  history: PricePoint[];
}

export interface PortfolioAsset {
  token: TokenInfo;
  balance: string;
  balanceUSD: number;
  allocation: number;
  change24h: number;
}

export interface PricePoint {
  timestamp: number;
  value: number;
}

export interface TransactionStats {
  total: number;
  pending: number;
  completed: number;
  failed: number;
  volume24h: string;
  volume24hUSD: number;
}

// ============================================================================
// API Response Types
// ============================================================================

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: APIError;
  timestamp: number;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface PaginatedResponse<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

// ============================================================================
// Theme Types
// ============================================================================

export interface Theme {
  mode: 'light' | 'dark';
  colors: ThemeColors;
  spacing: ThemeSpacing;
  typography: ThemeTypography;
  borderRadius: ThemeBorderRadius;
}

export interface ThemeColors {
  primary: string;
  secondary: string;
  background: string;
  surface: string;
  surfaceVariant: string;
  text: string;
  textSecondary: string;
  textTertiary: string;
  border: string;
  borderLight: string;
  success: string;
  warning: string;
  error: string;
  info: string;
  positive: string;
  negative: string;
}

export interface ThemeSpacing {
  xs: number;
  sm: number;
  md: number;
  lg: number;
  xl: number;
  xxl: number;
}

export interface ThemeTypography {
  fontFamily: string;
  fontSize: {
    xs: number;
    sm: number;
    md: number;
    lg: number;
    xl: number;
    xxl: number;
    xxxl: number;
  };
  fontWeight: {
    regular: number;
    medium: number;
    semibold: number;
    bold: number;
  };
}

export interface ThemeBorderRadius {
  sm: number;
  md: number;
  lg: number;
  xl: number;
  full: number;
}

// ============================================================================
// Notification Types
// ============================================================================

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data?: Record<string, unknown>;
  timestamp: number;
  read: boolean;
}

export type NotificationType = 
  | 'transaction' 
  | 'swap' 
  | 'stake' 
  | 'alert' 
  | 'system';

// ============================================================================
// Biometric Types
// ============================================================================

export interface BiometricAuthResult {
  success: boolean;
  error?: string;
  publicKey?: string;
}

// ============================================================================
// Security Types
// ============================================================================

export interface SecuritySettings {
  biometricEnabled: boolean;
  pinEnabled: boolean;
  pinHash?: string;
  autoLockTimeout: number;
  showBalance: boolean;
  privacyMode: boolean;
  trustedApps: string[];
}

export interface BackupInfo {
  hasBackup: boolean;
  backupMethod?: 'cloud' | 'manual';
  lastBackupAt?: number;
}

// ============================================================================
// Export all types
// ============================================================================

export type {
  BigNumber,
};
