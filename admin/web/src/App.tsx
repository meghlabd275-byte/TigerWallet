// TigerWallet Admin - Web App Main Component
import React, { useState, useEffect } from 'react';
import { ThemeContext } from './index';

// Types
interface User {
  id: string;
  email: string;
  name: string;
  kycStatus: string;
  createdAt: string;
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

interface SystemService {
  name: string;
  status: string;
  uptime: string;
  latency: string;
}

// Sidebar Component
const Sidebar = ({ currentPage, setCurrentPage }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'kyc', label: 'KYC Verification', icon: '✅' },
    { id: 'tokens', label: 'Token Management', icon: '🪙' },
    { id: 'fees', label: 'Fee Configuration', icon: '💰' },
    { id: 'system', label: 'System Status', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <aside className="w-64 bg-red-900 border-r border-red-800 flex flex-col min-h-screen">
      <div className="p-6 border-b border-red-800">
        <div className="flex items-center space-x-3">
          <span className="text-3xl">🔧</span>
          <div>
            <h1 className="text-xl font-bold">Admin Panel</h1>
            <p className="text-xs text-red-300">Platform Management</p>
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
                ? 'bg-red-600 text-white'
                : 'text-red-200 hover:bg-red-800 hover:text-white'
            }`}
          >
            <span>{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="p-4 border-t border-red-800">
        <div className="bg-red-800 rounded-lg p-3">
          <div className="text-xs text-red-300">Admin Level</div>
          <div className="font-medium">Super Admin</div>
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
          placeholder="Search users, transactions, tokens..."
          className="px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm w-96 focus:outline-none focus:border-red-500"
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <button className="px-4 py-2 bg-red-600 rounded-lg hover:bg-red-700 text-sm">
          🔔 Notifications (5)
        </button>
        <button 
          onClick={toggleTheme}
          className="p-2 bg-gray-800 rounded-lg hover:bg-gray-700"
        >
          {isDarkMode ? '☀️' : '🌙'}
        </button>
        <div className="w-8 h-8 bg-red-600 rounded-full flex items-center justify-center">
          <span className="text-sm font-bold">A</span>
        </div>
      </div>
    </header>
  );
};

// Dashboard Page
const Dashboard = () => {
  const stats = {
    totalUsers: 12450,
    totalVolume: '$45.2M',
    pendingKYC: 89,
    systemHealth: '99.9%',
    dailyTransactions: 1523,
    activeUsers: 8234,
  };

  const recentActivity = [
    { type: 'user_verified', message: 'New user verified', email: 'user@example.com', time: '2 min ago' },
    { type: 'transaction', message: 'Large transaction detected', amount: '$50,000', time: '5 min ago' },
    { type: 'kyc', message: 'KYC submitted', email: 'new@example.com', time: '10 min ago' },
    { type: 'token', message: 'Token listed', name: 'TIGER', time: '15 min ago' },
    { type: 'suspicious', message: 'Suspicious activity', details: 'Multiple failed logins', time: '20 min ago' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      
      <div className="grid grid-cols-6 gap-4">
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">👥 Total Users</div>
          <div className="text-3xl font-bold">{stats.totalUsers.toLocaleString()}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">💰 Total Volume</div>
          <div className="text-3xl font-bold">{stats.totalVolume}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">⏳ Pending KYC</div>
          <div className="text-3xl font-bold text-orange-500">{stats.pendingKYC}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">❤️ System Health</div>
          <div className="text-3xl font-bold text-green-500">{stats.systemHealth}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">📜 Daily Tx</div>
          <div className="text-3xl font-bold">{stats.dailyTransactions.toLocaleString()}</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-5">
          <div className="text-gray-400 mb-2">👤 Active Users</div>
          <div className="text-3xl font-bold">{stats.activeUsers.toLocaleString()}</div>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-4">
            <button className="p-4 bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors">👥 Manage Users</button>
            <button className="p-4 bg-green-600 rounded-lg hover:bg-green-700 transition-colors">📜 View Transactions</button>
            <button className="p-4 bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors">✅ Review KYC</button>
            <button className="p-4 bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors">🪙 Manage Tokens</button>
            <button className="p-4 bg-red-600 rounded-lg hover:bg-red-700 transition-colors">⚙️ System Config</button>
            <button className="p-4 bg-cyan-600 rounded-lg hover:bg-cyan-700 transition-colors">📊 Analytics</button>
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Admin Actions</h2>
          <div className="space-y-3">
            {recentActivity.map((activity, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
                <div className="flex items-center space-x-3">
                  <span className="text-xl">
                    {activity.type === 'user_verified' ? '👤' : 
                     activity.type === 'transaction' ? '💰' :
                     activity.type === 'kyc' ? '✅' :
                     activity.type === 'token' ? '🪙' : '⚠️'}
                  </span>
                  <div>
                    <div className="font-medium">{activity.message}</div>
                    <div className="text-xs text-gray-400">
                      {activity.email || activity.amount || activity.name || activity.details}
                    </div>
                  </div>
                </div>
                <div className="text-xs text-gray-400">{activity.time}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

// Users Page
const Users = () => {
  const users: User[] = [
    { id: '1', email: 'user1@example.com', name: 'John Doe', kycStatus: 'Verified', createdAt: '2024-01-15' },
    { id: '2', email: 'user2@example.com', name: 'Jane Smith', kycStatus: 'Pending', createdAt: '2024-01-16' },
    { id: '3', email: 'user3@example.com', name: 'Bob Wilson', kycStatus: 'Verified', createdAt: '2024-01-14' },
    { id: '4', email: 'user4@example.com', name: 'Alice Brown', kycStatus: 'Rejected', createdAt: '2024-01-13' },
    { id: '5', email: 'user5@example.com', name: 'Charlie Davis', kycStatus: 'Verified', createdAt: '2024-01-12' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Users</h1>
        <button className="px-4 py-2 bg-red-500 rounded-lg hover:bg-red-600">➕ Add User</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">Email</th>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">KYC Status</th>
              <th className="px-6 py-3 text-left">Created</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map(user => (
              <tr key={user.id} className="border-b border-gray-700 hover:bg-gray-750">
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4">{user.name}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${
                    user.kycStatus === 'Verified' ? 'bg-green-500' : 
                    user.kycStatus === 'Pending' ? 'bg-orange-500' : 'bg-red-500'
                  }`}>
                    {user.kycStatus}
                  </span>
                </td>
                <td className="px-6 py-4">{user.createdAt}</td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline mr-3">Edit</button>
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

// Transactions Page
const Transactions = () => {
  const transactions: Transaction[] = [
    { id: '1', hash: '0x742d35Cc6634C0532925a3b844Bc9e7595f', type: 'Transfer', amount: '$50,000', status: 'Completed', from: '0x111', to: '0x222', timestamp: '2 min ago' },
    { id: '2', hash: '0x1111111111111111111111111111111111111111', type: 'Swap', amount: '$12,500', status: 'Pending', from: '0x333', to: '0x444', timestamp: '5 min ago' },
    { id: '3', hash: '0x2222222222222222222222222222222222222222', type: 'Transfer', amount: '$3,200', status: 'Completed', from: '0x555', to: '0x666', timestamp: '10 min ago' },
    { id: '4', hash: '0x3333333333333333333333333333333333333333', type: 'Stake', amount: '$8,000', status: 'Flagged', from: '0x777', to: '0x888', timestamp: '15 min ago' },
    { id: '5', hash: '0x4444444444444444444444444444444444444444', type: 'Transfer', amount: '$1,500', status: 'Completed', from: '0x999', to: '0xaaa', timestamp: '20 min ago' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Transactions</h1>
      
      <div className="bg-gray-800 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">Hash</th>
              <th className="px-6 py-3 text-left">Type</th>
              <th className="px-6 py-3 text-left">Amount</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map(tx => (
              <tr key={tx.id} className="border-b border-gray-700 hover:bg-gray-750">
                <td className="px-6 py-4 font-mono text-sm">{tx.hash.substring(0, 20)}...</td>
                <td className="px-6 py-4">{tx.type}</td>
                <td className="px-6 py-4 font-bold">{tx.amount}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${
                    tx.status === 'Completed' ? 'bg-green-500' : 
                    tx.status === 'Pending' ? 'bg-orange-500' : 'bg-red-500'
                  }`}>
                    {tx.status}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline mr-3">View</button>
                  <button className="text-red-500 hover:underline">Flag</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// System Page
const System = () => {
  const services: SystemService[] = [
    { name: 'API Gateway', status: 'Running', uptime: '99.99%', latency: '12ms' },
    { name: 'Wallet Service', status: 'Running', uptime: '99.95%', latency: '15ms' },
    { name: 'Transaction Engine', status: 'Running', uptime: '99.99%', latency: '8ms' },
    { name: 'Price Feed', status: 'Running', uptime: '99.90%', latency: '25ms' },
    { name: 'PostgreSQL', status: 'Running', uptime: '99.99%', latency: '5ms' },
    { name: 'Redis Cache', status: 'Running', uptime: '99.95%', latency: '1ms' },
    { name: 'Ethereum RPC', status: 'Running', uptime: '99.80%', latency: '150ms' },
    { name: 'BSC RPC', status: 'Running', uptime: '99.85%', latency: '120ms' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">System Status</h1>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Services</h2>
          <div className="space-y-3">
            {services.slice(0, 4).map((service, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
                <div className="flex items-center space-x-3">
                  <span className="text-green-500">✅</span>
                  <span className="font-medium">{service.name}</span>
                </div>
                <div className="text-right">
                  <div className="text-sm">{service.uptime}</div>
                  <div className="text-xs text-gray-400">{service.latency}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Database</h2>
          <div className="space-y-3">
            {services.slice(4, 6).map((service, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
                <div className="flex items-center space-x-3">
                  <span className="text-green-500">✅</span>
                  <span className="font-medium">{service.name}</span>
                </div>
                <div className="text-right">
                  <div className="text-sm">{service.uptime}</div>
                  <div className="text-xs text-gray-400">{service.latency}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6 col-span-2">
          <h2 className="text-lg font-semibold mb-4">Network</h2>
          <div className="grid grid-cols-2 gap-4">
            {services.slice(6).map((service, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-gray-700 rounded-lg">
                <div className="flex items-center space-x-3">
                  <span className="text-green-500">✅</span>
                  <span className="font-medium">{service.name}</span>
                </div>
                <div className="text-right">
                  <div className="text-sm">{service.uptime}</div>
                  <div className="text-xs text-gray-400">{service.latency}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
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
            className={`w-14 h-7 rounded-full transition-colors ${isDarkMode ? 'bg-red-500' : 'bg-gray-500'}`}
          >
            <div className={`w-5 h-5 bg-white rounded-full transform transition-transform ${isDarkMode ? 'translate-x-7' : 'translate-x-1'}`} />
          </button>
        </div>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Platform Settings</h2>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">💰 Fee Configuration</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🪙 Token Listing</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">✅ KYC Levels</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🔒 Security Policies</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Security</h2>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">👥 Admin Users</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🔑 Permissions</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🔐 API Keys</button>
        <button className="w-full text-left px-4 py-3 bg-gray-700 rounded hover:bg-gray-600">🛡️ Two-Factor Auth</button>
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
          {currentPage === 'users' && <Users />}
          {currentPage === 'transactions' && <Transactions />}
          {currentPage === 'system' && <System />}
          {currentPage === 'settings' && <SettingsPage isDarkMode={isDarkMode} toggleTheme={toggleTheme} />}
        </main>
      </div>
    </div>
  );
};

export default App;
