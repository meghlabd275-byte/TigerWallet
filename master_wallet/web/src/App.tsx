// TigerWallet Master - Web App Main Component
import React, { useState, useEffect } from 'react';
import { ThemeContext } from './index';

// Types
interface MasterWallet {
  address: string;
  totalVolume: string;
  subWalletCount: number;
  userCount: number;
  pendingTx: number;
}

interface SubWallet {
  id: string;
  name: string;
  address: string;
  balance: string;
  status: 'Active' | 'Inactive';
  userCount: number;
}

interface Transaction {
  id: string;
  hash: string;
  type: string;
  amount: string;
  status: string;
  from: string;
  to: string;
  timestamp: string;
}

interface AutoSignRule {
  id: string;
  name: string;
  maxAmount: string;
  enabled: boolean;
  chain: string;
}

// Sidebar Component
const Sidebar = ({ currentPage, setCurrentPage }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'wallets', label: 'Sub-Wallets', icon: '💼' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'auto-sign', label: 'Auto Sign', icon: '🔑' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <aside className="w-64 bg-gray-900 border-r border-gray-800 flex flex-col min-h-screen">
      <div className="p-6 border-b border-gray-800">
        <div className="flex items-center space-x-3">
          <span className="text-3xl">🏦</span>
          <div>
            <h1 className="text-xl font-bold">MasterWallet</h1>
            <p className="text-xs text-gray-400">Enterprise</p>
          </div>
        </div>
      </div>
      
      <nav className="flex-1 p-4">
        {menuItems.map(item => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors ${
              currentPage === item.id
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:bg-gray-800 hover:text-white'
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="p-4 border-t border-gray-800">
        <div className="bg-gray-800 rounded-lg p-3">
          <div className="text-xs text-gray-400">Master Address</div>
          <div className="font-mono text-sm">0x742d...12eB3</div>
        </div>
      </div>
    </aside>
  );
};

// Header Component
const Header = ({ toggleTheme, isDarkMode }) => {
  return (
    <header className="h-16 bg-gray-900 border-b border-gray-800 flex items-center justify-between px-6">
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search wallets, users, transactions..."
          className="px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm w-96 focus:outline-none focus:border-blue-500"
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <button className="px-4 py-2 bg-blue-600 rounded-lg hover:bg-blue-700 text-sm">
          + Create Sub-Wallet
        </button>
        <button 
          onClick={toggleTheme}
          className="p-2 bg-gray-800 rounded-lg hover:bg-gray-700"
        >
          {isDarkMode ? '☀️' : '🌙'}
        </button>
        <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
          <span className="text-sm font-bold">M</span>
        </div>
      </div>
    </header>
  );
};

// API Service
import { masterWalletAPI, SubWallet, Transaction } from './api';

// Dashboard Page
const Dashboard = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState({
    totalWallets: 15,
    totalVolume: '$12.5M',
    totalUsers: 8,
    pendingTx: 3,
    activeRules: 5,
  };

  const recentTransactions: Transaction[] = [
    { id: '1', hash: '0x742d35Cc6634C0532925a3b844Bc9e7595f', type: 'Transfer', amount: '$5,000', status: 'Confirmed', from: '0x111', to: '0x222', timestamp: '2 min ago' },
    { id: '2', hash: '0x1111111111111111111111111111111111111111', type: 'Swap', amount: '$12,500', status: 'Pending', from: '0x333', to: '0x444', timestamp: '5 min ago' },
    { id: '3', hash: '0x2222222222222222222222222222222222222222', type: 'Transfer', amount: '$3,200', status: 'Confirmed', from: '0x555', to: '0x666', timestamp: '10 min ago' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      
      <div className="grid grid-cols-5 gap-4">
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">💼 Total Wallets</div>
          <div className="text-3xl font-bold">{stats.totalWallets}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">💰 Total Volume</div>
          <div className="text-3xl font-bold">{stats.totalVolume}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">👥 Total Users</div>
          <div className="text-3xl font-bold">{stats.totalUsers}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">⏳ Pending Tx</div>
          <div className="text-3xl font-bold text-orange-500">{stats.pendingTx}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">🔑 Active Rules</div>
          <div className="text-3xl font-bold text-green-500">{stats.activeRules}</div>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-4">
            <button className="p-4 bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors">➕ Create Wallet</button>
            <button className="p-4 bg-green-600 rounded-lg hover:bg-green-700 transition-colors">👤 Add User</button>
            <button className="p-4 bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors">🔑 Auto Sign</button>
            <button className="p-4 bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors">📊 Analytics</button>
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Transactions</h2>
          <div className="space-y-3">
            {recentTransactions.map(tx => (
              <div key={tx.id} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
                <div className="flex items-center space-x-3">
                  <span className="text-xl">📤</span>
                  <div>
                    <div className="font-medium">{tx.type}</div>
                    <div className="text-xs text-gray-400 font-mono">{tx.hash.substring(0, 16)}...</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-bold">{tx.amount}</div>
                  <div className={`text-xs ${tx.status === 'Confirmed' ? 'text-green-500' : 'text-orange-500'}`}>{tx.status}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

// Wallets Page
const Wallets = () => {
  const wallets: SubWallet[] = [
    { id: '1', name: 'Trading Wallet', address: '0x1111111111111111111111111111111111111111', balance: '$45,000', status: 'Active', userCount: 3 },
    { id: '2', name: 'Staking Wallet', address: '0x2222222222222222222222222222222222222222', balance: '$23,500', status: 'Active', userCount: 2 },
    { id: '3', name: 'Reserve Wallet', address: '0x3333333333333333333333333333333333333333', balance: '$12,000', status: 'Inactive', userCount: 1 },
    { id: '4', name: 'Marketing Wallet', address: '0x4444444444444444444444444444444444444444', balance: '$8,750', status: 'Active', userCount: 4 },
    { id: '5', name: 'Development Wallet', address: '0x5555555555555555555555555555555555555555', balance: '$5,200', status: 'Active', userCount: 2 },
  ];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Sub-Wallets</h1>
        <button className="px-4 py-2 bg-blue-500 rounded-lg hover:bg-blue-600">➕ Create Sub-Wallet</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Address</th>
              <th className="px-6 py-3 text-left">Balance</th>
              <th className="px-6 py-3 text-left">Users</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {wallets.map(wallet => (
              <tr key={wallet.id} className="border-b border-gray-700 hover:bg-gray-750">
                <td className="px-6 py-4 font-medium">{wallet.name}</td>
                <td className="px-6 py-4 font-mono text-sm">{wallet.address.substring(0, 18)}...</td>
                <td className="px-6 py-4">{wallet.balance}</td>
                <td className="px-6 py-4">{wallet.userCount}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${wallet.status === 'Active' ? 'bg-green-500' : 'bg-gray-500'}`}>
                    {wallet.status}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline mr-3">Manage</button>
                  <button className="text-red-500 hover:underline">Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// AutoSign Page
const AutoSignPage = () => {
  const rules: AutoSignRule[] = [
    { id: '1', name: 'Small Transfers', maxAmount: '$1,000', enabled: true, chain: 'Ethereum' },
    { id: '2', name: 'Weekly Payouts', maxAmount: '$5,000', enabled: true, chain: 'BNB' },
    { id: '3', name: 'Gas Top-ups', maxAmount: '$100', enabled: false, chain: 'All' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Auto-Sign Rules</h1>
        <button className="px-4 py-2 bg-blue-500 rounded-lg hover:bg-blue-600">➕ Create Rule</button>
      </div>
      
      <div className="grid grid-cols-1 gap-4">
        {rules.map(rule => (
          <div key={rule.id} className="bg-gray-800 rounded-xl p-5 flex items-center justify-between">
            <div>
              <div className="font-semibold text-lg">{rule.name}</div>
              <div className="text-gray-400">Max: {rule.maxAmount} • Chain: {rule.chain}</div>
            </div>
            <div className="flex items-center space-x-4">
              <span className={`px-3 py-1 rounded ${rule.enabled ? 'bg-green-500' : 'bg-gray-500'}`}>
                {rule.enabled ? 'Enabled' : 'Disabled'}
              </span>
              <button className="text-blue-500 hover:underline">Edit</button>
              <button className="text-red-500 hover:underline">Delete</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// Settings Page
const SettingsPage = ({ isDarkMode, toggleTheme }) => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Appearance</h2>
        <div className="flex items-center justify-between p-4 bg-gray-700 rounded-lg">
          <span>Dark Mode</span>
          <button 
            onClick={toggleTheme}
            className={`w-14 h-7 rounded-full transition-colors ${isDarkMode ? 'bg-blue-500' : 'bg-gray-500'}`}
          >
            <div className={`w-5 h-5 bg-white rounded-full transform transition-transform ${isDarkMode ? 'translate-x-7' : 'translate-x-1'}`} />
          </button>
        </div>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Security</h2>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🔑 Auto-Sign Rules</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">👥 User Permissions</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🔐 API Keys</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🛡️ Two-Factor Auth</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Network</h2>
        <select className="w-full px-4 py-3 bg-gray-700 rounded-lg">
          <option>Ethereum Mainnet</option>
          <option>BNB Smart Chain</option>
          <option>Polygon</option>
          <option>Arbitrum</option>
        </select>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">About</h2>
        <div className="flex justify-between py-2">
          <span>Version</span>
          <span className="text-gray-400">1.0.0</span>
        </div>
        <div className="flex justify-between py-2">
          <span>Build</span>
          <span className="text-gray-400">2024.1</span>
        </div>
      </div>
    </div>
  );
};

// Main App
const App = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [isDarkMode, setIsDarkMode] = useState(true);
  
  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
  };
  
  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDarkMode);
  }, [isDarkMode]);

  return (
    <div className={`flex min-h-screen ${isDarkMode ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <Sidebar currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <div className="flex-1 flex flex-col">
        <Header toggleTheme={toggleTheme} isDarkMode={isDarkMode} />
        <main className="flex-1 p-6 overflow-auto">
          {currentPage === 'dashboard' && <Dashboard />}
          {currentPage === 'wallets' && <Wallets />}
          {currentPage === 'auto-sign' && <AutoSignPage />}
          {currentPage === 'settings' && <SettingsPage isDarkMode={isDarkMode} toggleTheme={toggleTheme} />}
        </main>
      </div>
    </div>
  );
};

export default App;
