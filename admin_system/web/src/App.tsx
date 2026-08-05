/**
 * Admin System Web Application
 */
import React, { useState } from 'react';
import { ThemeProvider, useTheme } from '../context/ThemeContext';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import Config from './pages/Config';
import AuditLogs from './pages/AuditLogs';
import Notifications from './pages/Notifications';
import FeatureFlags from './pages/FeatureFlags';
import APIKeys from './pages/APIKeys';
import Settings from './pages/Settings';

type Page = 'dashboard' | 'users' | 'config' | 'audit' | 'notifications' | 'features' | 'apikeys' | 'settings';

function AppContent() {
  const { theme, toggleTheme } = useTheme();
  const [currentPage, setCurrentPage] = useState<Page>('dashboard');

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard': return <Dashboard />;
      case 'users': return <Users />;
      case 'config': return <Config />;
      case 'audit': return <AuditLogs />;
      case 'notifications': return <Notifications />;
      case 'features': return <FeatureFlags />;
      case 'apikeys': return <APIKeys />;
      case 'settings': return <Settings />;
      default: return <Dashboard />;
    }
  };

  const navItems = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'users', label: 'Users' },
    { id: 'config', label: 'Configuration' },
    { id: 'audit', label: 'Audit Logs' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'features', label: 'Feature Flags' },
    { id: 'apikeys', label: 'API Keys' },
    { id: 'settings', label: 'Settings' },
  ];

  return (
    <div className={`min-h-screen ${theme === 'dark' ? 'dark bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <div className="flex">
        <aside className="w-64 bg-white dark:bg-gray-800 shadow-md min-h-screen">
          <div className="p-4 border-b dark:border-gray-700">
            <h1 className="text-xl font-bold">Admin System</h1>
          </div>
          <nav className="p-2">
            {navItems.map((item) => (
              <button key={item.id} onClick={() => setCurrentPage(item.id as Page)} className={`w-full text-left px-4 py-2 rounded mb-1 ${currentPage === item.id ? 'bg-blue-600 text-white' : 'hover:bg-gray-100 dark:hover:bg-gray-700'}`}>
                {item.label}
              </button>
            ))}
          </nav>
        </aside>
        <main className="flex-1">
          <header className="bg-white dark:bg-gray-800 shadow-sm p-4 flex justify-between items-center">
            <h2 className="text-lg font-semibold">{navItems.find(n => n.id === currentPage)?.label}</h2>
            <button onClick={toggleTheme} className="px-4 py-2 rounded bg-gray-200 dark:bg-gray-700">{theme === 'light' ? '🌙 Dark' : '☀️ Light'}</button>
          </header>
          <div className="p-6">{renderPage()}</div>
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
