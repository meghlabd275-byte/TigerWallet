/**
 * TigerWallet Super Admin - Layout Component
 * Main layout with sidebar, header, and theme toggle
 */

import React, { useState, useEffect } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useTheme } from '../context/ThemeContext';
import superAdminApi from '../services/api';

const menuItems = [
  { path: '/', icon: '📊', label: 'Dashboard' },
  { path: '/users', icon: '👥', label: 'Users' },
  { path: '/kyc', icon: '🛡️', label: 'KYC' },
  { path: '/transactions', icon: '💸', label: 'Transactions' },
  { path: '/withdrawals', icon: '💰', label: 'Withdrawals' },
  { path: '/tokens', icon: '🪙', label: 'Tokens' },
  { path: '/blockchains', icon: '⛓️', label: 'Blockchains' },
  { path: '/pairs', icon: '🔄', label: 'Trading Pairs' },
  { path: '/fees', icon: '💵', label: 'Fees' },
  { path: '/whitelabels', icon: '🏢', label: 'White Labels' },
  { path: '/admins', icon: '👤', label: 'Admins' },
  { path: '/tickets', icon: '🎫', label: 'Tickets' },
  { path: '/knowledge-base', icon: '📚', label: 'Knowledge Base' },
  { path: '/workflows', icon: '✅', label: 'Workflows' },
  { path: '/reports', icon: '📈', label: 'Reports' },
  { path: '/security', icon: '🔒', label: 'Security' },
  { path: '/api-keys', icon: '🔑', label: 'API Keys' },
  { path: '/webhooks', icon: '🪝', label: 'Webhooks' },
  { path: '/audit-logs', icon: '📝', label: 'Audit Logs' },
  { path: '/system', icon: '⚙️', label: 'System' },
  { path: '/settings', icon: '🔧', label: 'Settings' },
];

export default function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [notifications, setNotifications] = useState(0);
  const { theme, setTheme, resolvedTheme } = useTheme();
  const navigate = useNavigate();

  useEffect(() => {
    loadNotifications();
  }, []);

  const loadNotifications = async () => {
    try {
      const result = await superAdminApi.getNotifications({ page_size: 5 });
      const unread = result.data.filter((n: any) => !n.is_read).length;
      setNotifications(unread);
    } catch (error) {
      console.error('Failed to load notifications:', error);
    }
  };

  const handleLogout = async () => {
    try {
      await superAdminApi.logout();
    } catch (error) {
      console.error('Logout error:', error);
    }
    localStorage.removeItem('super_admin_token');
    navigate('/login');
  };

  const cycleTheme = () => {
    if (theme === 'light') setTheme('dark');
    else if (theme === 'dark') setTheme('system');
    else setTheme('light');
  };

  const getThemeIcon = () => {
    if (theme === 'dark') return '🌙';
    if (theme === 'light') return '☀️';
    return '💻';
  };

  return (
    <div className="flex min-h-screen bg-primary">
      {/* Sidebar */}
      <aside 
        className={`fixed left-0 top-0 h-full bg-secondary border-r border-primary transition-all duration-300 z-50 ${
          sidebarOpen ? 'w-64' : 'w-16'
        }`}
      >
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="flex items-center justify-between p-4 border-b border-primary">
            {sidebarOpen && (
              <div className="flex items-center gap-2">
                <span className="text-2xl">🐯</span>
                <span className="font-bold text-lg text-primary">Super Admin</span>
              </div>
            )}
            <button 
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 rounded hover:bg-tertiary text-secondary"
            >
              {sidebarOpen ? '◀' : '▶'}
            </button>
          </div>

          {/* Menu */}
          <nav className="flex-1 overflow-y-auto py-4">
            {menuItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-4 py-3 mx-2 rounded-lg transition-fast ${
                    isActive
                      ? 'bg-accent-primary text-white'
                      : 'text-secondary hover:bg-tertiary hover:text-primary'
                  }`
                }
              >
                <span className="text-xl">{item.icon}</span>
                {sidebarOpen && <span>{item.label}</span>}
              </NavLink>
            ))}
          </nav>

          {/* User Section */}
          <div className="p-4 border-t border-primary">
            {sidebarOpen ? (
              <div className="flex items-center justify-between">
                <span className="text-sm text-secondary">Admin</span>
                <button
                  onClick={handleLogout}
                  className="text-sm text-error hover:underline"
                >
                  Logout
                </button>
              </div>
            ) : (
              <button
                onClick={handleLogout}
                className="w-full p-2 text-center text-error"
                title="Logout"
              >
                🚪
              </button>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className={`flex-1 transition-all duration-300 ${sidebarOpen ? 'ml-64' : 'ml-16'}`}>
        {/* Header */}
        <header className="sticky top-0 z-40 bg-secondary border-b border-primary">
          <div className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center gap-4">
              <h1 className="text-xl font-semibold text-primary">
                TigerWallet Super Admin
              </h1>
            </div>

            <div className="flex items-center gap-4">
              {/* Theme Toggle */}
              <button
                onClick={cycleTheme}
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-tertiary hover:bg-border-primary transition-fast"
                title={`Current: ${theme}`}
              >
                <span>{getThemeIcon()}</span>
                <span className="text-sm text-secondary capitalize">{theme}</span>
              </button>

              {/* Notifications */}
              <button className="relative p-2 rounded-lg bg-tertiary hover:bg-border-primary transition-fast">
                <span>🔔</span>
                {notifications > 0 && (
                  <span className="absolute -top-1 -right-1 w-5 h-5 bg-error text-white text-xs rounded-full flex items-center justify-center">
                    {notifications}
                  </span>
                )}
              </button>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="p-6">
          <Outlet />
        </main>
      </div>

      {/* Theme Toggle Floating Button */}
      <button
        onClick={cycleTheme}
        className="theme-toggle"
        title={`Switch theme (current: ${theme})`}
      >
        {getThemeIcon()}
      </button>
    </div>
  );
}
