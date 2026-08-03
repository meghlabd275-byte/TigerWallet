import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useTheme, ThemeToggle } from '../context/ThemeContext';
import { authService } from '../services/api';

interface LayoutProps {
  children: React.ReactNode;
}

const menuItems = [
  { path: '/dashboard', label: 'Dashboard', icon: '📊' },
  { path: '/users', label: 'Users', icon: '👥' },
  { path: '/transactions', label: 'Transactions', icon: '💳' },
  { path: '/tokens', label: 'Tokens', icon: '🪙' },
  { path: '/pairs', label: 'Trading Pairs', icon: '⚖️' },
  { path: '/kyc', label: 'KYC', icon: '✅' },
  { path: '/withdrawals', label: 'Withdrawals', icon: '💸' },
  { path: '/chains', label: 'Chains', icon: '🔗' },
  { path: '/fees', label: 'Fees', icon: '💰' },
  { path: '/whitelabels', label: 'White Labels', icon: '🏢' },
  { path: '/settings', label: 'Settings', icon: '⚙️' },
];

export default function Layout({ children }: LayoutProps) {
  const { isDark } = useTheme();
  const location = useLocation();

  const handleLogout = async () => {
    try {
      await authService.logout();
      window.location.href = '/login';
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className={`w-64 ${isDark ? 'bg-gray-800' : 'bg-white'} shadow-lg flex flex-col`}>
        {/* Logo */}
        <div className={`p-6 border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            TigerWallet Admin
          </h1>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto p-4">
          <ul className="space-y-1">
            {menuItems.map((item) => {
              const isActive = location.pathname === item.path;
              return (
                <li key={item.path}>
                  <Link
                    to={item.path}
                    className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                      isActive
                        ? 'bg-blue-600 text-white'
                        : isDark
                        ? 'text-gray-300 hover:bg-gray-700'
                        : 'text-gray-700 hover:bg-gray-100'
                    }`}
                  >
                    <span className="text-lg">{item.icon}</span>
                    <span>{item.label}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* Theme Toggle & Logout */}
        <div className={`p-4 border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <div className="flex items-center justify-between mb-4">
            <span className={isDark ? 'text-gray-400' : 'text-gray-600'}>Theme</span>
            <ThemeToggle />
          </div>
          <button
            onClick={handleLogout}
            className={`w-full py-2 px-4 rounded-lg ${
              isDark
                ? 'bg-red-600 hover:bg-red-700 text-white'
                : 'bg-red-500 hover:bg-red-600 text-white'
            } transition-colors`}
          >
            Logout
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className={`flex-1 overflow-y-auto ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        {children}
      </main>
    </div>
  );
}
