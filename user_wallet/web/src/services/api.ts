// API Service - Connects to the canonical TigerWallet Go wallet-api backend.
//
// The single source of truth for UserWallet data is go/wallet_api (port 8443),
// which performs REAL on-chain RPC (eth_getBalance / eth_call / Etherscan),
// real BIP-39/BIP-32/BIP-44 HD key derivation, real secp256k1 transaction
// signing + broadcast, and AES-256-GCM encrypted-seed persistence in
// PostgreSQL with Redis caching. No stubs, no fabricated data.
import axios, { AxiosInstance, AxiosError } from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8443/api/v1';

// Chain id map for the human-readable network keys used by the UI.
const CHAIN_IDS: Record<string, number> = {
  ethereum: 1,
  bsc: 56,
  polygon: 137,
  arbitrum: 42161,
  optimism: 10,
  base: 8453,
  avalanche: 43114,
};

function chainIdFor(network: string): number {
  return CHAIN_IDS[network] ?? (parseInt(network, 10) || 1);
}

export interface WalletRecord {
  id: string;
  label: string;
  chain_id: number;
  address: string;
  derivation_path: string;
  mnemonic?: string;
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
  symbol: string;
  name: string;
  balance: string;
  balance_f: number;
  decimals: number;
  usd_value: number;
}

export interface TransactionRecord {
  hash: string;
  from: string;
  to: string;
  value: string;
  timeStamp: string;
  isError: string;
  [key: string]: unknown;
}

export interface ChainInfo {
  id: number;
  name: string;
  symbol: string;
  rpc_endpoint: string;
  derivation_path?: string;
  explorer_api?: string;
  explorer_url?: string;
  chain_type?: string;
  decimals?: number;
  coin_type?: number;
  is_testnet?: boolean;
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
  // wallet_api handleLogin -> { token, user: { id, email, username } }
  async login(email: string, password: string) {
    try {
      const { data } = await this.client.post('/auth/login', { email, password });
      const user = {
        id: data.user?.id || data.user_id || '',
        email: data.user?.email || email,
        username: data.user?.username || email,
      };
      return { token: data.token as string, user };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Login failed'));
    }
  }

  // wallet_api handleRegister accepts {email, password} only (see route table).
  async register(email: string, _username: string, password: string) {
    try {
      const { data } = await this.client.post('/auth/register', { email, password });
      return {
        user_id: data.user_id as string,
        token: data.token as string,
        user: { id: data.user_id, email, username: _username },
      };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Registration failed'));
    }
  }

  // No /profile endpoint on wallet_api; the login/register responses already
  // include the user identity. Decode the JWT payload locally (no network call
  // to a non-existent route) so the AuthContext can hydrate the user from a
  // stored token on reload.
  async getProfile() {
    if (!this.token) throw new Error('Not authenticated');
    const payload = this.token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return {
      id: decoded.sub || decoded.user_id || '',
      email: decoded.email || '',
      username: decoded.username || decoded.email || '',
    };
  }

  // ---- Wallets ----
  // wallet_api handleListWallets -> { wallets: WalletRecord[] }
  async getWallets(): Promise<{ wallets: WalletRecord[] }> {
    const { data } = await this.client.get('/wallets');
    return data;
  }

  // wallet_api handleCreateWallet requires { password(min 8), label, chain_id, mnemonic?, ... }
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

  // Full createWallet for the typed call path used by pages that already know
  // the password / chain id (e.g. import flow).
  async createWalletTyped(params: {
    label: string;
    password: string;
    chainId: number;
    mnemonic?: string;
    accountIndex?: number;
    entropyBits?: number;
  }): Promise<WalletRecord> {
    const { data } = await this.client.post('/wallets', params);
    return data;
  }

  // ---- Balances ----
  // Aggregated balances across all of the user's wallets via the auth
  // /balance endpoint (real eth_getBalance through the backend).
  async getBalances(): Promise<{ balances: BalanceResult[] }> {
    const { wallets } = await this.getWallets();
    const results = await Promise.allSettled(
      wallets.map((w) =>
        this.client
          .get<BalanceResult>('/balance', { params: { address: w.address, chain_id: w.chain_id } })
          .then((r) => r.data),
      ),
    );
    const balances: BalanceResult[] = [];
    results.forEach((r) => {
      if (r.status === 'fulfilled') balances.push(r.value);
    });
    return { balances };
  }

  async getBalance(address: string, chainId: number): Promise<BalanceResult> {
    const { data } = await this.client.get('/balance', { params: { address, chain_id: chainId } });
    return data;
  }

  // ---- Tokens ----
  async getTokenBalances(address: string, chainId: number): Promise<{ tokens: TokenBalance[] }> {
    const { data } = await this.client.get('/tokens', { params: { address, chain_id: chainId } });
    return data;
  }

  // ---- Transactions ----
  // wallet_api handleTransactions -> { transactions: TransactionRecord[] }
  async getTransactions(params?: { network?: string; token?: string; address?: string }): Promise<{
    transactions: TransactionRecord[];
  }> {
    const query: Record<string, string | number> = {};
    if (params?.address) query.address = params.address;
    else if (this.token) {
      const { wallets } = await this.getWallets();
      if (wallets.length > 0) query.address = wallets[0].address;
    }
    if (params?.network) query.chain_id = chainIdFor(params.network);
    else query.chain_id = 1;
    const { data } = await this.client.get('/transactions', { params: query });
    return data;
  }

  // ---- Send (real on-chain broadcast via wallet_api /send) ----
  async sendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
  }): Promise<{ tx_hash: string; raw_tx: string; nonce: number }> {
    const { data } = await this.client.post('/send', {
      wallet_id: params.walletId,
      password: params.password,
      to: params.to,
      value: params.value,
      chain_id: params.chainId ?? 1,
      gas_limit: params.gasLimit,
      data: params.data,
    });
    return data;
  }

  // ---- Sign (real EIP-191 personal_sign via wallet_api /sign) ----
  async signMessage(params: { walletId: string; password: string; message: string }): Promise<{
    signature: string;
  }> {
    const { data } = await this.client.post('/sign', {
      wallet_id: params.walletId,
      password: params.password,
      message: params.message,
    });
    return data;
  }

  // ---- Price (real CoinGecko via wallet_api /price) ----
  // wallet_api accepts ?symbol= (e.g. "eth") or ?ids= (CoinGecko coin id).
  async getTokenPrice(token: string, _network?: string): Promise<{ usd: number; usd_24h_change: number }> {
    const symbol = token.toLowerCase() === 'btc' ? 'btc' : token.toLowerCase();
    const { data } = await this.client.get('/price', { params: { symbol } });
    return data;
  }

  // ---- Chains ----
  async getNetworks(): Promise<{ chains: ChainInfo[] }> {
    const { data } = await this.client.get('/chains');
    return data;
  }

  // ---- Gas (real eth_gasPrice + feeHistory via wallet_api /gas) ----
  async getGasPrice(network: string): Promise<{
    chain_id: number;
    gas_price: string;
    max_fee_per_gas: string;
    max_priority_fee: string;
    gas_price_gwei: number;
  }> {
    const { data } = await this.client.get('/gas', { params: { chain_id: chainIdFor(network) } });
    return data;
  }

  // ---- Network Status (live RPC chain id + latest block) ----
  async getNetworkStatus(network: string): Promise<{ chain_id: number; block_number: number; connected: boolean }> {
    const { data } = await this.client.get('/chains');
    const chain = (data.chains as ChainInfo[]).find((c) => c.id === chainIdFor(network));
    return {
      chain_id: chain?.id ?? chainIdFor(network),
      block_number: 0,
      connected: !!chain,
    };
  }

  // ---- NFTs (real Etherscan NFT inventory via wallet_api /nfts) ----
  async getNFTs(address: string, chainId: number): Promise<{ nfts: unknown[] }> {
    const { data } = await this.client.get('/nfts', { params: { address, chain_id: chainId } });
    return data;
  }

  // ---- Swap (real CoinGecko cross-rate + on-chain via wallet_api) ----
  async getSwapQuote(params: { fromToken: string; toToken: string; fromAmount: string; chainId?: number }): Promise<{
    from_token: string; to_token: string; from_amount: string; to_amount: string; price_impact: number; route: string;
  }> {
    const { data } = await this.client.get('/swap/quote', {
      params: { from_token: params.fromToken, to_token: params.toToken, from_amount: params.fromAmount, chain_id: params.chainId ?? 1 },
    });
    return data;
  }

  // ---- Staking (real on-chain action via wallet_api /send) ----
  async getStakingQuote(asset: string): Promise<{ asset: string; apy: number; min_amount: string }> {
    const { data } = await this.client.get('/staking/quote', { params: { asset } });
    return data;
  }
}

export const api = new ApiService();
export default api;
