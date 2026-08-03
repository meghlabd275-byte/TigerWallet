// TigerAdmin - Desktop Admin Application
import React, { useState, useEffect } from 'react';

// Types
interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  status: string;
}

interface Transaction {
  id: string;
  hash: string;
  type: string;
  amount: string;
  status: string;
}

interface SystemService {
  name: string;
  status: string;
  uptime: string;
}

// Theme Context
const AdminThemeContext = React.createContext<any>(null);

const AdminThemeProvider = ({ children }: { children: React.ReactNode }) => {
  const [isDarkMode, setIsDarkMode] = useState(true);
  
  useEffect(() => {
    const stored = localStorage.getItem('admin_theme');
    if (stored) setIsDarkMode(stored === 'dark');
  }, []);
  
  const toggleTheme = () => {
    const newTheme = !isDarkMode;
    setIsDarkMode(newTheme);
    localStorage.setItem('admin_theme', newTheme ? 'dark' : 'light');
  };
  
  return (
    <AdminThemeContext.Provider value={{ isDarkMode, toggleTheme }}>
      {children}
    </AdminThemeContext.Provider>
  );
};

// Sidebar Component
const AdminSidebar = ({ currentPage, setCurrentPage }: { currentPage: string; setCurrentPage: (page: string) => void }) => {
  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'users', label: 'Users', icon: '👥' },
    { id: 'transactions', label: 'Transactions', icon: '📜' },
    { id: 'system', label: 'System', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ];
  
  return (
    <div className="w-64 bg-red-900 border-r border-red-800 flex flex-col">
      <div className="p-4 border-b border-red-800">
        <div className="flex items-center space-x-3">
          <span className="text-2xl">🔧</span>
          <span className="text-xl font-bold">Admin Panel</span>
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
    </div>
  );
};

// Header Component
const AdminHeader = ({ onToggleTheme, isDarkMode }: { onToggleTheme: () => void; isDarkMode: boolean }) => {
  return (
    <header className="h-16 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-6">
      <div className="flex items-center space-x-4">
        <input
          type="text"
          placeholder="Search users, transactions..."
          className="px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm w-96"
        />
      </div>
      
      <div className="flex items-center space-x-4">
        <button className="px-4 py-2 bg-red-600 rounded-lg hover:bg-red-700">
          🔔 Notifications (3)
        </button>
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
const AdminDashboard = () => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      
      <div className="grid grid-cols-4 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">👥 Total Users</div>
          <div className="text-3xl font-bold">12,450</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">💰 Total Volume</div>
          <div className="text-3xl font-bold">$45.2M</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">⏳ Pending KYC</div>
          <div className="text-3xl font-bold">89</div>
        </div>
        <div className="bg-gray-800 rounded-xl p-6">
          <div className="text-gray-400 mb-2">❤️ System Health</div>
          <div className="text-3xl font-bold text-green-500">99.9%</div>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-4">
            <button className="p-4 bg-blue-600 rounded-lg hover:bg-blue-700">👥 Manage Users</button>
            <button className="p-4 bg-green-600 rounded-lg hover:bg-green-700">📜 View Transactions</button>
            <button className="p-4 bg-purple-600 rounded-lg hover:bg-purple-700">⚙️ System Config</button>
            <button className="p-4 bg-orange-600 rounded-lg hover:bg-orange-700">📊 Analytics</button>
          </div>
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Admin Actions</h2>
          {[1,2,3,4,5].map(i => (
            <div key={i} className="flex items-center justify-between py-2 border-b border-gray-700">
              <div className="flex items-center space-x-2">
                <span>👤</span>
                <span>New user verified</span>
              </div>
              <span className="text-gray-400 text-sm">2 min ago</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Users Component
const AdminUsers = () => {
  const users: AdminUser[] = [
    { id: '1', email: 'user1@example.com', name: 'John Doe', role: 'User', status: 'Verified' },
    { id: '2', email: 'user2@example.com', name: 'Jane Smith', role: 'User', status: 'Pending' },
    { id: '3', email: 'user3@example.com', name: 'Bob Wilson', role: 'User', status: 'Verified' },
    { id: '4', email: 'user4@example.com', name: 'Alice Brown', role: 'User', status: 'Verified' },
    { id: '5', email: 'user5@example.com', name: 'Charlie Davis', role: 'User', status: 'Rejected' },
  ];
  
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Users</h1>
        <button className="px-4 py-2 bg-blue-500 rounded-lg hover:bg-blue-600">➕ Add User</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">Email</th>
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Role</th>
              <th className="px-6 py-3 text-left">Status</th>
              <th className="px-6 py-3 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map(user => (
              <tr key={user.id} className="border-b border-gray-700">
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4">{user.name}</td>
                <td className="px-6 py-4">{user.role}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${
                    user.status === 'Verified' ? 'bg-green-500' : 
                    user.status === 'Pending' ? 'bg-orange-500' : 'bg-red-500'
                  }`}>
                    {user.status}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline mr-2">Edit</button>
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

// Transactions Component
const AdminTransactions = () => {
  const transactions: Transaction[] = [
    { id: '1', hash: '0x742d35Cc6634C0532925a3b844Bc9e7595f', type: 'Transfer', amount: '$5,000', status: 'Confirmed' },
    { id: '2', hash: '0x1111111111111111111111111111111111111111', type: 'Swap', amount: '$12,500', status: 'Pending' },
    { id: '3', hash: '0x2222222222222222222222222222222222222222', type: 'Transfer', amount: '$3,200', status: 'Confirmed' },
    { id: '4', hash: '0x3333333333333333333333333333333333333333', type: 'Stake', amount: '$8,000', status: 'Confirmed' },
    { id: '5', hash: '0x4444444444444444444444444444444444444444', type: 'Transfer', amount: '$1,500', status: 'Failed' },
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
              <tr key={tx.id} className="border-b border-gray-700">
                <td className="px-6 py-4 font-mono text-sm">{tx.hash.substring(0, 20)}...</td>
                <td className="px-6 py-4">{tx.type}</td>
                <td className="px-6 py-4">{tx.amount}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 rounded text-xs ${
                    tx.status === 'Confirmed' ? 'bg-green-500' : 
                    tx.status === 'Pending' ? 'bg-orange-500' : 'bg-red-500'
                  }`}>
                    {tx.status}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-blue-500 hover:underline">View</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// System Component
const AdminSystem = () => {
  const services: SystemService[] = [
    { name: 'API Gateway', status: 'Running', uptime: '99.99%' },
    { name: 'Wallet Service', status: 'Running', uptime: '99.95%' },
    { name: 'Transaction Engine', status: 'Running', uptime: '99.99%' },
    { name: 'Price Feed', status: 'Running', uptime: '99.90%' },
    { name: 'PostgreSQL', status: 'Running', uptime: '99.99%' },
    { name: 'Redis Cache', status: 'Running', uptime: '99.95%' },
  ];
  
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">System Status</h1>
      
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Services</h2>
          {services.slice(0, 4).map((service, i) => (
            <div key={i} className="flex items-center justify-between py-2 border-b border-gray-700">
              <div className="flex items-center space-x-2">
                <span className="text-green-500">✅</span>
                <span>{service.name}</span>
              </div>
              <span className="text-gray-400">{service.uptime}</span>
            </div>
          ))}
        </div>
        
        <div className="bg-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Database</h2>
          {services.slice(4).map((service, i) => (
            <div key={i} className="flex items-center justify-between py-2 border-b border-gray-700">
              <div className="flex items-center space-x-2">
                <span className="text-green-500">✅</span>
                <span>{service.name}</span>
              </div>
              <span className="text-gray-400">{service.uptime}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Settings Component  
const AdminSettings = ({ isDarkMode, toggleTheme }: { isDarkMode: boolean; toggleTheme: () => void }) => {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Appearance</h2>
        <div className="flex items-center justify-between">
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
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Fee Configuration</button>
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Token Listing</button>
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">KYC Levels</button>
      </div>
      
      <div className="bg-gray-800 rounded-xl p-6 space-y-4">
        <h2 className="text-lg font-semibold">Security</h2>
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Admin Users</button>
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">Permissions</button>
        <button className="w-full text-left px-4 py-2 bg-gray-700 rounded hover:bg-gray-600">API Keys</button>
      </div>
    </div>
  );
};

// Main App
const AdminDesktopApp = () => {
  const [currentPage, setCurrentPage] = useState('dashboard');
  const [isDarkMode, setIsDarkMode] = useState(true);
  
  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
    localStorage.setItem('admin_theme', !isDarkMode ? 'dark' : 'light');
  };
  
  useEffect(() => {
    const stored = localStorage.getItem('admin_theme');
    if (stored) setIsDarkMode(stored === 'dark');
  }, []);
  
  return (
    <div className={`flex h-screen ${isDarkMode ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <AdminSidebar currentPage={currentPage} setCurrentPage={setCurrentPage} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <AdminHeader onToggleTheme={toggleTheme} isDarkMode={isDarkMode} />
        <main className="flex-1 overflow-auto p-6">
          {currentPage === 'dashboard' && <AdminDashboard />}
          {currentPage === 'users' && <AdminUsers />}
          {currentPage === 'transactions' && <AdminTransactions />}
          {currentPage === 'system' && <AdminSystem />}
          {currentPage === 'settings' && <AdminSettings isDarkMode={isDarkMode} toggleTheme={toggleTheme} />}
        </main>
      </div>
    </div>
  );
};

export default AdminDesktopApp;
