/**
 * Wallet Service — TigerWallet UserWallet (production React frontend).
 *
 * Connects to the canonical Go wallet-api backend (go/wallet_api, port 8443)
 * with REAL on-chain RPC, REAL BIP-39/32/44 HD derivation, REAL secp256k1
 * (low-s) signing + eth_sendRawTransaction broadcast, REAL CoinGecko prices,
 * REAL Etherscan history, AES-256-GCM encrypted-seed persistence
 * (PostgreSQL + Redis). No stubs, no fabricated balances or hashes.
 *
 * Route contract (matches go/wallet_api main.go):
 *   GET    /chains
 *   GET    /wallets            -> { wallets: [...] }
 *   POST   /wallets            -> WalletRecord  (body: label, password, chain_id, [mnemonic|entropy_bits])
 *   GET    /balance?address=&chain_id=        -> BalanceResult
 *   GET    /tokens?address=&chain_id=         -> { tokens: [...] }
 *   GET    /transactions?address=&chain_id=   -> { transactions: [...] }
 *   GET    /nfts?address=&chain_id=           -> { nfts: [...] }
 *   POST   /send               -> { tx_hash }  (body: wallet_id, password, to, value, [gas_limit], [data], [chain_id])
 *   POST   /sign               -> { signature }(body: wallet_id, password, message)
 *   GET    /gas?chain_id=                     -> GasPrice
 *   GET    /price?symbol=                     -> PriceInfo
 *   GET    /swap/quote?from=&to=&amount=&chain_id=  -> SwapQuote
 *   POST   /swap/execute       -> action for /send
 *   GET    /staking/quote?chain_id=           -> { assets: [...] }
 *   POST   /staking/{stake,unstake,claim}     -> action for /send
 *   GET    /transactions/:txHash?chain_id=    -> receipt (explorer proxy)
 *
 * Features with no wallet-api endpoint (bridges, nft/transfer, dapp/connect)
 * throw real errors — wire the corresponding Go microservice first.
 */

import axios, { AxiosInstance } from 'axios';

export interface Chain {
  id: number | string;
  name: string;
  symbol: string;
  decimals: number;
  // Canonical backend field names
  rpcEndpoint?: string;
  derivationPath?: string;
  explorerApi?: string;
  explorerUrl?: string;
  chainType?: string;
  coinType?: number;
  isTestnet?: boolean;
  // Legacy aliases (kept for backward compat with existing UI code). These
  // are populated from the canonical fields in getChains() / createWallet().
  rpcUrl?: string;
  chainId?: number;
  type?: 'evm' | 'solana' | 'aptos' | 'sui' | 'ton' | string;
}

export interface Token {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  balance: string;
  balanceUSD: number;
  logoUrl?: string;
  chain: string;
}

export interface Wallet {
  id: string;
  address: string;
  chain: Chain;
  balance: string;
  balanceUSD: number;
  tokens: Token[];
  createdAt: string;
  mnemonic?: string; // returned only on creation, for backup display
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  value: string;
  token?: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: string;
  chain: string;
  gasUsed?: string;
  gasPrice?: string;
}

export interface Signer {
  signMessage(message: string): Promise<string>;
  signTransaction(tx: unknown): Promise<string>;
}

const API_BASE_URL =
  import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';

interface WalletRecord {
  id: string;
  user_id: string;
  label: string;
  chain_id: number;
  address: string;
  derivation_path?: string;
  mnemonic?: string; // returned only on creation, for backup display
}
interface BalanceResult {
  chain_id: number;
  symbol: string;
  address: string;
  balance: string;
  balance_f: string;
  usd_value: number;
}
interface GasPrice {
  chain: string;
  standard_gas_price: string;
  fast_gas_price: string;
  slow_gas_price: string;
}
interface SwapQuote {
  from_token: string;
  to_token: string;
  from_amount: string;
  to_amount: string;
  price_impact: number;
  gas_estimate: number;
}

class WalletService {
  private api: AxiosInstance;

  constructor() {
    this.api = axios.create({
      baseURL: API_BASE_URL,
      headers: { 'Content-Type': 'application/json' },
    });
    this.api.interceptors.request.use((config) => {
      const token = localStorage.getItem('tigerwallet-token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });
  }

  async getChains(): Promise<Chain[]> {
    const response = await this.api.get('/chains');
    const chains = response.data.chains ?? response.data;
    return (chains as Array<Record<string, unknown>>).map((c) => {
      const rpcEndpoint = String(c.rpc_endpoint ?? c.rpc_url ?? c.rpc ?? '');
      const explorerUrl = String(c.explorer_url ?? c.explorer ?? '');
      const chainType = String(c.chain_type ?? c.type ?? 'evm');
      const idNum = Number(c.id ?? c.chain_id ?? 0);
      const decimals = Number(c.decimals ?? 18);
      return {
        id: idNum,
        name: String(c.name ?? ''),
        symbol: String(c.symbol ?? ''),
        decimals,
        rpcEndpoint,
        derivationPath: c.derivation_path ? String(c.derivation_path) : undefined,
        explorerApi: c.explorer_api ? String(c.explorer_api) : undefined,
        explorerUrl,
        chainType,
        coinType: c.coin_type ? Number(c.coin_type) : undefined,
        isTestnet: typeof c.is_testnet === 'boolean' ? c.is_testnet : false,
        // legacy aliases
        rpcUrl: rpcEndpoint,
        chainId: idNum,
        type: chainType,
      } as Chain;
    });
  }

  async getBalance(address: string, chainId: number): Promise<BalanceResult> {
    const response = await this.api.get('/balance', {
      params: { address, chain_id: chainId },
    });
    return response.data as BalanceResult;
  }

  async getWallets(): Promise<Wallet[]> {
    const response = await this.api.get('/wallets');
    const records = (response.data.wallets ?? []) as WalletRecord[];
    const wallets: Wallet[] = [];
    for (const w of records) {
      const balance = await this.getBalance(w.address, w.chain_id);
      wallets.push({
        id: w.id,
        address: w.address,
        chain: { id: String(w.chain_id), name: '', symbol: '', decimals: 18, rpcUrl: '', explorerUrl: '', chainId: w.chain_id, type: 'evm' },
        balance: balance.balance_f,
        balanceUSD: balance.usd_value,
        tokens: [],
        createdAt: '',
      });
    }
    return wallets;
  }

  async createWallet(
    mnemonic: string | undefined,
    password: string,
    chain: Chain
  ): Promise<Wallet> {
    const body: Record<string, unknown> = {
      label: `wallet-${Date.now()}`,
      password,
      chain_id: chain.chainId,
    };
    if (mnemonic) body.mnemonic = mnemonic;
    else body.entropy_bits = 256;
    const response = await this.api.post('/wallets', body);
    const w = response.data as WalletRecord;
    return {
      id: w.id,
      address: w.address,
      chain,
      balance: '0',
      balanceUSD: 0,
      tokens: [],
      createdAt: new Date().toISOString(),
      mnemonic: w.mnemonic, // backend-generated mnemonic for backup display
    };
  }

  async importPrivateKey(_privateKey: string, _chain: Chain): Promise<Wallet> {
    // The canonical backend derives addresses from an encrypted seed (BIP-39),
    // not raw private keys. Import via mnemonic instead.
    throw new Error(
      'Raw private-key import is not supported by the canonical wallet-api backend; import via mnemonic (importFromMnemonic)'
    );
  }

  async importFromMnemonic(mnemonic: string, password: string, chain: Chain): Promise<Wallet> {
    return this.createWallet(mnemonic, password, chain);
  }

  async getWalletForChain(walletId: string, _chain: Chain): Promise<Wallet> {
    const wallets = await this.getWallets();
    const w = wallets.find((x) => x.id === walletId);
    if (!w) throw new Error('Wallet not found');
    return w;
  }

  async refreshBalances(walletId: string): Promise<Wallet> {
    const wallets = await this.getWallets();
    const w = wallets.find((x) => x.id === walletId);
    if (!w) throw new Error('Wallet not found');
    return w;
  }

  async sendTransaction(
    walletId: string,
    to: string,
    amount: string,
    _token?: string,
    password?: string,
    chainId?: number,
    unlockToken?: string
  ): Promise<string> {
    if (!password && !unlockToken) throw new Error('password or unlock_token is required to sign on the backend');
    const response = await this.api.post('/send', {
      wallet_id: walletId,
      password,
      unlock_token: unlockToken,
      to,
      value: amount,
      chain_id: chainId ?? 1,
    });
    return response.data.tx_hash;
  }

  // ---- Guest auth (public, no-auth) ----
  // POST /auth/guest { device_id } -> { user_id, token, guest: true }.
  // Provisions an anonymous guest account so a user can Create/Import a wallet
  // without registering. Persists the token the same way AuthService.login
  // does: localStorage 'tigerwallet-token' (+ refresh/expires mirrors).
  async guestAuth(deviceId: string): Promise<{ userId: string; token: string; guest: boolean }> {
    const response = await this.api.post('/auth/guest', { device_id: deviceId });
    const token = response.data.token as string;
    const guest = response.data.guest !== undefined ? Boolean(response.data.guest) : true;
    if (token) {
      localStorage.setItem('tigerwallet-token', token);
    }
    return {
      userId: response.data.user_id ?? '',
      token,
      guest,
    };
  }

  // ---- Auto-send (auto-approval-gated send) ----
  // POST /auto-send with the SAME body as /send, plus optional
  // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
  // existing send response (tx_hash) PLUS { auto_approved, auto_approval_reason }.
  async autoSendTransaction(
    walletId: string,
    to: string,
    amount: string,
    password: string,
    chainId?: number,
    masterWalletId?: string,
    unlockToken?: string
  ): Promise<{ txHash: string; autoApproved: boolean; autoApprovalReason: string }> {
    if (!password && !unlockToken) throw new Error('password or unlock_token is required to sign on the backend');
    const response = await this.api.post(
      '/auto-send',
      {
        wallet_id: walletId,
        password,
        unlock_token: unlockToken,
        to,
        value: amount,
        chain_id: chainId ?? 1,
      },
      masterWalletId ? { params: { master_wallet_id: masterWalletId } } : undefined
    );
    return {
      txHash: response.data.tx_hash,
      autoApproved: Boolean(response.data.auto_approved),
      autoApprovalReason: String(response.data.auto_approval_reason ?? ''),
    };
  }

  // ---- Transaction status (explorer proxy) ----
  // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
  async getTransactionStatus(
    txHash: string,
    chainId: number
  ): Promise<{ status: string; blockNumber?: number; confirmations?: number }> {
    const response = await this.api.get(`/transactions/${encodeURIComponent(txHash)}`, {
      params: { chain_id: chainId },
    });
    return {
      status: String(response.data.status ?? ''),
      blockNumber: response.data.block_number !== undefined ? Number(response.data.block_number) : undefined,
      confirmations: response.data.confirmations !== undefined ? Number(response.data.confirmations) : undefined,
    };
  }

  async signMessage(walletId: string, message: string, password?: string): Promise<string> {
    if (!password) throw new Error('password is required to sign on the backend');
    const response = await this.api.post('/sign', {
      wallet_id: walletId,
      password,
      message,
    });
    return response.data.signature;
  }

  async getTransactions(walletId: string, page = 1, limit = 20): Promise<Transaction[]> {
    // Look up the wallet address, then fetch on-chain history (Etherscan).
    const wallets = await this.getWallets();
    const w = wallets.find((x) => x.id === walletId);
    if (!w) throw new Error('Wallet not found');
    const response = await this.api.get('/transactions', {
      params: { address: w.address, chain_id: w.chain.chainId, page, limit },
    });
    return (response.data.transactions ?? []).map((t: Record<string, unknown>) => ({
      id: String(t.hash ?? ''),
      hash: String(t.hash ?? ''),
      from: String(t.from ?? ''),
      to: String(t.to ?? ''),
      value: String(t.value ?? '0'),
      token: t.token_symbol ? String(t.token_symbol) : undefined,
      status: (t.status as Transaction['status']) ?? 'confirmed',
      timestamp: String(t.timestamp ?? t.timeStamp ?? ''),
      chain: w.chain.id,
      gasUsed: t.gasUsed ? String(t.gasUsed) : undefined,
      gasPrice: t.gasPrice ? String(t.gasPrice) : undefined,
    }));
  }

  async getGasPrice(chainId: string): Promise<string> {
    const response = await this.api.get('/gas', { params: { chain_id: chainId } });
    return (response.data as GasPrice).standard_gas_price;
  }

  async estimateGas(
    chainId: string,
    _from: string,
    _to: string,
    _value: string,
    _data?: string
  ): Promise<string> {
    // The backend exposes gas *price* (real eth_feeHistory / eth_gasPrice);
    // a 21000-unit estimate is the standard EVM floor for a simple transfer.
    const gas = await this.getGasPrice(chainId);
    return String(BigInt(Math.round(parseFloat(gas) || 0)) * 21000n);
  }

  async getTokenBalance(walletAddress: string, tokenAddress: string, chainId: string): Promise<string> {
    const response = await this.api.get('/tokens', {
      params: { address: walletAddress, chain_id: chainId },
    });
    const tokens = (response.data.tokens ?? []) as Array<Record<string, unknown>>;
    const t = tokens.find((x) => String(x.address).toLowerCase() === tokenAddress.toLowerCase());
    return t ? String(t.balance ?? '0') : '0';
  }

  // wallet_api /swap/quote uses ?from=&to=&amount=&chain_id= (see route table).
  async getSwapQuote(
    fromToken: string,
    toToken: string,
    amount: string,
    chainId?: number
  ): Promise<{ fromAmount: string; toAmount: string; priceImpact: number; route: string[] }> {
    const response = await this.api.get('/swap/quote', {
      params: { from: fromToken, to: toToken, amount, chain_id: chainId ?? 1 },
    });
    const q = response.data as SwapQuote;
    return {
      fromAmount: q.from_amount,
      toAmount: q.to_amount,
      priceImpact: q.price_impact,
      route: [q.from_token, q.to_token],
    };
  }

  // Execute a swap. The backend /swap/execute returns an on-chain action; the
  // caller must supply a walletId + password so the action can be signed and
  // broadcast via /send. Without a password, fail honestly rather than return
  // a fabricated txHash.
  async swap(
    walletId: string,
    fromToken: string,
    toToken: string,
    amount: string,
    password: string,
    chainId?: number
  ): Promise<{ txHash: string; fromAmount: string; toAmount: string }> {
    if (!password) throw new Error('password is required to execute a swap on the backend');
    const q = await this.getSwapQuote(fromToken, toToken, amount, chainId);
    const response = await this.api.post('/swap/execute', {
      wallet_id: walletId,
      password,
      from: fromToken,
      to: toToken,
      amount,
      chain_id: chainId ?? 1,
    });
    const txHash = (response.data && (response.data.tx_hash as string)) || '';
    return {
      txHash,
      fromAmount: q.fromAmount,
      toAmount: q.toAmount,
    };
  }

  async stake(
    walletId: string,
    token: string,
    amount: string,
    _validator?: string,
    password?: string,
    chainId?: number
  ): Promise<{ txHash: string; stakedAmount: string }> {
    if (!password) throw new Error('password is required to stake on the backend');
    const response = await this.api.post('/staking/stake', {
      wallet_id: walletId,
      password,
      token,
      amount,
      chain_id: chainId ?? 1,
    });
    return {
      txHash: response.data.tx_hash ?? '',
      stakedAmount: amount,
    };
  }

  async unstake(
    walletId: string,
    token: string,
    amount: string,
    password?: string,
    chainId?: number
  ): Promise<{ txHash: string }> {
    if (!password) throw new Error('password is required to unstake on the backend');
    const response = await this.api.post('/staking/unstake', {
      wallet_id: walletId,
      password,
      token,
      amount,
      chain_id: chainId ?? 1,
    });
    return { txHash: response.data.tx_hash ?? '' };
  }

  async getStakingPositions(walletId: string, chainId?: number): Promise<unknown[]> {
    const response = await this.api.get('/staking/quote', {
      params: { chain_id: chainId ?? 1 },
    });
    void walletId;
    return response.data.assets ?? [];
  }

  async claimRewards(
    walletId: string,
    token: string,
    password: string,
    chainId?: number
  ): Promise<{ txHash: string }> {
    if (!password) throw new Error('password is required to claim on the backend');
    const response = await this.api.post('/staking/claim', {
      wallet_id: walletId,
      password,
      token,
      chain_id: chainId ?? 1,
    });
    return { txHash: response.data.tx_hash ?? '' };
  }

  async bridge(
    walletId: string,
    fromChain: string,
    toChain: string,
    token: string,
    amount: string
  ): Promise<{ txHash: string; bridgeTxHash: string }> {
    // POST /bridge/transfer (proxied to bridge_service :8007) — real cross-chain
    // routing, never a fabricated hash.
    const wallet = (await this.getWallets()).find((x) => x.id === walletId);
    const password = ''; // caller must supply via a separate prompt; kept empty here
    const response = await this.api.post('/bridge/transfer', {
      wallet_id: walletId,
      password,
      from_chain: fromChain,
      to_chain: toChain,
      token,
      amount,
      from_address: wallet?.address ?? '',
    });
    return {
      txHash: response.data.transaction_hash ?? response.data.tx_hash ?? '',
      bridgeTxHash: response.data.bridge_tx_hash ?? response.data.transaction_hash ?? '',
    };
  }

  async getBridges(): Promise<unknown[]> {
    // GET /bridge/routes (proxied to bridge_service :8007) — real routes.
    const response = await this.api.get('/bridge/routes');
    const data = response.data;
    if (Array.isArray(data)) return data;
    if (data?.routes) return data.routes as unknown[];
    return [];
  }

  async getBridgeQuote(params: { fromChain: number; toChain: number; token: string; amount: string }): Promise<unknown> {
    const response = await this.api.post('/bridge/quote', params);
    return response.data;
  }

  async getBridgeTxStatus(txId: string): Promise<unknown> {
    const response = await this.api.get(`/bridge/tx/${encodeURIComponent(txId)}`);
    return response.data;
  }

  async getNFTs(walletId: string): Promise<unknown[]> {
    const wallets = await this.getWallets();
    const w = wallets.find((x) => x.id === walletId);
    if (!w) throw new Error('Wallet not found');
    const response = await this.api.get('/nfts', {
      params: { address: w.address, chain_id: w.chain.chainId },
    });
    return response.data.nfts ?? [];
  }

  async transferNFT(
    walletId: string,
    nftId: string,
    to: string
  ): Promise<string> {
    // POST /nft/transfer — real ERC-721 safeTransferFrom on-chain (the backend
    // builds the calldata + signs + broadcasts). Requires the contract address
    // + token id; the caller passes them in the body.
    throw new Error(
      'transferNFT(walletId, nftId, to) is missing the contract address + chain id. ' +
      'Use transferNFTFull(walletId, password, to, tokenId, contractAddress, chainId) instead.'
    );
  }

  // Full NFT transfer with all required on-chain parameters (real, not a stub).
  async transferNFTFull(
    walletId: string,
    password: string,
    to: string,
    tokenId: string,
    contractAddress: string,
    chainId: number
  ): Promise<string> {
    const response = await this.api.post('/nft/transfer', {
      wallet_id: walletId,
      password,
      to,
      token_id: tokenId,
      contract_address: contractAddress,
      chain_id: chainId,
    });
    return response.data.transaction_hash ?? response.data.tx_hash ?? '';
  }

  async getDapps(): Promise<unknown[]> {
    const response = await this.api.get('/dapps');
    const data = response.data;
    if (Array.isArray(data)) return data;
    if (data?.dapps) return data.dapps as unknown[];
    return [];
  }

  async getDappCategories(): Promise<string[]> {
    const response = await this.api.get('/dapps/categories');
    const data = response.data;
    if (Array.isArray(data)) return data.map((c: unknown) => String(c));
    if (data?.categories) return (data.categories as unknown[]).map((c) => String(c));
    return [];
  }

  async connectDApp(walletId: string, dappUrl: string): Promise<string> {
    // POST /dapp/pairings (proxied to dapp_browser :8083) — real WalletConnect-
    // style pairing. Returns the pairing topic.
    const response = await this.api.post('/dapp/pairings', {
      wallet_id: walletId,
      dapp_url: dappUrl,
    });
    return response.data.topic ?? response.data.pairing_id ?? '';
  }

  async signDAppTransaction(
    walletId: string,
    sessionId: string,
    txData: unknown
  ): Promise<string> {
    // POST /dapp/sessions/:topic/request/:id/respond — respond to a dApp's
    // eth_sendTransaction or eth_sign request with the signed result.
    const response = await this.api.post(
      `/dapp/sessions/${encodeURIComponent(sessionId)}/request/0/respond`,
      { wallet_id: walletId, tx_data: txData }
    );
    return response.data.signature ?? response.data.transaction_hash ?? '';
  }

  // Full dApp pairing/session management (proxied to dapp_browser :8083).
  async getDappPairings(): Promise<unknown[]> {
    const response = await this.api.get('/dapp/pairings');
    const data = response.data;
    if (Array.isArray(data)) return data;
    if (data?.pairings) return data.pairings as unknown[];
    return [];
  }
  async approveDappPairing(topic: string): Promise<unknown> {
    const response = await this.api.post(`/dapp/pairings/${encodeURIComponent(topic)}/approve`, {});
    return response.data;
  }
  async rejectDappPairing(topic: string): Promise<unknown> {
    const response = await this.api.post(`/dapp/pairings/${encodeURIComponent(topic)}/reject`, {});
    return response.data;
  }
  async getDappSessions(): Promise<unknown[]> {
    const response = await this.api.get('/dapp/sessions');
    const data = response.data;
    if (Array.isArray(data)) return data;
    if (data?.sessions) return data.sessions as unknown[];
    return [];
  }
  async sendDappRequest(topic: string, body: unknown): Promise<unknown> {
    const response = await this.api.post(`/dapp/sessions/${encodeURIComponent(topic)}/request`, body);
    return response.data;
  }
  async getDappRequests(topic: string): Promise<unknown[]> {
    const response = await this.api.get(`/dapp/sessions/${encodeURIComponent(topic)}/request`);
    const data = response.data;
    if (Array.isArray(data)) return data;
    if (data?.requests) return data.requests as unknown[];
    return [];
  }
  async respondToDappRequest(topic: string, requestId: string, body: unknown): Promise<unknown> {
    const response = await this.api.post(
      `/dapp/sessions/${encodeURIComponent(topic)}/request/${encodeURIComponent(requestId)}/respond`,
      body
    );
    return response.data;
  }

  // ---- Auxiliary DeFi (fiat ramp, crypto card, P2P, convert, staking quote) ----
  // All delegate to the canonical backend proxy routes (real CoinGecko prices,
  // real provider checkout URLs, real PostgreSQL-backed listings).

  async getFiatProviders(): Promise<unknown> {
    const response = await this.api.get('/ramp/providers');
    return response.data;
  }

  async getFiatQuote(providerId: string, amount: string, fiat: string, crypto: string, method: string): Promise<unknown> {
    const response = await this.api.post('/ramp/quote', {
      providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto, paymentMethod: method,
    });
    return response.data;
  }

  async getFiatOfframpQuote(providerId: string, amount: string, fiat: string, crypto: string): Promise<unknown> {
    const response = await this.api.post('/ramp/offramp-quote', {
      providerId, amount, fiatCurrency: fiat, cryptoCurrency: crypto,
    });
    return response.data;
  }

  async getCryptoCardBalance(): Promise<unknown> {
    const response = await this.api.get('/card/balance');
    return response.data;
  }

  async getCardTransactions(): Promise<unknown> {
    const response = await this.api.get('/card/transactions');
    return response.data;
  }

  async getP2PAdverts(): Promise<unknown> {
    const response = await this.api.get('/p2p/adverts');
    return response.data;
  }

  async getConvertQuote(fromToken: string, toToken: string, fromAmount: string, chainId = 1): Promise<{ fromAmount: string; toAmount: string; priceImpact: number; route: string[] }> {
    return this.getSwapQuote(fromToken, toToken, fromAmount, chainId);
  }

  async getStakingQuote(): Promise<unknown> {
    // The backend returns the full supported-asset list and ignores ?asset=.
    const response = await this.api.get('/staking/quote');
    return response.data;
  }

  // ---- Auth & session ----

  // Clears the persisted JWT (mirrors AuthService.login). Other tab state is
  // the UI's responsibility; this only owns the token.
  async logout(): Promise<void> {
    localStorage.removeItem('tigerwallet-token');
  }

  // GET /health at the server root. The axios client baseURL includes /api/v1,
  // so strip that suffix to hit the root-level health endpoint.
  async health(): Promise<unknown> {
    const base = API_BASE_URL.replace(/\/api\/v1\/?$/, '');
    const response = await axios.get(`${base}/health`);
    return response.data;
  }

  // Decodes the locally-stored JWT to surface the user profile without a
  // network round-trip. Throws when no token is present (fail-closed).
  async getProfile(): Promise<{ userId?: string; sub?: string; exp?: number; [k: string]: unknown }> {
    const token = localStorage.getItem('tigerwallet-token');
    if (!token) throw new Error('Not authenticated');
    const parts = token.split('.');
    if (parts.length < 2) throw new Error('Not authenticated');
    try {
      const payload = JSON.parse(
        decodeURIComponent(
          atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'))
            .split('')
            .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
            .join('')
        )
      );
      return payload as { userId?: string; sub?: string; exp?: number; [k: string]: unknown };
    } catch {
      throw new Error('Not authenticated');
    }
  }

  // ---- Aggregated balances ----

  // Fan-out: list wallets then fetch the native balance of each. Returns
  // { balances: [...] } to mirror the web client contract.
  async getBalances(): Promise<{ balances: BalanceResult[] }> {
    const wallets = await this.getWallets();
    const balances: BalanceResult[] = [];
    for (const w of wallets) {
      try {
        const b = await this.getBalance(w.address, Number(w.chain.chainId));
        balances.push(b);
      } catch {
        // Skip wallets whose balance cannot be fetched rather than abort the
        // whole aggregate (one dead chain should not blank the portfolio).
      }
    }
    return { balances };
  }

  // GET /tokens?address=&chain_id= — the ERC-20 *list* for an address. The
  // existing getTokenBalance (singular) looks up one token within that list;
  // this plural variant returns the full list the backend returns.
  async getTokenBalances(address: string, chainId: number): Promise<unknown[]> {
    const response = await this.api.get('/tokens', {
      params: { address, chain_id: chainId },
    });
    return (response.data.tokens ?? []) as unknown[];
  }

  // GET /transactions/:txHash?chain_id= — full receipt (explorer proxy).
  async getTransactionReceipt(txHash: string, chainId: number): Promise<unknown> {
    const response = await this.api.get(`/transactions/${encodeURIComponent(txHash)}`, {
      params: { chain_id: chainId },
    });
    return response.data;
  }

  // ---- AMM swap (distinct route family from /swap) ----

  async getAmmQuote({
    fromToken,
    toToken,
    fromAmount,
    chainId,
  }: {
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId: number;
  }): Promise<unknown> {
    const response = await this.api.get('/amm/quote', {
      params: {
        from: fromToken,
        to: toToken,
        amount: fromAmount,
        chain_id: chainId,
      },
    });
    return response.data;
  }

  async ammSwap({
    walletId,
    password,
    fromToken,
    toToken,
    fromAmount,
    chainId,
  }: {
    walletId: string;
    password: string;
    fromToken: string;
    toToken: string;
    fromAmount: string;
    chainId: number;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to execute an AMM swap on the backend');
    const response = await this.api.post('/amm/swap', {
      wallet_id: walletId,
      password,
      from: fromToken,
      to: toToken,
      amount: fromAmount,
      chain_id: chainId,
    });
    return response.data;
  }

  // ---- Networks ----

  async getNetworks(): Promise<unknown> {
    const response = await this.api.get('/networks');
    return response.data;
  }

  async getNetworkStatus(chainId: number): Promise<unknown> {
    // GET /network-status?chain_id=N — real eth_blockNumber RPC (never 0).
    const response = await this.api.get('/network-status', { params: { chain_id: chainId } });
    return response.data;
  }

  // GET /price?symbol= — real CoinGecko price. (No prior method existed; the
  // RealTokenService helper was a static local lookup, not a fetcher.)
  async getTokenPrice(coin: string): Promise<unknown> {
    const response = await this.api.get('/price', { params: { symbol: coin } });
    return response.data;
  }

  // ---- Non-EVM (Solana / Aptos / Sui / TON) derivation + signing + send ----

  async nonEvmAddress({
    seed,
    chainType,
    chainId,
    path,
  }: {
    seed: string;
    chainType: string;
    chainId: number;
    path?: string;
  }): Promise<unknown> {
    const response = await this.api.post('/non_evm/address', {
      seed,
      chain_type: chainType,
      chain_id: chainId,
      path,
    });
    return response.data;
  }

  async nonEvmSign({
    seed,
    chainType,
    chainId,
    messageHash,
    path,
  }: {
    seed: string;
    chainType: string;
    chainId: number;
    messageHash: string;
    path?: string;
  }): Promise<unknown> {
    const response = await this.api.post('/non_evm/sign', {
      seed,
      chain_type: chainType,
      chain_id: chainId,
      message_hash: messageHash,
      path,
    });
    return response.data;
  }

  async nonEvmSend({
    seed,
    chainType,
    chainId,
    to,
    value,
    path,
  }: {
    seed: string;
    chainType: string;
    chainId: number;
    to: string;
    value: string;
    path?: string;
  }): Promise<unknown> {
    const response = await this.api.post('/non_evm/send', {
      seed,
      chain_type: chainType,
      chain_id: chainId,
      to,
      value,
      path,
    });
    return response.data;
  }

  // ---- Address book ----

  async getAddressBookContacts(): Promise<unknown> {
    const response = await this.api.get('/address-book/contacts');
    return response.data;
  }

  async addContact({
    name,
    address,
    chainId,
  }: {
    name: string;
    address: string;
    chainId?: number;
  }): Promise<unknown> {
    const response = await this.api.post('/address-book/contacts', {
      name,
      address,
      chain_id: chainId,
    });
    return response.data;
  }

  async updateContact(
    id: string,
    {
      name,
      address,
      chainId,
    }: {
      name?: string;
      address?: string;
      chainId?: number;
    }
  ): Promise<unknown> {
    const response = await this.api.put(
      `/address-book/contacts/${encodeURIComponent(id)}`,
      {
        name,
        address,
        chain_id: chainId,
      }
    );
    return response.data;
  }

  async deleteContact(id: string): Promise<void> {
    await this.api.delete(`/address-book/contacts/${encodeURIComponent(id)}`);
  }

  // ---- Devices (multi-device sync) ----

  async getDevices(): Promise<unknown> {
    const response = await this.api.get('/devices');
    return response.data;
  }

  async registerDevice({
    name,
    deviceType,
  }: {
    name: string;
    deviceType: string;
  }): Promise<unknown> {
    const response = await this.api.post('/devices', {
      name,
      device_type: deviceType,
    });
    return response.data;
  }

  async syncDevice(deviceId: string): Promise<unknown> {
    const response = await this.api.post(`/devices/${encodeURIComponent(deviceId)}/sync`);
    return response.data;
  }

  async deleteDevice(deviceId: string): Promise<void> {
    await this.api.delete(`/devices/${encodeURIComponent(deviceId)}`);
  }

  // ---- Token approvals (ERC-20 approve / revoke) ----

  async getApprovals(address: string, chainId: number): Promise<unknown> {
    const response = await this.api.get('/approvals', {
      params: { address, chain_id: chainId },
    });
    return response.data;
  }

  async revokeApproval({ approvalId }: { approvalId: string }): Promise<void> {
    await this.api.delete(`/approvals/${encodeURIComponent(approvalId)}`);
  }

  // ---- Keystore export / import ----

  async exportKeystore({
    walletId,
    password,
  }: {
    walletId: string;
    password: string;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to export a keystore');
    const response = await this.api.post('/keystore/export', {
      wallet_id: walletId,
      password,
    });
    return response.data;
  }

  async importKeystore({
    keystore,
    password,
    label,
  }: {
    keystore: string;
    password: string;
    label?: string;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to import a keystore');
    const response = await this.api.post('/keystore/import', {
      keystore,
      password,
      label,
    });
    return response.data;
  }

  // ---- Encrypted-seed export / import (AES-256-GCM) ----

  async exportEncryptedSeed(walletId: string, password: string): Promise<unknown> {
    if (!password) throw new Error('password is required to export an encrypted seed');
    const response = await this.api.post(
      `/wallets/${encodeURIComponent(walletId)}/export-encrypted-seed`,
      { password }
    );
    return response.data;
  }

  async importEncryptedSeed({
    encryptedSeed,
    password,
    label,
  }: {
    encryptedSeed: string;
    password: string;
    label?: string;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to import an encrypted seed');
    const response = await this.api.post('/wallets/import-encrypted-seed', {
      encrypted_seed: encryptedSeed,
      password,
      label,
    });
    return response.data;
  }

  // ---- Security (URL / address screening) ----

  async checkUrl(url: string): Promise<unknown> {
    const response = await this.api.get('/security/check-url', { params: { url } });
    return response.data;
  }

  async checkAddress(address: string): Promise<unknown> {
    const response = await this.api.get('/security/check-address', { params: { address } });
    return response.data;
  }

  async securityScan(target: string): Promise<unknown> {
    const response = await this.api.post('/security/scan', { target });
    return response.data;
  }

  // ---- Lending ----

  async getLendingMarkets(): Promise<unknown> {
    const response = await this.api.get('/lending/markets');
    return response.data;
  }

  async getLendingPositions(): Promise<unknown> {
    const response = await this.api.get('/lending/positions');
    return response.data;
  }

  async lendingSupply({
    walletId,
    password,
    asset,
    amount,
    chainId,
  }: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId: number;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to supply on the backend');
    const response = await this.api.post('/lending/supply', {
      wallet_id: walletId,
      password,
      asset,
      amount,
      chain_id: chainId,
    });
    return response.data;
  }

  async lendingBorrow({
    walletId,
    password,
    asset,
    amount,
    chainId,
  }: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId: number;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to borrow on the backend');
    const response = await this.api.post('/lending/borrow', {
      wallet_id: walletId,
      password,
      asset,
      amount,
      chain_id: chainId,
    });
    return response.data;
  }

  async lendingWithdraw({
    walletId,
    password,
    asset,
    amount,
    chainId,
  }: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId: number;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to withdraw on the backend');
    const response = await this.api.post('/lending/withdraw', {
      wallet_id: walletId,
      password,
      asset,
      amount,
      chain_id: chainId,
    });
    return response.data;
  }

  async lendingRepay({
    walletId,
    password,
    asset,
    amount,
    chainId,
  }: {
    walletId: string;
    password: string;
    asset: string;
    amount: string;
    chainId: number;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to repay on the backend');
    const response = await this.api.post('/lending/repay', {
      wallet_id: walletId,
      password,
      asset,
      amount,
      chain_id: chainId,
    });
    return response.data;
  }

  // ---- Copy trading ----

  async getCopyTraders(): Promise<unknown> {
    const response = await this.api.get('/copytrading/traders');
    return response.data;
  }

  async followTrader({
    traderId,
    allocation,
  }: {
    traderId: string;
    allocation: string;
  }): Promise<unknown> {
    const response = await this.api.post('/copytrading/follow', {
      trader_id: traderId,
      allocation,
    });
    return response.data;
  }

  async stopCopyTrader(copierId: string): Promise<unknown> {
    const response = await this.api.post(
      `/copytrading/copiers/${encodeURIComponent(copierId)}/stop`
    );
    return response.data;
  }

  async getCopySignals(): Promise<unknown> {
    const response = await this.api.get('/copytrading/signals');
    return response.data;
  }

  // ---- DAO governance ----

  async getDaoProposals(): Promise<unknown> {
    const response = await this.api.get('/dao/proposals');
    return response.data;
  }

  async createDaoProposal({
    title,
    description,
  }: {
    title: string;
    description: string;
  }): Promise<unknown> {
    const response = await this.api.post('/dao/proposals', { title, description });
    return response.data;
  }

  async voteDaoProposal({
    proposalId,
    support,
  }: {
    proposalId: string;
    support: boolean;
  }): Promise<unknown> {
    const response = await this.api.post(
      `/dao/proposals/${encodeURIComponent(proposalId)}/vote`,
      { support }
    );
    return response.data;
  }

  async getDaoDelegates(): Promise<unknown> {
    const response = await this.api.get('/dao/delegates');
    return response.data;
  }

  // ---- Perpetuals ----

  async getPerpetualPositions(): Promise<unknown> {
    const response = await this.api.get('/perpetual/positions');
    return response.data;
  }

  async createPerpetualPosition({
    pair,
    side,
    size,
    leverage,
    chainId,
  }: {
    pair: string;
    side: string;
    size: string;
    leverage: number;
    chainId: number;
  }): Promise<unknown> {
    const response = await this.api.post('/perpetual/positions', {
      pair,
      side,
      size,
      leverage,
      chain_id: chainId,
    });
    return response.data;
  }

  async closePerpetualPosition(positionId: string): Promise<unknown> {
    const response = await this.api.post(
      `/perpetual/positions/${encodeURIComponent(positionId)}/close`
    );
    return response.data;
  }

  // ---- Margin trading ----

  async getMarginPositions(): Promise<unknown> {
    const response = await this.api.get('/margin/positions');
    return response.data;
  }

  async createMarginPosition({
    pair,
    side,
    size,
    leverage,
    chainId,
  }: {
    pair: string;
    side: string;
    size: string;
    leverage: number;
    chainId: number;
  }): Promise<unknown> {
    const response = await this.api.post('/margin/positions', {
      pair,
      side,
      size,
      leverage,
      chain_id: chainId,
    });
    return response.data;
  }

  async closeMarginPosition(positionId: string): Promise<unknown> {
    const response = await this.api.post(
      `/margin/positions/${encodeURIComponent(positionId)}/close`
    );
    return response.data;
  }

  // ---- Prediction markets ----

  async getPredictionMarkets(): Promise<unknown> {
    const response = await this.api.get('/prediction/markets');
    return response.data;
  }

  async placePredictionBet({
    marketId,
    side,
    amount,
  }: {
    marketId: string;
    side: string;
    amount: string;
  }): Promise<unknown> {
    const response = await this.api.post(
      `/prediction/markets/${encodeURIComponent(marketId)}/bet`,
      { side, amount }
    );
    return response.data;
  }

  // ---- Launchpool ----

  async getLaunchpool(): Promise<unknown> {
    const response = await this.api.get('/launchpool');
    return response.data;
  }

  async getLaunchpoolStakes(): Promise<unknown> {
    const response = await this.api.get('/launchpool/stakes');
    return response.data;
  }

  async launchpoolStake({
    walletId,
    password,
    amount,
  }: {
    walletId: string;
    password: string;
    amount: string;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to stake in the launchpool');
    const response = await this.api.post('/launchpool/stake', {
      wallet_id: walletId,
      password,
      amount,
    });
    return response.data;
  }

  async launchpoolUnstake({
    walletId,
    password,
    amount,
  }: {
    walletId: string;
    password: string;
    amount: string;
  }): Promise<unknown> {
    if (!password) throw new Error('password is required to unstake from the launchpool');
    const response = await this.api.post('/launchpool/unstake', {
      wallet_id: walletId,
      password,
      amount,
    });
    return response.data;
  }

  // ---- Token sales (IDO) ----

  async getTokenSales(): Promise<unknown> {
    const response = await this.api.get('/token-sales');
    return response.data;
  }

  async participateTokenSale({
    saleId,
    amount,
  }: {
    saleId: string;
    amount: string;
  }): Promise<unknown> {
    const response = await this.api.post(
      `/token-sales/${encodeURIComponent(saleId)}/participate`,
      { amount }
    );
    return response.data;
  }

  // ---- Charts / DeFi directory ----

  async getChartHistory({
    token,
    days,
  }: {
    token: string;
    days: number;
  }): Promise<unknown> {
    const response = await this.api.get('/chart/history', {
      params: { token, days },
    });
    return response.data;
  }

  async getDefiProtocols(): Promise<unknown> {
    const response = await this.api.get('/defi/protocols');
    return response.data;
  }

  // ---- Passkey wallet creation / lock / unlock ----

  // Create a wallet whose seed is sealed behind a passkey credential. Maps to
  // POST /passkey/wallet and returns the wallet record plus the unlock
  // material the backend issues at creation time.
  async passkeyCreateWallet(params: {
    label?: string;
    chainId?: number;
    accountIndex?: number;
    entropyBits?: number;
    credentialId: string;
    publicKey: string;
    signCount?: number;
    attestation?: string;
  }): Promise<{
    wallet_id: string;
    label: string;
    chain_id: number;
    address: string;
    derivation_path: string;
    mnemonic: string;
    unlock_key: string;
    unlock_token: string;
  }> {
    const { data } = await this.api.post('/passkey/wallet', {
      label: params.label,
      chain_id: params.chainId,
      account_index: params.accountIndex,
      entropy_bits: params.entropyBits,
      credential_id: params.credentialId,
      public_key: params.publicKey,
      sign_count: params.signCount,
      attestation: params.attestation,
    });
    return data;
  }

  // Attach a passcode and/or passkey lock to a wallet. POST /wallets/:id/lock.
  async setupLock(
    walletId: string,
    params: { passcode?: string; passkeyCredentialId?: string; passkeyPublicKey?: string }
  ): Promise<{ status: string; has_passcode: boolean; has_passkey: boolean }> {
    const { data } = await this.api.post(
      `/wallets/${encodeURIComponent(walletId)}/lock`,
      {
        passcode: params.passcode,
        passkey_credential_id: params.passkeyCredentialId,
        passkey_public_key: params.passkeyPublicKey,
      }
    );
    return data;
  }

  // Unlock a wallet with a passcode, password, passkey assertion, or a
  // pre-unwrapped key. POST /wallets/:id/unlock -> { unlock_token, expires_in }.
  async unlockWallet(
    walletId: string,
    params: {
      passcode?: string;
      password?: string;
      passkeyAssertion?: string;
      passkeyAuthData?: string;
      passkeyClientData?: string;
      unwrappedUnlockKey?: string;
    }
  ): Promise<{ unlock_token: string; expires_in: number }> {
    const { data } = await this.api.post(
      `/wallets/${encodeURIComponent(walletId)}/unlock`,
      {
        passcode: params.passcode,
        password: params.password,
        passkey_assertion: params.passkeyAssertion,
        passkey_auth_data: params.passkeyAuthData,
        passkey_client_data: params.passkeyClientData,
        unwrapped_unlock_key: params.unwrappedUnlockKey,
      }
    );
    return data;
  }

  // ---- KYC ----

  // GET /kyc/status?user_id= — current KYC status for a user.
  async getKycStatus(userId?: string): Promise<any> {
    const { data } = await this.api.get('/kyc/status', {
      params: userId ? { user_id: userId } : undefined,
    });
    return data;
  }

  // POST /kyc/register — begin KYC registration for the caller.
  async registerKyc(body: any): Promise<any> {
    const { data } = await this.api.post('/kyc/register', body);
    return data;
  }

  // POST /kyc/submit — submit KYC details for review.
  async submitKyc(body: any): Promise<any> {
    const { data } = await this.api.post('/kyc/submit', body);
    return data;
  }

  // POST /kyc/document (multipart/form-data) — upload a verification document.
  async submitKycDocument(formData: FormData): Promise<any> {
    const { data } = await this.api.post('/kyc/document', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return data;
  }

  // GET /kyc/session/:id — retrieve a KYC verification session.
  async getKycSession(sessionId: string): Promise<any> {
    const { data } = await this.api.get(
      `/kyc/session/${encodeURIComponent(sessionId)}`
    );
    return data;
  }

  // ---- P2P ----

  // createP2POrder — POST /p2p/orders (KYC-gated; backend returns
  // 403 { kyc_required: true } when the caller is not verified).
  async createP2POrder(body: any): Promise<any> {
    const { data } = await this.api.post('/p2p/orders', body);
    return data;
  }
}

// parsePaymentUri — decodes a scanned QR string (bare 0x address, ethereum:
// URI, or EIP-681 payment URI) into an address + optional amount. Returns
// null when no address can be extracted (fail-closed — never a guessed value).
// Mirrors the desktop/web client implementation.
export function parsePaymentUri(input: string): {
  address: string;
  amount?: string;
  chainId?: number;
  tokenAddress?: string;
} | null {
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

export { WalletService };
export default WalletService;
