// TigerWallet Master - Desktop App
import React, { useState, useEffect } from 'react';

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
}

// Theme Context
const MasterThemeContext = React.createContext<any>(null);

const MasterThemeProvider = ({ children }: { children: React.ReactNode }) => {
  const [isDarkMode, setIsDarkMode] = useState(true);
  
  useEffect(() => {
    const stored = localStorage.getItem('master_wallet_theme');
    if (stored) setIsDarkMode(stored === 'dark');
  }, []);
  
  const toggleTheme = () => {
    const newTheme = !isDarkMode;
    setIsDarkMode(newTheme);
    localStorage.setItem('master_wallet_theme', newTheme ? 'dark' : 'light');
  };
  
  return (
    <MasterThemeContext.Provider value={{ isDarkMode, toggleTheme }}>
      {children}
    </MasterThemeContext.Provider>
  );
};

// Sidebar Component
const MasterSidebar = ({ currentPage, setCurrentPage }: { currentPage: string; setCurrentPage: (page: string) => void }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'wallets', label: 'Wallets', icon: '💼' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'auto-sign', label: 'Auto Sign', icon: '🔑' },
    { id: 'analytics', label: 'Analytics', icon: '📈' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];
  
  return (
    <div className="w-64 bg-gray-800 border-r border-gray-700 flex flex-col">
      <div className="p-4 border-b border-gray-700">
        <div className="flex items-center space-x-3">
          <span className="text-2xl">🏦</span>
          <span className="text-xl font-bold">MasterWallet</span>
        </div>
      </div>
      
      <nav className="flex-1 p-4">
        {menuItems.map(item => (
          <button
            key={item.id}
            onClick={() => setCurrentPage(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg mb-2 transition-colors ${
              currentPage === item.id
                ? 'bg-blue-500 text-white'
                : 'text-gray-400 hover:bg-gray-700 hover:text-white'
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
    </div>
  );
};

// Header Component
const MasterHeader = ({ onToggleTheme, isDarkMode }: { onToggleTheme: () => void; isDarkMode: boolean }) => {
  const [stats, setStats] = useState<MasterWallet>({
    address: '0x742d...12eB3',
    totalVolume: '$12.5M',
    subWalletCount: 15,
    userCount: 8,
    pendingTx: 3
  });
  
  return (
    <header className="h-16 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-6">
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search wallets, users, transactions..."
          className="px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm w-96"
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <div className="text-right">
          <div className="text-sm text-gray-400">Total Volume</div>
          <div className="font-bold">{stats.totalVolume}</div>
        </div>
        
        <button 
          onClick={onToggleTheme}
          className="p-2 bg-gray-700 rounded-lg hover:bg-gray-600"
        >
          {isDarkMode ? '☀️' : '🌙'}
        </button>
      </div>
    </header>
  );
};

// Dashboard Component
const MasterDashboard = () => {
  const stats = {
    totalWallets: 15,
    totalVolume: '$12.5M',
    totalUsers: 8,
    pendingTx: 3
  };
  
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      
      <div className="grid grid-cols-4 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">💼 Total Wallets</div>
          <div className="text-3xl font-bold">{stats.totalWallets}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">💰 Total Volume</div>
          <div className="text-3xl font-bold">{stats.totalVolume}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">👥 Total Users</div>
          <div className="text-3xl font-bold">{stats.totalUsers}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">⏳ Pending Tx</div>
          <div className="text-3xl font-bold">{stats.pendingTx}</div>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-4">
            <button className="p-4 bg-blue-600 rounded-lg hover:bg-blue-700">➕ Create Wallet</button>
            <button className="p-4 bg-green-600 rounded-lg hover:bg-green-700">👤 Add User</button>
            <button className="p-4 bg-orange-600 rounded-lg hover:bg-orange-700">🔑 Auto Sign</button>
            <button className="p-4 bg-purple-600 rounded-lg hover:bg-purple-700">📊 Analytics</button>
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
          {[1,2,3,4,5].map(i => (
            <div key={i} className="flex items-center justify-between py-2 border-b border-gray-700">
              <div className="flex items-center space-x-2">
                <span>📤</span>
                <span>Transaction Sent</span>
              </div>
              <span className="text-green-500">+$5,000</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Wallets Component
const MasterWallets = () => {
  const wallets: SubWallet[] = [
    { id: '1', name: 'User Wallet 1', address: '0x111...111', balance: '$45,000', status: 'Active' },
    { id: '2', name: 'User Wallet 2', address: '0x222...222', balance: '$23,500', status: 'Active' },
    { id: '3', name: 'User Wallet 3', address: '0x333...333', balance: '$12,000', status: 'Inactive' },
    { id: '4', name: 'User Wallet 4', address: '0x444...444', balance: '$8,750', status: 'Active' },
    { id: '5', name: 'User Wallet 5', address: '0x555...555', balance: '$5,200', status: 'Active' },
  ];
  
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Sub-Wallets</h1>
        <button className="px-4 py-2 bg-blue-500 rounded-lg hover:bg-blue-600">➕ Add Wallet</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Address</th>
              <th className="px-6 py-3 text-left">Balance</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {wallets.map(wallet => (
              <tr key={wallet.id} className="border-b border-gray-700">
                <td className="px-6 py-4">{wallet.name}</td>
                <td className="px-6 py-4 font-mono text-sm">{wallet.address}</td>
                <td className="px-6 py-4">{wallet.balance}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${wallet.status === 'Active' ? 'bg-green-500' : 'bg-gray-500'}`}>
                    {wallet.status}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline">Edit</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Settings Component  
const MasterSettings = ({ isDarkMode, toggleTheme }: { isDarkMode: boolean; toggleTheme: () => void }) => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Appearance</h2>
        <div className="flex items-center justify-between">
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
        <div className="space-y-2">
          <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Auto-Sign Rules</button>
          <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">User Permissions</button>
          <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">API Keys</button>
          <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Two-Factor Auth</button>
        </div>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">About</h2>
        <div className="flex justify-between">
          <span>Version</span>
          <span className="text-gray-400">1.0.0</span>
        </div>
      </div>
    </div>
  );
};

// Main App
const MasterDesktopApp = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [isDarkMode, setIsDarkMode] = useState(true);
  
  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
    localStorage.setItem('master_wallet_theme', !isDarkMode ? 'dark' : 'light');
  };
  
  useEffect(() => {
    const stored = localStorage.getItem('master_wallet_theme');
    if (stored) setIsDarkMode(stored === 'dark');
  }, []);
  
  return (
    <div className={`flex h-screen ${isDarkMode ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <MasterSidebar currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <MasterHeader onToggleTheme={toggleTheme} isDarkMode={isDarkMode} />
        <main className="flex-1 overflow-auto p-6">
          {currentPage === 'dashboard' && <MasterDashboard />}
          {currentPage === 'wallets' && <MasterWallets />}
          {currentPage === 'settings' && <MasterSettings isDarkMode={isDarkMode} toggleTheme={toggleTheme} />}
        </main>
      </div>
    </div>
  );
};

export default MasterDesktopApp;
