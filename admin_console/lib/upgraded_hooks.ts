// TigerWallet Admin Console - UPGRADED HOOKS
// React 19 compatible with Next.js 15, optimized for performance

import { useQuery, useMutation } from '@tanstack/react-query';
import { useCallback, useEffect, useState } from 'react';
import { ethers } from 'ethers';

/**
 * Hook to fetch wallet accounts
 */
export const useWalletAccounts = (walletId: string) => {
  return useQuery({
    queryKey: ['wallet-accounts', walletId],
    queryFn: async () => {
      const response = await fetch(`/api/wallets/${walletId}/accounts`);
      if (!response.ok) throw new Error('Failed to fetch accounts');
      return response.json();
    },
    staleTime: 30000,
    gcTime: 5 * 60 * 1000,
  });
};

/**
 * Hook to fetch portfolio balance with cache
 */
export const usePortfolioBalance = (walletId: string, refreshInterval?: number) => {
  return useQuery({
    queryKey: ['portfolio-balance', walletId],
    queryFn: async () => {
      const response = await fetch(`/api/wallets/${walletId}/portfolio`);
      if (!response.ok) throw new Error('Failed to fetch portfolio');
      return response.json();
    },
    refetchInterval: refreshInterval || 60000,
    staleTime: 10000,
  });
};

/**
 * Hook to execute transaction with retry logic
 */
export const useExecuteTransaction = () => {
  return useMutation({
    mutationFn: async (params: {
      walletId: string;
      to: string;
      value: string;
      data?: string;
      gasLimit?: string;
    }) => {
      const response = await fetch('/api/transactions/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });
      if (!response.ok) throw new Error('Transaction failed');
      return response.json();
    },
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};

/**
 * Hook to swap tokens via DEX
 */
export const useSwapTokens = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const swap = useCallback(
    async (params: {
      fromToken: string;
      toToken: string;
      amount: string;
      slippageTolerance?: number;
      walletId: string;
    }) => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/swap', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(params),
        });

        if (!response.ok) {
          throw new Error('Swap failed');
        }

        const data = await response.json();
        return data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return { swap, loading, error };
};

/**
 * Hook for gas estimation with network detection
 */
export const useGasEstimate = (chainId: number) => {
  return useQuery({
    queryKey: ['gas-estimate', chainId],
    queryFn: async () => {
      const response = await fetch(`/api/gas/estimate?chainId=${chainId}`);
      if (!response.ok) throw new Error('Gas estimate failed');
      return response.json();
    },
    refetchInterval: 15000,
    staleTime: 5000,
  });
};

/**
 * Hook for multi-sig transaction approval
 */
export const useMultiSigApproval = () => {
  return useMutation({
    mutationFn: async (params: {
      txId: string;
      signature: string;
      walletId: string;
    }) => {
      const response = await fetch('/api/multisig/approve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });
      if (!response.ok) throw new Error('Approval failed');
      return response.json();
    },
  });
};

/**
 * Hook for blockchain RPC calls with fallback
 */
export const useBlockchainRPC = (chainId: number) => {
  const call = useCallback(
    async (method: string, params: any[]) => {
      const rpcUrls = getRPCUrls(chainId);
      
      for (const rpcUrl of rpcUrls) {
        try {
          const response = await fetch(rpcUrl, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              jsonrpc: '2.0',
              id: 1,
              method,
              params,
            }),
          });
          const data = await response.json();
          if (data.result) return data.result;
        } catch (err) {
          // Try next RPC URL
          continue;
        }
      }
      throw new Error('All RPC endpoints failed');
    },
    [chainId]
  );

  return { call };
};

/**
 * Get RPC URLs by chain ID
 */
function getRPCUrls(chainId: number): string[] {
  const rpcMap: Record<number, string[]> = {
    1: [
      'https://eth-mainnet.g.alchemy.com/v2/',
      'https://mainnet.infura.io/v3/',
      'https://rpc.ankr.com/eth',
    ],
    137: [
      'https://polygon-rpc.com/',
      'https://rpc.ankr.com/polygon',
    ],
    42161: [
      'https://arb1.arbitrum.io/rpc',
      'https://rpc.ankr.com/arbitrum',
    ],
  };
  return rpcMap[chainId] || ['https://rpc.ankr.com/eth'];
}

export default {
  useWalletAccounts,
  usePortfolioBalance,
  useExecuteTransaction,
  useSwapTokens,
  useGasEstimate,
  useMultiSigApproval,
  useBlockchainRPC,
};
