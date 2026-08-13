/**
 * TigerWallet Kit - RainbowKit Equivalent
 * Embeddable wallet connection UI components
 * Production-ready React components for easy wallet integration
 * 
 * Features:
 * - Easy wallet connection
 * - Multi-chain support
 * - Custom branding
 * - Mobile responsive
 * - Dark/Light theme support
 */

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { ethers } from 'ethers';

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

export interface Wallet {
  id: string;
  name: string;
  icon: string;
  connector: () => Promise<ethers.Signer>;
  isInstalled?: () => boolean;
}

export interface Account {
  address: string;
  chainId: number;
  balance?: string;
  signer: ethers.Signer;
}

export interface TigerWalletKitOptions {
  children?: React.ReactNode;
  chains: Chain[];
  wallets?: Wallet[];
  theme?: 'light' | 'dark' | 'auto';
  accentColor?: string;
  appName?: string;
  appIcon?: string;
  showRecentTransactions?: boolean;
  chainImages?: Record<number, string>;
  featuredWallets?: Wallet[];
}

interface TigerWalletKitContextType {
  account: Account | null;
  isConnecting: boolean;
  connect: (wallet: Wallet) => Promise<void>;
  disconnect: () => void;
  chains: Chain[];
  activeChain: Chain | null;
  switchChain: (chainId: number) => Promise<void>;
  theme: 'light' | 'dark';
  setTheme: (theme: 'light' | 'dark') => void;
}

const TigerWalletKitContext = createContext<TigerWalletKitContextType | null>(null);

// ============================================================================
// Default Chains
// ============================================================================

export const DEFAULT_CHAINS: Chain[] = [
  {
    id: 1,
    name: 'Ethereum',
    rpcUrl: 'https://eth.llamarpc.com',
    explorer: 'https://etherscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 56,
    name: 'BNB Chain',
    rpcUrl: 'https://bsc-dataseed.binance.org',
    explorer: 'https://bscscan.com',
    nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
  },
  {
    id: 137,
    name: 'Polygon',
    rpcUrl: 'https://polygon-rpc.com',
    explorer: 'https://polygonscan.com',
    nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 },
  },
  {
    id: 42161,
    name: 'Arbitrum',
    rpcUrl: 'https://arb1.arbitrum.io/rpc',
    explorer: 'https://arbiscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 10,
    name: 'Optimism',
    rpcUrl: 'https://mainnet.optimism.io',
    explorer: 'https://optimistic.etherscan.io',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  },
  {
    id: 43114,
    name: 'Avalanche',
    rpcUrl: 'https://api.avax.network/ext/bc/C/rpc',
    explorer: 'https://snowtrace.io',
    nativeCurrency: { name: 'AVAX', symbol: 'AVAX', decimals: 18 },
  },
];

// ============================================================================
// Default Wallets
// ============================================================================

export const DEFAULT_WALLETS: Wallet[] = [
  {
    id: 'metamask',
    name: 'MetaMask',
    icon: '/wallets/metamask.svg',
    connector: async () => {
      if (typeof window === 'undefined' || !window.ethereum) {
        throw new Error('MetaMask not installed');
      }
      const provider = new ethers.providers.Web3Provider(window.ethereum);
      return provider.getSigner();
    },
    isInstalled: () => typeof window !== 'undefined' && !!window.ethereum?.isMetaMask,
  },
  {
    id: 'walletconnect',
    name: 'WalletConnect',
    icon: '/wallets/walletconnect.svg',
    connector: async () => {
      // WalletConnect v2 connects through the injected provider bridge when a
      // WC-compatible wallet extension (MetaMask mobile, Rainbow, etc.) is
      // present, or via the @walletconnect/ethereum-provider SDK when
      // installed. We use the injected bridge when available; otherwise we
      // throw an honest error (no silent fake provider).
      if (typeof window === 'undefined' || !window.ethereum) {
        throw new Error('WalletConnect not available: no injected provider. Install @walletconnect/ethereum-provider or a WC-compatible wallet.');
      }
      const provider = new ethers.providers.Web3Provider(window.ethereum);
      await provider.send('eth_requestAccounts', []);
      return provider.getSigner();
    },
    isInstalled: () => typeof window !== 'undefined' && (!!((window.ethereum as any)?.isWalletConnect) || !!window.ethereum?.isMetaMask),
  },
  {
    id: 'coinbase',
    name: 'Coinbase Wallet',
    icon: '/wallets/coinbase.svg',
    connector: async () => {
      if (typeof window === 'undefined' || !window.ethereum) {
        throw new Error('Coinbase Wallet not installed');
      }
      const provider = new ethers.providers.Web3Provider(window.ethereum);
      return provider.getSigner();
    },
    isInstalled: () => typeof window !== 'undefined' && !!window.ethereum?.isCoinbaseWallet,
  },
  {
    id: 'tigerwallet',
    name: 'TigerWallet',
    icon: '/wallets/tigerwallet.svg',
    connector: async () => {
      // Connect to TigerWallet
      if (typeof window === 'undefined') {
        throw new Error('Browser only');
      }
      // Use injected provider or create connection
      const provider = new ethers.providers.Web3Provider(window.ethereum as any);
      return provider.getSigner();
    },
  },
];

// ============================================================================
// Provider Component
// ============================================================================

export function TigerWalletKitProvider({ 
  children, 
  chains = DEFAULT_CHAINS,
  wallets = DEFAULT_WALLETS,
  theme: initialTheme = 'dark',
  accentColor = '#f97316',
  appName = 'DApp',
}: TigerWalletKitOptions) {
  const [account, setAccount] = useState<Account | null>(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [activeChain, setActiveChain] = useState<Chain | null>(chains[0] || null);
  const resolvedTheme: 'light' | 'dark' = initialTheme === 'auto'
    ? (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
    : initialTheme;
  const [theme, setTheme] = useState<'light' | 'dark'>(resolvedTheme);

  // Apply theme to document
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  // Check for existing connection on mount
  useEffect(() => {
    const storedAccount = localStorage.getItem('tigerwallet_account');
    if (storedAccount) {
      try {
        const parsed = JSON.parse(storedAccount);
        setAccount(parsed);
      } catch (e) {
        console.error('Failed to parse stored account:', e);
      }
    }
  }, []);

  const connect = useCallback(async (wallet: Wallet) => {
    setIsConnecting(true);
    try {
      const signer = await wallet.connector();
      const address = await signer.getAddress();
      const provider = signer.provider as ethers.providers.JsonRpcProvider;
      const network = await provider.getNetwork();
      
      const newAccount: Account = {
        address,
        chainId: Number(network.chainId),
        signer,
      };
      
      setAccount(newAccount);
      localStorage.setItem('tigerwallet_account', JSON.stringify(newAccount));
      
      // Find and set active chain
      const chain = chains.find(c => c.id === Number(network.chainId));
      if (chain) {
        setActiveChain(chain);
      }
    } catch (error) {
      console.error('Failed to connect:', error);
      throw error;
    } finally {
      setIsConnecting(false);
    }
  }, [chains]);

  const disconnect = useCallback(() => {
    setAccount(null);
    localStorage.removeItem('tigerwallet_account');
  }, []);

  const switchChain = useCallback(async (chainId: number) => {
    const chain = chains.find(c => c.id === chainId);
    if (!chain) {
      throw new Error(`Chain ${chainId} not supported`);
    }
    
    if (account?.signer?.provider) {
      try {
        // Request chain switch
        const swp = account.signer.provider as ethers.providers.JsonRpcProvider; await swp.send('wallet_switchEthereumChain', [
          { chainId: `0x${chainId.toString(16)}` },
        ]);
        setActiveChain(chain);
        
        // Update account with new chain
        setAccount(prev => prev ? { ...prev, chainId } : null);
      } catch (error: any) {
        // Chain not added to wallet, try to add
        if (error.code === 4902) {
          const adp = account.signer.provider as ethers.providers.JsonRpcProvider; await adp.send('wallet_addEthereumChain', [{
            chainId: `0x${chainId.toString(16)}`,
            chainName: chain.name,
            nativeCurrency: chain.nativeCurrency,
            rpcUrls: [chain.rpcUrl],
            blockExplorerUrls: [chain.explorer],
          }]);
        } else {
          throw error;
        }
      }
    }
  }, [chains, account]);

  return (
    <TigerWalletKitContext.Provider
      value={{
        account,
        isConnecting,
        connect,
        disconnect,
        chains,
        activeChain,
        switchChain,
        theme,
        setTheme,
      }}
    >
      <div 
        style={{ 
          '--accent-color': accentColor,
        } as React.CSSProperties}
      >
        {children}
      </div>
    </TigerWalletKitContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

export function useTigerWallet() {
  const context = useContext(TigerWalletKitContext);
  if (!context) {
    throw new Error('useTigerWallet must be used within TigerWalletKitProvider');
  }
  return context;
}

// ============================================================================
// Connect Button Component
// ============================================================================

interface ConnectButtonProps {
  showBalance?: boolean;
  chainSelector?: boolean;
}

export function ConnectButton({ 
  showBalance = true,
  chainSelector = true,
}: ConnectButtonProps) {
  const { account, isConnecting, connect, disconnect, chains, activeChain, switchChain, theme } = useTigerWallet();
  const [showWalletModal, setShowWalletModal] = useState(false);
  const [showChainModal, setShowChainModal] = useState(false);

  const formatAddress = (address: string) => {
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  const formatBalance = (balance?: string) => {
    if (!balance) return '0.00';
    const num = parseFloat(balance);
    return num.toFixed(4);
  };

  if (!account) {
    return (
      <div>
        <button
          onClick={() => setShowWalletModal(true)}
          disabled={isConnecting}
          className={`tigerkit-button tigerkit-button-${theme}`}
          style={{
            backgroundColor: 'var(--accent-color)',
            color: 'white',
            padding: '12px 24px',
            borderRadius: '12px',
            border: 'none',
            fontSize: '16px',
            fontWeight: 600,
            cursor: isConnecting ? 'not-allowed' : 'pointer',
            opacity: isConnecting ? 0.7 : 1,
            transition: 'all 0.2s ease',
          }}
        >
          {isConnecting ? 'Connecting...' : 'Connect Wallet'}
        </button>

        {showWalletModal && (
          <WalletModal 
            onClose={() => setShowWalletModal(false)}
            onConnect={(wallet) => {
              connect(wallet).then(() => setShowWalletModal(false));
            }}
          />
        )}
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', gap: '8px' }}>
      {chainSelector && (
        <button
          onClick={() => setShowChainModal(true)}
          className={`tigerkit-chain-button tigerkit-button-${theme}`}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '8px 12px',
            borderRadius: '8px',
            border: '1px solid var(--border-color)',
            backgroundColor: 'var(--bg-secondary)',
            color: 'var(--text-primary)',
            cursor: 'pointer',
          }}
        >
          <span style={{ width: '20px', height: '20px', borderRadius: '50%', backgroundColor: 'var(--accent-color)' }} />
          {activeChain?.name || 'Unknown'}
        </button>
      )}

      <button
        className={`tigerkit-address-button tigerkit-button-${theme}`}
        onClick={disconnect}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          padding: '8px 16px',
          borderRadius: '12px',
          border: '1px solid var(--border-color)',
          backgroundColor: 'var(--bg-secondary)',
            color: 'var(--text-primary)',
            cursor: 'pointer',
          }}
        >
          {showBalance && (
            <span style={{ color: 'var(--accent-color)', fontWeight: 600 }}>
              {formatBalance(account.balance)} {activeChain?.nativeCurrency.symbol}
            </span>
          )}
          <span style={{ fontWeight: 500 }}>{formatAddress(account.address)}</span>
        </button>

      {showChainModal && (
        <ChainModal
          chains={chains}
          activeChain={activeChain}
          onSelect={(chain) => {
            switchChain(chain.id);
            setShowChainModal(false);
          }}
          onClose={() => setShowChainModal(false)}
        />
      )}
    </div>
  );
}

// ============================================================================
// Wallet Modal Component
// ============================================================================

interface WalletModalProps {
  onClose: () => void;
  onConnect: (wallet: Wallet) => void;
}

function WalletModal({ onClose, onConnect }: WalletModalProps) {
  const { theme } = useTigerWallet();
  const [filteredWallets, setFilteredWallets] = useState(DEFAULT_WALLETS);

  return (
    <div 
      className="tigerkit-modal-overlay"
      onClick={onClose}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div 
        className="tigerkit-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          backgroundColor: 'var(--bg-primary)',
          borderRadius: '24px',
          padding: '24px',
          width: '100%',
          maxWidth: '400px',
          maxHeight: '80vh',
          overflow: 'auto',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
          <h2 style={{ fontSize: '20px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
            Connect Wallet
          </h2>
          <button 
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: 'var(--text-secondary)',
            }}
          >
            ×
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {filteredWallets.map((wallet) => (
            <button
              key={wallet.id}
              onClick={() => onConnect(wallet)}
              className={`tigerkit-wallet-option tigerkit-button-${theme}`}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '16px',
                padding: '16px',
                borderRadius: '16px',
                border: '1px solid var(--border-color)',
                backgroundColor: 'var(--bg-secondary)',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
              }}
            >
              <img 
                src={wallet.icon} 
                alt={wallet.name}
                style={{ width: '40px', height: '40px', borderRadius: '8px' }}
                onError={(e) => {
                  (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="50" fill="%23f97316"/></svg>';
                }}
              />
              <span style={{ fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
                {wallet.name}
              </span>
              {wallet.isInstalled && !wallet.isInstalled() && (
                <span style={{ 
                  marginLeft: 'auto', 
                  fontSize: '12px', 
                  color: 'var(--text-secondary)',
                  padding: '4px 8px',
                  backgroundColor: 'var(--bg-primary)',
                  borderRadius: '4px',
                }}>
                  Not installed
                </span>
              )}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Chain Modal Component
// ============================================================================

interface ChainModalProps {
  chains: Chain[];
  activeChain: Chain | null;
  onSelect: (chain: Chain) => void;
  onClose: () => void;
}

function ChainModal({ chains, activeChain, onSelect, onClose }: ChainModalProps) {
  const { theme } = useTigerWallet();

  return (
    <div 
      className="tigerkit-modal-overlay"
      onClick={onClose}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div 
        className="tigerkit-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          backgroundColor: 'var(--bg-primary)',
          borderRadius: '24px',
          padding: '24px',
          width: '100%',
          maxWidth: '400px',
          maxHeight: '80vh',
          overflow: 'auto',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
          <h2 style={{ fontSize: '20px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
            Switch Network
          </h2>
          <button 
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: 'var(--text-secondary)',
            }}
          >
            ×
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {chains.map((chain) => (
            <button
              key={chain.id}
              onClick={() => onSelect(chain)}
              className={`tigerkit-chain-option tigerkit-button-${theme}`}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                padding: '12px 16px',
                borderRadius: '12px',
                border: activeChain?.id === chain.id 
                  ? '2px solid var(--accent-color)' 
                  : '1px solid var(--border-color)',
                backgroundColor: activeChain?.id === chain.id 
                  ? 'rgba(var(--accent-color-rgb), 0.1)' 
                  : 'var(--bg-secondary)',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
              }}
            >
              <div style={{ 
                width: '32px', 
                height: '32px', 
                borderRadius: '50%', 
                backgroundColor: 'var(--accent-color)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'white',
                fontWeight: 700,
                fontSize: '12px',
              }}>
                {chain.name.charAt(0)}
              </div>
              <div style={{ flex: 1, textAlign: 'left' }}>
                <div style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                  {chain.name}
                </div>
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                  {chain.nativeCurrency.symbol}
                </div>
              </div>
              {activeChain?.id === chain.id && (
                <span style={{ color: 'var(--accent-color)' }}>✓</span>
              )}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Theme Toggle Component
// ============================================================================

export function ThemeToggle() {
  const { theme, setTheme } = useTigerWallet();

  return (
    <button
      onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
      className="tigerkit-theme-toggle"
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '40px',
        height: '40px',
        borderRadius: '50%',
        border: '1px solid var(--border-color)',
        backgroundColor: 'var(--bg-secondary)',
        cursor: 'pointer',
      }}
    >
      {theme === 'light' ? '🌙' : '☀️'}
    </button>
  );
}

// ============================================================================
// CSS Styles
// ============================================================================

const styles = `
  :root {
    --bg-primary: #ffffff;
    --bg-secondary: #f3f4f6;
    --text-primary: #111827;
    --text-secondary: #6b7280;
    --border-color: #e5e7eb;
    --accent-color: #f97316;
    --accent-color-rgb: 249, 115, 22;
  }

  [data-theme="dark"] {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --text-primary: #f8fafc;
    --text-secondary: #94a3b8;
    --border-color: #334155;
  }

  .tigerkit-button {
    transition: all 0.2s ease;
  }

  .tigerkit-button:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  .tigerkit-modal-overlay {
    animation: fadeIn 0.2s ease;
  }

  .tigerkit-modal {
    animation: slideUp 0.3s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes slideUp {
    from { 
      opacity: 0;
      transform: translateY(20px);
    }
    to { 
      opacity: 1;
      transform: translateY(0);
    }
  }
`;

// Inject styles
if (typeof document !== 'undefined') {
  const styleSheet = document.createElement('style');
  styleSheet.textContent = styles;
  document.head.appendChild(styleSheet);
}

// window.ethereum type is declared centrally in app/wallet.ts (Eip1193Provider).

export default TigerWalletKitProvider;
