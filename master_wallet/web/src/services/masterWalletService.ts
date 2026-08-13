/**
 * MasterWalletService - Web (React/TypeScript)
 * Real BIP-39 mnemonic generation (ethers.js) + backend wiring for wallet
 * creation, balance lookups, and transaction signing/broadcast. No fake
 * balances, no fabricated transaction hashes — all on-chain state comes from
 * the canonical backend (port 8450).
 */

import { ethers } from 'ethers';
import {
  masterWalletAPI,
  BalanceResponse,
  SignTransactionResponse,
} from '../api';

export interface ChainConfig {
  id: number;
  name: string;
  symbol: string;
  decimals: number;
}

// Curated EVM chain metadata (display/symbol only; RPC + signing live on the
// backend). Keep in sync with backend chains.go.
const CHAIN_CONFIGS: Record<number, ChainConfig> = {
  1: { id: 1, name: 'Ethereum', symbol: 'ETH', decimals: 18 },
  56: { id: 56, name: 'BNB Smart Chain', symbol: 'BNB', decimals: 18 },
  137: { id: 137, name: 'Polygon', symbol: 'POL', decimals: 18 },
  42161: { id: 42161, name: 'Arbitrum One', symbol: 'ETH', decimals: 18 },
  10: { id: 10, name: 'Optimism', symbol: 'ETH', decimals: 18 },
  43114: { id: 43114, name: 'Avalanche C-Chain', symbol: 'AVAX', decimals: 18 },
  8453: { id: 8453, name: 'Base', symbol: 'ETH', decimals: 18 },
  42220: { id: 42220, name: 'Celo', symbol: 'CELO', decimals: 18 },
  250: { id: 250, name: 'Fantom', symbol: 'FTM', decimals: 18 },
  25: { id: 25, name: 'Cronos', symbol: 'CRO', decimals: 18 },
};

export interface GeneratedWallet {
  walletId: string;
  address: string;
  mnemonic: string;
}

export interface SendResult {
  transaction_hash: string;
  status: string;
  from?: string;
  chain_id?: number;
}

class MasterWalletService {
  /**
   * Generate a fresh BIP-39 mnemonic + derived EVM address (m/44'/60'/0'/0/0).
   * Used to preview a mnemonic before POSTing it to the backend for creation.
   */
  generateMnemonic(): { mnemonic: string; address: string } {
    const wallet = ethers.Wallet.createRandom();
    const phrase = wallet.mnemonic?.phrase ?? '';
    if (!phrase) {
      throw new Error('Failed to generate mnemonic');
    }
    const masterNode = ethers.HDNodeWallet.fromMnemonic(ethers.Mnemonic.fromPhrase(phrase));
    const derived = masterNode.derivePath("m/44'/60'/0'/0/0");
    return { mnemonic: phrase, address: derived.address };
  }

  /**
   * Validate a BIP-39 mnemonic (real checksum via ethers.js).
   */
  isValidMnemonic(mnemonic: string): boolean {
    return ethers.Mnemonic.isValidMnemonic(mnemonic);
  }

  /**
   * Derive the EVM address for an existing mnemonic (m/44'/60'/0'/0/0).
   */
  deriveAddressFromMnemonic(mnemonic: string): string {
    if (!ethers.Mnemonic.isValidMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic');
    }
    const masterNode = ethers.HDNodeWallet.fromMnemonic(
      ethers.Mnemonic.fromPhrase(mnemonic)
    );
    return masterNode.derivePath("m/44'/60'/0'/0/0").address;
  }

  /**
   * Create a master wallet on the backend. The backend generates (or imports
   * the provided) mnemonic, derives the key, encrypts the seed with the
   * password, persists, and returns the mnemonic once.
   */
  async createMasterWallet(
    name: string,
    password: string,
    chainId: number,
    mnemonic?: string
  ): Promise<GeneratedWallet> {
    const res = await masterWalletAPI.createMasterWallet(
      name,
      password,
      chainId,
      mnemonic
    );
    // backend returns { wallet_id, address, mnemonic, ... }
    const anyRes = res as unknown as {
      wallet_id?: string;
      id?: string;
      address: string;
      mnemonic?: string;
    };
    const walletId = anyRes.wallet_id ?? anyRes.id ?? '';
    return { walletId, address: anyRes.address, mnemonic: anyRes.mnemonic ?? '' };
  }

  /**
   * Import an existing mnemonic by creating a backend master wallet with it.
   */
  async importWallet(
    mnemonic: string,
    password: string,
    chainId: number,
    name = 'Imported Wallet'
  ): Promise<GeneratedWallet> {
    if (!this.isValidMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic');
    }
    return this.createMasterWallet(name, password, chainId, mnemonic);
  }

  /**
   * Fetch the real native + token balances for a master wallet from the
   * backend (real RPC, no client-side providers).
   */
  async getBalance(walletId: string): Promise<BalanceResponse> {
    return masterWalletAPI.getMasterWalletBalance(walletId);
  }

  /**
   * Fetch the real balance for a sub-wallet from the backend.
   */
  async getSubWalletBalance(
    masterId: string,
    subId: string
  ): Promise<BalanceResponse> {
    return masterWalletAPI.getSubWalletBalance(masterId, subId);
  }

  /**
   * Sign + broadcast a transaction through the backend (real secp256k1 signing
   * with the password-decrypted key). Returns the real on-chain tx hash.
   */
  async sendTransaction(
    walletId: string,
    to: string,
    amount: string,
    password: string,
    token?: string
  ): Promise<SendResult> {
    const res: SignTransactionResponse = await masterWalletAPI.signTransaction(
      walletId,
      { to, amount, password, token }
    );
    return {
      transaction_hash: res.transaction_hash,
      status: res.status,
      from: res.from,
      chain_id: res.chain_id,
    };
  }

  /**
   * Sign + broadcast from a sub-wallet via the backend.
   */
  async sendFromSubWallet(
    masterId: string,
    subId: string,
    to: string,
    amount: string,
    password: string,
    token?: string
  ): Promise<SendResult> {
    const res = await masterWalletAPI.transferFromSubWallet(masterId, subId, {
      to,
      amount,
      password,
      token,
    });
    return {
      transaction_hash: res.transaction_hash,
      status: res.status,
      from: res.from,
      chain_id: res.chain_id,
    };
  }

  /**
   * Supported chains metadata (display only; RPC resolution is server-side).
   */
  getSupportedChains(): ChainConfig[] {
    return Object.values(CHAIN_CONFIGS).sort((a, b) => a.id - b.id);
  }

  getChainConfig(chainId: number): ChainConfig | undefined {
    return CHAIN_CONFIGS[chainId];
  }
}

export const masterWalletService = new MasterWalletService();
export default masterWalletService;
