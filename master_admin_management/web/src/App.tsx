/**
 * Master Admin Web Application
 * Main entry point with routing and theme
 */

import React, { useState } from 'react';
import { ThemeProvider, useTheme } from '../context/ThemeContext';
import Dashboard from './pages/Dashboard';
import WhiteLabels from './pages/WhiteLabels';
import MasterAdmins from './pages/MasterAdmins';
import Users from './pages/Users';
import Transactions from './pages/Transactions';
import Analytics from './pages/Analytics';
import System from './pages/System';
import Settings from './pages/Settings';

type Page = 'dashboard' | 'whitelabels' | 'masters' | 'users' | 'transactions' | 'analytics' | 'system' | 'settings';

function AppContent() {
  const { theme, toggleTheme } = useTheme();
  const [currentPage, setCurrentPage] = useState<Page>('dashboard');

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard': return <Dashboard />;
      case 'whitelabels': return <WhiteLabels />;
      case 'masters': return <MasterAdmins />;
      case 'users': return <Users />;
      case 'transactions': return <Transactions />;
      case 'analytics': return <Analytics />;
      case 'system': return <System />;
      case 'settings': return <Settings />;
      default: return <Dashboard />;
    }
  };

  const navItems = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'whitelabels', label: 'White Labels' },
    { id: 'masters', label: 'Master Admins' },
    { id: 'users', label: 'Users' },
    { id: 'transactions', label: 'Transactions' },
    { id: 'analytics', label: 'Analytics' },
    { id: 'system', label: 'System' },
    { id: 'settings', label: 'Settings' },
  ];

  return (
    <div className={`min-h-screen ${theme === 'dark' ? 'dark bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <div className="flex">
        {/* Sidebar */}
        <aside className="w-64 bg-white dark:bg-gray-800 shadow-md min-h-screen">
          <div className="p-4 border-b dark:border-gray-700">
            <h1 className="text-xl font-bold">Master Admin</h1>
          </div>
          <nav className="p-2">
            {navItems.map((item) => (
              <button
                key={item.id}
                onClick={() => setCurrentPage(item.id as Page)}
                className={`w-full text-left px-4 py-2 rounded mb-1 ${
                  currentPage === item.id
                    ? 'bg-blue-600 text-white'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                }`}
              >
                {item.label}
              </button>
            ))}
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1">
          {/* Header */}
          <header className="bg-white dark:bg-gray-800 shadow-sm p-4 flex justify-between items-center">
            <h2 className="text-lg font-semibold">{navItems.find(n => n.id === currentPage)?.label}</h2>
            <button
              onClick={toggleTheme}
              className="px-4 py-2 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600"
            >
              {theme === 'light' ? '🌙 Dark' : '☀️ Light'}
            </button>
          </header>

          {/* Page Content */}
          <div className="p-6">
            {renderPage()}
          </div>
        </main>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  );
}
