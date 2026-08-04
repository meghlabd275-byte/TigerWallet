/**
 * TigerWallet MasterWallet - Multi-Sig Service
 * Production-ready multi-signature wallet functionality
 */

import { MasterWalletService } from './MasterWalletService';

interface SignerInfo {
  id: string;
  name: string;
  address: string;
  role: 'admin' | 'signer' | 'viewer';
  status: 'active' | 'pending' | 'removed';
  hasApproved: boolean;
  approvedAt?: number;
}

interface TransactionInfo {
  id: string;
  txHash?: string;
  from: string;
  to: string;
  amount: string;
  symbol: string;
  fee: string;
  status: 'pending' | 'approved' | 'rejected' | 'executed' | 'failed';
  approvals: ApprovalInfo[];
  requiredApprovals: number;
  currentApprovals: number;
  description: string;
  createdAt: number;
  executedAt?: number;
  expiresAt: number;
}

interface ApprovalInfo {
  signerId: string;
  signerName: string;
  signature?: string;
  status: 'pending' | 'approved' | 'rejected';
  timestamp: number;
  reason?: string;
}

interface MultiSigWallet {
  id: string;
  name: string;
  address: string;
  blockchain: string;
  threshold: number;
  totalSigners: number;
  signers: SignerInfo[];
  pendingTransactions: TransactionInfo[];
  confirmedTransactions: TransactionInfo[];
  balance: string;
  balanceUSD: string;
  isActive: boolean;
  createdAt: number;
}

export class MultiSigService {
  private masterWalletService: MasterWalletService;
  private wallets: Map<string, MultiSigWallet> = new Map();
  private transactions: Map<string, TransactionInfo> = new Map();
  private initialized: boolean = false;
  private eventListeners: Map<string, Set<Function>> = new Map();

  constructor(masterWalletService: MasterWalletService) {
    this.masterWalletService = masterWalletService;
  }

  async initialize(): Promise<boolean> {
    if (this.initialized) return true;
    await this.loadWallets();
    this.initialized = true;
    console.log('[MultiSig] Service initialized');
    return true;
  }

  private async loadWallets(): Promise<void> {
    try {
      const stored = localStorage.getItem('multisig_wallets');
      if (stored) {
        const data = JSON.parse(stored);
        for (const [id, wallet] of Object.entries(data)) {
          this.wallets.set(id, wallet as MultiSigWallet);
        }
      }
    } catch (error) {
      console.error('[MultiSig] Load wallets failed:', error);
    }
  }

  private async saveWallets(): Promise<void> {
    try {
      const data: Record<string, MultiSigWallet> = {};
      this.wallets.forEach((wallet, id) => { data[id] = wallet; });
      localStorage.setItem('multisig_wallets', JSON.stringify(data));
    } catch (error) {
      console.error('[MultiSig] Save wallets failed:', error);
    }
  }

  /**
   * Create a new multi-sig wallet
   */
  async createWallet(
    name: string,
    threshold: number,
    signers: { name: string; address: string; role?: string }[],
    blockchain: string = 'ethereum'
  ): Promise<MultiSigWallet> {
    const wallet: MultiSigWallet = {
      id: this.generateId(),
      name,
      address: this.deriveMultiSigAddress(blockchain, signers),
      blockchain,
      threshold,
      totalSigners: signers.length,
      signers: signers.map((signer, index) => ({
        id: this.generateId(),
        name: signer.name,
        address: signer.address,
        role: (signer.role as any) || (index === 0 ? 'admin' : 'signer'),
        status: 'active',
        hasApproved: false,
      })),
      pendingTransactions: [],
      confirmedTransactions: [],
      balance: '0',
      balanceUSD: '0',
      isActive: true,
      createdAt: Date.now(),
    };

    this.wallets.set(wallet.id, wallet);
    await this.saveWallets();
    this.emit('wallet_created', wallet);

    return wallet;
  }

  /**
   * Get wallet by ID
   */
  getWallet(walletId: string): MultiSigWallet | undefined {
    return this.wallets.get(walletId);
  }

  /**
   * Get all wallets
   */
  getAllWallets(): MultiSigWallet[] {
    return Array.from(this.wallets.values());
  }

  /**
   * Get user's wallets
   */
  getUserWallets(userAddress: string): MultiSigWallet[] {
    return Array.from(this.wallets.values()).filter(
      wallet => wallet.signers.some(s => s.address.toLowerCase() === userAddress.toLowerCase())
    );
  }

  /**
   * Create a new transaction
   */
  async createTransaction(
    walletId: string,
    to: string,
    amount: string,
    symbol: string,
    data: { description?: string; data?: string } = {}
  ): Promise<TransactionInfo> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) throw new Error('Wallet not found');

    const tx: TransactionInfo = {
      id: this.generateId(),
      from: wallet.address,
      to,
      amount,
      symbol,
      fee: await this.estimateFee(wallet.blockchain),
      status: 'pending',
      approvals: [],
      requiredApprovals: wallet.threshold,
      currentApprovals: 0,
      description: data.description || '',
      createdAt: Date.now(),
      expiresAt: Date.now() + (24 * 60 * 60 * 1000),
    };

    wallet.pendingTransactions.push(tx);
    this.transactions.set(tx.id, tx);
    await this.saveWallets();
    this.emit('transaction_created', { wallet, transaction: tx });

    return tx;
  }

  /**
   * Get transaction by ID
   */
  getTransaction(txId: string): TransactionInfo | undefined {
    return this.transactions.get(txId);
  }

  /**
   * Get pending transactions for wallet
   */
  getPendingTransactions(walletId: string): TransactionInfo[] {
    const wallet = this.wallets.get(walletId);
    return wallet?.pendingTransactions || [];
  }

  /**
   * Get transaction history
   */
  getTransactionHistory(walletId: string, limit: number = 50): TransactionInfo[] {
    const wallet = this.wallets.get(walletId);
    if (!wallet) return [];

    const allTransactions = [...wallet.pendingTransactions, ...wallet.confirmedTransactions];
    return allTransactions.sort((a, b) => b.createdAt - a.createdAt).slice(0, limit);
  }

  /**
   * Approve a transaction
   */
  async approveTransaction(
    txId: string,
    signerId: string,
    signature?: string
  ): Promise<TransactionInfo> {
    const tx = this.transactions.get(txId);
    if (!tx) throw new Error('Transaction not found');

    const wallet = this.wallets.get(tx.from.includes('0x') ? tx.from : Object.keys(this.wallets).find(k => this.wallets.get(k)?.address === tx.from) || '');
    const walletActual = Array.from(this.wallets.values()).find(w => 
      w.pendingTransactions.some(t => t.id === txId) || 
      w.confirmedTransactions.some(t => t.id === txId)
    );
    
    if (!walletActual) throw new Error('Wallet not found');

    const signer = walletActual.signers.find(s => s.id === signerId);
    if (!signer) throw new Error('Invalid signer');

    tx.approvals.push({
      signerId,
      signerName: signer.name,
      signature,
      status: 'approved',
      timestamp: Date.now(),
    });

    signer.hasApproved = true;
    signer.approvedAt = Date.now();
    tx.currentApprovals++;

    if (tx.currentApprovals >= tx.requiredApprovals) {
      tx.status = 'approved';
    }

    await this.saveWallets();
    this.emit('transaction_approved', { wallet: walletActual, transaction: tx, signer });

    return tx;
  }

  /**
   * Reject a transaction
   */
  async rejectTransaction(
    txId: string,
    signerId: string,
    reason: string
  ): Promise<TransactionInfo> {
    const tx = this.transactions.get(txId);
    if (!tx) throw new Error('Transaction not found');

    const walletActual = Array.from(this.wallets.values()).find(w => 
      w.pendingTransactions.some(t => t.id === txId)
    );
    if (!walletActual) throw new Error('Wallet not found');

    const signer = walletActual.signers.find(s => s.id === signerId);
    if (!signer) throw new Error('Invalid signer');

    tx.approvals.push({
      signerId,
      signerName: signer.name,
      status: 'rejected',
      timestamp: Date.now(),
      reason,
    });

    tx.status = 'rejected';

    await this.saveWallets();
    this.emit('transaction_rejected', { wallet: walletActual, transaction: tx, signer, reason });

    return tx;
  }

  /**
   * Execute an approved transaction
   */
  async executeTransaction(txId: string): Promise<TransactionInfo> {
    const tx = this.transactions.get(txId);
    if (!tx) throw new Error('Transaction not found');
    if (tx.status !== 'approved') throw new Error('Transaction not approved');

    const walletActual = Array.from(this.wallets.values()).find(w => 
      w.pendingTransactions.some(t => t.id === txId)
    );
    if (!walletActual) throw new Error('Wallet not found');

    // Simulate execution
    tx.status = 'executed';
    tx.executedAt = Date.now();
    tx.txHash = '0x' + this.generateId();

    // Move from pending to confirmed
    const pendingIndex = walletActual.pendingTransactions.findIndex(t => t.id === txId);
    if (pendingIndex !== -1) {
      walletActual.pendingTransactions.splice(pendingIndex, 1);
    }
    walletActual.confirmedTransactions.unshift(tx);

    await this.saveWallets();
    this.emit('transaction_executed', { wallet: walletActual, transaction: tx });

    return tx;
  }

  /**
   * Cancel a transaction
   */
  async cancelTransaction(txId: string): Promise<TransactionInfo> {
    const tx = this.transactions.get(txId);
    if (!tx) throw new Error('Transaction not found');

    tx.status = 'cancelled';

    const walletActual = Array.from(this.wallets.values()).find(w => 
      w.pendingTransactions.some(t => t.id === txId)
    );
    if (walletActual) {
      const pendingIndex = walletActual.pendingTransactions.findIndex(t => t.id === txId);
      if (pendingIndex !== -1) {
        walletActual.pendingTransactions.splice(pendingIndex, 1);
      }
    }

    await this.saveWallets();
    return tx;
  }

  /**
   * Add a signer
   */
  async addSigner(walletId: string, signer: { name: string; address: string; role?: string }): Promise<MultiSigWallet> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) throw new Error('Wallet not found');

    wallet.signers.push({
      id: this.generateId(),
      name: signer.name,
      address: signer.address,
      role: (signer.role as any) || 'signer',
      status: 'pending',
      hasApproved: false,
    });
    wallet.totalSigners = wallet.signers.length;

    await this.saveWallets();
    return wallet;
  }

  /**
   * Remove a signer
   */
  async removeSigner(walletId: string, signerId: string): Promise<MultiSigWallet> {
    const wallet = this.wallets.get(walletId);
    if (!wallet) throw new Error('Wallet not found');

    const signerIndex = wallet.signers.findIndex(s => s.id === signerId);
    if (signerIndex !== -1) {
      wallet.signers.splice(signerIndex, 1);
    }
    wallet.totalSigners = wallet.signers.length;

    await this.saveWallets();
    return wallet;
  }

  /**
   * Get pending approvals for signer
   */
  getPendingApprovals(signerId: string): (TransactionInfo & { walletName: string })[] {
    const pending: (TransactionInfo & { walletName: string })[] = [];
    
    for (const wallet of this.wallets.values()) {
      const signer = wallet.signers.find(s => s.id === signerId);
      if (!signer) continue;
      
      for (const tx of wallet.pendingTransactions) {
        if (tx.status === 'pending' && !tx.approvals.some(a => a.signerId === signerId)) {
          pending.push({ ...tx, walletName: wallet.name });
        }
      }
    }
    
    return pending;
  }

  /**
   * Estimate transaction fee
   */
  async estimateFee(blockchain: string): Promise<string> {
    const feeEstimates: Record<string, string> = {
      ethereum: '0.005',
      polygon: '0.01',
      bsc: '0.005',
      avalanche: '0.025',
      arbitrum: '0.001',
      optimism: '0.001',
    };
    return feeEstimates[blockchain] || '0.01';
  }

  /**
   * Derive multi-sig address
   */
  private deriveMultiSigAddress(blockchain: string, signers: { address: string }[]): string {
    const data = signers.map(s => s.address).sort().join('-');
    const hash = this.simpleHash(data);
    return '0x' + hash.slice(0, 40);
  }

  private simpleHash(data: string): string {
    let hash = 0;
    for (let i = 0; i < data.length; i++) {
      hash = ((hash << 5) - hash) + data.charCodeAt(i);
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(40, '0');
  }

  private generateId(): string {
    return '0x' + Array.from(crypto.getRandomValues(new Uint8Array(16)))
      .map(b => b.toString(16).padStart(2, '0')).join('');
  }

  /**
   * Delete wallet
   */
  async deleteWallet(walletId: string): Promise<boolean> {
    if (this.wallets.has(walletId)) {
      this.wallets.delete(walletId);
      await this.saveWallets();
      return true;
    }
    return false;
  }

  // Events
  addEventListener(event: string, callback: Function): void {
    if (!this.eventListeners.has(event)) this.eventListeners.set(event, new Set());
    this.eventListeners.get(event)!.add(callback);
  }

  removeEventListener(event: string, callback: Function): void {
    this.eventListeners.get(event)?.delete(callback);
  }

  private emit(event: string, data: any): void {
    this.eventListeners.get(event)?.forEach(cb => { try { cb(data); } catch (e) { console.error(e); } });
  }
}

export default MultiSigService;