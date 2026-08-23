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

// Decode the JWT payload of a locally-held token to derive a user identity.
// Mirrors web api.getProfile: no network call, purely a best-effort decode of
// the session token that the auth context already holds. Falls back to null
// when there is no token or it is not a parseable JWT.
function profileFromToken(token) {
  if (!token || typeof token !== 'string' || !token.includes('.')) return null;
  try {
    const payload = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = payload.padEnd(payload.length + (4 - (payload.length % 4)) % 4, '=');
    const json = JSON.parse(atob(padded));
    return {
      id: json.sub || json.user_id || json.uid || null,
      email: json.email || json.mail || '',
      username: json.username || json.name || json.email || '',
    };
  } catch {
    return null;
  }
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

  // Mirror of web api.getProfile: decode the locally-stored JWT to obtain the
  // user identity (sub/email/username) without an extra network round-trip. The
  // desktop /auth/register endpoint does not always issue a JWT, so this is a
  // safe best-effort identity extractor used by OnboardingContext/Settings.
  getProfile() {
    const token = authToken;
    return Promise.resolve(profileFromToken(token));
  },

  // expose the module-level token accessors on the api object so callers
  // (e.g. OnboardingContext) can use api.setToken / api.getToken like the web
  // reference. They delegate to the same module-level authToken used by request().
  setToken,
  getToken,
  // chain id helper used by the onboarding flow to map network names → ids.
  chainIdFor,

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

  // Typed create path used by the no-registration onboarding flow which already
  // knows the label / password / chain id (and an optional mnemonic for import).
  // Posts to the REAL /wallets endpoint — no stubs.
  async createWalletTyped({ label, password, chainId, mnemonic, passphrase }) {
    return request('/wallets', {
      method: 'POST',
      body: JSON.stringify({
        label,
        password,
        chain_id: chainId,
        mnemonic,
        passphrase,
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
  // WL GET /wallets/:id/transactions -> { transactions: TransactionRecord[] }
  // Mirrors the web reference: per-wallet list, client-side filter by network/token.
  async getTransactions(params = {}) {
    const { walletId, network, token } = params;
    if (!walletId) {
      // No wallet selected — return an empty list honestly (no fabricated txs).
      return { transactions: [] };
    }
    const data = await request(`/wallets/${encodeURIComponent(walletId)}/transactions`);
    let txs = data.transactions || [];
    if (network) {
      const cid = chainIdFor(network);
      txs = txs.filter((t) => t.chain_id === cid);
    }
    if (token) {
      const tok = token.toUpperCase();
      txs = txs.filter((t) => (t.token || '').toUpperCase() === tok || (!t.token && tok === 'ETH'));
    }
    return { transactions: txs };
  },

  // ---- Send / Sign (real on-chain) ----
  // WL POST /wallets/:id/send -> { transaction_hash, status, from }
  async sendTransaction({ walletId, password, to, value, chainId, gasLimit, data: txData, maxFeeGwei, maxPriorityGwei }) {
    const body = {
      to,
      amount: value,
      password,
      gas_limit: gasLimit,
      data: txData,
      chain_id: chainId,
    };
    // Optional EIP-1559 fee overrides (gwei strings) — only sent when set.
    if (maxFeeGwei) body.max_fee_gwei = maxFeeGwei;
    if (maxPriorityGwei) body.max_priority_gwei = maxPriorityGwei;
    return request(`/wallets/${encodeURIComponent(walletId)}/send`, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  // WL POST /wallets/:id/auto-send -> { transaction_hash, auto_approved, auto_approval_reason }
  // The backend auto-signs + auto-approves (fast path, license alive + non-treasury tx);
  // returns auto_approved=false + reason when two-party co-sign is required.
  async autoSendTransaction({ walletId, password, to, value, chainId, gasLimit, data: txData, unlockToken, maxFeeGwei, maxPriorityGwei }) {
    const body = {
      to,
      amount: value,
      password,
      gas_limit: gasLimit,
      data: txData,
      chain_id: chainId,
      unlock_token: unlockToken,
    };
    // Optional EIP-1559 fee overrides (gwei strings) — only sent when set.
    if (maxFeeGwei) body.max_fee_gwei = maxFeeGwei;
    if (maxPriorityGwei) body.max_priority_gwei = maxPriorityGwei;
    return request(`/wallets/${encodeURIComponent(walletId)}/auto-send`, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  // ---- Transaction simulation (pre-sign dry-run) ----
  // WL POST /simulate { chain_id, from, to, value?, data? } -> { success,
  // gas_estimate, will_revert, revert_reason?, estimated_cost_wei?, ... }.
  // Lets the send form preview success/gas/revert BEFORE signing.
  async simulateTransaction({ chainId, from, to, value, data: txData }) {
    return request('/simulate', {
      method: 'POST',
      body: JSON.stringify({
        chain_id: chainId ?? 1,
        from,
        to,
        value,
        data: txData,
      }),
    });
  },

  // ---- ENS (real on-chain lookups via the backend) ----
  // WL GET /ens/resolve?name=alice.eth -> { name, address } (forward resolution).
  async resolveENS(name) {
    return request(`/ens/resolve?name=${encodeURIComponent(name)}`);
  },

  // WL GET /ens/lookup?address=0x... -> { address, name } (reverse resolution).
  async lookupENS(address) {
    return request(`/ens/lookup?address=${encodeURIComponent(address)}`);
  },

  async signMessage({ walletId, password, message }) {
    return request(`/wallets/${encodeURIComponent(walletId)}/sign`, {
      method: 'POST',
      body: JSON.stringify({ message, password }),
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
  // P2P adverts — the backend route is /p2p/adverts (no /p2p/listings route).
  async getP2PListings() { return request('/p2p/adverts'); },
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
