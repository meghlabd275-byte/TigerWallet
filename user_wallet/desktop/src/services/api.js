// Desktop API client — connects to the canonical TigerWallet Go wallet-api
// backend (go/wallet_api, port 8443). Real on-chain RPC, real BIP-39/32/44
// HD derivation, real secp256k1 signing + broadcast, PostgreSQL + Redis.
// No stubs, no fabricated data.

const API_BASE_URL =
  (typeof process !== 'undefined' && process.env && process.env.REACT_APP_API_URL) ||
  'http://localhost:8443/api/v1';

const CHAIN_IDS = {
  ethereum: 1,
  bsc: 56,
  polygon: 137,
  arbitrum: 42161,
  optimism: 10,
  base: 8453,
  avalanche: 43114,
};

function chainIdFor(network) {
  return CHAIN_IDS[network] || (parseInt(network, 10) || 1);
}

let authToken = null;

export function setToken(token) {
  authToken = token;
}

export function getToken() {
  return authToken;
}

async function request(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (authToken) headers.Authorization = `Bearer ${authToken}`;
  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || `Request failed: ${res.status}`);
  }
  return res.json();
}

export const api = {
  // ---- Auth ----
  async login(email, password) {
    const data = await request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    return { token: data.token, user: data.user || { email } };
  },

  async register(email, _username, password) {
    // Canonical /auth/register accepts {email, password} only (see route table).
    const data = await request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    return { user_id: data.user_id, token: data.token };
  },

  // ---- Wallets ----
  async getWallets() {
    return request('/wallets');
  },

  async createWallet({ label, password, chainId, mnemonic, accountIndex, entropyBits }) {
    return request('/wallets', {
      method: 'POST',
      body: JSON.stringify({
        label,
        password,
        chain_id: chainId,
        mnemonic,
        account_index: accountIndex,
        entropy_bits: entropyBits,
      }),
    });
  },

  // ---- Balances ----
  // Aggregated balances via the auth /balance endpoint (real eth_getBalance).
  async getBalances() {
    const { wallets } = await this.getWallets();
    const results = await Promise.allSettled(
      wallets.map((w) =>
        request(`/balance?address=${w.address}&chain_id=${w.chain_id}`),
      ),
    );
    return {
      balances: results
        .filter((r) => r.status === 'fulfilled')
        .map((r) => r.value),
    };
  },

  async getBalance(address, chainId) {
    return request(`/balance?address=${address}&chain_id=${chainId}`);
  },

  // ---- Tokens ----
  async getTokenBalances(address, chainId) {
    return request(`/tokens?address=${address}&chain_id=${chainId}`);
  },

  // ---- Transactions ----
  async getTransactions({ network, address } = {}) {
    const query = {};
    if (address) query.address = address;
    else if (authToken) {
      const { wallets } = await this.getWallets();
      if (wallets.length > 0) query.address = wallets[0].address;
    }
    query.chain_id = network ? chainIdFor(network) : 1;
    const qs = new URLSearchParams(query).toString();
    return request(`/transactions?${qs}`);
  },

  // ---- Send / Sign (real on-chain) ----
  async sendTransaction({ walletId, password, to, value, chainId, gasLimit, data }) {
    return request('/send', {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        to,
        value,
        chain_id: chainId ?? 1,
        gas_limit: gasLimit,
        data,
      }),
    });
  },

  async signMessage({ walletId, password, message }) {
    return request('/sign', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, message }),
    });
  },

  // ---- Price / Gas / Chains ----
  // wallet_api /price accepts ?symbol= (e.g. "eth") or ?ids= (CoinGecko coin id).
  async getPrice(coin = 'eth') {
    return request(`/price?symbol=${coin}`);
  },

  async getTokenPrice(coin = 'eth') {
    return request(`/price?symbol=${coin}`);
  },

  async getGasPrice(network) {
    return request(`/gas?chain_id=${chainIdFor(network)}`);
  },

  async getNetworks() {
    return request('/chains');
  },

  async getNetworkStatus(chainId = 1) {
    // The backend exposes the chains registry but no dedicated block-height
    // endpoint; block_number is honestly 0 (never fabricated) and connected
    // reflects whether the chain is present in the registry.
    const data = await request('/chains');
    const chain = (data.chains || []).find((c) => c.id === Number(chainId));
    return { chain_id: Number(chainId), block_number: 0, connected: !!chain };
  },

  async getNFTs(address, chainId) {
    return request(`/nfts?address=${address}&chain_id=${chainId}`);
  },

  async getSwapQuote({ fromToken, toToken, fromAmount, chainId = 1 }) {
    return request(`/swap/quote?from_token=${fromToken}&to_token=${toToken}&from_amount=${fromAmount}&chain_id=${chainId}`);
  },

  async getStakingQuote(_asset) {
    // Backend returns { success, assets[], apy, min_stake, lock_period } and
    // ignores ?asset=; the full supported-asset list is returned.
    return request('/staking/quote');
  },

  // ---- Auxiliary DeFi (fiat ramp, crypto card, P2P, convert) ----
  async getFiatProviders() { return request('/ramp/providers'); },
  async getFiatQuote(providerId, amount, fiat, crypto, method) {
    return request('/ramp/quote', { method: 'POST', body: JSON.stringify({ providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto, paymentMethod: method }) });
  },
  async getFiatOfframpQuote(providerId, amount, fiat, crypto) {
    return request('/ramp/offramp-quote', { method: 'POST', body: JSON.stringify({ providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto }) });
  },
  async getCryptoCardRates() { return request('/cards/rates'); },
  async getP2PListings() { return request('/p2p/listings'); },
  async getConvertQuote({ fromToken, toToken, fromAmount, chainId = 1 }) {
    return request(`/swap/quote?from_token=${fromToken}&to_token=${toToken}&from_amount=${fromAmount}&chain_id=${chainId}`);
  },

  async logout() {
    authToken = null;
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem('tigerwallet-token');
    }
  },
};

// parsePaymentUri — decodes a scanned QR string (bare 0x address, ethereum:
// URI, or EIP-681 payment URI) into an address + optional amount. Returns
// null when no address can be extracted (fail-closed — never a guessed value).
export function parsePaymentUri(input) {
  const s = (input || '').trim();
  if (!s) return null;
  if (/^0x[a-fA-F0-9]{40}$/.test(s)) return { address: s };
  let body;
  if (s.startsWith('ethereum:')) body = s.slice('ethereum:'.length);
  else return null;
  const qIdx = body.indexOf('?');
  const target = qIdx >= 0 ? body.slice(0, qIdx) : body;
  const query = qIdx >= 0 ? body.slice(qIdx + 1) : '';
  let address, tokenAddress = null;
  if (target.includes('/')) {
    const [addr, func] = target.split('/');
    address = addr;
    if (func.startsWith('transfer')) tokenAddress = '';
  } else {
    address = target;
  }
  if (!/^0x[a-fA-F0-9]{40}$/.test(address)) return null;
  let amount, chainId;
  query.split('&').forEach((pair) => {
    const [k, v] = pair.split('=');
    if (k === 'value') amount = v;
    else if (k === 'chainId') chainId = Number(v);
    else if (k === 'address' && tokenAddress !== null) tokenAddress = v;
  });
  return { address, amount, chainId, tokenAddress: tokenAddress || undefined };
}

export default api;
