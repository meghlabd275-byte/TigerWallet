/**
 * Wallet Service - Real Backend Integration
 * Connects to Go backend for blockchain operations
 */

import axios, { AxiosInstance } from 'axios';
import { ethers } from 'ethers';
import { Connection, Transaction, PublicKey } from '@solana/web3.js';

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
  signTransaction(tx: any): Promise<string>;
}

// API Base URL - would be configured per environment
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

class WalletService {
  private api: AxiosInstance;
  private providers: Map<string, ethers.JsonRpcProvider> = new Map();
  private solanaConnections: Map<string, Connection> = new Map();

  constructor() {
    this.api = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth interceptor
    this.api.interceptors.request.use((config) => {
      const token = localStorage.getItem('tigerwallet-token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });
  }

  // Get all supported chains
  async getChains(): Promise<Chain[]> {
    const response = await this.api.get('/chains');
    return response.data;
  }

  // Get user's wallets
  async getWallets(): Promise<Wallet[]> {
    const response = await this.api.get('/wallets');
    return response.data.wallets;
  }

  // Create new wallet
  async createWallet(mnemonic: string, password: string, chain: Chain): Promise<Wallet> {
    const response = await this.api.post('/wallets', {
      mnemonic,
      password,
      chain: chain.id,
    });
    return response.data.wallet;
  }

  // Import wallet from private key
  async importPrivateKey(privateKey: string, chain: Chain): Promise<Wallet> {
    const response = await this.api.post('/wallets/import', {
      privateKey,
      chain: chain.id,
    });
    return response.data.wallet;
  }

  // Import wallet from mnemonic
  async importFromMnemonic(mnemonic: string, password: string, chain: Chain): Promise<Wallet> {
    const response = await this.api.post('/wallets/import-mnemonic', {
      mnemonic,
      password,
      chain: chain.id,
    });
    return response.data.wallet;
  }

  // Get wallet for specific chain
  async getWalletForChain(walletId: string, chain: Chain): Promise<Wallet> {
    const response = await this.api.get(`/wallets/${walletId}/chain/${chain.id}`);
    return response.data.wallet;
  }

  // Refresh wallet balances
  async refreshBalances(walletId: string): Promise<Wallet> {
    const response = await this.api.post(`/wallets/${walletId}/refresh`);
    return response.data.wallet;
  }

  // Send transaction
  async sendTransaction(
    walletId: string,
    to: string,
    amount: string,
    token?: string
  ): Promise<string> {
    const response = await this.api.post(`/wallets/${walletId}/send`, {
      to,
      amount,
      token,
    });
    return response.data.txHash;
  }

  // Sign message
  async signMessage(walletId: string, message: string): Promise<string> {
    const response = await this.api.post(`/wallets/${walletId}/sign`, {
      message,
    });
    return response.data.signature;
  }

  // Get transaction history
  async getTransactions(walletId: string, page = 1, limit = 20): Promise<Transaction[]> {
    const response = await this.api.get(`/wallets/${walletId}/transactions`, {
      params: { page, limit },
    });
    return response.data.transactions;
  }

  // Get gas price for chain
  async getGasPrice(chainId: string): Promise<string> {
    const response = await this.api.get(`/chains/${chainId}/gas-price`);
    return response.data.gasPrice;
  }

  // Estimate gas
  async estimateGas(
    chainId: string,
    from: string,
    to: string,
    value: string,
    data?: string
  ): Promise<string> {
    const response = await this.api.post(`/chains/${chainId}/estimate-gas`, {
      from,
      to,
      value,
      data,
    });
    return response.data.gasEstimate;
  }

  // Get token balance
  async getTokenBalance(walletAddress: string, tokenAddress: string, chainId: string): Promise<string> {
    const response = await this.api.get(`/wallets/${walletAddress}/token/${tokenAddress}`, {
      params: { chain: chainId },
    });
    return response.data.balance;
  }

  // Swap tokens
  async swap(
    walletId: string,
    fromToken: string,
    toToken: string,
    amount: string,
    slippage: number = 0.5
  ): Promise<{ txHash: string; fromAmount: string; toAmount: string }> {
    const response = await this.api.post(`/wallets/${walletId}/swap`, {
      fromToken,
      toToken,
      amount,
      slippage,
    });
    return response.data;
  }

  // Get swap quote
  async getSwapQuote(
    fromToken: string,
    toToken: string,
    amount: string
  ): Promise<{ fromAmount: string; toAmount: string; priceImpact: number; route: string[] }> {
    const response = await this.api.get('/swap/quote', {
      params: { fromToken, toToken, amount },
    });
    return response.data;
  }

  // Stake tokens
  async stake(
    walletId: string,
    token: string,
    amount: string,
    validator?: string
  ): Promise<{ txHash: string; stakedAmount: string }> {
    const response = await this.api.post(`/wallets/${walletId}/stake`, {
      token,
      amount,
      validator,
    });
    return response.data;
  }

  // Unstake tokens
  async unstake(
    walletId: string,
    token: string,
    amount: string
  ): Promise<{ txHash: string }> {
    const response = await this.api.post(`/wallets/${walletId}/unstake`, {
      token,
      amount,
    });
    return response.data;
  }

  // Get staking positions
  async getStakingPositions(walletId: string): Promise<any[]> {
    const response = await this.api.get(`/wallets/${walletId}/staking`);
    return response.data.positions;
  }

  // Bridge tokens
  async bridge(
    walletId: string,
    fromChain: string,
    toChain: string,
    token: string,
    amount: string
  ): Promise<{ txHash: string; bridgeTxHash: string }> {
    const response = await this.api.post(`/wallets/${walletId}/bridge`, {
      fromChain,
      toChain,
      token,
      amount,
    });
    return response.data;
  }

  // Get supported bridges
  async getBridges(): Promise<any[]> {
    const response = await this.api.get('/bridges');
    return response.data.bridges;
  }

  // NFT operations
  async getNFTs(walletId: string): Promise<any[]> {
    const response = await this.api.get(`/wallets/${walletId}/nfts`);
    return response.data.nfts;
  }

  async transferNFT(
    walletId: string,
    nftId: string,
    to: string
  ): Promise<string> {
    const response = await this.api.post(`/wallets/${walletId}/nft/transfer`, {
      nftId,
      to,
    });
    return response.data.txHash;
  }

  // DApp connection
  async connectDApp(walletId: string, dappUrl: string): Promise<string> {
    const response = await this.api.post(`/wallets/${walletId}/dapp/connect`, {
      dappUrl,
    });
    return response.data.sessionId;
  }

  async signDAppTransaction(
    walletId: string,
    sessionId: string,
    txData: any
  ): Promise<string> {
    const response = await this.api.post(`/wallets/${walletId}/dapp/sign`, {
      sessionId,
      txData,
    });
    return response.data.txHash;
  }

  // Utility methods
  private getProvider(chain: Chain): ethers.JsonRpcProvider {
    if (!this.providers.has(chain.id)) {
      const provider = new ethers.JsonRpcProvider(chain.rpcUrl);
      this.providers.set(chain.id, provider);
    }
    return this.providers.get(chain.id)!;
  }

  private getSolanaConnection(rpcUrl: string): Connection {
    if (!this.solanaConnections.has(rpcUrl)) {
      const connection = new Connection(rpcUrl);
      this.solanaConnections.set(rpcUrl, connection);
    }
    return this.solanaConnections.get(rpcUrl)!;
  }
}

export default WalletService;
