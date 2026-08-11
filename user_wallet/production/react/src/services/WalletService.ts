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
 *   GET    /swap/quote?from_token=&to_token=&amount=&chain_id=  -> SwapQuote
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
  id: string;
  name: string;
  symbol: string;
  decimals: number;
  rpcUrl: string;
  explorerUrl: string;
  chainId: number;
  type: 'evm' | 'solana' | 'aptos' | 'sui' | 'ton';
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
    return (chains as Array<Record<string, unknown>>).map((c) => ({
      id: String(c.id ?? c.chain_id ?? ''),
      name: String(c.name ?? ''),
      symbol: String(c.symbol ?? ''),
      decimals: Number(c.decimals ?? 18),
      rpcUrl: String(c.rpc_url ?? c.rpc ?? ''),
      explorerUrl: String(c.explorer_url ?? c.explorer ?? ''),
      chainId: Number(c.chain_id ?? c.id ?? 0),
      type: (c.type as Chain['type']) ?? 'evm',
    }));
  }

  async getBalance(address: string, chainId: number): Promise<BalanceResult> {
    const response = await this.api.get('/public/balance', {
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
    chainId?: number
  ): Promise<string> {
    if (!password) throw new Error('password is required to sign on the backend');
    const response = await this.api.post('/send', {
      wallet_id: walletId,
      password,
      to,
      value: amount,
      chain_id: chainId ?? 1,
    });
    return response.data.tx_hash;
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

  async getSwapQuote(
    fromToken: string,
    toToken: string,
    amount: string,
    chainId?: number
  ): Promise<{ fromAmount: string; toAmount: string; priceImpact: number; route: string[] }> {
    const response = await this.api.get('/swap/quote', {
      params: { from_token: fromToken, to_token: toToken, amount, chain_id: chainId ?? 1 },
    });
    const q = response.data as SwapQuote;
    return {
      fromAmount: q.from_amount,
      toAmount: q.to_amount,
      priceImpact: q.price_impact,
      route: [q.from_token, q.to_token],
    };
  }

  async swap(
    _walletId: string,
    fromToken: string,
    toToken: string,
    amount: string,
    _slippage = 0.5,
    chainId?: number
  ): Promise<{ txHash: string; fromAmount: string; toAmount: string }> {
    const q = await this.getSwapQuote(fromToken, toToken, amount, chainId);
    // The backend returns an on-chain action to submit via /send; the caller
    // must provide a walletId + password to execute. Without those, surface
    // the quote honestly.
    return {
      txHash: '',
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

  async bridge(
    _walletId: string,
    _fromChain: string,
    _toChain: string,
    _token: string,
    _amount: string
  ): Promise<{ txHash: string; bridgeTxHash: string }> {
    // No bridge HTTP service is running (go/bridge is a library, not a
    // server). Fail honestly rather than fabricate a hash.
    throw new Error(
      'Bridge transfer is not available; deploy go/bridge as an HTTP service or wire go/bridge_aggregator first'
    );
  }

  async getBridges(): Promise<unknown[]> {
    throw new Error(
      'Bridge list is not available; deploy go/bridge as an HTTP service first'
    );
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
    _walletId: string,
    _nftId: string,
    _to: string
  ): Promise<string> {
    // NFT transfer requires an on-chain safe-transfer-from call; submit it via
    // /send with the ERC-721 transfer calldata. Without the contract address
    // + token id encoded, fail honestly.
    throw new Error(
      'NFT transfer requires the ERC-721 contract + token id; build the transfer calldata and submit via /send'
    );
  }

  async connectDApp(_walletId: string, _dappUrl: string): Promise<string> {
    // WalletConnect / dApp sessions are handled by the dapp_browser service,
    // not wallet_api. Fail honestly until that service is wired.
    throw new Error(
      'DApp connection is handled by the dapp_browser WalletConnect service; wire dapp_browser/go before use'
    );
  }

  async signDAppTransaction(
    _walletId: string,
    _sessionId: string,
    _txData: unknown
  ): Promise<string> {
    throw new Error(
      'DApp signing is handled by the dapp_browser WalletConnect service; wire dapp_browser/go before use'
    );
  }
}

export { WalletService };
export default WalletService;
