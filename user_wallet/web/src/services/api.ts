// API Service - Connects to the standalone WL-UserWallet backend (port 8461).
//
// WL-UserWallet runs INDEPENDENTLY in the WL client's own environment: own
// PostgreSQL, own BIP-39/32/44 EVM key derivation, real secp256k1 signing +
// on-chain broadcast, AES-256-GCM encrypted-seed persistence, and a
// fail-closed license gate. RESTful /wallets/:id/* routes. No stubs, no
// fabricated data — every value comes from a real WL backend fetch.
import axios, { AxiosInstance, AxiosError } from 'axios';

// CRA (react-scripts) exposes REACT_APP_* env vars. Default to the WL
// standalone backend host port (docker-compose maps 8461:8443).
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8461/api/v1';

// /health lives at the server root (outside /api/v1). Derive it by stripping
// the /api/v1 suffix from the configured base, falling back to the WL host.
const HEALTH_URL = (API_BASE_URL.replace(/\/api\/v1\/?$/, '') || 'http://localhost:8461') + '/health';

// Chain id map for the human-readable network keys used by the UI. The WL
// backend derives the native symbol from chain_id; we mirror that mapping
// client-side for display only (no price feed — usd values stay 0, honestly).
const CHAIN_IDS: Record<string, number> = {
  ethereum: 1,
  bsc: 56,
  polygon: 137,
  arbitrum: 42161,
  optimism: 10,
  base: 8453,
  avalanche: 43114,
};

const CHAIN_SYMBOLS: Record<number, string> = {
  1: 'ETH',
  56: 'BNB',
  137: 'MATIC',
  42161: 'ETH',
  10: 'ETH',
  8453: 'ETH',
  43114: 'AVAX',
};

function chainIdFor(network: string): number {
  return CHAIN_IDS[network] ?? (parseInt(network, 10) || 1);
}

function symbolFor(chainId: number): string {
  return CHAIN_SYMBOLS[chainId] ?? 'ETH';
}

// Convert a wei string (balance_wei from the WL /balance endpoint) to a
// human-readable float in native units. Big-number safe via string parsing.
function weiToFloat(wei: string): number {
  if (!wei) return 0;
  const neg = wei.startsWith('-');
  const digits = neg ? wei.slice(1) : wei;
  const padded = digits.padStart(19, '0');
  const whole = padded.slice(0, -18);
  const frac = padded.slice(-18).replace(/0+$/, '');
  const num = parseFloat(`${whole}.${frac}`);
  return neg ? -num : num;
}

export interface WalletRecord {
  id: string;
  label: string;
  chain_id: number;
  address: string;
  created_at?: string;
  mnemonic?: string; // only present on the create response (generated mnemonic)
}

export interface BalanceResult {
  wallet_id: string;
  chain_id: number;
  symbol: string;
  address: string;
  balance_wei: string;
  balance_f: number;
  // The WL backend exposes no price feed; usd_value stays 0 (honest, never
  // fabricated). Kept in the interface for UI compatibility.
  usd_value: number;
}

export interface TransactionRecord {
  id: string;
  tx_hash: string;
  type: string;
  status: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  chain_id: number;
  created_at: string;
  [key: string]: unknown;
}

class ApiService {
  private client: AxiosInstance;
  private token: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: { 'Content-Type': 'application/json' },
    });

    this.client.interceptors.request.use((config) => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });
  }

  setToken(token: string) {
    this.token = token;
  }

  getToken(): string | null {
    return this.token;
  }

  private errMsg(err: unknown, fallback: string): string {
    if (err instanceof AxiosError && err.response?.data?.error) {
      return err.response.data.error;
    }
    if (err instanceof Error && err.message) return err.message;
    return fallback;
  }

  // ---- Auth ----
  // WL POST /auth/login -> { token, user_id, email }
  async login(email: string, password: string) {
    try {
      const { data } = await this.client.post('/auth/login', { email, password });
      const user = {
        id: data.user_id || '',
        email: data.email || email,
        username: data.email || email,
      };
      return { token: data.token as string, user };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Login failed'));
    }
  }

  // WL POST /auth/register accepts {email, password} and returns { id, email }
  // — it does NOT return a JWT. Callers must login afterwards to obtain a
  // token (handled by AuthContext.register).
  async register(email: string, _username: string, password: string) {
    try {
      const { data } = await this.client.post('/auth/register', { email, password });
      return {
        user_id: data.id as string,
        email: data.email as string,
        token: '',
        user: { id: data.id, email: data.email, username: data.email },
      };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Registration failed'));
    }
  }

  // The WL backend has no /profile route. The login/register responses
  // already include the user identity; decode the JWT payload locally (no
  // network call) so AuthContext can hydrate the user from a stored token.
  async getProfile() {
    if (!this.token) throw new Error('Not authenticated');
    const payload = this.token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return {
      id: decoded.sub || decoded.user_id || '',
      email: decoded.email || '',
      username: decoded.email || decoded.username || '',
    };
  }

  // ---- Health ----
  // WL GET /health (server root) -> { status, service, licensed, wl_client_id }
  async health(): Promise<{ status: string; service: string; licensed: boolean; wl_client_id: string }> {
    const { data } = await this.client.get(HEALTH_URL);
    return data;
  }

  // ---- Wallets ----
  // WL GET /wallets -> { wallets: WalletRecord[] }
  async getWallets(): Promise<{ wallets: WalletRecord[] }> {
    const { data } = await this.client.get('/wallets');
    return data;
  }

  // WL POST /wallets { label, password, chain_id, mnemonic?, passphrase? }
  // -> 201 { id, label, address, chain_id, mnemonic? }
  async createWallet(name: string, walletType: string, _networks: string[]): Promise<WalletRecord> {
    const password = window.prompt('Enter a wallet password (min 8 chars) to encrypt your seed:') || '';
    if (password.length < 8) throw new Error('Password must be at least 8 characters');
    const { data } = await this.client.post('/wallets', {
      label: name,
      password,
      chain_id: chainIdFor(walletType),
    });
    return data;
  }

  // Typed create path used by pages that already know the password / chain id.
  async createWalletTyped(params: {
    label: string;
    password: string;
    chainId: number;
    mnemonic?: string;
    passphrase?: string;
  }): Promise<WalletRecord> {
    const { data } = await this.client.post('/wallets', {
      label: params.label,
      password: params.password,
      chain_id: params.chainId,
      mnemonic: params.mnemonic,
      passphrase: params.passphrase,
    });
    return data;
  }

  // ---- Balances ----
  // Aggregated balances across all of the user's wallets. WL exposes balance
  // per wallet id (GET /wallets/:id/balance -> { address, balance_wei,
  // chain_id }), so we list wallets then fan out one real balance fetch each.
  async getBalances(): Promise<{ balances: BalanceResult[] }> {
    const { wallets } = await this.getWallets();
    const results = await Promise.allSettled(
      wallets.map((w) =>
        this.client
          .get<{ address: string; balance_wei: string; chain_id: number }>(`/wallets/${w.id}/balance`)
          .then((r) => r.data)
          .then((b) => ({
            wallet_id: w.id,
            chain_id: b.chain_id,
            symbol: symbolFor(b.chain_id),
            address: b.address,
            balance_wei: b.balance_wei,
            balance_f: weiToFloat(b.balance_wei),
            usd_value: 0,
          })),
      ),
    );
    const balances: BalanceResult[] = [];
    results.forEach((r) => {
      if (r.status === 'fulfilled') balances.push(r.value);
    });
    return { balances };
  }

  // WL GET /wallets/:id/balance -> { address, balance_wei, chain_id }
  async getBalance(walletId: string, _chainId?: number): Promise<BalanceResult> {
    const { data } = await this.client.get<{ address: string; balance_wei: string; chain_id: number }>(
      `/wallets/${walletId}/balance`,
    );
    return {
      wallet_id: walletId,
      chain_id: data.chain_id,
      symbol: symbolFor(data.chain_id),
      address: data.address,
      balance_wei: data.balance_wei,
      balance_f: weiToFloat(data.balance_wei),
      usd_value: 0,
    };
  }

  // ---- Transactions ----
  // WL GET /wallets/:id/transactions -> { transactions: TransactionRecord[] }
  async getTransactions(params?: { walletId: string; network?: string; token?: string }): Promise<{
    transactions: TransactionRecord[];
  }> {
    if (!params?.walletId) {
      // No wallet selected — return an empty list honestly (no fabricated txs).
      return { transactions: [] };
    }
    const { data } = await this.client.get<{ transactions: TransactionRecord[] }>(
      `/wallets/${params.walletId}/transactions`,
    );
    let txs = data.transactions || [];
    if (params.network) {
      const cid = chainIdFor(params.network);
      txs = txs.filter((t) => t.chain_id === cid);
    }
    if (params.token) {
      const tok = params.token.toUpperCase();
      txs = txs.filter((t) => (t.token || '').toUpperCase() === tok || (!t.token && tok === 'ETH'));
    }
    return { transactions: txs };
  }

  // ---- Send (real EVM signing + broadcast via WL POST /wallets/:id/send) ----
  // WL expects { to, amount (human-readable native units), password, gas_limit }
  // -> { transaction_hash, status, from }
  async sendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
  }): Promise<{ transaction_hash: string; status: string; from: string }> {
    const { data } = await this.client.post(`/wallets/${params.walletId}/send`, {
      to: params.to,
      amount: params.value,
      password: params.password,
      gas_limit: params.gasLimit,
    });
    return data;
  }

  // ---- Guest auth (public, no-auth) ----
  // POST /auth/guest { device_id } -> { user_id, token, guest: true }. Provisions
  // an anonymous guest account so a user can Create/Import a wallet without
  // registering. The token is persisted exactly like a login token (setToken).
  async guestAuth(deviceId: string): Promise<{ user_id: string; token: string; guest: boolean }> {
    try {
      const { data } = await this.client.post('/auth/guest', { device_id: deviceId });
      const token = data.token as string;
      if (token) this.setToken(token);
      return {
        user_id: data.user_id || '',
        token,
        guest: data.guest !== undefined ? Boolean(data.guest) : true,
      };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Guest auth failed'));
    }
  }

  // ---- Auto-send (auto-approval-gated send) ----
  // POST /auto-send with the SAME body as /send, plus optional
  // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
  // existing send response PLUS { auto_approved, auto_approval_reason }.
  async autoSendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
    masterWalletId?: string;
  }): Promise<{ transaction_hash: string; status: string; from: string; auto_approved: boolean; auto_approval_reason: string }> {
    const query = params.masterWalletId ? { master_wallet_id: params.masterWalletId } : undefined;
    const { data } = await this.client.post('/auto-send', {
      wallet_id: params.walletId,
      password: params.password,
      to: params.to,
      value: params.value,
      chain_id: params.chainId,
      gas_limit: params.gasLimit,
      data: params.data,
    }, { params: query });
    return data;
  }

  // ---- Transaction status (explorer proxy) ----
  // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
  async getTransactionStatus(txHash: string, chainId: number): Promise<{
    status: string;
    block_number?: number;
    confirmations?: number;
  }> {
    const { data } = await this.client.get(`/transactions/${encodeURIComponent(txHash)}`, {
      params: { chain_id: chainId },
    });
    return data;
  }

  // ---- Sign (real EIP-191 personal_sign via WL POST /wallets/:id/sign) ----
  // WL expects { message, password } -> { signature, address }
  async signMessage(params: { walletId: string; password: string; message: string }): Promise<{
    signature: string;
    address: string;
  }> {
    const { data } = await this.client.post(`/wallets/${params.walletId}/sign`, {
      message: params.message,
      password: params.password,
    });
    return data;
  }
}

// parsePaymentUri — decodes a scanned QR string (bare 0x address, ethereum:
// URI, or EIP-681 payment URI) into an address + optional amount. Returns
// null when no address can be extracted (fail-closed — never a guessed value).
export function parsePaymentUri(input: string): { address: string; amount?: string; chainId?: number; tokenAddress?: string } | null {
  const s = (input || '').trim();
  if (!s) return null;
  if (/^0x[a-fA-F0-9]{40}$/.test(s)) {
    return { address: s };
  }
  let body: string;
  if (s.startsWith('ethereum:')) body = s.slice('ethereum:'.length);
  else return null;
  const qIdx = body.indexOf('?');
  const target = qIdx >= 0 ? body.slice(0, qIdx) : body;
  const query = qIdx >= 0 ? body.slice(qIdx + 1) : '';
  let address: string;
  let tokenAddress: string | null = null;
  if (target.includes('/')) {
    const [addr, func] = target.split('/');
    address = addr;
    if (func.startsWith('transfer')) tokenAddress = '';
  } else {
    address = target;
  }
  if (!/^0x[a-fA-F0-9]{40}$/.test(address)) return null;
  let amount: string | undefined;
  let chainId: number | undefined;
  query.split('&').forEach((pair) => {
    const [k, v] = pair.split('=');
    if (k === 'value') amount = v;
    else if (k === 'chainId') chainId = Number(v);
    else if (k === 'address' && tokenAddress !== null) tokenAddress = v;
  });
  return { address, amount, chainId, tokenAddress: tokenAddress || undefined };
}

export const api = new ApiService();
export default api;
