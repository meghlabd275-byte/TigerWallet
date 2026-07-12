// Mock wallet hook for development
import { useState, useEffect } from 'react';

export interface WalletState {
  isConnected: boolean;
  address: string | null;
  chainId: number;
  balance: string;
}

export function useWallet() {
  const [state, setState] = useState<WalletState>({
    isConnected: false,
    address: null,
    chainId: 1,
    balance: '0',
  });

  const connect = async () => {
    // Mock connection
    setState({
      isConnected: true,
      address: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E',
      chainId: 1,
      balance: '1.5',
    });
  };

  const disconnect = () => {
    setState({
      isConnected: false,
      address: null,
      chainId: 1,
      balance: '0',
    });
  };

  return {
    ...state,
    connect,
    disconnect,
  };
}
