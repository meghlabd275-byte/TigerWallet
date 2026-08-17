// API Service - Connects to the canonical TigerWallet Go wallet-api backend
// (go/wallet_api, port 8443). Real on-chain RPC, real BIP-39/32/44 HD
// derivation, real secp256k1 signing + broadcast, PostgreSQL + Redis, plus
// transparent reverse-proxies to the auxiliary DeFi microservices
// (lending/copytrading/governance/prediction). No stubs, no fabricated data —
// every value comes from a real backend fetch.
//
// This is the canonical UserWallet client contract shared across all 7
// UserWallet platforms (web/desktop/extension/production-react/android/ios/
// rust): the SAME method names + endpoint shapes, so every app exposes the
// SAME feature set.
import axios, { AxiosInstance, AxiosError } from 'axios';

// CRA (react-scripts) exposes REACT_APP_* env vars. Default to the canonical
// wallet_api host port 8443 (docker-compose maps 8443:8443).
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8443/api/v1';

// /health lives at the server root (outside /api/v1). Derive it by stripping
// the /api/v1 suffix from the configured base, falling back to the host.
const HEALTH_URL = (API_BASE_URL.replace(/\/api\/v1\/?$/, '') || 'http://localhost:8443') + '/health';

// Chain id map for the human-readable network keys used by the UI. The backend
// derives the native symbol from chain_id; we mirror that mapping client-side
// for display only.
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

// Convert a wei string to a human-readable float in native units. Big-number
// safe via string parsing.
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
    if (err instanceof AxiosError && err.response?.data) {
      const d = err.response.data as { error?: string; message?: string };
      if (d.error) return d.error;
      if (d.message) return d.message;
    }
    if (err instanceof Error && err.message) return err.message;
    return fallback;
  }

  // ==================== Auth ====================
  // POST /auth/login -> { token, user_id, email }
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

  // POST /auth/register accepts {email, password} and returns { id, email, token }
  async register(email: string, _username: string, password: string) {
    try {
      const { data } = await this.client.post('/auth/register', { email, password });
      return {
        user_id: data.user_id || data.id || '',
        email: data.email as string,
        token: data.token as string,
        user: { id: data.user_id || data.id || '', email: data.email, username: data.email },
      };
    } catch (err) {
      throw new Error(this.errMsg(err, 'Registration failed'));
    }
  }

  // POST /auth/guest { device_id } -> { user_id, token, guest: true }.
  // Public (no auth). Provisions an anonymous guest account so a user can
  // Create/Import a wallet WITHOUT registering. Token persisted like login.
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

  async logout() {
    this.token = null;
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem('tigerwallet-token');
    }
  }

  // The backend has no /profile route. The login/register responses already
  // include the user identity; decode the JWT payload locally (no network call).
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

  // ==================== Health ====================
  async health(): Promise<{ status: string; service: string }> {
    const { data } = await this.client.get(HEALTH_URL);
    return data;
  }

  // ==================== Wallets ====================
  // GET /wallets -> { wallets: WalletRecord[] }
  async getWallets(): Promise<{ wallets: WalletRecord[] }> {
    const { data } = await this.client.get('/wallets');
    return data;
  }

  // POST /wallets { label, password, chain_id, mnemonic?, passphrase? } -> 201 { id, label, address, chain_id, mnemonic? }
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

  // Import a wallet from a BIP-39 mnemonic (POST /wallets with mnemonic).
  async importWallet(params: {
    label: string;
    password: string;
    mnemonic: string;
    chainId?: number;
    passphrase?: string;
  }): Promise<WalletRecord> {
    const { data } = await this.client.post('/wallets', {
      label: params.label,
      password: params.password,
      chain_id: params.chainId ?? 1,
      mnemonic: params.mnemonic,
      passphrase: params.passphrase,
    });
    return data;
  }

  // ==================== Balances & Tokens ====================
  // Aggregated balances across all of the user's wallets via the auth
  // /balance endpoint (real eth_getBalance).
  async getBalances(): Promise<{ balances: BalanceResult[] }> {
    const { wallets } = await this.getWallets();
    const results = await Promise.allSettled(
      wallets.map((w) =>
        this.client
          .get<{ address: string; balance_wei: string; chain_id: number }>(
            `/balance?address=${w.address}&chain_id=${w.chain_id}`,
          )
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

  // GET /balance?address=&chain_id= -> { address, balance_wei, chain_id }
  async getBalance(walletIdOrAddress: string, chainId?: number): Promise<BalanceResult> {
    // /balance takes an address; if given a wallet id, resolve to its address.
    let address = walletIdOrAddress;
    let cid = chainId ?? 1;
    if (!/^0x[a-fA-F0-9]{40}$/.test(address)) {
      const { wallets } = await this.getWallets();
      const w = wallets.find((x) => x.id === walletIdOrAddress);
      if (w) {
        address = w.address;
        cid = w.chain_id;
      }
    }
    const { data } = await this.client.get<{ address: string; balance_wei: string; chain_id: number }>(
      `/balance?address=${address}&chain_id=${cid}`,
    );
    return {
      wallet_id: walletIdOrAddress,
      chain_id: data.chain_id,
      symbol: symbolFor(data.chain_id),
      address: data.address,
      balance_wei: data.balance_wei,
      balance_f: weiToFloat(data.balance_wei),
      usd_value: 0,
    };
  }

  // GET /tokens?address=&chain_id= -> ERC-20 token balances (real eth_call).
  async getTokenBalances(address: string, chainId: number): Promise<{ tokens: unknown[] }> {
    const { data } = await this.client.get(`/tokens?address=${address}&chain_id=${chainId}`);
    return data;
  }

  // GET /nfts?address=&chain_id= -> ERC-721 NFTs (real on-chain reads).
  async getNFTs(address: string, chainId: number): Promise<{ nfts: unknown[] }> {
    const { data } = await this.client.get(`/nfts?address=${address}&chain_id=${chainId}`);
    return data;
  }

  // POST /nft/transfer { wallet_id, password, to, token_id, contract_address, chain_id }
  async transferNFT(params: {
    walletId: string;
    password: string;
    to: string;
    tokenId: string;
    contractAddress: string;
    chainId: number;
  }): Promise<{ transaction_hash: string; status: string }> {
    const { data } = await this.client.post('/nft/transfer', {
      wallet_id: params.walletId,
      password: params.password,
      to: params.to,
      token_id: params.tokenId,
      contract_address: params.contractAddress,
      chain_id: params.chainId,
    });
    return data;
  }

  // ==================== Transactions & Signing ====================
  // GET /transactions?address=&chain_id= -> { transactions: TransactionRecord[] }
  async getTransactions(params?: { walletId?: string; network?: string; token?: string }): Promise<{
    transactions: TransactionRecord[];
  }> {
    const query: Record<string, string> = {};
    let address = '';
    if (params?.walletId) {
      const { wallets } = await this.getWallets();
      const w = wallets.find((x) => x.id === params.walletId);
      if (w) address = w.address;
    } else if (this.token) {
      const { wallets } = await this.getWallets();
      if (wallets.length > 0) address = wallets[0].address;
    }
    if (address) query.address = address;
    query.chain_id = params?.network ? String(chainIdFor(params.network)) : '1';
    const { data } = await this.client.get<{ transactions: TransactionRecord[] }>(
      `/transactions?${new URLSearchParams(query).toString()}`,
    );
    let txs = data.transactions || [];
    if (params?.token) {
      const tok = params.token.toUpperCase();
      txs = txs.filter((t) => (t.token || '').toUpperCase() === tok || (!t.token && tok === 'ETH'));
    }
    return { transactions: txs };
  }

  // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }
  async getTransactionReceipt(txHash: string, chainId: number): Promise<{
    status: string;
    block_number?: number;
    confirmations?: number;
  }> {
    const { data } = await this.client.get(`/transactions/${encodeURIComponent(txHash)}`, {
      params: { chain_id: chainId },
    });
    return data;
  }

  // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }
  async getTransactionStatus(txHash: string, chainId: number): Promise<{
    status: string;
    block_number?: number;
    confirmations?: number;
  }> {
    return this.getTransactionReceipt(txHash, chainId);
  }

  // POST /send { wallet_id, password, to, value, chain_id, gas_limit, data }
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

  // POST /auto-send (same body as /send, optional ?master_wallet_id=). Returns
  // the send response PLUS { auto_approved, auto_approval_reason }.
  async autoSendTransaction(params: {
    walletId: string;
    password: string;
    to: string;
    value: string;
    chainId?: number;
    gasLimit?: number;
    data?: string;
    masterWalletId?: string;
  }): Promise<{
    transaction_hash: string;
    status: string;
    from: string;
    auto_approved: boolean;
    auto_approval_reason: string;
  }> {
    const query = params.masterWalletId ? { master_wallet_id: params.masterWalletId } : undefined;
    const { data } = await this.client.post(
      '/auto-send',
      {
        wallet_id: params.walletId,
        password: params.password,
        to: params.to,
        value: params.value,
        chain_id: params.chainId ?? 1,
        gas_limit: params.gasLimit,
        data: params.data,
      },
      { params: query },
    );
    return data;
  }

  // POST /sign { wallet_id, password, message } -> { signature, address } (real EIP-191).
  async signMessage(params: { walletId: string; password: string; message: string }): Promise<{
    signature: string;
    address: string;
  }> {
    const { data } = await this.client.post('/sign', {
      wallet_id: params.walletId,
      password: params.password,
      message: params.message,
    });
    return data;
  }

  // ==================== Gas / Price / Chains / Network ====================
  async getGasPrice(network: string): Promise<{ gas_price: string }> {
    const { data } = await this.client.get(`/gas?chain_id=${chainIdFor(network)}`);
    return data;
  }

  // /price accepts ?symbol= (e.g. "eth") or ?ids= (CoinGecko coin id).
  async getTokenPrice(coin = 'eth'): Promise<{ price: number }> {
    const { data } = await this.client.get(`/price?symbol=${coin}`);
    return data;
  }

  async getPrice(coin = 'eth'): Promise<{ price: number }> {
    return this.getTokenPrice(coin);
  }

  async getChains(): Promise<{ chains: unknown[] }> {
    const { data } = await this.client.get('/chains');
    return data;
  }

  async getNetworks(): Promise<{ chains: unknown[] }> {
    return this.getChains();
  }

  // No dedicated block-height endpoint; block_number is honestly 0 (never
  // fabricated) and connected reflects whether the chain is in the registry.
  async getNetworkStatus(chainId = 1): Promise<{ chain_id: number; block_number: number; connected: boolean }> {
    const data = await this.getChains();
    const chain = (data.chains as Array<{ id?: number }>).find((c) => c.id === Number(chainId));
    return { chain_id: Number(chainId), block_number: 0, connected: !!chain };
  }

  async estimateGas(params: {
    from: string;
    to: string;
    value?: string;
    data?: string;
    chainId?: number;
  }): Promise<{ gas_limit: number }> {
    const { data } = await this.client.post('/gas/estimate', {
      from: params.from,
      to: params.to,
      value: params.value,
      data: params.data,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  // ==================== Swap / Convert / AMM ====================
  // GET /swap/quote (CoinGecko cross-rate, indicative).
  async getSwapQuote(params: {
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.get(
      `/swap/quote?from_token=${params.fromToken}&to_token=${params.toToken}&from_amount=${params.fromAmount}&chain_id=${params.chainId ?? 1}`,
    );
    return data;
  }

  // POST /swap/execute -> on-chain action for /send (no fabricated hash).
  async executeSwap(params: {
    walletId: string;
    password: string;
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/swap/execute', {
      wallet_id: params.walletId,
      password: params.password,
      from_token: params.fromToken,
      to_token: params.toToken,
      from_amount: params.fromAmount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  // Convert is a cross-token conversion; reuses /swap/quote.
  async getConvertQuote(params: {
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId?: number;
  }): Promise<unknown> {
    return this.getSwapQuote(params);
  }

  // GET /amm/quote (real on-chain Uniswap-V2 getAmountsOut).
  async getAmmQuote(params: {
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.get(
      `/amm/quote?from_token=${params.fromToken}&to_token=${params.toToken}&from_amount=${params.fromAmount}&chain_id=${params.chainId ?? 1}`,
    );
    return data;
  }

  // POST /amm/swap (real on-chain swapExactTokensForTokens calldata).
  async ammSwap(params: {
    walletId: string;
    password: string;
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/amm/swap', {
      wallet_id: params.walletId,
      password: params.password,
      from_token: params.fromToken,
      to_token: params.toToken,
      from_amount: params.fromAmount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  // ==================== Staking ====================
  // GET /staking/quote -> { success, assets[], apy, min_stake, lock_period }
  async getStakingQuote(): Promise<unknown> {
    const { data } = await this.client.get('/staking/quote');
    return data;
  }

  async stake(params: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/staking/stake', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async unstake(params: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/staking/unstake', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async claim(params: {
    walletId: string;
    password: string;
    asset: string;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/staking/claim', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  // ==================== Fiat / Card / P2P ====================
  async getFiatProviders(): Promise<unknown> {
    const { data } = await this.client.get('/ramp/providers');
    return data;
  }

  async getFiatQuote(params: {
    providerId: string;
    amount: string;
    fiat: string;
    crypto: string;
    method?: string;
  }): Promise<unknown> {
    const { data } = await this.client.post('/ramp/quote', {
      providerId: params.providerId,
      amount: params.amount,
      fiatCurrency: params.fiat,
      cryptoCurrency: params.crypto,
      paymentMethod: params.method,
    });
    return data;
  }

  async getFiatOfframpQuote(params: {
    providerId: string;
    amount: string;
    fiat: string;
    crypto: string;
  }): Promise<unknown> {
    const { data } = await this.client.post('/ramp/offramp-quote', {
      providerId: params.providerId,
      amount: params.amount,
      fiatCurrency: params.fiat,
      cryptoCurrency: params.crypto,
    });
    return data;
  }

  async getCryptoCardRates(): Promise<unknown> {
    const { data } = await this.client.get('/cards/rates');
    return data;
  }

  async getCryptoCardBalance(cardId: string): Promise<unknown> {
    const { data } = await this.client.get(`/cards/${cardId}/balance`);
    return data;
  }

  async getCardTransactions(cardId: string): Promise<unknown> {
    const { data } = await this.client.get(`/cards/${cardId}/transactions`);
    return data;
  }

  async getP2PAdverts(): Promise<unknown> {
    const { data } = await this.client.get('/p2p/listings');
    return data;
  }

  // ==================== Non-EVM (Solana / Bitcoin / Cosmos) ====================
  // POST /non_evm/address { seed, chain_type, chain_id } -> { address }
  async nonEvmAddress(params: {
    seed: string;
    chainType: string;
    chainId: number;
    path?: string;
  }): Promise<{ address: string }> {
    const { data } = await this.client.post('/non_evm/address', {
      seed: params.seed,
      chain_type: params.chainType,
      chain_id: params.chainId,
      path: params.path,
    });
    return data;
  }

  // POST /non_evm/sign { seed, chain_type, message_hash } -> { signature }
  async nonEvmSign(params: {
    seed: string;
    chainType: string;
    chainId: number;
    messageHash: string;
    path?: string;
  }): Promise<{ signature: string }> {
    const { data } = await this.client.post('/non_evm/sign', {
      seed: params.seed,
      chain_type: params.chainType,
      chain_id: params.chainId,
      message_hash: params.messageHash,
      path: params.path,
    });
    return data;
  }

  // POST /non_evm/send { seed, chain_type, to, value, ... } -> { signature, raw_tx }
  async nonEvmSend(params: {
    seed: string;
    chainType: string;
    chainId: number;
    to: string;
    value: string;
    path?: string;
  }): Promise<{ signature: string; raw_tx?: string; tx_hash?: string }> {
    const { data } = await this.client.post('/non_evm/send', {
      seed: params.seed,
      chain_type: params.chainType,
      chain_id: params.chainId,
      to: params.to,
      value: params.value,
      path: params.path,
    });
    return data;
  }

  // ==================== Address Book ====================
  async getAddressBookContacts(): Promise<{ contacts: unknown[] }> {
    const { data } = await this.client.get('/address-book/contacts');
    return data;
  }

  async addContact(params: { name: string; address: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.post('/address-book/contacts', {
      name: params.name,
      address: params.address,
      chain_id: params.chainId,
    });
    return data;
  }

  async updateContact(id: string, params: { name?: string; address?: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.put(`/address-book/contacts/${id}`, {
      name: params.name,
      address: params.address,
      chain_id: params.chainId,
    });
    return data;
  }

  async deleteContact(id: string): Promise<unknown> {
    const { data } = await this.client.delete(`/address-book/contacts/${id}`);
    return data;
  }

  // ==================== Devices ====================
  async getDevices(): Promise<{ devices: unknown[] }> {
    const { data } = await this.client.get('/devices');
    return data;
  }

  async registerDevice(params: { name: string; deviceType: string }): Promise<unknown> {
    const { data } = await this.client.post('/devices', {
      name: params.name,
      device_type: params.deviceType,
    });
    return data;
  }

  async syncDevice(deviceId: string): Promise<unknown> {
    const { data } = await this.client.post(`/devices/${deviceId}/sync`, {});
    return data;
  }

  async deleteDevice(deviceId: string): Promise<unknown> {
    const { data } = await this.client.delete(`/devices/${deviceId}`);
    return data;
  }

  // ==================== Token Approvals ====================
  async getApprovals(address: string, chainId: number): Promise<{ approvals: unknown[] }> {
    const { data } = await this.client.get(`/approvals?address=${address}&chain_id=${chainId}`);
    return data;
  }

  async revokeApproval(params: { approvalId: string }): Promise<unknown> {
    const { data } = await this.client.delete(`/approvals/${params.approvalId}`);
    return data;
  }

  // ==================== Keystore V3 (Web3 Secret Storage) ====================
  // POST /keystore/export { wallet_id, password } -> { keystore } (real scrypt + AES-CTR + keccak MAC).
  async exportKeystore(params: { walletId: string; password: string }): Promise<{ keystore: string }> {
    const { data } = await this.client.post('/keystore/export', {
      wallet_id: params.walletId,
      password: params.password,
    });
    return data;
  }

  // POST /keystore/import { keystore, password, label } -> { wallet_id, address }
  async importKeystore(params: { keystore: string; password: string; label?: string }): Promise<{ wallet_id: string; address: string }> {
    const { data } = await this.client.post('/keystore/import', {
      keystore: params.keystore,
      password: params.password,
      label: params.label,
    });
    return data;
  }

  // ==================== Encrypted-seed backup (Google Drive) ====================
  // POST /wallets/:id/export-encrypted-seed { password } -> { encrypted_seed, salt, nonce }
  // (real AES-256-GCM; the user uploads this blob to Google Drive themselves
  // via the native Google Picker / Drive API on the client — the backend never
  // receives Drive credentials).
  async exportEncryptedSeed(walletId: string, password: string): Promise<{ encrypted_seed: string; salt: string; nonce: string }> {
    const { data } = await this.client.post(`/wallets/${walletId}/export-encrypted-seed`, {
      password,
    });
    return data;
  }

  // POST /wallets/import-encrypted-seed { encrypted_seed, password, label } -> { wallet_id, address }
  async importEncryptedSeed(params: {
    encryptedSeed: string;
    password: string;
    label?: string;
  }): Promise<{ wallet_id: string; address: string }> {
    const { data } = await this.client.post('/wallets/import-encrypted-seed', {
      encrypted_seed: params.encryptedSeed,
      password: params.password,
      label: params.label,
    });
    return data;
  }

  // ==================== Security scan (scam URL/address check) ====================
  // GET /security/check-url?url= -> { safe, reason }
  async checkUrl(url: string): Promise<{ safe: boolean; reason?: string }> {
    const { data } = await this.client.get(`/security/check-url?url=${encodeURIComponent(url)}`);
    return data;
  }

  // GET /security/check-address?address= -> { safe, reason }
  async checkAddress(address: string): Promise<{ safe: boolean; reason?: string }> {
    const { data } = await this.client.get(`/security/check-address?address=${encodeURIComponent(address)}`);
    return data;
  }

  // POST /security/scan { target } -> { safe, threats[] }
  async securityScan(target: string): Promise<{ safe: boolean; threats: unknown[] }> {
    const { data } = await this.client.post('/security/scan', { target });
    return data;
  }

  // ==================== DeFi services (proxied through :8443) ====================
  // Lending (go/lending_service, /api/v1/lending/*)
  async getLendingMarkets(): Promise<unknown> {
    const { data } = await this.client.get('/lending/markets');
    return data;
  }

  async getLendingPositions(): Promise<unknown> {
    const { data } = await this.client.get('/lending/positions');
    return data;
  }

  async lendingSupply(params: { walletId: string; password: string; asset: string; amount: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.post('/lending/supply', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async lendingBorrow(params: { walletId: string; password: string; asset: string; amount: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.post('/lending/borrow', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async lendingWithdraw(params: { walletId: string; password: string; asset: string; amount: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.post('/lending/withdraw', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async lendingRepay(params: { walletId: string; password: string; asset: string; amount: string; chainId?: number }): Promise<unknown> {
    const { data } = await this.client.post('/lending/repay', {
      wallet_id: params.walletId,
      password: params.password,
      asset: params.asset,
      amount: params.amount,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  // Copy trading (go/copy_trading_service, /api/v1/copytrading/*)
  async getCopyTraders(): Promise<unknown> {
    const { data } = await this.client.get('/copytrading/traders');
    return data;
  }

  async followTrader(params: { traderId: string; allocation?: string }): Promise<unknown> {
    const { data } = await this.client.post('/copytrading/follow', {
      trader_id: params.traderId,
      allocation: params.allocation,
    });
    return data;
  }

  async stopCopyTrader(copierId: string): Promise<unknown> {
    const { data } = await this.client.post(`/copytrading/copiers/${copierId}/stop`, {});
    return data;
  }

  async getCopySignals(): Promise<unknown> {
    const { data } = await this.client.get('/copytrading/signals');
    return data;
  }

  // Governance / DAO (go/governance_service, /api/v1/governance/* + wallet_api /dao/*)
  async getDaoProposals(): Promise<{ proposals: unknown[] }> {
    const { data } = await this.client.get('/dao/proposals');
    return data;
  }

  async createDaoProposal(params: { title: string; description: string }): Promise<unknown> {
    const { data } = await this.client.post('/dao/proposals', {
      title: params.title,
      description: params.description,
    });
    return data;
  }

  async voteDaoProposal(params: { proposalId: string; support: boolean }): Promise<unknown> {
    const { data } = await this.client.post(`/dao/proposals/${params.proposalId}/vote`, {
      support: params.support,
    });
    return data;
  }

  async getDaoDelegates(): Promise<{ delegates: unknown[] }> {
    const { data } = await this.client.get('/dao/delegates');
    return data;
  }

  // Perpetual positions (wallet_api /perpetual/*)
  async getPerpetualPositions(): Promise<{ positions: unknown[] }> {
    const { data } = await this.client.get('/perpetual/positions');
    return data;
  }

  async createPerpetualPosition(params: {
    pair: string;
    side: string;
    size: string;
    leverage: number;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/perpetual/positions', {
      pair: params.pair,
      side: params.side,
      size: params.size,
      leverage: params.leverage,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async closePerpetualPosition(positionId: string): Promise<unknown> {
    const { data } = await this.client.post(`/perpetual/positions/${positionId}/close`, {});
    return data;
  }

  // Margin positions (wallet_api /margin/*)
  async getMarginPositions(): Promise<{ positions: unknown[] }> {
    const { data } = await this.client.get('/margin/positions');
    return data;
  }

  async createMarginPosition(params: {
    pair: string;
    side: string;
    size: string;
    leverage: number;
    chainId?: number;
  }): Promise<unknown> {
    const { data } = await this.client.post('/margin/positions', {
      pair: params.pair,
      side: params.side,
      size: params.size,
      leverage: params.leverage,
      chain_id: params.chainId ?? 1,
    });
    return data;
  }

  async closeMarginPosition(positionId: string): Promise<unknown> {
    const { data } = await this.client.post(`/margin/positions/${positionId}/close`, {});
    return data;
  }

  // Prediction markets (go/prediction_service, /api/v1/prediction/*)
  async getPredictionMarkets(): Promise<{ markets: unknown[] }> {
    const { data } = await this.client.get('/prediction/markets');
    return data;
  }

  async placePredictionBet(params: { marketId: string; side: string; amount: string }): Promise<unknown> {
    const { data } = await this.client.post(`/prediction/markets/${params.marketId}/bet`, {
      side: params.side,
      amount: params.amount,
    });
    return data;
  }

  // Launchpool (wallet_api /launchpool/*)
  async getLaunchpool(): Promise<unknown> {
    const { data } = await this.client.get('/launchpool');
    return data;
  }

  async getLaunchpoolStakes(): Promise<unknown> {
    const { data } = await this.client.get('/launchpool/stakes');
    return data;
  }

  async launchpoolStake(params: { walletId: string; password: string; amount: string }): Promise<unknown> {
    const { data } = await this.client.post('/launchpool/stake', {
      wallet_id: params.walletId,
      password: params.password,
      amount: params.amount,
    });
    return data;
  }

  async launchpoolUnstake(params: { walletId: string; password: string; amount: string }): Promise<unknown> {
    const { data } = await this.client.post('/launchpool/unstake', {
      wallet_id: params.walletId,
      password: params.password,
      amount: params.amount,
    });
    return data;
  }

  // Token sales (wallet_api /token-sales/*)
  async getTokenSales(): Promise<{ sales: unknown[] }> {
    const { data } = await this.client.get('/token-sales');
    return data;
  }

  async participateTokenSale(params: { saleId: string; amount: string }): Promise<unknown> {
    const { data } = await this.client.post(`/token-sales/${params.saleId}/participate`, {
      amount: params.amount,
    });
    return data;
  }

  // ==================== dApps / Chart / DeFi protocols ====================
  async getDapps(): Promise<{ dapps: unknown[] }> {
    const { data } = await this.client.get('/dapps');
    return data;
  }

  async getDappCategories(): Promise<{ categories: unknown[] }> {
    const { data } = await this.client.get('/dapps/categories');
    return data;
  }

  async getChartHistory(params: { token: string; days?: number }): Promise<unknown> {
    const { data } = await this.client.get(
      `/chart/history?token=${encodeURIComponent(params.token)}&days=${params.days ?? 30}`,
    );
    return data;
  }

  async getDefiProtocols(): Promise<{ protocols: unknown[] }> {
    const { data } = await this.client.get('/defi/protocols');
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
