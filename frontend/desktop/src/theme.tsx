// TigerWallet Desktop - Theme Context
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

type Theme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (theme: Theme) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
}

export function DesktopThemeProvider({ children }: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>('dark');
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const storedTheme = localStorage.getItem('tiger-desktop-theme') as Theme;
    if (storedTheme) {
      setThemeState(storedTheme);
    } else {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      setThemeState(prefersDark ? 'dark' : 'light');
    }
  }, []);

  useEffect(() => {
    if (mounted) {
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem('tiger-desktop-theme', theme);
    }
  }, [theme, mounted]);

  const toggleTheme = () => {
    setThemeState(prev => prev === 'dark' ? 'light' : 'dark');
  };

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useDesktopTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useDesktopTheme must be used within DesktopThemeProvider');
  }
  return context;
}

// Desktop API Service
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

class DesktopAPIService {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status}`);
    }

    return response.json();
  }

  // Wallet APIs
  async getWallets() {
    return this.request<any>('/api/v1/wallets');
  }

  async getWallet(chainId: string) {
    return this.request<any>(`/api/v1/wallet/${chainId}`);
  }

  async createWallet(chainId: string, type: string = 'user') {
    return this.request<any>(`/api/v1/wallets`, {
      method: 'POST',
      body: JSON.stringify({ chain: chainId, type }),
    });
  }

  async getBalance(walletAddress: string, chainId: string) {
    return this.request<any>(`/api/v1/balance/${chainId}/${walletAddress}`);
  }

  // Transaction APIs
  async sendTransaction(to: string, amount: string, chainId: string, data?: string) {
    return this.request<any>('/api/v1/transactions', {
      method: 'POST',
      body: JSON.stringify({ to, amount, chain: chainId, data: data || '0x' }),
    });
  }

  async getTransactions(walletAddress: string, chainId: string, limit: number = 50) {
    return this.request<any>(`/api/v1/transactions/${chainId}/${walletAddress}?limit=${limit}`);
  }

  // Swap APIs
  async getSwapQuote(fromToken: string, toToken: string, amount: string, chainId: string) {
    return this.request<any>(
      `/api/v1/swap/quote?from=${fromToken}&to=${toToken}&amount=${amount}&chain=${chainId}`
    );
  }

  async executeSwap(quoteId: string) {
    return this.request<any>('/api/v1/swap/execute', {
      method: 'POST',
      body: JSON.stringify({ quoteId }),
    });
  }

  // Gas APIs
  async getGasPrice(chainId: string) {
    return this.request<any>(`/api/v1/gas/${chainId}`);
  }

  // Network APIs
  async getNetworkStatus(chainId: string) {
    return this.request<any>(`/api/v1/network/${chainId}/status`);
  }

  // Token APIs
  async getTokens(chainId: string) {
    return this.request<any>(`/api/v1/tokens/${chainId}`);
  }

  // NFT APIs
  async getNFTs(walletAddress: string, chainId: string) {
    return this.request<any>(`/api/v1/nfts/${chainId}/${walletAddress}`);
  }

  // Staking APIs
  async getStakingPositions(walletAddress: string, chainId: string) {
    return this.request<any>(`/api/v1/staking/${chainId}/${walletAddress}`);
  }
}

export const desktopAPI = new DesktopAPIService();
export type { Theme, ThemeContextType };
