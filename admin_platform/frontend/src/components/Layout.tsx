/**
 * TigerWallet Admin Platform - Main Layout Component
 * Provides consistent layout with theme switching for all pages
 */

import React, { useState, ReactNode } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { authService } from '../services/api';

interface LayoutProps {
  children: ReactNode;
}

interface NavItem {
  label: string;
  path: string;
  icon: string;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/dashboard', icon: '📊' },
  { label: 'Users', path: '/users', icon: '👥' },
  { label: 'KYC', path: '/kyc', icon: '🆔' },
  { label: 'Transactions', path: '/transactions', icon: '💸' },
  { label: 'Trading Pairs', path: '/pairs', icon: '🔄' },
  { label: 'Liquidity', path: '/liquidity', icon: '🌊' },
  { label: 'Blockchains', path: '/chains', icon: '⛓️' },
  { label: 'Fees', path: '/fees', icon: '💰' },
  { label: 'Bots', path: '/bots', icon: '🤖' },
  { label: 'CEX', path: '/cex', icon: '🏦' },
  { label: 'DEX', path: '/dex', icon: '🔀' },
  { label: 'Token Listings', path: '/token-listings', icon: '📋' },
  { label: 'API Keys', path: '/api-keys', icon: '🔑' },
];

const superAdminItems: NavItem[] = [
  { label: 'Admins', path: '/admins', icon: '👤' },
  { label: 'White Labels', path: '/white-labels', icon: '🏷️' },
  { label: 'Audit Logs', path: '/audit', icon: '📝' },
  { label: 'Settings', path: '/settings', icon: '⚙️' },
];

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const { theme, toggleTheme } = useTheme();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [currentPage, setCurrentPage] = useState('Dashboard');
  const user = authService.getUser();
  const isSuperAdmin = user?.role === 'super_admin';

  const handleLogout = async () => {
    await authService.logout();
    window.location.href = '/login';
  };

  return (
    <div className={`min-h-screen flex ${theme === 'dark' ? 'bg-gray-900' : 'bg-gray-50'}`}>
      {/* Sidebar */}
      <aside
        className={`fixed left-0 top-0 h-full transition-all duration-300 z-50 ${
          sidebarOpen ? 'w-64' : 'w-16'
        } ${theme === 'dark' ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-r`}
      >
        {/* Logo */}
        <div className={`h-16 flex items-center justify-center border-b ${
          theme === 'dark' ? 'border-gray-700' : 'border-gray-200'
        }`}>
          <span className="text-xl font-bold text-blue-500">🐯 TigerWallet</span>
        </div>

        {/* Navigation */}
        <nav className="p-2 overflow-y-auto h-[calc(100vh-4rem)]">
          {/* Main Navigation */}
          <div className="mb-4">
            <h3 className={`px-3 py-2 text-xs font-semibold uppercase ${
              theme === 'dark' ? 'text-gray-400' : 'text-gray-500'
            }`}>
              {sidebarOpen ? 'Main Menu' : '⬡'}
            </h3>
            {navItems.map((item) => (
              <button
                key={item.path}
                onClick={() => setCurrentPage(item.label)}
                className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                  currentPage === item.label
                    ? 'bg-blue-500 text-white'
                    : theme === 'dark'
                    ? 'text-gray-300 hover:bg-gray-700'
                    : 'text-gray-700 hover:bg-gray-100'
                }`}
              >
                <span className="text-lg">{item.icon}</span>
                {sidebarOpen && <span>{item.label}</span>}
              </button>
            ))}
          </div>

          {/* Super Admin Navigation */}
          {isSuperAdmin && (
            <div>
              <h3 className={`px-3 py-2 text-xs font-semibold uppercase ${
                theme === 'dark' ? 'text-gray-400' : 'text-gray-500'
              }`}>
                {sidebarOpen ? 'Super Admin' : '⭐'}
              </h3>
              {superAdminItems.map((item) => (
                <button
                  key={item.path}
                  onClick={() => setCurrentPage(item.label)}
                  className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                    currentPage === item.label
                      ? 'bg-blue-500 text-white'
                      : theme === 'dark'
                      ? 'text-gray-300 hover:bg-gray-700'
                      : 'text-gray-700 hover:bg-gray-100'
                  }`}
                >
                  <span className="text-lg">{item.icon}</span>
                  {sidebarOpen && <span>{item.label}</span>}
                </button>
              ))}
            </div>
          )}
        </nav>

        {/* Toggle Sidebar Button */}
        <button
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className={`absolute -right-3 top-20 w-6 h-6 rounded-full flex items-center justify-center ${
            theme === 'dark' ? 'bg-gray-700 text-gray-300' : 'bg-white border border-gray-200 text-gray-500'
          } shadow-md`}
        >
          {sidebarOpen ? '◀' : '▶'}
        </button>
      </aside>

      {/* Main Content */}
      <main
        className={`flex-1 transition-all duration-300 ${
          sidebarOpen ? 'ml-64' : 'ml-16'
        }`}
      >
        {/* Header */}
        <header className={`h-16 flex items-center justify-between px-6 border-b ${
          theme === 'dark' ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'
        }`}>
          <h1 className={`text-xl font-semibold ${
            theme === 'dark' ? 'text-white' : 'text-gray-800'
          }`}>
            {currentPage}
          </h1>

          <div className="flex items-center gap-4">
            {/* Theme Toggle */}
            <button
              onClick={toggleTheme}
              className={`p-2 rounded-lg transition-colors ${
                theme === 'dark'
                  ? 'hover:bg-gray-700 text-yellow-400'
                  : 'hover:bg-gray-100 text-gray-600'
              }`}
              title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} theme`}
            >
              {theme === 'dark' ? '☀️' : '🌙'}
            </button>

            {/* Notifications */}
            <button className={`p-2 rounded-lg transition-colors ${
              theme === 'dark' ? 'hover:bg-gray-700 text-gray-300' : 'hover:bg-gray-100 text-gray-600'
            }`}>
              🔔
            </button>

            {/* User Menu */}
            <div className="flex items-center gap-3">
              <div className={`text-sm ${theme === 'dark' ? 'text-gray-300' : 'text-gray-700'}`}>
                <span className="font-medium">{user?.username || 'Admin'}</span>
                <span className={`mx-2 ${theme === 'dark' ? 'text-gray-500' : 'text-gray-400'}`}>|</span>
                <span className={`text-xs ${theme === 'dark' ? 'text-gray-500' : 'text-gray-400'}`}>
                  {user?.role || 'admin'}
                </span>
              </div>
              
              <button
                onClick={handleLogout}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  theme === 'dark'
                    ? 'bg-red-600 hover:bg-red-700 text-white'
                    : 'bg-red-500 hover:bg-red-600 text-white'
                }`}
              >
                Logout
              </button>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <div className="p-6">
          {children}
        </div>
      </main>
    </div>
  );
};

export default Layout;
