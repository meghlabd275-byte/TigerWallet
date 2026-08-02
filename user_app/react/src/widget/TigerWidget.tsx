/**
 * TigerWallet Widget SDK
 * Production-ready embeddable wallet widgets
 * No mock data, no simulations - fully functional
 */

import React, { useState, useEffect, useCallback, createContext, useContext } from 'react';
import { walletApi, Transaction, Wallet } from '../services/api';

// ============================================================================
// Types
// ============================================================================

export interface WidgetConfig {
  apiKey: string;
  theme?: 'light' | 'dark' | 'auto';
  network?: 'mainnet' | 'testnet';
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left';
  language?: 'en' | 'es' | 'zh' | 'ja' | 'ko';
}

export interface WidgetState {
  isConnected: boolean;
  wallet: Wallet | null;
  balance: string;
  address: string;
}

export interface TransactionRequest {
  to: string;
  amount: string;
  token?: string;
  data?: string;
}

export interface SignMessageRequest {
  message: string;
}

// ============================================================================
// Context
// ============================================================================

interface WidgetContextType {
  config: WidgetConfig;
  state: WidgetState;
  connect: () => Promise<void>;
  disconnect: () => Promise<void>;
  sendTransaction: (req: TransactionRequest) => Promise<string>;
  signMessage: (req: SignMessageRequest) => Promise<string>;
  getBalance: () => Promise<string>;
  switchNetwork: (chainId: number) => Promise<void>;
}

const WidgetContext = createContext<WidgetContextType | null>(null);

export const useWidget = () => {
  const context = useContext(WidgetContext);
  if (!context) {
    throw new Error('useWidget must be used within a TigerWidgetProvider');
  }
  return context;
};

// ============================================================================
// Provider Component
// ============================================================================

interface TigerWidgetProviderProps {
  config: WidgetConfig;
  children: React.ReactNode;
  onConnect?: (wallet: Wallet) => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
}

export const TigerWidgetProvider: React.FC<TigerWidgetProviderProps> = ({
  config,
  children,
  onConnect,
  onDisconnect,
  onError
}) => {
  const [state, setState] = useState<WidgetState>({
    isConnected: false,
    wallet: null,
    balance: '0',
    address: ''
  });
  const [loading, setLoading] = useState(false);

  // Check for existing session on mount
  useEffect(() => {
    checkExistingSession();
  }, []);

  const checkExistingSession = async () => {
    try {
      const savedWallet = localStorage.getItem('widget_wallet');
      if (savedWallet) {
        const wallet = JSON.parse(savedWallet);
        setState({
          isConnected: true,
          wallet,
          balance: wallet.balance || '0',
          address: wallet.address
        });
      }
    } catch (error) {
      console.error('Failed to check session:', error);
    }
  };

  const connect = useCallback(async () => {
    setLoading(true);
    try {
      // Get wallets from API
      const wallets = await walletApi.getWallets();
      
      if (wallets.length > 0) {
        const wallet = wallets[0];
        
        // Save to localStorage for persistence
        localStorage.setItem('widget_wallet', JSON.stringify(wallet));
        
        setState({
          isConnected: true,
          wallet,
          balance: wallet.balance || '0',
          address: wallet.address
        });
        
        onConnect?.(wallet);
      } else {
        // Create a new wallet if none exists
        const newWallet = await walletApi.createWallet('ethereum', 'Widget Wallet');
        localStorage.setItem('widget_wallet', JSON.stringify(newWallet));
        
        setState({
          isConnected: true,
          wallet: newWallet,
          balance: newWallet.balance || '0',
          address: newWallet.address
        });
        
        onConnect?.(newWallet);
      }
    } catch (error) {
      onError?.(error as Error);
      console.error('Failed to connect:', error);
    } finally {
      setLoading(false);
    }
  }, [onConnect, onError]);

  const disconnect = useCallback(async () => {
    try {
      localStorage.removeItem('widget_wallet');
      setState({
        isConnected: false,
        wallet: null,
        balance: '0',
        address: ''
      });
      onDisconnect?.();
    } catch (error) {
      onError?.(error as Error);
    }
  }, [onDisconnect, onError]);

  const sendTransaction = useCallback(async (req: TransactionRequest): Promise<string> => {
    if (!state.wallet) {
      throw new Error('Wallet not connected');
    }

    try {
      const result = await walletApi.sendTransaction(
        state.wallet.id,
        req.to,
        req.amount,
        req.token
      );
      return result.hash;
    } catch (error) {
      onError?.(error as Error);
      throw error;
    }
  }, [state.wallet, onError]);

  const signMessage = useCallback(async (req: SignMessageRequest): Promise<string> => {
    if (!state.wallet) {
      throw new Error('Wallet not connected');
    }

    try {
      // Sign message via API
      const result = await walletApi.signMessage(
        state.wallet.id,
        req.message
      );
      return result.signature;
    } catch (error) {
      onError?.(error as Error);
      throw error;
    }
  }, [state.wallet, onError]);

  const getBalance = useCallback(async (): Promise<string> => {
    if (!state.wallet) {
      return '0';
    }

    try {
      const balanceData = await walletApi.getBalance(state.wallet.id);
      const newBalance = balanceData.balance;
      
      setState(prev => ({
        ...prev,
        balance: newBalance
      }));
      
      return newBalance;
    } catch (error) {
      onError?.(error as Error);
      return state.balance;
    }
  }, [state.wallet, state.balance, onError]);

  const switchNetwork = useCallback(async (chainId: number) => {
    if (!state.wallet) {
      throw new Error('Wallet not connected');
    }

    try {
      await walletApi.switchNetwork(state.wallet.id, chainId);
    } catch (error) {
      onError?.(error as Error);
      throw error;
    }
  }, [state.wallet, onError]);

  const value: WidgetContextType = {
    config,
    state,
    connect,
    disconnect,
    sendTransaction,
    signMessage,
    getBalance,
    switchNetwork
  };

  return (
    <WidgetContext.Provider value={value}>
      {children}
    </WidgetContext.Provider>
  );
};

// ============================================================================
// Connect Button Widget
// ============================================================================

interface ConnectButtonProps {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'small' | 'medium' | 'large';
  label?: string;
  showBalance?: boolean;
}

export const ConnectButton: React.FC<ConnectButtonProps> = ({
  variant = 'primary',
  size = 'medium',
  label = 'Connect Wallet',
  showBalance = true
}) => {
  const { state, connect, disconnect } = useWidget();

  const buttonStyles: React.CSSProperties = {
    padding: size === 'small' ? '8px 16px' : size === 'large' ? '16px 32px' : '12px 24px',
    borderRadius: '8px',
    border: 'none',
    cursor: 'pointer',
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.2s ease',
    ...(variant === 'primary' && {
      background: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)',
      color: 'white'
    }),
    ...(variant === 'secondary' && {
      background: '#1e293b',
      color: 'white'
    }),
    ...(variant === 'ghost' && {
      background: 'transparent',
      color: '#64748b'
    })
  };

  const formatAddress = (addr: string): string => {
    if (!addr) return '';
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  if (state.isConnected) {
    return (
      <button style={buttonStyles} onClick={disconnect}>
        <span>🔗</span>
        <span>{formatAddress(state.address)}</span>
        {showBalance && (
          <span style={{ opacity: 0.8, fontSize: '0.9em' }}>
            (${state.balance})
          </span>
        )}
      </button>
    );
  }

  return (
    <button style={buttonStyles} onClick={connect}>
      <span>👛</span>
      <span>{label}</span>
    </button>
  );
};

// ============================================================================
// Balance Widget
// ============================================================================

interface BalanceWidgetProps {
  showToken?: boolean;
  decimals?: number;
}

export const BalanceWidget: React.FC<BalanceWidgetProps> = ({
  showToken = true,
  decimals = 4
}) => {
  const { state, getBalance } = useWidget();
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    await getBalance();
    setRefreshing(false);
  };

  const formatBalance = (balance: string): string => {
    const num = parseFloat(balance);
    return num.toFixed(decimals);
  };

  if (!state.isConnected) {
    return <div style={{ color: '#64748b' }}>Not connected</div>;
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
      <span style={{ fontSize: '1.5em' }}>💰</span>
      <div>
        <div style={{ fontSize: '1.25rem', fontWeight: 600 }}>
          ${formatBalance(state.balance)}
        </div>
        {showToken && (
          <div style={{ fontSize: '0.875rem', color: '#64748b' }}>
            ETH
          </div>
        )}
      </div>
      <button
        onClick={handleRefresh}
        disabled={refreshing}
        style={{
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          fontSize: '1rem',
          opacity: refreshing ? 0.5 : 1
        }}
      >
        {refreshing ? '⏳' : '🔄'}
      </button>
    </div>
  );
};

// ============================================================================
// Send Transaction Widget
// ============================================================================

interface SendWidgetProps {
  defaultToken?: string;
  onSuccess?: (hash: string) => void;
  onError?: (error: Error) => void;
}

export const SendWidget: React.FC<SendWidgetProps> = ({
  defaultToken = 'ETH',
  onSuccess,
  onError
}) => {
  const { state, sendTransaction } = useWidget();
  const [to, setTo] = useState('');
  const [amount, setAmount] = useState('');
  const [sending, setSending] = useState(false);
  const [txHash, setTxHash] = useState<string | null>(null);

  const handleSend = async () => {
    if (!to || !amount) return;
    
    setSending(true);
    setTxHash(null);
    
    try {
      const hash = await sendTransaction({ to, amount, token: defaultToken });
      setTxHash(hash);
      setTo('');
      setAmount('');
      onSuccess?.(hash);
    } catch (error) {
      onError?.(error as Error);
    } finally {
      setSending(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '12px',
    borderRadius: '8px',
    border: '1px solid #e2e8f0',
    marginBottom: '12px',
    fontSize: '1rem'
  };

  const buttonStyle: React.CSSProperties = {
    width: '100%',
    padding: '12px',
    borderRadius: '8px',
    border: 'none',
    background: sending ? '#ccc' : 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)',
    color: 'white',
    fontWeight: 600,
    cursor: sending ? 'not-allowed' : 'pointer'
  };

  if (!state.isConnected) {
    return <div style={{ color: '#64748b' }}>Connect wallet first</div>;
  }

  return (
    <div style={{ background: 'white', padding: '20px', borderRadius: '12px', boxShadow: '0 2px 8px rgba(0,0,0,0.1)' }}>
      <h3 style={{ marginBottom: '16px' }}>Send {defaultToken}</h3>
      
      <input
        type="text"
        placeholder="Recipient address (0x...)"
        value={to}
        onChange={(e) => setTo(e.target.value)}
        style={inputStyle}
      />
      
      <input
        type="number"
        placeholder={`Amount (${defaultToken})`}
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        style={inputStyle}
      />
      
      <button
        onClick={handleSend}
        disabled={sending || !to || !amount}
        style={buttonStyle}
      >
        {sending ? 'Sending...' : `Send ${defaultToken}`}
      </button>
      
      {txHash && (
        <div style={{ marginTop: '12px', fontSize: '0.875rem', color: '#22c55e' }}>
          ✅ Transaction sent: {txHash.slice(0, 10)}...
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Quick Swap Widget
// ============================================================================

interface QuickSwapProps {
  onSwap?: (hash: string) => void;
}

export const QuickSwap: React.FC<QuickSwapProps> = ({ onSwap }) => {
  const { state } = useWidget();
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [amount, setAmount] = useState('');
  const [swapping, setSwapping] = useState(false);

  const handleSwap = async () => {
    if (!amount || !state.wallet) return;
    
    setSwapping(true);
    try {
      // Execute swap via API
      const result = await walletApi.swap(
        state.wallet.id,
        fromToken,
        toToken,
        amount
      );
      onSwap?.(result.hash);
    } catch (error) {
      console.error('Swap failed:', error);
    } finally {
      setSwapping(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    padding: '12px',
    borderRadius: '8px',
    border: '1px solid #e2e8f0',
    fontSize: '1rem'
  };

  if (!state.isConnected) {
    return <div style={{ color: '#64748b' }}>Connect wallet to swap</div>;
  }

  return (
    <div style={{ background: 'white', padding: '20px', borderRadius: '12px' }}>
      <h3 style={{ marginBottom: '16px' }}>Quick Swap</h3>
      
      <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
        <input
          type="number"
          placeholder="Amount"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          style={{ ...inputStyle, flex: 1 }}
        />
        <select
          value={fromToken}
          onChange={(e) => setFromToken(e.target.value)}
          style={inputStyle}
        >
          <option value="ETH">ETH</option>
          <option value="USDT">USDT</option>
          <option value="USDC">USDC</option>
          <option value="BNB">BNB</option>
        </select>
      </div>
      
      <div style={{ textAlign: 'center', margin: '8px 0' }}>↓</div>
      
      <select
        value={toToken}
        onChange={(e) => setToToken(e.target.value)}
        style={{ ...inputStyle, width: '100%', marginBottom: '12px' }}
      >
        <option value="ETH">ETH</option>
        <option value="USDT">USDT</option>
        <option value="USDC">USDC</option>
        <option value="BNB">BNB</option>
      </select>
      
      <button
        onClick={handleSwap}
        disabled={swapping || !amount}
        style={{
          width: '100%',
          padding: '12px',
          borderRadius: '8px',
          border: 'none',
          background: swapping ? '#ccc' : '#f97316',
          color: 'white',
          fontWeight: 600,
          cursor: swapping ? 'not-allowed' : 'pointer'
        }}
      >
        {swapping ? 'Swapping...' : `Swap ${fromToken} → ${toToken}`}
      </button>
    </div>
  );
};

// ============================================================================
// Export
// ============================================================================

export default {
  Provider: TigerWidgetProvider,
  ConnectButton,
  BalanceWidget,
  SendWidget,
  QuickSwap,
  useWidget
};
