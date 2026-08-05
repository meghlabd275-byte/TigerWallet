// TigerWallet Super Admin - Type Definitions
// Complete TypeScript types for all Super Admin operations

// ==================== User Types ====================

export interface User {
  id: string;
  email: string;
  username: string;
  wallet_address: string;
  kyc_status: 'none' | 'pending' | 'verified' | 'rejected';
  status: 'active' | 'suspended' | 'banned';
  created_at: string;
  last_login: string;
  balance: Record<string, number>;
  two_factor_enabled: boolean;
  ip_address: string;
  country: string;
  risk_score: number;
  total_volume: number;
  verification_level: number;
}

export interface UserCreateInput {
  email: string;
  username: string;
  password: string;
  role?: string;
}

export interface UserUpdateInput {
  email?: string;
  username?: string;
  status?: string;
  kyc_status?: string;
}

// ==================== KYC Types ====================

export interface KYCRequest {
  id: string;
  user_id: string;
  user_email: string;
  doc_type: 'identity' | 'address' | 'selfie' | 'passport' | 'drivers_license';
  status: 'pending' | 'approved' | 'rejected' | 'needs_review';
  document_url: string;
  document_front?: string;
  document_back?: string;
  selfie_url?: string;
  submitted_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  reject_reason?: string;
  notes?: string;
  risk_level: 'low' | 'medium' | 'high';
}

export interface KYCApproveInput {
  notes?: string;
}

export interface KYCRejectInput {
  reason: string;
  notes?: string;
}

// ==================== Transaction Types ====================

export interface Transaction {
  id: string;
  user_id: string;
  user_email: string;
  type: 'deposit' | 'withdrawal' | 'transfer' | 'swap' | 'trade' | 'fee' | 'refund';
  amount: number;
  currency: string;
  status: 'pending' | 'completed' | 'failed' | 'cancelled' | 'flagged';
  from_address: string;
  to_address: string;
  tx_hash: string;
  chain: string;
  timestamp: string;
  fee: number;
  fee_currency: string;
  chain_id: number;
  confirmations: number;
  risk_score: number;
  is_suspicious: boolean;
  flags: string[];
}

export interface TransactionFlagInput {
  reason: string;
  flags: string[];
}

export interface TransactionListParams {
  page?: number;
  page_size?: number;
  status?: string;
  type?: string;
  chain?: string;
  currency?: string;
  user_id?: string;
  start_date?: string;
  end_date?: string;
  is_suspicious?: boolean;
  min_amount?: number;
  max_amount?: number;
}

// ==================== Withdrawal Types ====================

export interface Withdrawal {
  id: string;
  user_id: string;
  user_email: string;
  amount: number;
  currency: string;
  status: 'pending' | 'approved' | 'rejected' | 'processing' | 'completed' | 'failed';
  to_address: string;
  chain: string;
  tx_hash?: string;
  fee: number;
  fee_currency: string;
  requested_at: string;
  processed_at?: string;
  approved_by?: string;
  rejected_by?: string;
  reject_reason?: string;
  notes?: string;
  risk_score: number;
  is_urgent: boolean;
}

export interface WithdrawalApproveInput {
  notes?: string;
}

export interface WithdrawalRejectInput {
  reason: string;
  notes?: string;
}

// ==================== Token Types ====================

export interface Token {
  id: string;
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  chain: string;
  chain_id: number;
  total_supply: number;
  circulating_supply: number;
  price_usd: number;
  market_cap: number;
  volume_24h: number;
  status: 'active' | 'paused' | 'delisted' | 'pending';
  is_verified: boolean;
  is_fiat: boolean;
  logo_url?: string;
  website_url?: string;
  created_at: string;
  updated_at: string;
}

export interface TokenCreateInput {
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  chain: string;
  chain_id: number;
  total_supply: number;
  logo_url?: string;
  website_url?: string;
}

export interface TokenUpdateInput {
  name?: string;
  status?: string;
  logo_url?: string;
  website_url?: string;
}

// ==================== Blockchain Types ====================

export interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chain_id: number;
  is_evm: boolean;
  rpc_url: string;
  explorer_url: string;
  native_token: string;
  decimals: number;
  is_active: boolean;
  avg_gas_price_gwei: number;
  block_time_seconds: number;
  logo_url?: string;
  created_at: string;
}

export interface BlockchainCreateInput {
  name: string;
  symbol: string;
  chain_id: number;
  is_evm: boolean;
  rpc_url: string;
  explorer_url: string;
  native_token: string;
  decimals: number;
  avg_gas_price_gwei?: number;
  block_time_seconds?: number;
  logo_url?: string;
}

// ==================== Trading Pair Types ====================

export interface TradingPair {
  id: string;
  base: string;
  quote: string;
  pair_name: string;
  price: number;
  price_change_24h: number;
  volume_24h: number;
  liquidity: number;
  status: 'active' | 'suspended' | 'halted' | 'delisted';
  chain_id: number;
  chain: string;
  min_trade_amount: number;
  max_trade_amount: number;
  created_at: string;
  updated_at: string;
}

export interface TradingPairCreateInput {
  base: string;
  quote: string;
  chain_id: number;
  min_trade_amount?: number;
  max_trade_amount?: number;
}

// ==================== Fee Types ====================

export interface FeeStructure {
  id: string;
  fee_type: 'withdrawal' | 'deposit' | 'trading' | 'swap' | 'transfer' | 'conversion';
  asset: string;
  chain?: string;
  fee_percent: number;
  fee_fixed: number;
  min_fee: number;
  max_fee?: number;
  tier: string;
  is_active: boolean;
  effective_from: string;
}

export interface FeeCreateInput {
  fee_type: string;
  asset: string;
  chain?: string;
  fee_percent: number;
  fee_fixed: number;
  min_fee: number;
  max_fee?: number;
  tier: string;
  effective_from?: string;
}

// ==================== White Label Types ====================

export interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  domain_verified: boolean;
  api_key: string;
  fee_percent: number;
  status: 'pending' | 'active' | 'suspended' | 'revoked';
  plan: 'starter' | 'professional' | 'enterprise';
  features: string[];
  custom_branding: boolean;
  primary_color?: string;
  secondary_color?: string;
  logo_url?: string;
  owner_name: string;
  owner_email: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
  expires_at?: string;
}

export interface WhiteLabelCreateInput {
  name: string;
  domain: string;
  owner_name: string;
  owner_email: string;
  plan?: string;
  fee_percent?: number;
  primary_color?: string;
  secondary_color?: string;
  logo_url?: string;
}

export interface WhiteLabelUpdateInput {
  name?: string;
  domain?: string;
  status?: string;
  fee_percent?: number;
  plan?: string;
  features?: string[];
  primary_color?: string;
  secondary_color?: string;
  logo_url?: string;
}

// ==================== Admin Types ====================

export interface Admin {
  id: string;
  username: string;
  email: string;
  role: 'super_admin' | 'admin' | 'manager' | 'support' | 'auditor';
  status: 'active' | 'suspended' | 'inactive';
  permissions: string[];
  security_level: number;
  two_factor_enabled: boolean;
  created_at: string;
  last_login?: string;
  failed_attempts: number;
  locked_until?: string;
}

export interface AdminCreateInput {
  username: string;
  email: string;
  password: string;
  role: string;
  permissions?: string[];
}

export interface AdminUpdateInput {
  username?: string;
  email?: string;
  role?: string;
  status?: string;
  permissions?: string[];
}

// ==================== Ticket Types ====================

export interface Ticket {
  id: string;
  title: string;
  description: string;
  category: 'technical' | 'billing' | 'kyc' | 'security' | 'feature_request' | 'bug' | 'other';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed';
  user_id: string;
  user_email: string;
  assigned_to?: string;
  assigned_to_email?: string;
  messages: TicketMessage[];
  created_at: string;
  updated_at: string;
  resolved_at?: string;
}

export interface TicketMessage {
  id: string;
  ticket_id: string;
  sender_type: 'user' | 'admin';
  sender_id: string;
  sender_email: string;
  message: string;
  is_internal: boolean;
  created_at: string;
}

export interface TicketCreateInput {
  title: string;
  description: string;
  category: string;
  priority?: string;
}

export interface TicketMessageInput {
  message: string;
  is_internal?: boolean;
}

// ==================== Knowledge Base Types ====================

export interface Article {
  id: string;
  title: string;
  content: string;
  category: string;
  tags: string[];
  status: 'draft' | 'published' | 'archived';
  author_id: string;
  author_email: string;
  views: number;
  created_at: string;
  updated_at: string;
  published_at?: string;
}

export interface ArticleCreateInput {
  title: string;
  content: string;
  category: string;
  tags?: string[];
  status?: string;
}

// ==================== Approval Workflow Types ====================

export interface ApprovalWorkflow {
  id: string;
  name: string;
  description: string;
  trigger_type: string;
  trigger_condition: Record<string, any>;
  approvers: string[];
  approval_threshold: number;
  timeout_hours: number;
  status: 'active' | 'inactive' | 'draft';
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalRequest {
  id: string;
  workflow_id: string;
  workflow_name: string;
  requester_id: string;
  requester_email: string;
  request_data: Record<string, any>;
  status: 'pending' | 'approved' | 'rejected' | 'cancelled';
  current_approvers: string[];
  approvals: Approval[];
  created_at: string;
  updated_at: string;
  expires_at: string;
}

export interface Approval {
  id: string;
  request_id: string;
  approver_id: string;
  approver_email: string;
  status: 'pending' | 'approved' | 'rejected';
  comments?: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalActionInput {
  action: 'approve' | 'reject';
  comments?: string;
}

// ==================== Analytics Types ====================

export interface DashboardStats {
  total_users: number;
  active_users: number;
  total_transactions: number;
  transaction_volume_24h: number;
  total_volume: number;
  revenue_24h: number;
  total_revenue: number;
  pending_withdrawals: number;
  pending_kyc: number;
  active_white_labels: number;
  system_health: 'healthy' | 'degraded' | 'critical';
}

export interface AnalyticsData {
  period: string;
  users: {
    total: number;
    new: number;
    active: number;
    kyc_verified: number;
  };
  transactions: {
    count: number;
    volume: number;
    fees: number;
  };
  revenue: {
    total: number;
    by_type: Record<string, number>;
  };
  chains: {
    name: string;
    volume: number;
    transactions: number;
  }[];
}

export interface ComplianceReport {
  id: string;
  type: 'kyc' | 'aml' | 'transaction' | 'audit';
  period_start: string;
  period_end: string;
  status: 'generating' | 'ready' | 'failed';
  download_url?: string;
  created_at: string;
}

export interface FinanceReport {
  id: string;
  type: 'revenue' | 'expenses' | 'profit' | 'tax';
  period: string;
  status: 'generating' | 'ready' | 'failed';
  amount?: number;
  download_url?: string;
  created_at: string;
}

export interface SecurityAlert {
  id: string;
  type: 'suspicious_activity' | 'large_transaction' | 'multiple_failed_logins' | 'unusual_pattern';
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'new' | 'investigating' | 'resolved' | 'false_positive';
  description: string;
  related_entities: string[];
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

// ==================== API Key Types ====================

export interface APIKey {
  id: string;
  name: string;
  key: string;
  user_id: string;
  user_email: string;
  permissions: string[];
  rate_limit_per_minute: number;
  rate_limit_per_day: number;
  is_active: boolean;
  expires_at: string;
  last_used_at?: string;
  created_at: string;
}

// ==================== Webhook Types ====================

export interface Webhook {
  id: string;
  name: string;
  url: string;
  events: string[];
  secret: string;
  is_active: boolean;
  failure_count: number;
  last_triggered_at?: string;
  created_by: string;
  created_at: string;
}

export interface WebhookCreateInput {
  name: string;
  url: string;
  events: string[];
}

// ==================== Audit Log Types ====================

export interface AuditLog {
  id: string;
  admin_id: string;
  admin_email: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  details: Record<string, any>;
  ip_address: string;
  user_agent: string;
  status: 'success' | 'failure';
  created_at: string;
}

export interface AuditLogParams {
  page?: number;
  page_size?: number;
  admin_id?: string;
  action?: string;
  resource_type?: string;
  start_date?: string;
  end_date?: string;
  status?: string;
}

// ==================== System Types ====================

export interface SystemStatus {
  name: string;
  status: 'running' | 'stopped' | 'error' | 'degraded';
  uptime: string;
  latency_ms: number;
  cpu_percent: number;
  memory_percent: number;
  last_check: string;
}

export interface SystemMetrics {
  total_requests: number;
  requests_per_second: number;
  average_latency_ms: number;
  error_rate: number;
  active_connections: number;
}

// ==================== Notification Types ====================

export interface Notification {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error' | 'alert';
  is_read: boolean;
  link?: string;
  created_at: string;
}

// ==================== Pagination Types ====================

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// ==================== Theme Types ====================

export type Theme = 'light' | 'dark' | 'system';

export interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolvedTheme: 'light' | 'dark';
}
