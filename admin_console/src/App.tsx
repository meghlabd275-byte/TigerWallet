/**
 * Admin Console Web Application
 */
import React, { useState } from 'react';
import { ThemeProvider, useTheme } from '../context/ThemeContext';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import KYC from './pages/KYC';
import Transactions from './pages/Transactions';
import Tokens from './pages/Tokens';
import Settings from './pages/Settings';

type Page = 'dashboard' | 'users' | 'kyc' | 'transactions' | 'tokens' | 'settings';

function AppContent() {
  const { theme, toggleTheme } = useTheme();
  const [page, setPage] = useState<Page>('dashboard');
  const pages = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'users', label: 'Users' },
    { id: 'kyc', label: 'KYC' },
    { id: 'transactions', label: 'Transactions' },
    { id: 'tokens', label: 'Tokens' },
    { id: 'settings', label: 'Settings' },
  ];
  const render = () => {
    switch (page) {
      case 'dashboard': return <Dashboard />;
      case 'users': return <Users />;
      case 'kyc': return <KYC />;
      case 'transactions': return <Transactions />;
      case 'tokens': return <Tokens />;
      case 'settings': return <Settings />;
      default: return <Dashboard />;
    }
  };
  return (
    <div className={`min-h-screen ${theme === 'dark' ? 'dark bg-gray-900 text-white' : 'bg-gray-100 text-gray-900'}`}>
      <div className="flex">
        <aside className="w-64 bg-white dark:bg-gray-800 shadow-md min-h-screen">
          <div className="p-4 border-b dark:border-gray-700"><h1 className="text-xl font-bold">Admin Console</h1></div>
          <nav className="p-2">
            {pages.map(p => (<button key={p.id} onClick={() => setPage(p.id as Page)} className={`w-full text-left px-4 py-2 rounded mb-1 ${page === p.id ? 'bg-blue-600 text-white' : 'hover:bg-gray-100 dark:hover:bg-gray-700'}`}>{p.label}</button>))}
          </nav>
        </aside>
        <main className="flex-1">
          <header className="bg-white dark:bg-gray-800 shadow-sm p-4 flex justify-between items-center">
            <h2 className="text-lg font-semibold">{pages.find(p => p.id === page)?.label}</h2>
            <button onClick={toggleTheme} className="px-4 py-2 rounded bg-gray-200 dark:bg-gray-700">{theme === 'light' ? '🌙 Dark' : '☀️ Light'}</button>
          </header>
          <div className="p-6">{render()}</div>
        </main>
      </div>
    </div>
  );
}

export default function App() {
  return <ThemeProvider><AppContent /></ThemeProvider>;
}
