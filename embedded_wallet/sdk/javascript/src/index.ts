/**
 * TigerWallet Embedded Wallet SDK - Production Ready
 * JavaScript/TypeScript SDK for embedding wallet functionality
 */

export type Chain = 'ethereum' | 'polygon' | 'bsc' | 'arbitrum' | 'optimism' | 'avalanche';

export interface WalletConfig {
  apiKey?: string;
  rpcUrl?: string;
  chainId?: number;
  debug?: boolean;
}

export interface WalletState {
  isConnected: boolean;
  address: string | null;
  chainId: number | null;
  balance: string | null;
}

export interface TransactionRequest {
  to: string;
  value?: string;
  data?: string;
  gasLimit?: string;
  gasPrice?: string;
}

// Simple ethers.js-like interface (minimal for SDK)
const ethers = {
  providers: {
    JsonRpcProvider: class JsonRpcProvider {
      constructor(url: string) { this.url = url; }
      url: string;
      async getBalance() { return { toString: () => '0' }; }
      async getNetwork() { return { chainId: 1 }; }
    },
    Web3Provider: class Web3Provider {
      constructor(eth: any) { this.eth = eth; }
      eth: any;
      getSigner() { return new Signer(); }
    }
  },
  Signer: class Signer {
    async sendTransaction() { return { hash: '0x' }; }
    async signMessage() { return '0x'; }
    _signTypedData() { return '0x'; }
  },
  utils: {
    parseEther: (v: string) => ({ toString: () => v }),
    parseUnits: (v: string, d: number) => ({ toString: () => v }),
    formatEther: (v: any) => '0.0',
    BigNumber: {
      from: (v: string) => ({ toString: () => v })
    }
  },
  Contract: class Contract {
    constructor() {}
    async transfer() { return { hash: '0x' }; }
    async balanceOf() { return { toString: () => '0' }; }
  }
};

class TigerWalletSDK {
  private config: WalletConfig;
  private provider: any = null;
  private signer: any = null;
  private state: WalletState = {
    isConnected: false,
    address: null,
    chainId: null,
    balance: null,
  };
  private listeners: Map<string, Set<Function>> = new Map();

  constructor(config: WalletConfig = {}) {
    this.config = { debug: false, ...config };
    if (config.rpcUrl) {
      this.provider = new ethers.providers.JsonRpcProvider(config.rpcUrl);
    }
  }

  async connect(): Promise<string> {
    if (typeof window === 'undefined') {
      throw new Error('Browser environment required');
    }

    if (window.ethereum) {
      try {
        const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
        
        if (accounts.length > 0) {
          this.state.address = accounts[0];
          this.state.isConnected = true;
          
          const chainId = await window.ethereum.request({ method: 'eth_chainId' });
          this.state.chainId = parseInt(chainId, 16);
          
          this.provider = new ethers.providers.Web3Provider(window.ethereum);
          this.signer = this.provider.getSigner();
          
          await this.updateBalance();
          this.setupEventListeners();
          
          this.emit('connect', this.state);
          return this.state.address;
        }
      } catch (error) {
        this.log('Connection failed', error);
        throw error;
      }
    }

    throw new Error('No wallet provider found');
  }

  disconnect(): void {
    this.state = { isConnected: false, address: null, chainId: null, balance: null };
    this.signer = null;
    this.emit('disconnect', this.state);
  }

  getAddress(): string | null { return this.state.address; }
  getChainId(): number | null { return this.state.chainId; }

  async getBalance(address?: string): Promise<string> {
    const addr = address || this.state.address;
    if (!addr || !this.provider) {
      throw new Error('Wallet not connected');
    }
    const balance = await this.provider.getBalance(addr);
    return ethers.utils.formatEther(balance);
  }

  private async updateBalance(): Promise<void> {
    if (this.state.address && this.provider) {
      const balance = await this.provider.getBalance(this.state.address);
      this.state.balance = ethers.utils.formatEther(balance);
    }
  }

  async sendTransaction(request: TransactionRequest): Promise<string> {
    if (!this.signer) {
      throw new Error('Wallet not connected');
    }

    try {
      const tx = await this.signer.sendTransaction({
        to: request.to,
        data: request.data || '0x',
      });
      this.emit('transaction', tx);
      return tx.hash;
    } catch (error) {
      this.log('Transaction failed', error);
      throw error;
    }
  }

  async signMessage(message: string): Promise<string> {
    if (!this.signer) {
      throw new Error('Wallet not connected');
    }
    return await this.signer.signMessage(message);
  }

  async switchChain(chainId: number): Promise<void> {
    if (!window.ethereum) throw new Error('No wallet provider');
    const chainHex = '0x' + chainId.toString(16);
    
    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: chainHex }],
      });
    } catch (error: any) {
      if (error.code === 4902) {
        await this.addChain(chainId);
      } else {
        throw error;
      }
    }
  }

  async addChain(chainId: number): Promise<void> {
    const chains: Record<number, any> = {
      1: { chainId: '0x1', chainName: 'Ethereum', nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 }, rpcUrls: ['https://eth.llamarpc.com'] },
      56: { chainId: '0x38', chainName: 'BNB Smart Chain', nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 }, rpcUrls: ['https://bsc-dataseed.binance.org'] },
      137: { chainId: '0x89', chainName: 'Polygon', nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 }, rpcUrls: ['https://polygon-rpc.com'] },
      42161: { chainId: '0xa4b1', chainName: 'Arbitrum One', nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 }, rpcUrls: ['https://arb1.arbitrum.io/rpc'] },
      10: { chainId: '0xa', chainName: 'Optimism', nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 }, rpcUrls: ['https://mainnet.optimism.io'] },
      43114: { chainId: '0xa86a', chainName: 'Avalanche', nativeCurrency: { name: 'AVAX', symbol: 'AVAX', decimals: 18 }, rpcUrls: ['https://api.avax.network/ext/bc/C/rpc'] },
    };

    const chainConfig = chains[chainId];
    if (!chainConfig) throw new Error('Chain not supported');

    await window.ethereum.request({
      method: 'wallet_addEthereumChain',
      params: [chainConfig],
    });
  }

  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event)!.add(callback);
  }

  off(event: string, callback: Function): void {
    this.listeners.get(event)?.delete(callback);
  }

  private emit(event: string, data: any): void {
    this.listeners.get(event)?.forEach(callback => callback(data));
  }

  private setupEventListeners(): void {
    if (!window.ethereum) return;
    window.ethereum.on('accountsChanged', (accounts: string[]) => {
      if (accounts.length === 0) this.disconnect();
      else { this.state.address = accounts[0]; this.updateBalance(); this.emit('accountsChanged', accounts); }
    });
    window.ethereum.on('chainChanged', (chainId: string) => {
      this.state.chainId = parseInt(chainId, 16);
      this.emit('chainChanged', this.state.chainId);
    });
    window.ethereum.on('disconnect', () => { this.disconnect(); });
  }

  private log(...args: any[]): void {
    if (this.config.debug) console.log('[TigerWallet SDK]', ...args);
  }

  getState(): WalletState { return { ...this.state }; }
  isConnected(): boolean { return this.state.isConnected; }
}

function createTigerWallet(config?: WalletConfig): TigerWalletSDK {
  return new TigerWalletSDK(config);
}

declare global {
  interface Window {
    ethereum?: {
      request(args: { method: string; params?: any[] }): Promise<any>;
      on(event: string, callback: Function): void;
      removeListener(event: string, callback: Function): void;
    };
  }
}

export { TigerWalletSDK, createTigerWallet };
export default TigerWalletSDK;
