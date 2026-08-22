// MasterWallet Web — Canonical API client
// Targets the canonical Go backend (port 8450) per CANONICAL_API_CONTRACT.md.
// Base URL: http://localhost:8450 (override via MASTER_WALLET_API_URL env var).
// All protected routes send `Authorization: Bearer <JWT>`.

const API_BASE_URL: string =
  (typeof process !== 'undefined' && process.env && process.env.MASTER_WALLET_API_URL) ||
  'http://localhost:8450';

// ---------------- Types ----------------

export interface AuthResponse {
  token: string;
  user_id: string;
  email: string;
  role: string;
}

export interface MasterWallet {
  id: string;
  name: string;
  blockchain?: string;
  chain_id?: number;
  address: string;
  wallet_type?: string;
  created_at?: string;
  mnemonic?: string; // returned once on creation
}

export interface ChainConfig {
  chain_id: number;
  name: string;
  symbol: string;
  blockchain: string;
  rpc_env?: string;
  decimals: number;
  derivation_path: string;
  is_evm: boolean;
}

export interface TokenBalance {
  contract: string;
  symbol: string;
  balance: string;
  decimals: number;
}

export interface BalanceResponse {
  address?: string;
  chain_id?: number;
  balance: string;
  symbol?: string;
  decimals?: number;
  tokens?: TokenBalance[];
  source?: string;
}

export interface SubWallet {
  id: string;
  name: string;
  address: string;
  blockchain?: string;
  chain_id?: number;
  wallet_type?: string;
  balance?: string;
  status?: string;
  created_at?: string;
}

export interface Transaction {
  id: string;
  tx_hash?: string;
  tx_type?: string;
  status: string;
  blockchain?: string;
  chain_id?: number;
  from_address?: string;
  to_address?: string;
  amount: string;
  token_symbol?: string;
  created_at?: string;
  nonce?: number;
  required_signatures?: number;
  to?: string;
  value?: string;
  data?: string;
}

export interface Policy {
  id: string;
  master_wallet_id?: string;
  name: string;
  policy_type: string;
  is_active?: boolean;
  priority?: number;
  conditions?: Record<string, unknown>;
  actions?: Record<string, unknown>;
  created_at?: string;
}

export interface FeeConfig {
  id: string;
  fee_type: string;
  fee_percentage?: number;
  fee_fixed?: string;
  is_active?: boolean;
  created_at?: string;
}

export interface AutoSignRule {
  id: string;
  name: string;
  rule_type: string;
  max_amount?: string;
  is_active?: boolean;
  conditions?: Record<string, unknown>;
  created_at?: string;
}

export interface MasterUser {
  id: string;
  email: string;
  name: string;
  role: string;
  is_active?: boolean;
  last_login_at?: string;
  created_at?: string;
}

export interface AuditLog {
  id: string;
  event_type: string;
  event_category?: string;
  actor_type?: string;
  actor_id?: string;
  target_type?: string;
  target_id?: string;
  severity?: string;
  details?: Record<string, unknown>;
  created_at: string;
}

export interface NotificationItem {
  id: string;
  type?: string;
  category?: string;
  title: string;
  message: string;
  priority?: string;
  is_read?: boolean;
  created_at?: string;
}

export interface Webhook {
  id: string;
  name: string;
  url: string;
  events: string[];
  is_active?: boolean;
  is_verified?: boolean;
  total_delivered?: number;
  total_failed?: number;
  created_at?: string;
}

export interface MultisigWallet {
  id: string;
  name: string;
  chain_id: number;
  threshold: number;
  owners: string[];
  nonce?: number;
  created_at?: string;
}

export interface TreasuryOverview {
  master_wallet_id: string;
  address: string;
  chain_id: number;
  total_balance: string;
  total_balance_wei?: string;
  total_value_usd: number;
  native_symbol: string;
  updated_at: string;
}

export interface GasPriceResponse {
  chain_id: number;
  gas_price: string;
  max_fee: string;
  priority_fee: string;
  source?: string;
}

export interface PriceResponse {
  coin_id: string;
  usd: number;
  usd_24h_change?: number;
  market_cap?: number;
  source?: string;
}

export interface SignTransactionRequest {
  to: string;
  amount: string;
  password: string;
  token?: string;
  gas_limit?: number;
}

export interface SignTransactionResponse {
  transaction_hash: string;
  status: string;
  from?: string;
  chain_id?: number;
}

// Two-party revenue gate — matches the canonical Go backend handlers
// (RevenuePayout + WithdrawalRequest in backend/handlers.go).

export interface RevenuePayoutRequest {
  to: string;
  amount: string;
  password: string;
  gas_limit?: number;
  withdrawal_id: string;
}

export interface RevenuePayoutResponse {
  transaction_hash: string;
  status: string;
  withdrawal_id?: string;
  from?: string;
  chain_id?: number;
}

export interface WithdrawalRequestRequest {
  to_address: string;
  amount_wei: string;
  currency?: string;
  chain_id?: number;
}

export interface WithdrawalRequestResponse {
  withdrawal_id: string;
  status: string;
}

// ---------------- Token storage ----------------

const TOKEN_KEY = 'master_wallet_token';

export function getAuthToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setAuthToken(token: string): void {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch {
    /* ignore storage errors */
  }
}

export function clearAuthToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {
    /* ignore storage errors */
  }
}

// ---------------- API client ----------------

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(message: string, status: number, body: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
  }
}

class MasterWalletAPI {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL.replace(/\/$/, '');
  }

  get baseUrl(): string {
    return this.baseURL;
  }

  get wsUrl(): string {
    return this.baseURL.replace(/^http/, 'ws') + '/ws';
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    auth = true
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> | undefined),
    };
    if (auth) {
      const token = getAuthToken();
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
    }
    let response: Response;
    try {
      response = await fetch(url, { ...options, headers });
    } catch (err) {
      throw new ApiError(`Network error: ${String(err)}`, 0, null);
    }
    const text = await response.text();
    let body: unknown = null;
    if (text) {
      try {
        body = JSON.parse(text);
      } catch {
        body = text;
      }
    }
    if (!response.ok) {
      let message = `API Error: ${response.status}`;
      if (body && typeof body === 'object' && 'error' in (body as Record<string, unknown>)) {
        message = String((body as Record<string, unknown>).error) || message;
      } else if (typeof body === 'string' && body) {
        message = body;
      }
      throw new ApiError(message, response.status, body);
    }
    return body as T;
  }

  // ---------------- Auth ----------------

  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    return this.request<AuthResponse>(
      '/api/v1/auth/register',
      { method: 'POST', body: JSON.stringify({ email, password, name }) },
      false
    );
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>(
      '/api/v1/auth/login',
      { method: 'POST', body: JSON.stringify({ email, password }) },
      false
    );
    setAuthToken(res.token);
    return res;
  }

  logout(): void {
    clearAuthToken();
  }

  // ---------------- Master wallets ----------------

  async getMasterWallets(): Promise<MasterWallet[]> {
    const data = await this.request<{ wallets: MasterWallet[] }>('/api/v1/master-wallet');
    return data.wallets ?? [];
  }

  async createMasterWallet(
    name: string,
    password: string,
    chain_id: number,
    mnemonic?: string
  ): Promise<MasterWallet> {
    const payload: Record<string, unknown> = { name, password, chain_id };
    if (mnemonic) payload.mnemonic = mnemonic;
    return this.request<MasterWallet>('/api/v1/master-wallet', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getMasterWallet(id: string): Promise<MasterWallet> {
    return this.request<MasterWallet>(`/api/v1/master-wallet/${id}`);
  }

  async updateMasterWallet(
    id: string,
    req: {
      name?: string;
      is_active?: boolean;
      daily_limit?: string;
      per_transaction_limit?: string;
      metadata?: Record<string, unknown>;
    }
  ): Promise<{ id: string; updated: boolean }> {
    return this.request<{ id: string; updated: boolean }>(
      `/api/v1/master-wallet/${id}`,
      { method: 'PUT', body: JSON.stringify(req) }
    );
  }

  async deleteMasterWallet(id: string): Promise<void> {
    await this.request<{ id: string; deleted: boolean }>(
      `/api/v1/master-wallet/${id}`,
      { method: 'DELETE' }
    );
  }

  async getMasterWalletBalance(id: string): Promise<BalanceResponse> {
    return this.request<BalanceResponse>(`/api/v1/master-wallet/${id}/balance`);
  }

  async signTransaction(
    id: string,
    req: SignTransactionRequest
  ): Promise<SignTransactionResponse> {
    return this.request<SignTransactionResponse>(`/api/v1/master-wallet/${id}/sign`, {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  // ---------------- Two-party revenue gate ----------------

  async requestWithdrawal(
    masterWalletId: string,
    req: WithdrawalRequestRequest
  ): Promise<WithdrawalRequestResponse> {
    return this.request<WithdrawalRequestResponse>(
      `/api/v1/master-wallet/${masterWalletId}/withdrawal-request`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async revenuePayout(
    masterWalletId: string,
    req: RevenuePayoutRequest
  ): Promise<RevenuePayoutResponse> {
    return this.request<RevenuePayoutResponse>(
      `/api/v1/master-wallet/${masterWalletId}/revenue-payout`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  // ---------------- Sub-wallets ----------------

  async getSubWallets(masterId: string): Promise<SubWallet[]> {
    const data = await this.request<{ sub_wallets: SubWallet[] }>(
      `/api/v1/master-wallet/${masterId}/sub-wallets`
    );
    return data.sub_wallets ?? [];
  }

  async createSubWallet(
    masterId: string,
    name: string,
    password: string,
    chain_id: number
  ): Promise<SubWallet> {
    return this.request<SubWallet>(`/api/v1/master-wallet/${masterId}/sub-wallets`, {
      method: 'POST',
      body: JSON.stringify({ name, password, chain_id }),
    });
  }

  async getSubWalletBalance(
    masterId: string,
    subId: string
  ): Promise<BalanceResponse> {
    return this.request<BalanceResponse>(
      `/api/v1/master-wallet/${masterId}/sub-wallets/${subId}/balance`
    );
  }

  async transferFromSubWallet(
    masterId: string,
    subId: string,
    req: SignTransactionRequest
  ): Promise<SignTransactionResponse> {
    return this.request<SignTransactionResponse>(
      `/api/v1/master-wallet/${masterId}/sub-wallets/${subId}/transfer`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  // ---------------- Transactions ----------------

  async getTransactions(masterId: string): Promise<Transaction[]> {
    const data = await this.request<{ transactions: Transaction[] }>(
      `/api/v1/master-wallet/${masterId}/transactions`
    );
    return data.transactions ?? [];
  }

  async getTransaction(masterId: string, txId: string): Promise<Transaction> {
    const data = await this.request<{ transaction: Transaction }>(
      `/api/v1/master-wallet/${masterId}/transactions/${txId}`
    );
    return data.transaction;
  }

  async createTransaction(
    masterId: string,
    req: SignTransactionRequest
  ): Promise<SignTransactionResponse> {
    return this.request<SignTransactionResponse>(
      `/api/v1/master-wallet/${masterId}/transactions`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async approveTransaction(
    masterId: string,
    txId: string
  ): Promise<{ id: string; approved: boolean }> {
    return this.request(`/api/v1/master-wallet/${masterId}/transactions/${txId}/approve`, {
      method: 'POST',
    });
  }

  async rejectTransaction(
    masterId: string,
    txId: string
  ): Promise<{ id: string; rejected: boolean }> {
    return this.request(`/api/v1/master-wallet/${masterId}/transactions/${txId}/reject`, {
      method: 'POST',
    });
  }

  // ---------------- Policies ----------------

  async getPolicies(masterId: string): Promise<Policy[]> {
    const data = await this.request<{ policies: Policy[] }>(
      `/api/v1/master-wallet/${masterId}/policies`
    );
    return data.policies ?? [];
  }

  async createPolicy(masterId: string, policy: Partial<Policy>): Promise<Policy> {
    return this.request<Policy>(`/api/v1/master-wallet/${masterId}/policies`, {
      method: 'POST',
      body: JSON.stringify(policy),
    });
  }

  async updatePolicy(masterId: string, pid: string, policy: Partial<Policy>): Promise<Policy> {
    return this.request<Policy>(`/api/v1/master-wallet/${masterId}/policies/${pid}`, {
      method: 'PUT',
      body: JSON.stringify(policy),
    });
  }

  async deletePolicy(masterId: string, pid: string): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/policies/${pid}`, {
      method: 'DELETE',
    });
  }

  // ---------------- Fees ----------------

  async getFeeConfigs(masterId: string): Promise<FeeConfig[]> {
    const data = await this.request<{ fees: FeeConfig[] }>(
      `/api/v1/master-wallet/${masterId}/fees`
    );
    return data.fees ?? [];
  }

  async createFeeConfig(masterId: string, fee: Partial<FeeConfig>): Promise<FeeConfig> {
    return this.request<FeeConfig>(`/api/v1/master-wallet/${masterId}/fees`, {
      method: 'POST',
      body: JSON.stringify(fee),
    });
  }

  async updateFeeConfig(
    masterId: string,
    fid: string,
    updates: Partial<FeeConfig>
  ): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/fees/${fid}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteFeeConfig(masterId: string, fid: string): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/fees/${fid}`, {
      method: 'DELETE',
    });
  }

  // ---------------- Auto-sign rules ----------------

  async getAutoSignRules(masterId: string): Promise<AutoSignRule[]> {
    const data = await this.request<{ auto_sign_rules: AutoSignRule[] }>(
      `/api/v1/master-wallet/${masterId}/auto-sign`
    );
    return data.auto_sign_rules ?? [];
  }

  async createAutoSignRule(
    masterId: string,
    rule: Partial<AutoSignRule>
  ): Promise<AutoSignRule> {
    return this.request<AutoSignRule>(`/api/v1/master-wallet/${masterId}/auto-sign`, {
      method: 'POST',
      body: JSON.stringify(rule),
    });
  }

  async updateAutoSignRule(
    masterId: string,
    rid: string,
    updates: Partial<AutoSignRule>
  ): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/auto-sign/${rid}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteAutoSignRule(masterId: string, rid: string): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/auto-sign/${rid}`, {
      method: 'DELETE',
    });
  }

  // ---------------- Users ----------------

  async getUsers(masterId: string): Promise<MasterUser[]> {
    const data = await this.request<{ users: MasterUser[] }>(
      `/api/v1/master-wallet/${masterId}/users`
    );
    return data.users ?? [];
  }

  async createUser(
    masterId: string,
    user: { email: string; password: string; name?: string; role?: string }
  ): Promise<MasterUser> {
    return this.request<MasterUser>(`/api/v1/master-wallet/${masterId}/users`, {
      method: 'POST',
      body: JSON.stringify(user),
    });
  }

  async updateUser(
    masterId: string,
    uid: string,
    updates: { name?: string; role?: string; is_active?: boolean; password?: string }
  ): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/users/${uid}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteUser(masterId: string, uid: string): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/users/${uid}`, {
      method: 'DELETE',
    });
  }

  // ---------------- Audit + Analytics ----------------

  async getAuditLogs(masterId: string, limit?: number): Promise<AuditLog[]> {
    const query = limit ? `?limit=${limit}` : '';
    const data = await this.request<{ audit_logs: AuditLog[] }>(
      `/api/v1/master-wallet/${masterId}/audit${query}`
    );
    return data.audit_logs ?? [];
  }

  async getVolumeAnalytics(
    masterId: string
  ): Promise<{ master_wallet_id: string; total_volume: string; transaction_count: number }> {
    return this.request(`/api/v1/master-wallet/${masterId}/analytics/volume`);
  }

  async getTransactionAnalytics(
    masterId: string
  ): Promise<{ master_wallet_id: string; by_status: Record<string, number> }> {
    return this.request(`/api/v1/master-wallet/${masterId}/analytics/transactions`);
  }

  async getWalletAnalytics(masterId: string): Promise<{
    master_wallets: number;
    sub_wallets: number;
    users: number;
  }> {
    return this.request(`/api/v1/master-wallet/${masterId}/analytics/wallets`);
  }

  // ---------------- Notifications + Webhooks ----------------

  async getNotifications(masterId: string): Promise<NotificationItem[]> {
    const data = await this.request<{ notifications: NotificationItem[] }>(
      `/api/v1/master-wallet/${masterId}/notifications`
    );
    return data.notifications ?? [];
  }

  async createNotification(
    masterId: string,
    n: Partial<NotificationItem>
  ): Promise<NotificationItem> {
    return this.request<NotificationItem>(`/api/v1/master-wallet/${masterId}/notifications`, {
      method: 'POST',
      body: JSON.stringify(n),
    });
  }

  async updateNotification(
    masterId: string,
    nid: string,
    updates: { title?: string; message?: string; priority?: string; is_read?: boolean }
  ): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/notifications/${nid}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async getWebhooks(masterId: string): Promise<Webhook[]> {
    const data = await this.request<{ webhooks: Webhook[] }>(
      `/api/v1/master-wallet/${masterId}/webhooks`
    );
    return data.webhooks ?? [];
  }

  async createWebhook(masterId: string, wh: Partial<Webhook>): Promise<Webhook> {
    return this.request<Webhook>(`/api/v1/master-wallet/${masterId}/webhooks`, {
      method: 'POST',
      body: JSON.stringify(wh),
    });
  }

  async updateWebhook(
    masterId: string,
    wid: string,
    updates: Partial<Webhook>
  ): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/webhooks/${wid}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteWebhook(masterId: string, wid: string): Promise<void> {
    await this.request(`/api/v1/master-wallet/${masterId}/webhooks/${wid}`, {
      method: 'DELETE',
    });
  }

  // ---------------- Treasury ----------------

  async getTreasuryOverview(masterId: string): Promise<TreasuryOverview> {
    return this.request<TreasuryOverview>(`/api/v1/master-wallet/${masterId}/treasury`);
  }

  async getTreasuryTransactions(masterId: string): Promise<Transaction[]> {
    const data = await this.request<{ transactions: Transaction[] }>(
      `/api/v1/master-wallet/${masterId}/treasury/transactions`
    );
    return data.transactions ?? [];
  }

  async treasuryTransfer(
    masterId: string,
    req: { to: string; amount: string; password: string }
  ): Promise<SignTransactionResponse> {
    return this.request<SignTransactionResponse>(
      `/api/v1/master-wallet/${masterId}/treasury/transfer`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async treasurySweep(
    masterId: string,
    req: { to: string; password: string }
  ): Promise<SignTransactionResponse> {
    return this.request<SignTransactionResponse>(
      `/api/v1/master-wallet/${masterId}/treasury/sweep`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  // ---------------- Multisig ----------------

  async getMultisigWallets(masterId: string): Promise<MultisigWallet[]> {
    const data = await this.request<{ multisig_wallets: MultisigWallet[] }>(
      `/api/v1/master-wallet/${masterId}/multisig/wallets`
    );
    return data.multisig_wallets ?? [];
  }

  async createMultisigWallet(
    masterId: string,
    req: { name: string; owners: string[]; threshold: number; chain_id?: number }
  ): Promise<MultisigWallet> {
    return this.request<MultisigWallet>(
      `/api/v1/master-wallet/${masterId}/multisig/wallets`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async getMultisigWalletDetail(
    masterId: string,
    walletId: string
  ): Promise<MultisigWallet & { pending_transactions?: number }> {
    const data = await this.request<{ multisig_wallet: MultisigWallet & { pending_transactions?: number } }>(
      `/api/v1/master-wallet/${masterId}/multisig/wallets/${walletId}`
    );
    return data.multisig_wallet;
  }

  async getMultisigTransactions(
    masterId: string,
    walletId: string
  ): Promise<Transaction[]> {
    const data = await this.request<{ transactions: Transaction[] }>(
      `/api/v1/master-wallet/${masterId}/multisig/wallets/${walletId}/transactions`
    );
    return data.transactions ?? [];
  }

  async createMultisigTransaction(
    masterId: string,
    walletId: string,
    req: { to_address: string; value: string; data?: string }
  ): Promise<Transaction> {
    return this.request<Transaction>(
      `/api/v1/master-wallet/${masterId}/multisig/wallets/${walletId}/transactions`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async signMultisigTransaction(
    masterId: string,
    txId: string,
    req: { signature: string; signer: string; message_hash: string }
  ): Promise<{ id: string; status: string }> {
    return this.request(`/api/v1/master-wallet/${masterId}/multisig/transactions/${txId}/sign`, {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async executeMultisigTransaction(
    masterId: string,
    txId: string
  ): Promise<{ id: string; status: string }> {
    return this.request(`/api/v1/master-wallet/${masterId}/multisig/transactions/${txId}/execute`, {
      method: 'POST',
    });
  }

  // ---------------- User EVM chains ----------------

  async listUserEVMChains(masterId: string): Promise<unknown[]> {
    const data = await this.request<{ chains?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/user-chains/evm`
    );
    return data.chains ?? [];
  }

  async addUserEVMChain(
    masterId: string,
    chain: {
      chain_id: number;
      name: string;
      symbol: string;
      rpc_url: string;
      explorer_url?: string;
      decimals: number;
      derivation_path: string;
    }
  ): Promise<unknown> {
    return this.request<unknown>(`/api/v1/master-wallet/${masterId}/user-chains/evm`, {
      method: 'POST',
      body: JSON.stringify(chain),
    });
  }

  async updateUserEVMChain(
    masterId: string,
    chainId: string | number,
    chain: {
      chain_id?: number;
      name?: string;
      symbol?: string;
      rpc_url?: string;
      explorer_url?: string;
      decimals?: number;
      derivation_path?: string;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/user-chains/evm/${chainId}`,
      { method: 'PUT', body: JSON.stringify(chain) }
    );
  }

  async removeUserEVMChain(
    masterId: string,
    chainId: string | number
  ): Promise<void> {
    await this.request(
      `/api/v1/master-wallet/${masterId}/user-chains/evm/${chainId}`,
      { method: 'DELETE' }
    );
  }

  // ---------------- User non-EVM chains ----------------

  async listUserNonEVMChains(masterId: string): Promise<unknown[]> {
    const data = await this.request<{ chains?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/user-chains/nonevm`
    );
    return data.chains ?? [];
  }

  async addUserNonEVMChain(
    masterId: string,
    chain: {
      chain_id: number;
      name: string;
      symbol: string;
      chain_type: string;
      rpc_url: string;
      explorer_url?: string;
      decimals: number;
      derivation_path: string;
      address_prefix?: string;
    }
  ): Promise<unknown> {
    return this.request<unknown>(`/api/v1/master-wallet/${masterId}/user-chains/nonevm`, {
      method: 'POST',
      body: JSON.stringify(chain),
    });
  }

  async updateUserNonEVMChain(
    masterId: string,
    chainId: string | number,
    chain: {
      chain_id?: number;
      name?: string;
      symbol?: string;
      chain_type?: string;
      rpc_url?: string;
      explorer_url?: string;
      decimals?: number;
      derivation_path?: string;
      address_prefix?: string;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/user-chains/nonevm/${chainId}`,
      { method: 'PUT', body: JSON.stringify(chain) }
    );
  }

  async removeUserNonEVMChain(
    masterId: string,
    chainId: string | number
  ): Promise<void> {
    await this.request(
      `/api/v1/master-wallet/${masterId}/user-chains/nonevm/${chainId}`,
      { method: 'DELETE' }
    );
  }

  // ---------------- User tokens ----------------

  async listUserTokens(masterId: string, chainId?: number | string): Promise<unknown[]> {
    const query = chainId !== undefined ? `?chain_id=${chainId}` : '';
    const data = await this.request<{ tokens?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/user-tokens${query}`
    );
    return data.tokens ?? [];
  }

  async addUserToken(
    masterId: string,
    token: {
      chain_id: number;
      contract_address: string;
      symbol: string;
      name: string;
      decimals: number;
      logo_uri?: string;
      is_native?: boolean;
    }
  ): Promise<unknown> {
    return this.request<unknown>(`/api/v1/master-wallet/${masterId}/user-tokens`, {
      method: 'POST',
      body: JSON.stringify(token),
    });
  }

  async updateUserToken(
    masterId: string,
    tokenId: string | number,
    token: {
      chain_id?: number;
      contract_address?: string;
      symbol?: string;
      name?: string;
      decimals?: number;
      logo_uri?: string;
      is_native?: boolean;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/user-tokens/${tokenId}`,
      { method: 'PUT', body: JSON.stringify(token) }
    );
  }

  async removeUserToken(masterId: string, tokenId: string | number): Promise<void> {
    await this.request(
      `/api/v1/master-wallet/${masterId}/user-tokens/${tokenId}`,
      { method: 'DELETE' }
    );
  }

  // ---------------- Address derivation ----------------

  async deriveUserAddress(
    masterId: string,
    body: {
      mnemonic: string;
      chain_id: number;
      chain_type?: string;
      derivation_path: string;
      account_index?: number;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/derive-user-address`,
      { method: 'POST', body: JSON.stringify(body) }
    );
  }

  async listUserWalletAddresses(masterId: string): Promise<unknown[]> {
    const data = await this.request<{ addresses?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/user-wallet-addresses`
    );
    return data.addresses ?? [];
  }

  // ---------------- Auto-sign ----------------

  async autoSignTransaction(
    masterId: string,
    body: {
      mnemonic: string;
      chain_id: number;
      chain_type?: string;
      derivation_path: string;
      account_index?: number;
      tx_type?: string;
      to_address?: string;
      value?: string;
      token_address?: string;
      contract_address?: string;
      data?: string;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/auto-sign-transaction`,
      { method: 'POST', body: JSON.stringify(body) }
    );
  }

  async listAutoSignLogs(masterId: string): Promise<unknown[]> {
    const data = await this.request<{ logs?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/auto-sign-logs`
    );
    return data.logs ?? [];
  }

  // ---------------- Auto-sign bridge (MasterWallet-owner policy auto-approval) ----------------

  async userWalletAutoSign(masterId: string, body: Record<string, unknown>): Promise<unknown> {
    return this.request<{ [k: string]: unknown }>(
      `/api/v1/master-wallet/${masterId}/user-wallet-auto-sign`,
      { method: 'POST', body: JSON.stringify(body) }
    );
  }

  async checkAutoSignPolicy(masterId: string, body: Record<string, unknown>): Promise<unknown> {
    return this.request<{ [k: string]: unknown }>(
      `/api/v1/master-wallet/${masterId}/check-auto-sign-policy`,
      { method: 'POST', body: JSON.stringify(body) }
    );
  }

  // ---------------- Feature flags ----------------

  async listFeatureFlags(masterId: string): Promise<unknown[]> {
    const data = await this.request<{ flags?: unknown[] }>(
      `/api/v1/master-wallet/${masterId}/feature-flags`
    );
    return data.flags ?? [];
  }

  async addFeatureFlag(
    masterId: string,
    flag: {
      flag_key: string;
      flag_value?: unknown;
      description?: string;
      is_enabled?: boolean;
    }
  ): Promise<unknown> {
    return this.request<unknown>(`/api/v1/master-wallet/${masterId}/feature-flags`, {
      method: 'POST',
      body: JSON.stringify(flag),
    });
  }

  async updateFeatureFlag(
    masterId: string,
    flagId: string | number,
    flag: {
      flag_key?: string;
      flag_value?: unknown;
      description?: string;
      is_enabled?: boolean;
    }
  ): Promise<unknown> {
    return this.request<unknown>(
      `/api/v1/master-wallet/${masterId}/feature-flags/${flagId}`,
      { method: 'PUT', body: JSON.stringify(flag) }
    );
  }

  async removeFeatureFlag(masterId: string, flagId: string | number): Promise<void> {
    await this.request(
      `/api/v1/master-wallet/${masterId}/feature-flags/${flagId}`,
      { method: 'DELETE' }
    );
  }

  // ---------------- Public endpoints ----------------

  async getSupportedChains(): Promise<ChainConfig[]> {
    const data = await this.request<{ chains: ChainConfig[] }>('/api/v1/chains', {}, false);
    return data.chains ?? [];
  }

  async getGasPrice(chainId: number): Promise<GasPriceResponse> {
    return this.request<GasPriceResponse>(`/api/v1/gas?chain_id=${chainId}`, {}, false);
  }

  async getPrice(coinId: string): Promise<PriceResponse> {
    return this.request<PriceResponse>(`/api/v1/price?coin_id=${encodeURIComponent(coinId)}`, {}, false);
  }

  async getTransactionHistory(
    address: string,
    chainId: number
  ): Promise<Transaction[]> {
    const data = await this.request<{ transactions: Transaction[] }>(
      `/api/v1/transactions/history?address=${encodeURIComponent(address)}&chain_id=${chainId}`,
      {},
      false
    );
    return data.transactions ?? [];
  }

  async healthCheck(): Promise<boolean> {
    try {
      await this.request<{ status: string }>('/health', {}, false);
      return true;
    } catch {
      return false;
    }
  }

  async apiHealth(): Promise<{ status: string; [k: string]: unknown }> {
    return this.request<{ status: string; [k: string]: unknown }>(
      '/api/v1/health',
      {},
      false
    );
  }

  // ---------------- Passkey relying-party ----------------
  // The backend is the WebAuthn relying party; the client performs the
  // navigator.credentials ceremony and POSTs the result for server-side
  // verification.

  async registerPasskey(
    masterId: string,
    req: {
      credential_id: string;
      public_key: string;
      sign_count?: number;
      transports?: string[];
      label?: string;
    }
  ): Promise<{ passkey_id: string; credential_id: string; registered: boolean }> {
    return this.request(
      `/api/v1/master-wallet/${masterId}/passkey/register`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }

  async listPasskeys(
    masterId: string
  ): Promise<PasskeyCredential[]> {
    const data = await this.request<{ passkeys: PasskeyCredential[] }>(
      `/api/v1/master-wallet/${masterId}/passkey/credentials`
    );
    return data.passkeys ?? [];
  }

  async deletePasskey(masterId: string, credentialId: string): Promise<void> {
    await this.request(
      `/api/v1/master-wallet/${masterId}/passkey/credentials/${encodeURIComponent(credentialId)}`,
      { method: 'DELETE' }
    );
  }

  async verifyPasskeyAssertion(
    masterId: string,
    req: {
      credential_id: string;
      authenticator_data: string;
      client_data_json: string;
      signature: string;
    }
  ): Promise<{ verified: boolean; credential_id: string }> {
    return this.request(
      `/api/v1/master-wallet/${masterId}/passkey/verify-assertion`,
      { method: 'POST', body: JSON.stringify(req) }
    );
  }
}

export interface PasskeyCredential {
  id: string;
  credential_id: string;
  sign_count: number;
  transports: string[];
  label?: string;
  created_at: string;
  updated_at: string;
}

export const masterWalletAPI = new MasterWalletAPI();
export default masterWalletAPI;
