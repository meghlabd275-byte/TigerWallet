/**
 * TigerWallet - Hardware Wallet Service
 * Complete Ledger and Trezor integration for web app
 */

import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';

// ============================================================================
// Types
// ============================================================================

export interface HardwareWallet {
  id: string;
  type: 'ledger' | 'trezor';
  deviceName: string;
  addresses: WalletAddress[];
  isConnected: boolean;
  createdAt: string;
}

export interface WalletAddress {
  chain: string;
  chainId: number;
  address: string;
  derivation: string;
}

export interface DeviceInfo {
  type: string;
  deviceName: string;
  firmware: string;
  model: string;
  serial: string;
  capabilities: string[];
  chains: string[];
}

export interface TransactionRequest {
  chain: string;
  to: string;
  amount: string;
  tokenAddress?: string;
  gasLimit?: string;
  gasPrice?: string;
}

export interface SignedTransaction {
  txHash: string;
  rawTx: string;
  signature: string;
}

// ============================================================================
// Hardware Wallet Service
// ============================================================================

export class HardwareWalletService {
  private static instance: HardwareWalletService;
  private connectedDevice: DeviceInfo | null = null;

  static getInstance(): HardwareWalletService {
    if (!HardwareWalletService.instance) {
      HardwareWalletService.instance = new HardwareWalletService();
    }
    return HardwareWalletService.instance;
  }

  /**
   * Detect connected hardware wallet
   */
  static async detectDevice(): Promise<DeviceInfo | null> {
    try {
      const response = await axios.get(`${API_BASE_URL}/hardware-wallet/detect`);
      return response.data;
    } catch (error) {
      console.error('Failed to detect device:', error);
      return null;
    }
  }

  /**
   * Connect hardware wallet
   */
  static async connect(type: 'ledger' | 'trezor'): Promise<HardwareWallet> {
    try {
      const response = await axios.post(`${API_BASE_URL}/hardware-wallet/register`, {
        type
      });
      return response.data;
    } catch (error) {
      console.error('Failed to connect wallet:', error);
      throw error;
    }
  }

  /**
   * Get connected wallets
   */
  static async getWallets(): Promise<HardwareWallet[]> {
    try {
      const response = await axios.get(`${API_BASE_URL}/hardware-wallet/wallets`);
      return response.data.wallets;
    } catch (error) {
      console.error('Failed to get wallets:', error);
      return [];
    }
  }

  /**
   * Get wallet by ID
   */
  static async getWallet(walletId: string): Promise<HardwareWallet | null> {
    try {
      const response = await axios.get(`${API_BASE_URL}/hardware-wallet/wallets/${walletId}`);
      return response.data;
    } catch (error) {
      console.error('Failed to get wallet:', error);
      return null;
    }
  }

  /**
   * Sign transaction
   */
  static async signTransaction(walletId: string, tx: TransactionRequest): Promise<SignedTransaction> {
    try {
      const response = await axios.post(`${API_BASE_URL}/hardware-wallet/sign-tx`, {
        wallet_id: walletId,
        ...tx
      });
      return response.data;
    } catch (error) {
      console.error('Failed to sign transaction:', error);
      throw error;
    }
  }

  /**
   * Sign message
   */
  static async signMessage(walletId: string, chain: string, message: string): Promise<string> {
    try {
      const response = await axios.post(`${API_BASE_URL}/hardware-wallet/sign-message`, {
        wallet_id: walletId,
        chain,
        message
      });
      return response.data.signature;
    } catch (error) {
      console.error('Failed to sign message:', error);
      throw error;
    }
  }

  /**
   * Disconnect wallet
   */
  static async disconnect(walletId: string): Promise<void> {
    try {
      await axios.post(`${API_BASE_URL}/hardware-wallet/wallets/${walletId}/disconnect`);
    } catch (error) {
      console.error('Failed to disconnect:', error);
      throw error;
    }
  }

  /**
   * Get supported chains
   */
  static getSupportedChains(): string[] {
    return [
      'ethereum',
      'polygon',
      'arbitrum',
      'optimism',
      'avalanche',
      'bsc',
      'base',
      'solana',
      'bitcoin'
    ];
  }

  /**
   * Get derivation paths for chains
   */
  static getDerivationPaths(): Record<string, string> {
    return {
      ethereum: "m/44'/60'/0'/0/0",
      polygon: "m/44'/60'/0'/0/0",
      arbitrum: "m/44'/60'/0'/0/0",
      optimism: "m/44'/60'/0'/0/0",
      avalanche: "m/44'/60'/0'/0/0",
      bsc: "m/44'/60'/0'/0/0",
      base: "m/44'/60'/0'/0/0",
      solana: "m/44'/501'/0'/0'",
      bitcoin: "m/84'/0'/0'/0/0"
    };
  }
}

// ============================================================================
// React Hook
// ============================================================================

import { useState, useCallback, useEffect } from 'react';

export const useHardwareWallet = () => {
  const [wallets, setWallets] = useState<HardwareWallet[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadWallets = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const walletList = await HardwareWalletService.getWallets();
      setWallets(walletList);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const connect = useCallback(async (type: 'ledger' | 'trezor') => {
    setIsLoading(true);
    setError(null);
    try {
      await HardwareWalletService.connect(type);
      await loadWallets();
    } catch (err: any) {
      setError(err.message);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [loadWallets]);

  const disconnect = useCallback(async (walletId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      await HardwareWalletService.disconnect(walletId);
      await loadWallets();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  }, [loadWallets]);

  const signTransaction = useCallback(async (walletId: string, tx: TransactionRequest) => {
    setIsLoading(true);
    setError(null);
    try {
      return await HardwareWalletService.signTransaction(walletId, tx);
    } catch (err: any) {
      setError(err.message);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadWallets();
  }, [loadWallets]);

  return {
    wallets,
    isLoading,
    error,
    loadWallets,
    connect,
    disconnect,
    signTransaction,
    supportedChains: HardwareWalletService.getSupportedChains()
  };
};

export default HardwareWalletService;
