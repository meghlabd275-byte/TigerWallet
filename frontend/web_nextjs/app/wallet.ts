'use client';

import { useState, useEffect, useCallback } from 'react';
import { ethers } from 'ethers';

// EIP-1193 provider shape (MetaMask / injected wallets).
interface Eip1193Provider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  on?: (event: string, handler: (...args: unknown[]) => void) => void;
  removeListener?: (event: string, handler: (...args: unknown[]) => void) => void;
  isMetaMask?: boolean;
  isCoinbaseWallet?: boolean;
}

declare global {
  interface Window {
    ethereum?: Eip1193Provider;
  }
}

export interface WalletState {
  isConnected: boolean;
  address: string | null;
  chainId: number;
  balance: string;
}

const DISCONNECTED: WalletState = {
  isConnected: false,
  address: null,
  chainId: 1,
  balance: '0',
};

function getProvider(): Eip1193Provider | null {
  if (typeof window === 'undefined') return null;
  const eth = window.ethereum;
  if (eth && typeof eth.request === 'function') return eth;
  return null;
}

// Reads the live balance for `address` directly from the node via eth_getBalance.
// Returns ether as a decimal string. Falls back to '0' if the call fails.
async function fetchBalance(provider: Eip1193Provider, address: string): Promise<string> {
  try {
    const balHex = (await provider.request({
      method: 'eth_getBalance',
      params: [address, 'latest'],
    })) as string;
    const wei = BigInt(balHex);
    return ethers.utils.formatEther(wei);
  } catch {
    return '0';
  }
}

export function useWallet() {
  const [state, setState] = useState<WalletState>(DISCONNECTED);

  const refreshBalance = useCallback(async (address: string, chainId: number) => {
    const provider = getProvider();
    if (!provider) return;
    const balance = await fetchBalance(provider, address);
    setState({ isConnected: true, address, chainId, balance });
  }, []);

  // On mount, silently check if the wallet already permitted access (no popup).
  useEffect(() => {
    const provider = getProvider();
    if (!provider) return;

    (async () => {
      try {
        const accounts = (await provider.request({ method: 'eth_accounts' })) as string[];
        if (accounts && accounts.length > 0) {
          const chainIdHex = (await provider.request({ method: 'eth_chainId' })) as string;
          const chainId = Number(chainIdHex);
          const balance = await fetchBalance(provider, accounts[0]);
          setState({ isConnected: true, address: accounts[0], chainId, balance });
        }
      } catch {
        // Not connected; leave default state.
      }
    })();

    const handleAccountsChanged = (...args: unknown[]) => {
      const accounts = args[0] as string[];
      if (!accounts || accounts.length === 0) {
        setState(DISCONNECTED);
      } else {
        refreshBalance(accounts[0], state.chainId);
      }
    };

    const handleChainChanged = (...args: unknown[]) => {
      const chainIdHex = args[0] as string;
      const chainId = Number(chainIdHex);
      if (state.address) refreshBalance(state.address, chainId);
      else setState((s) => ({ ...s, chainId }));
    };

    if (provider.on) {
      provider.on('accountsChanged', handleAccountsChanged);
      provider.on('chainChanged', handleChainChanged);
    }

    return () => {
      if (provider.removeListener) {
        provider.removeListener('accountsChanged', handleAccountsChanged);
        provider.removeListener('chainChanged', handleChainChanged);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const connect = useCallback(async () => {
    const provider = getProvider();
    if (!provider) {
      throw new Error('No Ethereum wallet found. Please install MetaMask or a compatible wallet.');
    }

    const accounts = (await provider.request({ method: 'eth_requestAccounts' })) as string[];
    if (!accounts || accounts.length === 0) {
      throw new Error('No accounts returned by the wallet.');
    }

    const chainIdHex = (await provider.request({ method: 'eth_chainId' })) as string;
    const chainId = Number(chainIdHex);
    const balance = await fetchBalance(provider, accounts[0]);

    setState({ isConnected: true, address: accounts[0], chainId, balance });
  }, []);

  const disconnect = useCallback(() => {
    // EIP-1193 has no revoke permission call; we clear local state so the UI
    // reflects a disconnected wallet and the next connect re-prompts.
    setState(DISCONNECTED);
  }, []);

  return {
    ...state,
    connect,
    disconnect,
    refreshBalance,
  };
}
