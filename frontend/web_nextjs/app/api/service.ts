/**
 * TigerWallet API Service Layer
 * Comprehensive API client for frontend-backend connectivity
 * 
 * All endpoints connect to Go backend services
 */

// ============================================================================
// Configuration
// ============================================================================

// In the browser we talk same-origin to the Next.js API routes (which proxy
// to the Go wallet-api backend), avoiding CORS. On the server we hit the
// backend directly. NEXT_PUBLIC_API_URL can override either.
const isBrowser = typeof window !== 'undefined';
const API_CONFIG = {
  baseURL: isBrowser
    ? (process.env.NEXT_PUBLIC_API_URL || '')
    : (process.env.BACKEND_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443'),
  wsURL: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8443',
  timeout: 30000,
  retries: 3,
};

// ============================================================================
// Types
// ============================================================================

export interface User {
  id: string;
  email: string;
  username: string;
  kycStatus: string;
  kycLevel: number;
  emailVerified: boolean;
  phoneVerified: boolean;
  mfaEnabled: boolean;
  riskScore: number;
  createdAt: string;
  updatedAt: string;
}

export interface Wallet {
  id: string;
  userId: string;
  whiteLabelId?: string;
  walletType: 'user' | 'master';
  address: string;
  chainId: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Transaction {
  id: string;
  hash: string;
  fromAddress: string;
  toAddress: string;
  value: string;
  gasPrice: string;
  gasLimit: number;
  gasUsed?: number;
  chainId: number;
  status: 'pending' | 'broadcast' | 'confirmed' | 'failed';
  errorMessage?: string;
  blockNumber?: number;
  timestamp: string;
  createdAt: string;
}

export interface Token {
  id: string;
  address: string;
  chainId: number;
  name: string;
  symbol: string;
  decimals: number;
  isEnabled: boolean;
  isPopular: boolean;
  logoUrl?: string;
  priceUsd?: number;
}

export interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  email: string;
  status: 'active' | 'paused' | 'halted';
  feePercentage: number;
  allowedChains: string[];
  allowedFeatures: string[];
  customBranding: boolean;
  maxUsers: number;
  currentUsers: number;
  createdAt: string;
  updatedAt: string;
}

export interface Broker {
  id: string;
  name: string;
  email: string;
  whiteLabelId?: string;
  status: string;
  commissionRate: number;
  allowedIPs: string[];
  maxDailyVolume: number;
  currentVolume: number;
  createdAt: string;
  updatedAt: string;
}

export interface Institution {
  id: string;
  name: string;
  email: string;
  whiteLabelId?: string;
  status: string;
  kycStatus: string;
  accountType: 'retail' | 'professional' | 'institutional';
  tradingLimits: number;
  feeTier: number;
  allowedChains: string[];
  webhookUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface APIKey {
  id: string;
  userId: string;
  name: string;
  permissions: string[];
  rateLimit: number;
  expiresAt?: string;
  lastUsedAt?: string;
  isActive: boolean;
  createdAt: string;
}

export interface AuditLog {
  id: string;
  userId: string;
  action: string;
  resource: string;
  resourceId?: string;
  details?: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
}

// ============================================================================
// API Client
// ============================================================================

class APIClient {
  private baseURL: string;
  private token: string | null = null;
  private ws: WebSocket | null = null;
  private listeners: Map<string, Set<(data: any) => void>> = new Map();

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    this.loadToken();
  }

  // Token Management
  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('tigerwallet_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('tigerwallet_token');
    }
  }

  private loadToken() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('tigerwallet_token');
    }
  }

  // HTTP Methods
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  private get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  private post<T>(endpoint: string, data?: any): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  private put<T>(endpoint: string, data?: any): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  private delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  // WebSocket
  connectWebSocket() {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(API_CONFIG.wsURL);

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const type = data.type || 'default';
        
        const typeListeners = this.listeners.get(type);
        if (typeListeners) {
          typeListeners.forEach((listener) => listener(data.payload));
        }
      } catch (e) {
        console.error('WebSocket message parse error:', e);
      }
    };

    this.ws.onclose = () => {
      setTimeout(() => this.connectWebSocket(), 5000);
    };
  }

  subscribe(event: string, callback: (data: any) => void) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'subscribe', event }));
    }
  }

  unsubscribe(event: string, callback: (data: any) => void) {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      eventListeners.delete(callback);
      if (eventListeners.size === 0) {
        this.listeners.delete(event);
      }
    }
  }

  // ==========================================================================
  // Authentication
  // ==========================================================================

  async register(email: string, username: string, password: string): Promise<{ user: User; token: string }> {
    const result = await this.post<{ user: User; token: string }>('/api/v1/auth/register', {
      email,
      username,
      password,
    });
    this.setToken(result.token);
    return result;
  }

  async login(email: string, password: string): Promise<{ user: User; token: string }> {
    const result = await this.post<{ user: User; token: string }>('/api/v1/auth/login', {
      email,
      password,
    });
    this.setToken(result.token);
    return result;
  }

  async logout(): Promise<void> {
    await this.post('/api/v1/auth/logout');
    this.clearToken();
  }

  async refreshToken(): Promise<string> {
    const result = await this.post<{ token: string }>('/api/v1/auth/refresh');
    this.setToken(result.token);
    return result.token;
  }

  // ==========================================================================
  // User Profile
  // ==========================================================================

  async getProfile(): Promise<User> {
    return this.get<User>('/api/v1/profile');
  }

  async updateProfile(data: Partial<User>): Promise<User> {
    return this.put<User>('/api/v1/profile', data);
  }

  // ==========================================================================
  // Wallets
  // ==========================================================================

  async getWallets(): Promise<Wallet[]> {
    const result = await this.get<{ wallets: Wallet[] }>('/api/v1/wallets');
    return result.wallets;
  }

  async createWallet(walletType: string, chainId: number): Promise<Wallet> {
    const result = await this.post<{ wallet: Wallet }>('/api/v1/wallets', {
      walletType,
      chainId,
    });
    return result.wallet;
  }

  async getBalance(address: string, chainId: number): Promise<string> {
    const result = await this.get<{ balance: string }>(
      `/api/v1/wallets/${address}/balance?chainId=${chainId}`
    );
    return result.balance;
  }

  // ==========================================================================
  // Transactions
  // ==========================================================================

  async createTransaction(tx: {
    toAddress: string;
    value: string;
    gasPrice?: string;
    gasLimit?: number;
    chainId: number;
  }): Promise<Transaction> {
    const result = await this.post<{ transaction: Transaction }>('/api/v1/transactions', tx);
    return result.transaction;
  }

  async getTransactions(limit: number = 50): Promise<Transaction[]> {
    const result = await this.get<{ transactions: Transaction[] }>(
      `/api/v1/transactions?limit=${limit}`
    );
    return result.transactions;
  }

  async getTransaction(hash: string): Promise<Transaction> {
    const result = await this.get<{ transaction: Transaction }>(
      `/api/v1/transactions/${hash}`
    );
    return result.transaction;
  }

  // ==========================================================================
  // API Keys
  // ==========================================================================

  async getAPIKeys(): Promise<APIKey[]> {
    const result = await this.get<{ apiKeys: APIKey[] }>('/api/v1/api-keys');
    return result.apiKeys;
  }

  async createAPIKey(name: string, permissions: string[]): Promise<{ apiKey: APIKey; key: string }> {
    const result = await this.post<{ apiKey: APIKey; key: string }>('/api/v1/api-keys', {
      name,
      permissions,
    });
    return result;
  }

  async revokeAPIKey(id: string): Promise<void> {
    await this.delete(`/api/v1/api-keys/${id}`);
  }

  // ==========================================================================
  // White Label
  // ==========================================================================

  async getWhiteLabels(): Promise<WhiteLabel[]> {
    const result = await this.get<{ whiteLabels: WhiteLabel[] }>('/api/v1/white-labels');
    return result.whiteLabels;
  }

  async createWhiteLabel(data: {
    name: string;
    domain: string;
    email: string;
    feePercentage?: number;
  }): Promise<WhiteLabel> {
    const result = await this.post<{ whiteLabel: WhiteLabel }>('/api/v1/white-labels', data);
    return result.whiteLabel;
  }

  async updateWhiteLabel(id: string, data: Partial<WhiteLabel>): Promise<WhiteLabel> {
    const result = await this.put<{ whiteLabel: WhiteLabel }>(`/api/v1/white-labels/${id}`, data);
    return result.whiteLabel;
  }

  // ==========================================================================
  // Brokers
  // ==========================================================================

  async getBrokers(): Promise<Broker[]> {
    const result = await this.get<{ brokers: Broker[] }>('/api/v1/brokers');
    return result.brokers;
  }

  async createBroker(data: {
    name: string;
    email: string;
    commissionRate?: number;
  }): Promise<Broker> {
    const result = await this.post<{ broker: Broker }>('/api/v1/brokers', data);
    return result.broker;
  }

  // ==========================================================================
  // Institutions
  // ==========================================================================

  async getInstitutions(): Promise<Institution[]> {
    const result = await this.get<{ institutions: Institution[] }>('/api/v1/institutions');
    return result.institutions;
  }

  async createInstitution(data: {
    name: string;
    email: string;
    accountType?: string;
  }): Promise<Institution> {
    const result = await this.post<{ institution: Institution }>('/api/v1/institutions', data);
    return result.institution;
  }

  // ==========================================================================
  // Tokens
  // ==========================================================================

  async getTokens(chainId?: number): Promise<Token[]> {
    const url = chainId ? `/api/v1/tokens?chainId=${chainId}` : '/api/v1/tokens';
    const result = await this.get<{ tokens: Token[] }>(url);
    return result.tokens;
  }

  async createToken(data: {
    address: string;
    chainId: number;
    name: string;
    symbol: string;
    decimals: number;
  }): Promise<Token> {
    const result = await this.post<{ token: Token }>('/api/v1/tokens', data);
    return result.token;
  }

  // ==========================================================================
  // Audit Logs
  // ==========================================================================

  async getAuditLogs(limit: number = 100): Promise<AuditLog[]> {
    const result = await this.get<{ auditLogs: AuditLog[] }>(
      `/api/v1/audit-logs?limit=${limit}`
    );
    return result.auditLogs;
  }
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const api = new APIClient(API_CONFIG.baseURL);

// ============================================================================
// Blockchain Service (for direct blockchain interactions)
// ============================================================================

export class BlockchainService {
  private async httpError(res: Response, fallback: string): Promise<Error> {
    let detail = '';
    try {
      const body = await res.json();
      detail = body?.error || body?.message || '';
    } catch {
      // Non-JSON body — fall back below.
    }
    if (!detail) detail = fallback;
    return new Error(`${detail} (HTTP ${res.status})`);
  }

  async getBalance(address: string, chainId: number): Promise<string> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/public/balance?address=${address}&chain_id=${chainId}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch balance');
    const data = await res.json();
    return data.balance ?? '0';
  }

  async getGasPrice(chainId: number): Promise<string> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/gas?chain_id=${chainId}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch gas price');
    const data = await res.json();
    return data.gas_price ?? '0';
  }

  async getTransactionReceipt(txHash: string, chainId: number): Promise<unknown> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/public/transactions?address=&chain_id=${chainId}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch transactions');
    const data = await res.json();
    return (data.transactions ?? []).find((t: { hash: string }) => t.hash === txHash) ?? null;
  }
}

export const blockchain = new BlockchainService();

// ============================================================================
// Wallet Service (for wallet operations)
// ============================================================================

export class WalletService {
  /**
   * Create a new wallet via the real Go wallet-api backend. Generates a real
   * BIP-39 mnemonic server-side, derives the real secp256k1 key + EVM address,
   * and stores the encrypted seed in PostgreSQL. Returns the mnemonic once.
   */
  async createWallet(params: {
    password: string;
    label?: string;
    chainId?: number;
    mnemonic?: string;
    accountIndex?: number;
    entropyBits?: number;
  }): Promise<{
    id: string;
    label: string;
    chainId: number;
    address: string;
    derivationPath: string;
    mnemonic?: string;
  }> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/wallets`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify(params),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to create wallet');
    // The Go wallet-api returns snake_case (chain_id, derivation_path); normalize
    // to the camelCase shape the UI reads, so the created/imported wallet does not
    // silently lose its chainId/derivationPath.
    const data = await res.json();
    return normalizeWalletResp(data as Record<string, unknown>);
  }

  /**
   * Parse a non-2xx fetch response into a descriptive Error that surfaces the
   * backend's JSON `error` message (if any) plus the HTTP status, instead of a
   * generic "Failed to ..." string. This lets callers distinguish network
   * failures, validation errors (400), auth errors (401/403), not-found (404),
   * unavailable-upstream (502/503) and rate-limit (429) at the UI layer.
   */
  private async httpError(res: Response, fallback: string): Promise<Error> {
    let detail = '';
    try {
      const body = await res.json();
      detail = body?.error || body?.message || '';
    } catch {
      // Non-JSON body (e.g. plain text / empty) — fall back to status text.
    }
    if (!detail) detail = fallback;
    return new Error(`${detail} (HTTP ${res.status})`);
  }

  async listWallets(): Promise<{ wallets: WalletInfo[] }> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/wallets`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to list wallets');
    const data = (await res.json()) as { wallets?: Record<string, unknown>[] };
    return { wallets: (data.wallets ?? []).map(normalizeWalletResp) };
  }

  async getBalance(address: string, chainId: number): Promise<BalanceResult> {
    const res = await fetch(
      `${API_CONFIG.baseURL}/api/v1/public/balance?address=${address}&chain_id=${chainId}`
    );
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch balance');
    return res.json();
  }

  async getTokenBalances(address: string, chainId: number): Promise<{ tokens: TokenBalance[] }> {
    const res = await fetch(
      `${API_CONFIG.baseURL}/api/v1/public/tokens?address=${address}&chain_id=${chainId}`
    );
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch token balances');
    return res.json();
  }

  async getTransactions(address: string, chainId: number): Promise<{ transactions: TransactionHistory[] }> {
    const res = await fetch(
      `${API_CONFIG.baseURL}/api/v1/public/transactions?address=${address}&chain_id=${chainId}`
    );
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch transactions');
    return res.json();
  }

  async getNFTs(address: string, chainId: number): Promise<{ nfts: NFTAsset[] }> {
    const res = await fetch(
      `${API_CONFIG.baseURL}/api/v1/public/nfts?address=${address}&chain_id=${chainId}`
    );
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch NFTs');
    return res.json();
  }

  async sendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
  }): Promise<{ tx_hash: string; raw_tx: string; nonce: number }> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify(params),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to send transaction');
    return res.json();
  }

  async signMessage(params: { walletId: string; password: string; message: string }): Promise<{ signature: string }> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/sign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify(params),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to sign message');
    return res.json();
  }

  async getGasPrice(chainId: number): Promise<{
    chain_id: number;
    gas_price: string;
    max_fee_per_gas: string;
    max_priority_fee: string;
    gas_price_gwei: number;
  }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/gas?chain_id=${chainId}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch gas price');
    return res.json();
  }

  async getPrice(coin?: string): Promise<{ usd: number; usd_24h_change: number }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/price?coin=${coin || 'ethereum'}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch price');
    return res.json();
  }

  async getSupportedChains(): Promise<{ chains: ChainInfo[] }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/chains`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch supported chains');
    return res.json();
  }

  async login(email: string, password: string): Promise<{ token: string; user: unknown }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) throw await this.httpError(res, 'Login failed');
    const data = await res.json();
    if (data.token) localStorage.setItem('tigerwallet_token', data.token);
    return data;
  }

  async register(email: string, username: string, password: string): Promise<{ user_id: string; token: string }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, username, password }),
    });
    if (!res.ok) throw await this.httpError(res, 'Registration failed');
    const data = await res.json();
    if (data.token) localStorage.setItem('tigerwallet_token', data.token);
    return data;
  }

  /**
   * Guest authentication: provisions an anonymous account WITHOUT registration.
   * The UserWallet app opens directly to CreateWallet/ImportWallet — no
   * email/password. The client supplies a stable device id (from localStorage /
   * device fingerprint) so the same device re-gets the same guest account on
   * reconnect. Returns a JWT the app uses for all subsequent wallet calls.
   */
  async guestAuth(deviceId: string): Promise<{ user_id: string; token: string; guest: boolean }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/auth/guest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceId }),
    });
    if (!res.ok) throw await this.httpError(res, 'Guest auth failed');
    const data = await res.json();
    if (data.token) localStorage.setItem('tigerwallet_token', data.token);
    return data;
  }

  /**
   * Auto-send: self-sign + broadcast with MasterWallet-owner policy
   * auto-approval. The wallet_api asks the MasterWallet backend's auto-sign
   * rules (max_amount + active flag) server-to-server; if approved within a
   * second, the tx is self-signed with the user's own seed and broadcast, and
   * the response carries auto_approved=true so the UI can show
   * "transaction submitted to blockchain network (auto-approved by master
   * wallet)". The user always retains self-custody; the policy gate is a
   * gas-sponsorship/convenience layer, not a custody gate.
   */
  async autoSendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
    masterWalletId?: string;
  }): Promise<{ tx_hash: string; raw_tx: string; nonce: number; auto_approved: boolean; auto_approval_reason: string }> {
    const token = this.getAuthToken();
    const query = params.masterWalletId ? `?master_wallet_id=${encodeURIComponent(params.masterWalletId)}` : '';
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/auto-send${query}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify(params),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to send transaction');
    return res.json();
  }

  /**
   * Transaction status: poll a broadcast tx hash for confirmation on-chain.
   * Used to drive the "transaction submitted to blockchain network" →
   * "transaction confirmed" UX. Returns { status: 'pending'|'confirmed'|'failed' }.
   */
  async getTransactionStatus(txHash: string, chainId: number): Promise<{ status: string; block_number?: number; confirmations?: number }> {
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/transactions/${txHash}?chain_id=${chainId}`);
    if (!res.ok) throw await this.httpError(res, 'Failed to fetch transaction status');
    return res.json();
  }

  // Google Drive backup: export the encrypted seed blob (password-verified by backend).
  async exportEncryptedSeed(walletId: string, password: string): Promise<{
    encrypted_seed: string;
    wallet_id: string;
    address: string;
    chain_id: number;
    label: string;
    derivation_path: string;
    account_index: number;
  }> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/wallets/${walletId}/export-encrypted-seed`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ password }),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to export encrypted seed');
    return res.json();
  }

  // Google Drive restore: import an encrypted seed blob + password.
  async importEncryptedSeed(params: {
    encryptedSeed: string;
    password: string;
    label?: string;
    chainId?: number;
    derivationPath?: string;
    accountIndex?: number;
  }): Promise<WalletInfo> {
    const token = this.getAuthToken();
    const res = await fetch(`${API_CONFIG.baseURL}/api/v1/wallets/import-encrypted-seed`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({
        encrypted_seed: params.encryptedSeed,
        password: params.password,
        label: params.label,
        chain_id: params.chainId,
        derivation_path: params.derivationPath,
        account_index: params.accountIndex,
      }),
    });
    if (!res.ok) throw await this.httpError(res, 'Failed to restore wallet from backup');
    const data = (await res.json()) as Record<string, unknown>;
    return normalizeWalletResp(data);
  }

  logout(): void {
    localStorage.removeItem('tigerwallet_token');
  }

  private getAuthToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('tigerwallet_token');
  }
}

// ---- Types matching the Go backend ----

export interface WalletInfo {
  id: string;
  label: string;
  chainId: number;
  address: string;
  derivationPath: string;
}

// The Go wallet-api serializes wallet records with snake_case keys
// (chain_id, derivation_path) while the UI reads camelCase. Normalize both
// variants so wallet create/import/list never drop the chain id or path.
function normalizeWalletResp(w: Record<string, unknown>): WalletInfo {
  return {
    id: String(w.id ?? w.ID ?? ''),
    label: String(w.label ?? w.Label ?? ''),
    chainId: Number(w.chainId ?? w.chain_id ?? w.ChainID ?? 0),
    address: String(w.address ?? w.Address ?? ''),
    derivationPath: String(
      w.derivationPath ?? w.derivation_path ?? w.DerivationPath ?? ''
    ),
  };
}

export interface BalanceResult {
  chain_id: number;
  symbol: string;
  address: string;
  balance: string;
  balance_f: number;
  usd_value: number;
}

export interface TokenBalance {
  contract: string;
  symbol: string;
  name: string;
  decimals: number;
  balance: string;
  balance_f: number;
  usd_price: number;
  usd_value: number;
  logo?: string;
}

export interface TransactionHistory {
  hash: string;
  from: string;
  to: string;
  value: string;
  value_f: number;
  timestamp: number;
  status: string;
  direction: string;
  gas_used: string;
  is_token: boolean;
  token_symbol?: string;
}

export interface NFTAsset {
  contract: string;
  token_id: string;
  name: string;
  description: string;
  image_url: string;
  collection: string;
  standard: string;
}

export interface ChainInfo {
  id: number;
  name: string;
  symbol: string;
  rpc_endpoint: string;
  derivation_path: string;
  explorer_api: string;
  explorer_url: string;
  chain_type: string;
  decimals: number;
  coin_type: number;
  is_testnet: boolean;
}

export const walletService = new WalletService();

export default api;
