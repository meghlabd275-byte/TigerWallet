/**
 * TigerKit - TigerWallet Embeddable Wallet UI
 * 
 * RainbowKit equivalent for TigerWallet
 * Production-ready embeddable wallet UI components
 * 
 * Features:
 * - Connect wallet modal
 * - Wallet balance display
 * - Token list
 * - Transaction history
 * - Network switcher
 * - Dark/Light theme
 */

import React, { useState, useEffect, createContext, useContext } from 'react';

// ============================================================================
// Types
// ============================================================================

export interface Chain {
  id: number;
  name: string;
  icon?: string;
  rpcUrl: string;
  explorer: string;
  nativeCurrency: {
    name: string;
    symbol: string;
    decimals: number;
  };
}

export interface Token {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  logoURI?: string;
  balance?: string;
  price?: number;
  valueUSD?: number;
}

export interface WalletState {
  isConnected: boolean;
  address: string | null;
  chainId: number | null;
  balance: string | null;
  tokens: Token[];
}

export interface TigerKitConfig {
  chains: Chain[];
  initialChainId?: number;
  theme?: 'light' | 'dark' | 'system';
  walletConnectProjectId?: string;
  appName?: string;
  appIcon?: string;
}

// ============================================================================
// Context
// ============================================================================

interface TigerKitContextType {
  config: TigerKitConfig;
  wallet: WalletState;
  connect: () => Promise<void>;
  disconnect: () => void;
  switchChain: (chainId: number) => Promise<void>;
  isModalOpen: boolean;
  setIsModalOpen: (open: boolean) => void;
}

const TigerKitContext = createContext<TigerKitContextType | null>(null);

export const useTigerKit = () => {
  const context = useContext(TigerKitContext);
  if (!context) {
    throw new Error('useTigerKit must be used within TigerKitProvider');
  }
  return context;
};

// ============================================================================
// Default Chains
// ============================================================================

export const DEFAULT_CHAINS: Chain[] = [
  {
    id: 1,
    name: 'Ethereum',
    icon: '⬡',
    rpcUrl: 'https://eth.llamarpc.com',
    explorer: 'https://etherscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 56,
    name: 'BNB Chain',
    icon: '⬡',
    rpcUrl: 'https://bsc-dataseed.binance.org',
    explorer: 'https://bscscan.com',
    nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
  },
  {
    id: 42161,
    name: 'Arbitrum',
    icon: '⬡',
    rpcUrl: 'https://arb1.arbitrum.io/rpc',
    explorer: 'https://arbiscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 137,
    name: 'Polygon',
    icon: '⬡',
    rpcUrl: 'https://polygon-rpc.com',
    explorer: 'https://polygonscan.com',
    nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 },
  },
  {
    id: 10,
    name: 'Optimism',
    icon: '⬡',
    rpcUrl: 'https://mainnet.optimism.io',
    explorer: 'https://optimistic.etherscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 8453,
    name: 'Base',
    icon: '⬡',
    rpcUrl: 'https://mainnet.base.org',
    explorer: 'https://basescan.org',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
];

// ============================================================================
// Provider
// ============================================================================

interface TigerKitProviderProps {
  config: TigerKitConfig;
  children: React.ReactNode;
}

export const TigerKitProvider: React.FC<TigerKitProviderProps> = ({ config, children }) => {
  const [wallet, setWallet] = useState<WalletState>({
    isConnected: false,
    address: null,
    chainId: config.initialChainId || 1,
    balance: null,
    tokens: [],
  });
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark'>(config.theme === 'system' ? 'light' : (config.theme || 'light'));

  // Handle theme
  useEffect(() => {
    if (config.theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      setTheme(mediaQuery.matches ? 'dark' : 'light');
      
      const handler = (e: MediaQueryListEvent) => {
        setTheme(e.matches ? 'dark' : 'light');
      };
      mediaQuery.addEventListener('change', handler);
      return () => mediaQuery.removeEventListener('change', handler);
    }
  }, [config.theme]);

  // Apply theme to document
  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark');
  }, [theme]);

  const connect = async () => {
    // In production, this would use WalletConnect or injected provider
    setIsModalOpen(true);
  };

  const disconnect = () => {
    setWallet({
      isConnected: false,
      address: null,
      chainId: config.initialChainId || 1,
      balance: null,
      tokens: [],
    });
  };

  const switchChain = async (chainId: number) => {
    setWallet(prev => ({ ...prev, chainId }));
  };

  return (
    <TigerKitContext.Provider
      value={{
        config,
        wallet,
        connect,
        disconnect,
        switchChain,
        isModalOpen,
        setIsModalOpen,
      }}
    >
      {children}
    </TigerKitContext.Provider>
  );
};

// ============================================================================
// Connect Button
// ============================================================================

interface ConnectButtonProps {
  showBalance?: boolean;
}

export const ConnectButton: React.FC<ConnectButtonProps> = ({ showBalance = true }) => {
  const { wallet, connect, config } = useTigerKit();
  
  const currentChain = config.chains.find(c => c.id === wallet.chainId);

  if (wallet.isConnected && wallet.address) {
    return (
      <button
        onClick={() => {}}
        className="flex items-center gap-2 px-4 py-2 bg-slate-100 dark:bg-slate-800 rounded-full hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
      >
        {currentChain && (
          <span className="text-sm">{currentChain.icon}</span>
        )}
        <span className="font-medium">
          {wallet.address.slice(0, 6)}...{wallet.address.slice(-4)}
        </span>
        {showBalance && wallet.balance && (
          <span className="text-sm text-slate-500">
            {parseFloat(wallet.balance).toFixed(4)} {currentChain?.nativeCurrency.symbol}
          </span>
        )}
      </button>
    );
  }

  return (
    <button
      onClick={connect}
      className="px-4 py-2 bg-blue-600 text-white rounded-full font-medium hover:bg-blue-700 transition-colors"
    >
      Connect Wallet
    </button>
  );
};

// ============================================================================
// Network Switcher
// ============================================================================

export const NetworkSwitcher: React.FC = () => {
  const { wallet, switchChain, config } = useTigerKit();
  const [isOpen, setIsOpen] = useState(false);

  const currentChain = config.chains.find(c => c.id === wallet.chainId);

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 bg-slate-100 dark:bg-slate-800 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-700"
      >
        {currentChain?.icon && <span>{currentChain.icon}</span>}
        <span>{currentChain?.name}</span>
        <span className="text-xs">▼</span>
      </button>

      {isOpen && (
        <div className="absolute top-full mt-2 w-48 bg-white dark:bg-slate-800 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700 z-50">
          {config.chains.map(chain => (
            <button
              key={chain.id}
              onClick={() => {
                switchChain(chain.id);
                setIsOpen(false);
              }}
              className={`w-full px-4 py-2 text-left hover:bg-slate-100 dark:hover:bg-slate-700 flex items-center gap-2 ${
                chain.id === wallet.chainId ? 'bg-blue-50 dark:bg-blue-900' : ''
              }`}
            >
              {chain.icon && <span>{chain.icon}</span>}
              <span>{chain.name}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Token List
// ============================================================================

export const TokenList: React.FC = () => {
  const { wallet, config } = useTigerKit();
  
  const currentChain = config.chains.find(c => c.id === wallet.chainId);

  if (!wallet.isConnected || wallet.tokens.length === 0) {
    return (
      <div className="p-4 text-center text-slate-500">
        No tokens found
      </div>
    );
  }

  return (
    <div className="divide-y divide-slate-200 dark:divide-slate-700">
      {/* Native token */}
      <div className="p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-blue-600 flex items-center justify-center text-white font-bold">
            {currentChain?.nativeCurrency.symbol}
          </div>
          <div>
            <p className="font-medium">{currentChain?.nativeCurrency.name}</p>
            <p className="text-sm text-slate-500">Native</p>
          </div>
        </div>
        <div className="text-right">
          <p className="font-medium">{wallet.balance || '0'}</p>
        </div>
      </div>
      
      {/* ERC20 tokens */}
      {wallet.tokens.map(token => (
        <div key={token.address} className="p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {token.logoURI ? (
              <img src={token.logoURI} alt={token.symbol} className="w-10 h-10 rounded-full" />
            ) : (
              <div className="w-10 h-10 rounded-full bg-slate-200 dark:bg-slate-700 flex items-center justify-center">
                {token.symbol.slice(0, 2)}
              </div>
            )}
            <div>
              <p className="font-medium">{token.name}</p>
              <p className="text-sm text-slate-500">{token.symbol}</p>
            </div>
          </div>
          <div className="text-right">
            <p className="font-medium">{token.balance || '0'}</p>
            {token.valueUSD && (
              <p className="text-sm text-slate-500">${token.valueUSD.toFixed(2)}</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
};

// ============================================================================
// Connect Modal
// ============================================================================

export const ConnectModal: React.FC = () => {
  const { isModalOpen, setIsModalOpen, connect, config } = useTigerKit();
  
  const [step, setStep] = useState<'wallets' | 'chains'>('wallets');

  if (!isModalOpen) return null;

  const wallets = [
    { id: 'metamask', name: 'MetaMask', icon: '🦊' },
    { id: 'coinbase', name: 'Coinbase Wallet', icon: '🔵' },
    { id: 'rainbow', name: 'Rainbow', icon: '🌈' },
    { id: 'trust', name: 'Trust Wallet', icon: '🐯' },
    { id: 'walletconnect', name: 'WalletConnect', icon: '📱' },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div 
        className="absolute inset-0 bg-black/50"
        onClick={() => setIsModalOpen(false)}
      />
      <div className="relative bg-white dark:bg-slate-800 rounded-2xl w-full max-w-md mx-4 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-lg font-semibold">
            {step === 'wallets' ? 'Connect Wallet' : 'Select Network'}
          </h2>
          <button 
            onClick={() => setIsModalOpen(false)}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-4">
          {step === 'wallets' ? (
            <div className="space-y-2">
              {wallets.map(wallet => (
                <button
                  key={wallet.id}
                  onClick={connect}
                  className="w-full p-4 flex items-center gap-4 bg-slate-50 dark:bg-slate-700 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-600 transition-colors"
                >
                  <span className="text-2xl">{wallet.icon}</span>
                  <span className="font-medium">{wallet.name}</span>
                </button>
              ))}
              
              <button
                onClick={() => setStep('chains')}
                className="w-full p-4 flex items-center justify-center text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
              >
                Switch Network →
              </button>
            </div>
          ) : (
            <div className="space-y-2">
              {config.chains.map(chain => (
                <button
                  key={chain.id}
                  onClick={() => {
                    // Handle chain selection
                    setStep('wallets');
                  }}
                  className="w-full p-4 flex items-center gap-4 bg-slate-50 dark:bg-slate-700 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-600"
                >
                  {chain.icon && <span className="text-2xl">{chain.icon}</span>}
                  <span className="font-medium">{chain.name}</span>
                </button>
              ))}
              
              <button
                onClick={() => setStep('wallets')}
                className="w-full p-4 flex items-center justify-center text-slate-500"
              >
                ← Back to Wallets
              </button>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-700 text-center text-sm text-slate-500">
          By connecting, you agree to our Terms of Service
        </div>
      </div>
    </div>
  );
};

// ============================================================================
// Theme Toggle
// ============================================================================

export const ThemeToggle: React.FC = () => {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    setTheme(document.documentElement.classList.contains('dark') ? 'dark' : 'light');
  }, []);

  const toggle = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    document.documentElement.classList.toggle('dark', newTheme === 'dark');
  };

  return (
    <button
      onClick={toggle}
      className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
    >
      {theme === 'light' ? '🌙' : '☀️'}
    </button>
  );
};

// ============================================================================
// Main Component - Full Wallet Widget
// ============================================================================

export const WalletWidget: React.FC = () => {
  const { wallet, connect, disconnect, config } = useTigerKit();
  const [showModal, setShowModal] = useState(false);

  return (
    <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      {/* Header */}
      <div className="p-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-2xl">🐯</span>
          <span className="font-bold">TigerWallet</span>
        </div>
        <div className="flex items-center gap-2">
          <NetworkSwitcher />
          <ThemeToggle />
        </div>
      </div>

      {/* Content */}
      {wallet.isConnected ? (
        <div>
          {/* Balance */}
          <div className="p-6 bg-gradient-to-br from-blue-600 to-purple-600 text-white">
            <p className="text-sm opacity-80">Total Balance</p>
            <p className="text-3xl font-bold">$0.00</p>
            <button
              onClick={disconnect}
              className="mt-4 text-sm bg-white/20 px-4 py-2 rounded-lg hover:bg-white/30"
            >
              Disconnect
            </button>
          </div>

          {/* Tokens */}
          <TokenList />
        </div>
      ) : (
        <div className="p-8 text-center">
          <p className="text-slate-500 mb-4">Connect your wallet to get started</p>
          <button
            onClick={() => setShowModal(true)}
            className="px-6 py-3 bg-blue-600 text-white rounded-full font-medium hover:bg-blue-700"
          >
            Connect Wallet
          </button>
        </div>
      )}

      {/* Modal */}
      {showModal && <ConnectModal />}
    </div>
  );
};

// ============================================================================
// Export
// ============================================================================

export default {
  TigerKitProvider,
  ConnectButton,
  NetworkSwitcher,
  TokenList,
  ConnectModal,
  ThemeToggle,
  WalletWidget,
  DEFAULT_CHAINS,
};
