// Desktop API client — connects to the canonical TigerWallet Go wallet-api
// backend (go/wallet_api, port 8443). Real on-chain RPC, real BIP-39/32/44
// HD derivation, real secp256k1 signing + broadcast, PostgreSQL + Redis.
// No stubs, no fabricated data.

const API_BASE_URL =
  (typeof process !== 'undefined' && process.env && process.env.REACT_APP_API_URL) ||
  'http://localhost:8443/api/v1';

const HEALTH_URL = API_BASE_URL.replace(/\/api\/v1\/?$/, '') + '/health';

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
  async sendTransaction({ walletId, password, to, value, chainId, gasLimit, data, unlockToken }) {
    return request('/send', {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        unlock_token: unlockToken,
        to,
        value,
        chain_id: chainId ?? 1,
        gas_limit: gasLimit,
        data,
      }),
    });
  },

  // ---- Guest auth (public, no-auth) ----
  // POST /auth/guest { device_id } -> { user_id, token, guest: true }.
  // Provisions an anonymous guest account so the user can Create/Import a
  // wallet without registering. Mirrors login(): returns { token, user } and
  // leaves token persistence to the caller (AuthContext stores
  // 'userwallet-token' + calls setToken), exactly like login/register.
  async guestAuth(deviceId) {
    const data = await request('/auth/guest', {
      method: 'POST',
      body: JSON.stringify({ device_id: deviceId }),
    });
    return {
      token: data.token,
      user_id: data.user_id,
      guest: data.guest !== undefined ? Boolean(data.guest) : true,
      user: data.user || { id: data.user_id, guest: true },
    };
  },

  // ---- Auto-send (auto-approval-gated send) ----
  // POST /auto-send with the SAME body as /send, plus optional
  // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
  // existing send response PLUS { auto_approved, auto_approval_reason }.
  async autoSendTransaction({ walletId, password, to, value, chainId, gasLimit, data, masterWalletId, unlockToken }) {
    const query = masterWalletId ? `?master_wallet_id=${encodeURIComponent(masterWalletId)}` : '';
    return request(`/auto-send${query}`, {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        unlock_token: unlockToken,
        to,
        value,
        chain_id: chainId ?? 1,
        gas_limit: gasLimit,
        data,
      }),
    });
  },

  // ---- Transaction status (explorer proxy) ----
  // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
  async getTransactionStatus(txHash, chainId) {
    const path = `/transactions/${encodeURIComponent(txHash)}?chain_id=${chainId}`;
    return request(path);
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
    // GET /network-status?chain_id=N — real eth_blockNumber RPC (never 0).
    return request(`/network-status?chain_id=${chainId}`);
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

  // ---- Wallet import ----
  async importWallet({ label, password, mnemonic, chainId, passphrase }) {
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

  // ---- Profile (local JWT decode) ----
  async getProfile() {
    if (!authToken) throw new Error('Not authenticated');
    const payloadB64 = authToken.split('.')[1];
    const payloadJson = JSON.parse(
      decodeURIComponent(
        atob(payloadB64)
          .split('')
          .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join(''),
      ),
    );
    return { id: payloadJson.id, email: payloadJson.email, username: payloadJson.username };
  },

  // ---- Health ----
  async health() {
    const headers = { 'Content-Type': 'application/json' };
    if (authToken) headers.Authorization = `Bearer ${authToken}`;
    const res = await fetch(HEALTH_URL, { headers });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `Request failed: ${res.status}`);
    }
    return res.json();
  },

  // ---- NFT ----
  // getNFTs already exists above.
  async transferNFT({ walletId, password, to, tokenId, contractAddress, chainId }) {
    return request('/nft/transfer', {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        to,
        token_id: tokenId,
        contract_address: contractAddress,
        chain_id: chainId,
      }),
    });
  },

  // ---- Transaction receipt ----
  async getTransactionReceipt(txHash, chainId) {
    return request(`/transactions/${encodeURIComponent(txHash)}?chain_id=${chainId}`);
  },

  // ---- Gas estimate ----
  async estimateGas({ from, to, value, data, chainId }) {
    return request('/gas/estimate', {
      method: 'POST',
      body: JSON.stringify({ from, to, value, data, chain_id: chainId }),
    });
  },

  // ---- Swap execution ----
  async executeSwap({ walletId, password, fromToken, toToken, fromAmount, chainId }) {
    return request('/swap/execute', {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        from_token: fromToken,
        to_token: toToken,
        from_amount: fromAmount,
        chain_id: chainId,
      }),
    });
  },

  // ---- AMM ----
  async getAmmQuote({ fromToken, toToken, fromAmount, chainId }) {
    return request(`/amm/quote?from_token=${encodeURIComponent(fromToken)}&to_token=${encodeURIComponent(toToken)}&from_amount=${encodeURIComponent(fromAmount)}&chain_id=${encodeURIComponent(chainId)}`);
  },

  async ammSwap({ walletId, password, fromToken, toToken, fromAmount, chainId }) {
    return request('/amm/swap', {
      method: 'POST',
      body: JSON.stringify({
        wallet_id: walletId,
        password,
        from_token: fromToken,
        to_token: toToken,
        from_amount: fromAmount,
        chain_id: chainId,
      }),
    });
  },

  // ---- Staking ----
  async stake({ walletId, password, asset, amount, chainId }) {
    return request('/staking/stake', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  async unstake({ walletId, password, asset, amount, chainId }) {
    return request('/staking/unstake', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  async claim({ walletId, password, asset, chainId }) {
    return request('/staking/claim', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, chain_id: chainId }),
    });
  },

  // ---- Crypto card ----
  async getCryptoCardBalance(cardId) {
    return request(`/cards/${encodeURIComponent(cardId)}/balance`);
  },

  async getCardTransactions(cardId) {
    return request(`/cards/${encodeURIComponent(cardId)}/transactions`);
  },

  // ---- P2P alias ----
  // GET /p2p/adverts -> marketplace adverts (canonical endpoint).
  async getP2PAdverts() {
    return request('/p2p/adverts');
  },

  // ---- Non-EVM ----
  async nonEvmAddress({ seed, chainType, chainId, path }) {
    return request('/non_evm/address', {
      method: 'POST',
      body: JSON.stringify({ seed, chain_type: chainType, chain_id: chainId, path }),
    });
  },

  async nonEvmSign({ seed, chainType, chainId, messageHash, path }) {
    return request('/non_evm/sign', {
      method: 'POST',
      body: JSON.stringify({ seed, chain_type: chainType, chain_id: chainId, message_hash: messageHash, path }),
    });
  },

  async nonEvmSend({ seed, chainType, chainId, to, value, path }) {
    return request('/non_evm/send', {
      method: 'POST',
      body: JSON.stringify({ seed, chain_type: chainType, chain_id: chainId, to, value, path }),
    });
  },

  // ---- Address book ----
  async getAddressBookContacts() {
    return request('/address-book/contacts');
  },

  async addContact({ name, address, chainId }) {
    return request('/address-book/contacts', {
      method: 'POST',
      body: JSON.stringify({ name, address, chain_id: chainId }),
    });
  },

  async updateContact(id, { name, address, chainId }) {
    return request(`/address-book/contacts/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify({ name, address, chain_id: chainId }),
    });
  },

  async deleteContact(id) {
    return request(`/address-book/contacts/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  // ---- Devices ----
  async getDevices() {
    return request('/devices');
  },

  async registerDevice({ name, deviceType }) {
    return request('/devices', {
      method: 'POST',
      body: JSON.stringify({ name, device_type: deviceType }),
    });
  },

  async syncDevice(deviceId) {
    return request(`/devices/${encodeURIComponent(deviceId)}/sync`, { method: 'POST' });
  },

  async deleteDevice(deviceId) {
    return request(`/devices/${encodeURIComponent(deviceId)}`, { method: 'DELETE' });
  },

  // ---- Approvals ----
  async getApprovals(address, chainId) {
    return request(`/approvals?address=${encodeURIComponent(address)}&chain_id=${encodeURIComponent(chainId)}`);
  },

  async revokeApproval({ approvalId }) {
    return request(`/approvals/${encodeURIComponent(approvalId)}`, { method: 'DELETE' });
  },

  // ---- Keystore ----
  async exportKeystore({ walletId, password }) {
    return request('/keystore/export', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password }),
    });
  },

  async importKeystore({ keystore, password, label }) {
    return request('/keystore/import', {
      method: 'POST',
      body: JSON.stringify({ keystore, password, label }),
    });
  },

  // ---- Encrypted seed ----
  async exportEncryptedSeed(walletId, password) {
    return request(`/wallets/${encodeURIComponent(walletId)}/export-encrypted-seed`, {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },

  async importEncryptedSeed({ encryptedSeed, password, label }) {
    return request('/wallets/import-encrypted-seed', {
      method: 'POST',
      body: JSON.stringify({ encrypted_seed: encryptedSeed, password, label }),
    });
  },

  // ---- Security ----
  async checkUrl(url) {
    return request(`/security/check-url?url=${encodeURIComponent(url)}`);
  },

  async checkAddress(address) {
    return request(`/security/check-address?address=${encodeURIComponent(address)}`);
  },

  async securityScan(target) {
    return request('/security/scan', {
      method: 'POST',
      body: JSON.stringify({ target }),
    });
  },

  // ---- Lending ----
  async getLendingMarkets() {
    return request('/lending/markets');
  },

  async getLendingPositions() {
    return request('/lending/positions');
  },

  async lendingSupply({ walletId, password, asset, amount, chainId }) {
    return request('/lending/supply', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  async lendingBorrow({ walletId, password, asset, amount, chainId }) {
    return request('/lending/borrow', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  async lendingWithdraw({ walletId, password, asset, amount, chainId }) {
    return request('/lending/withdraw', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  async lendingRepay({ walletId, password, asset, amount, chainId }) {
    return request('/lending/repay', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, asset, amount, chain_id: chainId }),
    });
  },

  // ---- Copy trading ----
  async getCopyTraders() {
    return request('/copytrading/traders');
  },

  async followTrader({ traderId, allocation }) {
    return request('/copytrading/follow', {
      method: 'POST',
      body: JSON.stringify({ trader_id: traderId, allocation }),
    });
  },

  async stopCopyTrader(copierId) {
    return request(`/copytrading/copiers/${encodeURIComponent(copierId)}/stop`, { method: 'POST' });
  },

  async getCopySignals() {
    return request('/copytrading/signals');
  },

  // ---- DAO ----
  async getDaoProposals() {
    return request('/dao/proposals');
  },

  async createDaoProposal({ title, description }) {
    return request('/dao/proposals', {
      method: 'POST',
      body: JSON.stringify({ title, description }),
    });
  },

  async voteDaoProposal({ proposalId, support }) {
    return request(`/dao/proposals/${encodeURIComponent(proposalId)}/vote`, {
      method: 'POST',
      body: JSON.stringify({ support }),
    });
  },

  async getDaoDelegates() {
    return request('/dao/delegates');
  },

  // ---- Perpetual ----
  async getPerpetualPositions() {
    return request('/perpetual/positions');
  },

  async createPerpetualPosition({ pair, side, size, leverage, chainId }) {
    return request('/perpetual/positions', {
      method: 'POST',
      body: JSON.stringify({ pair, side, size, leverage, chain_id: chainId }),
    });
  },

  async closePerpetualPosition(positionId) {
    return request(`/perpetual/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST' });
  },

  // ---- Margin ----
  async getMarginPositions() {
    return request('/margin/positions');
  },

  async createMarginPosition({ pair, side, size, leverage, chainId }) {
    return request('/margin/positions', {
      method: 'POST',
      body: JSON.stringify({ pair, side, size, leverage, chain_id: chainId }),
    });
  },

  async closeMarginPosition(positionId) {
    return request(`/margin/positions/${encodeURIComponent(positionId)}/close`, { method: 'POST' });
  },

  // ---- Prediction markets ----
  async getPredictionMarkets() {
    return request('/prediction/markets');
  },

  async placePredictionBet({ marketId, side, amount }) {
    return request(`/prediction/markets/${encodeURIComponent(marketId)}/bet`, {
      method: 'POST',
      body: JSON.stringify({ side, amount }),
    });
  },

  // ---- Launchpool ----
  async getLaunchpool() {
    return request('/launchpool');
  },

  async getLaunchpoolStakes() {
    return request('/launchpool/stakes');
  },

  async launchpoolStake({ walletId, password, amount }) {
    return request('/launchpool/stake', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, amount }),
    });
  },

  async launchpoolUnstake({ walletId, password, amount }) {
    return request('/launchpool/unstake', {
      method: 'POST',
      body: JSON.stringify({ wallet_id: walletId, password, amount }),
    });
  },

  // ---- Token sales ----
  async getTokenSales() {
    return request('/token-sales');
  },

  async participateTokenSale({ saleId, amount }) {
    return request(`/token-sales/${encodeURIComponent(saleId)}/participate`, {
      method: 'POST',
      body: JSON.stringify({ amount }),
    });
  },

  // ---- Dapps ----
  async getDapps() {
    return request('/dapps');
  },

  async getDappCategories() {
    return request('/dapps/categories');
  },

  // ---- Chart history ----
  async getChartHistory({ token, days }) {
    return request(`/chart/history?token=${encodeURIComponent(token)}&days=${encodeURIComponent(days)}`);
  },

  // ---- DeFi protocols ----
  async getDefiProtocols() {
    return request('/defi/protocols');
  },

  // ---- Token registry + trading terminal (public) ----
  async getTokenRegistry(chainId) {
    return request(chainId ? `/tokens/registry?chain_id=${chainId}` : '/tokens/registry');
  },

  async getTerminalKline(symbol, days = 1) {
    return request(`/terminal/kline/${encodeURIComponent(symbol)}?days=${days}`);
  },

  async getTerminalTicker(symbol) {
    return request(`/terminal/ticker/${encodeURIComponent(symbol)}`);
  },

  // ---- Passkey wallet creation ----
  // POST /passkey/wallet -> 201 { wallet_id, label, chain_id, address,
  // derivation_path, mnemonic, unlock_key, unlock_token }.
  async passkeyCreateWallet(params) {
    return request('/passkey/wallet', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  // ---- Wallet lock/unlock ----
  // POST /wallets/:id/lock { passcode?, passkey_credential_id?, passkey_public_key? }
  // -> 200 { status, has_passcode, has_passkey }.
  async setupLock(walletId, params) {
    return request(`/wallets/${walletId}/lock`, {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  // POST /wallets/:id/unlock { passcode?, password?, passkey_assertion?,
  // passkey_auth_data?, passkey_client_data?, unwrapped_unlock_key? }
  // -> 200 { unlock_token, expires_in }.
  async unlockWallet(walletId, params) {
    return request(`/wallets/${walletId}/unlock`, {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  // ---- KYC ----
  // GET /kyc/status?user_id= -> proxied KYC status.
  async getKycStatus(userId) {
    return request(`/kyc/status${userId ? '?user_id=' + encodeURIComponent(userId) : ''}`);
  },

  // POST /kyc/register (JSON body).
  async registerKyc(body) {
    return request('/kyc/register', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  // POST /kyc/submit (JSON body).
  async submitKyc(body) {
    return request('/kyc/submit', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  // POST /kyc/document (multipart) — raw fetch: NO Content-Type so the
  // browser sets the multipart boundary; only auth header injected.
  async submitKycDocument(formData) {
    const res = await fetch(`${API_BASE_URL}/kyc/document`, {
      method: 'POST',
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
      body: formData,
    });
    return res.json();
  },

  // GET /kyc/session/:id -> KYC session details.
  async getKycSession(sessionId) {
    return request(`/kyc/session/${sessionId}`);
  },

  // ---- P2P orders ----
  // POST /p2p/orders (JSON body) — KYC-gated; backend returns 403
  // { kyc_required: true } when KYC is incomplete.
  async createP2POrder(body) {
    return request('/p2p/orders', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  // ---- Bridge (proxied bridge_service :8007) ----
  async getBridges() {
    return request('/bridge/routes');
  },
  async getBridgeQuote(params) {
    return request('/bridge/quote', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },
  async initiateBridgeTransfer(body) {
    return request('/bridge/transfer', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },
  async getBridgeTxStatus(txId) {
    return request(`/bridge/tx/${txId}`);
  },
  async getBridgeHistory() {
    return request('/bridge/history');
  },

  // ---- dApp browser / WalletConnect (proxied dapp_browser :8083) ----
  async getDappPairings() {
    return request('/dapp/pairings');
  },
  async createDappPairing(body) {
    return request('/dapp/pairings', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },
  async approveDappPairing(topic) {
    return request(`/dapp/pairings/${topic}/approve`, { method: 'POST', body: '{}' });
  },
  async rejectDappPairing(topic) {
    return request(`/dapp/pairings/${topic}/reject`, { method: 'POST', body: '{}' });
  },
  async getDappSessions() {
    return request('/dapp/sessions');
  },
  async sendDappRequest(topic, body) {
    return request(`/dapp/sessions/${topic}/request`, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },
  async getDappRequests(topic) {
    return request(`/dapp/sessions/${topic}/request`);
  },
  async respondToDappRequest(topic, requestId, body) {
    return request(`/dapp/sessions/${topic}/request/${requestId}/respond`, {
      method: 'POST',
      body: JSON.stringify(body),
    });
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
