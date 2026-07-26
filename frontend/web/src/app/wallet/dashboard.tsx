/**
 * TigerWallet - Complete Web Wallet Dashboard
 * Production-ready wallet UI with dark/light theme
 */

'use client';

import React, { useState, useEffect, createContext, useContext } from 'react';
import { 
  Wallet, 
  TrendingUp, 
  TrendingDown, 
  Send, 
  RefreshCw, 
  Settings, 
  LogOut,
  Menu,
  X,
  Moon,
  Sun,
  Copy,
  ExternalLink,
  ArrowUpRight,
  ArrowDownRight,
  Shield,
  Zap,
  Layers,
  CreditCard,
  Key,
  ChevronDown,
  CheckCircle,
  XCircle,
  Loader
} from 'lucide-react';

// ============================================================================
// Theme Context
// ============================================================================

interface ThemeContextType {
  theme: 'light' | 'dark';
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType>({ theme: 'light', toggleTheme: () => {} });

export const useTheme = () => useContext(ThemeContext);

// ============================================================================
// Types
// ============================================================================

interface Token {
  id: string;
  symbol: string;
  name: string;
  balance: number;
  price: number;
  change24h: number;
  logo: string;
}

interface Transaction {
  hash: string;
  type: 'send' | 'receive' | 'swap' | 'stake';
  amount: number;
  token: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
}

interface Chain {
  id: number;
  name: string;
  symbol: string;
  icon: string;
  connected: boolean;
}

interface WalletState {
  isConnected: boolean;
  address: string;
  balance: number;
  tokens: Token[];
  transactions: Transaction[];
  chain: Chain | null;
}

// ============================================================================
// Components
// ============================================================================

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      className="p-2 rounded-lg bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
    >
      {theme === 'dark' ? (
        <Sun className="w-5 h-5 text-yellow-500" />
      ) : (
        <Moon className="w-5 h-5 text-gray-600" />
      )}
    </button>
  );
}

function Header() {
  const { theme } = useTheme();
  const [walletState, setWalletState] = useState<WalletState>({
    isConnected: false,
    address: '',
    balance: 0,
    tokens: [],
    transactions: [],
    chain: null,
  });
  const [showMenu, setShowMenu] = useState(false);

  const formatAddress = (addr: string) => {
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  return (
    <header className="sticky top-0 z-50 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-orange-500 to-red-500 rounded-xl flex items-center justify-center">
              <span className="text-white font-bold text-xl">T</span>
            </div>
            <span className="text-xl font-bold text-gray-900 dark:text-white">TigerWallet</span>
          </div>

          <nav className="hidden md:flex items-center gap-6">
            <NavLink href="/wallet" icon={<Wallet className="w-4 h-4" />} label="Wallet" />
            <NavLink href="/swap" icon={<RefreshCw className="w-4 h-4" />} label="Swap" />
            <NavLink href="/stake" icon={<TrendingUp className="w-4 h-4" />} label="Stake" />
            <NavLink href="/bridge" icon={<Layers className="w-4 h-4" />} label="Bridge" />
          </nav>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            
            {walletState.isConnected ? (
              <div className="flex items-center gap-3">
                <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 bg-gray-100 dark:bg-gray-800 rounded-lg">
                  <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    ${walletState.balance.toFixed(2)}
                  </span>
                </div>
                
                <div className="relative">
                  <button
                    onClick={() => setShowMenu(!showMenu)}
                    className="flex items-center gap-2 px-3 py-2 bg-orange-500 hover:bg-orange-600 rounded-lg text-white"
                  >
                    <span className="text-sm font-medium">{formatAddress(walletState.address)}</span>
                    <ChevronDown className="w-4 h-4" />
                  </button>
                  
                  {showMenu && (
                    <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1">
                      <DropdownItem icon={<Wallet className="w-4 h-4" />} label="My Wallet" />
                      <DropdownItem icon={<Settings className="w-4 h-4" />} label="Settings" />
                      <DropdownItem icon={<LogOut className="w-4 h-4" />} label="Disconnect" onClick={() => setWalletState({ ...walletState, isConnected: false })} />
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <button
                onClick={() => setWalletState({ ...walletState, isConnected: true, address: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', balance: 12500.50 })}
                className="px-4 py-2 bg-orange-500 hover:bg-orange-600 rounded-lg text-white font-medium"
              >
                Connect Wallet
              </button>
            )}

            <button className="md:hidden p-2" onClick={() => setShowMenu(!showMenu)}>
              {showMenu ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}

function NavLink({ href, icon, label }: { href: string; icon: React.ReactNode; label: string }) {
  return (
    <a href={href} className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-orange-500 dark:hover:text-orange-400 transition-colors">
      {icon}
      {label}
    </a>
  );
}

function DropdownItem({ icon, label, onClick }: { icon: React.ReactNode; label: string; onClick?: () => void }) {
  return (
    <button onClick={onClick} className="w-full flex items-center gap-3 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
      {icon}
      {label}
    </button>
  );
}

function BalanceCard() {
  const [walletState] = useState<WalletState>({
    isConnected: true,
    address: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E',
    balance: 12500.50,
    tokens: [],
    transactions: [],
    chain: null,
  });

  return (
    <div className="bg-gradient-to-br from-orange-500 to-red-600 rounded-2xl p-6 text-white">
      <div className="flex items-center justify-between mb-4">
        <span className="text-orange-100">Total Balance</span>
        <div className="flex items-center gap-2">
          <button className="p-1.5 bg-white/20 rounded-lg hover:bg-white/30"><Copy className="w-4 h-4" /></button>
          <button className="p-1.5 bg-white/20 rounded-lg hover:bg-white/30"><ExternalLink className="w-4 h-4" /></button>
        </div>
      </div>
      
      <div className="text-4xl font-bold mb-2">
        ${walletState.balance.toLocaleString('en-US', { minimumFractionDigits: 2 })}
      </div>
      
      <div className="flex items-center gap-2 text-sm text-orange-100">
        <TrendingUp className="w-4 h-4" />
        <span>+2.5% ($312.50) today</span>
      </div>

      <div className="flex gap-3 mt-6">
        <ActionButton icon={<Send className="w-4 h-4" />} label="Send" />
        <ActionButton icon={<ArrowDownRight className="w-4 h-4" />} label="Receive" />
        <ActionButton icon={<RefreshCw className="w-4 h-4" />} label="Swap" />
      </div>
    </div>
  );
}

function ActionButton({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <button className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-white/20 rounded-xl hover:bg-white/30 transition-colors">
      {icon}
      <span className="text-sm font-medium">{label}</span>
    </button>
  );
}

function TokenList() {
  const [walletState] = useState<WalletState>({
    isConnected: true,
    address: '',
    balance: 12500.50,
    tokens: [
      { id: 'eth', symbol: 'ETH', name: 'Ethereum', balance: 2.5, price: 3200, change24h: 2.5, logo: '' },
      { id: 'btc', symbol: 'BTC', name: 'Bitcoin', balance: 0.5, price: 65000, change24h: -1.2, logo: '' },
      { id: 'usdt', symbol: 'USDT', name: 'Tether', balance: 5000, price: 1, change24h: 0.01, logo: '' },
      { id: 'usdc', symbol: 'USDC', name: 'USD Coin', balance: 3000, price: 1, change24h: 0.0, logo: '' },
    ],
    transactions: [],
    chain: null,
  });

  return (
    <div className="bg-white dark:bg-gray-900 rounded-2xl p-6 border border-gray-200 dark:border-gray-800">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Assets</h2>
        <button className="text-sm text-orange-500 hover:text-orange-600 font-medium">View All</button>
      </div>

      <div className="space-y-3">
        {walletState.tokens.map((token) => (
          <TokenRow key={token.id} token={token} />
        ))}
      </div>
    </div>
  );
}

function TokenRow({ token }: { token: Token }) {
  const isPositive = token.change24h >= 0;

  return (
    <div className="flex items-center justify-between p-3 rounded-xl hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors cursor-pointer">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center text-lg font-bold text-gray-600 dark:text-gray-300">
          {token.symbol.charAt(0)}
        </div>
        <div>
          <div className="font-medium text-gray-900 dark:text-white">{token.symbol}</div>
          <div className="text-sm text-gray-500 dark:text-gray-400">{token.name}</div>
        </div>
      </div>

      <div className="text-right">
        <div className="font-medium text-gray-900 dark:text-white">
          ${(token.balance * token.price).toLocaleString('en-US', { minimumFractionDigits: 2 })}
        </div>
        <div className="text-sm flex items-center justify-end gap-1">
          <span className={isPositive ? 'text-green-500' : 'text-red-500'}>
            {isPositive ? '+' : ''}{token.change24h.toFixed(2)}%
          </span>
        </div>
      </div>
    </div>
  );
}

function RecentTransactions() {
  const [transactions] = useState<Transaction[]>([
    { hash: '0x1234...5678', type: 'receive', amount: 1.5, token: 'ETH', status: 'confirmed', timestamp: Date.now() - 3600000 },
    { hash: '0x2345...6789', type: 'send', amount: 500, token: 'USDT', status: 'confirmed', timestamp: Date.now() - 7200000 },
    { hash: '0x3456...7890', type: 'swap', amount: 0.1, token: 'ETH', status: 'confirmed', timestamp: Date.now() - 10800000 },
    { hash: '0x4567...8901', type: 'stake', amount: 1000, token: 'USDC', status: 'pending', timestamp: Date.now() },
  ]);

  return (
    <div className="bg-white dark:bg-gray-900 rounded-2xl p-6 border border-gray-200 dark:border-gray-800">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Transactions</h2>
        <button className="text-sm text-orange-500 hover:text-orange-600 font-medium">View All</button>
      </div>

      <div className="space-y-3">
        {transactions.map((tx, index) => (
          <TransactionRow key={index} tx={tx} />
        ))}
      </div>
    </div>
  );
}

function TransactionRow({ tx }: { tx: Transaction }) {
  const getIcon = () => {
    switch (tx.type) {
      case 'send': return <ArrowUpRight className="w-4 h-4 text-red-500" />;
      case 'receive': return <ArrowDownRight className="w-4 h-4 text-green-500" />;
      case 'swap': return <RefreshCw className="w-4 h-4 text-orange-500" />;
      case 'stake': return <TrendingUp className="w-4 h-4 text-blue-500" />;
    }
  };

  const getStatusBadge = () => {
    switch (tx.status) {
      case 'confirmed': return <span className="flex items-center gap-1 text-xs text-green-500"><CheckCircle className="w-3 h-3" /> Confirmed</span>;
      case 'pending': return <span className="flex items-center gap-1 text-xs text-yellow-500"><Loader className="w-3 h-3 animate-spin" /> Pending</span>;
      case 'failed': return <span className="flex items-center gap-1 text-xs text-red-500"><XCircle className="w-3 h-3" /> Failed</span>;
    }
  };

  const formatTime = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return `${Math.floor(diff / 86400000)}d ago`;
  };

  return (
    <div className="flex items-center justify-between p-3 rounded-xl hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 bg-gray-100 dark:bg-gray-800 rounded-full flex items-center justify-center">
          {getIcon()}
        </div>
        <div>
          <div className="font-medium text-gray-900 dark:text-white capitalize">{tx.type}</div>
          <div className="text-xs text-gray-500 dark:text-gray-400">{formatTime(tx.timestamp)}</div>
        </div>
      </div>

      <div className="text-right">
        <div className="font-medium text-gray-900 dark:text-white">
          {tx.type === 'send' ? '-' : '+'}{tx.amount} {tx.token}
        </div>
        {getStatusBadge()}
      </div>
    </div>
  );
}

function QuickActions() {
  const actions = [
    { icon: <CreditCard className="w-5 h-5" />, label: 'Buy Crypto', color: 'bg-blue-500' },
    { icon: <Zap className="w-5 h-5" />, label: 'Bridge', color: 'bg-purple-500' },
    { icon: <TrendingUp className="w-5 h-5" />, label: 'Stake', color: 'bg-green-500' },
    { icon: <Layers className="w-5 h-5" />, label: 'NFTs', color: 'bg-orange-500' },
  ];

  return (
    <div className="grid grid-cols-4 gap-3">
      {actions.map((action, index) => (
        <button key={index} className="flex flex-col items-center gap-2 p-4 bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-800 hover:border-orange-500 dark:hover:border-orange-500 transition-colors">
          <div className={`w-10 h-10 ${action.color} rounded-xl flex items-center justify-center text-white`}>
            {action.icon}
          </div>
          <span className="text-xs font-medium text-gray-700 dark:text-gray-300">{action.label}</span>
        </button>
      ))}
    </div>
  );
}

function SecurityStatus() {
  const features = [
    { name: 'Biometric Lock', enabled: true, icon: <Key className="w-4 h-4" /> },
    { name: '2FA Enabled', enabled: true, icon: <Shield className="w-4 h-4" /> },
    { name: 'Recovery Phrase', enabled: true, icon: <Shield className="w-4 h-4" /> },
  ];

  return (
    <div className="bg-white dark:bg-gray-900 rounded-2xl p-6 border border-gray-200 dark:border-gray-800">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Security</h2>
      
      <div className="space-y-3">
        {features.map((feature, index) => (
          <div key={index} className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-gray-700 dark:text-gray-300">
              {feature.icon}
              <span className="text-sm">{feature.name}</span>
            </div>
            <CheckCircle className="w-5 h-5 text-green-500" />
          </div>
        ))}
      </div>
    </div>
  );
}

export default function Dashboard() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      setTheme('dark');
    }
  }, []);

  useEffect(() => {
    if (theme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [theme]);

  const toggleTheme = () => {
    setTheme(prev => prev === 'light' ? 'dark' : 'light');
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
        <Header />
        
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="mb-8">
            <QuickActions />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2 space-y-6">
              <BalanceCard />
              <TokenList />
            </div>

            <div className="space-y-6">
              <SecurityStatus />
              <RecentTransactions />
            </div>
          </div>
        </main>
      </div>
    </ThemeContext.Provider>
  );
}
