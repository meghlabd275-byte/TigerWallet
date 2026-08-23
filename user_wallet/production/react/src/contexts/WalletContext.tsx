/**
 * Wallet Context - Complete Wallet Management
 * Integrates with backend for real blockchain operations
 */

import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import { WalletService, Wallet, Chain, Token, Transaction, Signer } from '../services/WalletService';
import { useAuth } from './AuthContext';

// Types
export interface WalletAccount {
  id: string;
  address: string;
  chain: Chain;
  balance: string;
  balanceUSD: number;
  tokens: Token[];
}

export interface WalletState {
  wallets: WalletAccount[];
  activeWallet: WalletAccount | null;
  isLoading: boolean;
  error: string | null;
}

// Optional EIP-1559 fee overrides (gwei strings) forwarded to /send.
export interface FeeOverrides {
  maxFeeGwei?: string;
  maxPriorityGwei?: string;
}

interface WalletContextType extends WalletState {
  createWallet: (mnemonic: string | undefined, password: string, chain: Chain) => Promise<Wallet>;
  importWallet: (mnemonic: string, password: string, chain: Chain) => Promise<void>;
  importFromMnemonic: (mnemonic: string, password: string, chain: Chain) => Promise<void>;
  switchChain: (chain: Chain) => Promise<void>;
  refreshBalances: () => Promise<void>;
  sendTransaction: (to: string, amount: string, token?: string, fees?: FeeOverrides) => Promise<string>;
  getAddress: (chain: Chain) => string;
  signMessage: (message: string) => Promise<string>;
}

const WalletContext = createContext<WalletContextType | undefined>(undefined);

export function WalletProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const [wallets, setWallets] = useState<WalletAccount[]>([]);
  const [activeWallet, setActiveWallet] = useState<WalletAccount | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [walletService] = useState(() => new WalletService());

  // Initialize wallet on mount
  useEffect(() => {
    if (user) {
      loadWallets();
    }
  }, [user]);

  const loadWallets = async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const loadedWallets = await walletService.getWallets();
      setWallets(loadedWallets);
      
      if (loadedWallets.length > 0 && !activeWallet) {
        setActiveWallet(loadedWallets[0]);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load wallets');
    } finally {
      setIsLoading(false);
    }
  };

  const createWallet = useCallback(async (mnemonic: string | undefined, password: string, chain: Chain) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const newWallet = await walletService.createWallet(mnemonic, password, chain);
      setWallets(prev => [...prev, newWallet]);
      setActiveWallet(newWallet);
      return newWallet;
    } catch (err: any) {
      setError(err.message || 'Failed to create wallet');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [walletService]);

  const importWallet = useCallback(async (mnemonic: string, password: string, chain: Chain) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const importedWallet = await walletService.importFromMnemonic(mnemonic, password, chain);
      setWallets(prev => [...prev, importedWallet]);
      setActiveWallet(importedWallet);
    } catch (err: any) {
      setError(err.message || 'Failed to import wallet');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [walletService]);

  const importFromMnemonic = useCallback(async (mnemonic: string, password: string, chain: Chain) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const importedWallet = await walletService.importFromMnemonic(mnemonic, password, chain);
      setWallets(prev => [...prev, importedWallet]);
      setActiveWallet(importedWallet);
    } catch (err: any) {
      setError(err.message || 'Failed to import wallet');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [walletService]);

  const switchChain = useCallback(async (chain: Chain) => {
    if (!activeWallet) return;
    
    setIsLoading(true);
    try {
      const walletForChain = await walletService.getWalletForChain(activeWallet.id, chain);
      if (walletForChain) {
        setActiveWallet(walletForChain);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to switch chain');
    } finally {
      setIsLoading(false);
    }
  }, [activeWallet, walletService]);

  const refreshBalances = useCallback(async () => {
    if (!activeWallet) return;
    
    setIsLoading(true);
    try {
      const updatedWallet = await walletService.refreshBalances(activeWallet.id);
      setWallets(prev => prev.map(w => w.id === activeWallet.id ? updatedWallet : w));
      setActiveWallet(updatedWallet);
    } catch (err: any) {
      setError(err.message || 'Failed to refresh balances');
    } finally {
      setIsLoading(false);
    }
  }, [activeWallet, walletService]);

  const sendTransaction = useCallback(async (to: string, amount: string, token?: string, fees?: FeeOverrides): Promise<string> => {
    if (!activeWallet) throw new Error('No active wallet');

    setIsLoading(true);
    setError(null);

    try {
      const txHash = await walletService.sendTransaction(
        activeWallet.id,
        to,
        amount,
        token,
        undefined,
        undefined,
        undefined,
        fees?.maxFeeGwei,
        fees?.maxPriorityGwei
      );
      
      // Refresh balances after transaction
      await refreshBalances();
      
      return txHash;
    } catch (err: any) {
      setError(err.message || 'Failed to send transaction');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [activeWallet, walletService, refreshBalances]);

  const getAddress = useCallback((chain: Chain): string => {
    if (!activeWallet) return '';
    return activeWallet.address;
  }, [activeWallet]);

  const signMessage = useCallback(async (message: string): Promise<string> => {
    if (!activeWallet) throw new Error('No active wallet');
    
    try {
      return await walletService.signMessage(activeWallet.id, message);
    } catch (err: any) {
      setError(err.message || 'Failed to sign message');
      throw err;
    }
  }, [activeWallet, walletService]);

  return (
    <WalletContext.Provider value={{
      wallets,
      activeWallet,
      isLoading,
      error,
      createWallet,
      importWallet,
      importFromMnemonic,
      switchChain,
      refreshBalances,
      sendTransaction,
      getAddress,
      signMessage,
    }}>
      {children}
    </WalletContext.Provider>
  );
}

export function useWallet() {
  const context = useContext(WalletContext);
  if (context === undefined) {
    throw new Error('useWallet must be used within a WalletProvider');
  }
  return context;
}

export default WalletContext;
