'use client';

/**
 * White Label Admin Web Application
 */

import React, { useState } from 'react';
import { ThemeProvider, useTheme } from './context/ThemeContext';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import KYC from './pages/KYC';
import Transactions from './pages/Transactions';
import Withdrawals from './pages/Withdrawals';
import Tokens from './pages/Tokens';
import Fees from './pages/Fees';
import Settings from './pages/Settings';
import Admins from './pages/Admins';
import Trading from './pages/Trading';
import P2PFiat from './pages/P2PFiat';
import BotsManagement from './pages/BotsManagement';
import Listings from './pages/Listings';
import Liquidity from './pages/Liquidity';
import WalletManagement from './pages/WalletManagement';
import CustomerService from './pages/CustomerService';
import Marketing from './pages/Marketing';
import Compliance from './pages/Compliance';
import Rewards from './pages/Rewards';
import Security from './pages/Security';
import CryptoCard from './pages/CryptoCard';
import Futures from './pages/Futures';
import Options from './pages/Options';
import CopyTrading from './pages/CopyTrading';
import Convert from './pages/Convert';
import Onramp from './pages/Onramp';
import Offramp from './pages/Offramp';
import Partners from './pages/Partners';

type Page =
  | 'dashboard' | 'users' | 'kyc' | 'transactions' | 'withdrawals'
  | 'tokens' | 'fees' | 'admins' | 'settings'
  | 'trading' | 'p2p' | 'bots' | 'listings' | 'liquidity' | 'wallet'
  | 'customer-service' | 'marketing' | 'compliance' | 'rewards'
  | 'security' | 'crypto-card'
  | 'futures' | 'options' | 'copy-trading' | 'convert'
  | 'onramp' | 'offramp' | 'partners';

interface NavSection { title: string; items: { id: Page; label: string }[]; }

function AppContent() {
  const { theme, isDark, toggleTheme } = useTheme();
  const [currentPage, setCurrentPage] = useState<Page>('dashboard');

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard': return <Dashboard />;
      case 'users': return <Users />;
      case 'kyc': return <KYC />;
      case 'transactions': return <Transactions />;
      case 'withdrawals': return <Withdrawals />;
      case 'tokens': return <Tokens />;
      case 'fees': return <Fees />;
      case 'admins': return <Admins />;
      case 'settings': return <Settings />;
      case 'trading': return <Trading />;
      case 'p2p': return <P2PFiat />;
      case 'bots': return <BotsManagement />;
      case 'listings': return <Listings />;
      case 'liquidity': return <Liquidity />;
      case 'wallet': return <WalletManagement />;
      case 'customer-service': return <CustomerService />;
      case 'marketing': return <Marketing />;
      case 'compliance': return <Compliance />;
      case 'rewards': return <Rewards />;
      case 'security': return <Security />;
      case 'crypto-card': return <CryptoCard />;
      case 'futures': return <Futures />;
      case 'options': return <Options />;
      case 'copy-trading': return <CopyTrading />;
      case 'convert': return <Convert />;
      case 'onramp': return <Onramp />;
      case 'offramp': return <Offramp />;
      case 'partners': return <Partners />;
      default: return <Dashboard />;
    }
  };

  const navSections: NavSection[] = [
    {
      title: 'System',
      items: [
        { id: 'dashboard', label: 'Dashboard' },
        { id: 'admins', label: 'Admins & Roles' },
        { id: 'settings', label: 'Settings' },
      ],
    },
    {
      title: 'Core',
      items: [
        { id: 'users', label: 'Users' },
        { id: 'transactions', label: 'Transactions' },
        { id: 'withdrawals', label: 'Withdrawals' },
        { id: 'tokens', label: 'Tokens' },
        { id: 'fees', label: 'Fees' },
      ],
    },
    {
      title: 'Products',
      items: [
        { id: 'trading', label: 'Trading' },
        { id: 'futures', label: 'Futures' },
        { id: 'options', label: 'Options' },
        { id: 'copy-trading', label: 'Copy Trading' },
        { id: 'convert', label: 'Convert' },
        { id: 'onramp', label: 'Onramp' },
        { id: 'offramp', label: 'Offramp' },
        { id: 'p2p', label: 'P2P & Fiat' },
        { id: 'partners', label: 'Partners' },
        { id: 'bots', label: 'Bots Management' },
        { id: 'listings', label: 'Listings' },
        { id: 'liquidity', label: 'Liquidity' },
        { id: 'wallet', label: 'Wallet Management' },
      ],
    },
    {
      title: 'Operations',
      items: [
        { id: 'customer-service', label: 'Customer Service' },
        { id: 'marketing', label: 'Marketing' },
        { id: 'kyc', label: 'KYC' },
        { id: 'crypto-card', label: 'Crypto Card' },
        { id: 'rewards', label: 'Rewards' },
        { id: 'security', label: 'Security' },
        { id: 'compliance', label: 'Compliance' },
      ],
    },
  ];

  const sidebarBg = isDark ? 'bg-gray-800' : 'bg-white';
  const headerBg = isDark ? 'bg-gray-800' : 'bg-white';
  const pageBg = isDark ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-900';
  const sectionLabel = isDark ? 'text-gray-500' : 'text-gray-400';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const allItems = navSections.flatMap((s) => s.items);

  return (
    <div className={`min-h-screen ${pageBg}`}>
      <div className="flex">
        <aside className={`w-64 ${sidebarBg} shadow-md min-h-screen`}>
          <div className={`p-4 border-b ${border}`}>
            <h1 className="text-xl font-bold">White Label Admin</h1>
          </div>
          <nav className="p-2 overflow-y-auto" style={{ maxHeight: 'calc(100vh - 64px)' }}>
            {navSections.map((section) => (
              <div key={section.title} className="mb-3">
                <p className={`px-4 pt-2 pb-1 text-xs font-semibold uppercase tracking-wide ${sectionLabel}`}>{section.title}</p>
                {section.items.map((item) => (
                  <button key={item.id} onClick={() => setCurrentPage(item.id)}
                    className={`w-full text-left px-4 py-2 rounded mb-1 ${currentPage === item.id ? 'bg-blue-600 text-white' : (isDark ? 'hover:bg-gray-700 text-gray-200' : 'hover:bg-gray-100 text-gray-800')}`}>
                    {item.label}
                  </button>
                ))}
              </div>
            ))}
          </nav>
        </aside>
        <main className="flex-1">
          <header className={`${headerBg} shadow-sm p-4 flex justify-between items-center`}>
            <h2 className="text-lg font-semibold">{allItems.find((n) => n.id === currentPage)?.label || 'Dashboard'}</h2>
            <button onClick={toggleTheme} className={`px-4 py-2 rounded ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}>
              {theme === 'light' ? 'Dark mode' : 'Light mode'}
            </button>
          </header>
          <div>{renderPage()}</div>
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
