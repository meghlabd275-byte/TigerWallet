// Layout Component
import React from 'react';
import { Outlet, Link, useLocation } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';

export default function Layout() {
  const { theme, toggleTheme } = useTheme();
  const location = useLocation();

  const navItems = [
    { path: '/dashboard', label: 'Dashboard' },
    { path: '/wallets', label: 'Wallets' },
    { path: '/send', label: 'Send' },
    { path: '/receive', label: 'Receive' },
    { path: '/swap', label: 'Swap' },
    { path: '/staking', label: 'Staking' },
    { path: '/kyc', label: 'KYC' },
    { path: '/nfts', label: 'NFTs' },
    { path: '/bridge', label: 'Bridge' },
    { path: '/defi', label: 'DeFi' },
    { path: '/address-book', label: 'Address Book' },
    { path: '/devices', label: 'Devices' },
    { path: '/approvals', label: 'Approvals' },
    { path: '/keystore', label: 'Keystore' },
    { path: '/trading', label: 'Trading' },
    { path: '/launchpool', label: 'Launchpool' },
    { path: '/token-sales', label: 'Token Sales' },
    { path: '/p2p', label: 'P2P' },
    { path: '/cards', label: 'Cards' },
    { path: '/price-alerts', label: 'Price Alerts' },
    { path: '/dapps', label: 'dApps' },
    { path: '/dao', label: 'DAO' },
    { path: '/ramp', label: 'Fiat Ramp' },
    { path: '/copy-trading', label: 'Copy Trading' },
    { path: '/prediction', label: 'Prediction' },
    { path: '/ens', label: 'ENS' },
    { path: '/security', label: 'Security' },
    { path: '/terminal', label: 'Terminal' },
    { path: '/multisig', label: 'Multisig' },
    { path: '/non-evm', label: 'Non-EVM' },
    { path: '/transactions', label: 'Transactions' },
    { path: '/settings', label: 'Settings' }
  ];

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>UserWallet</h2>
        <ul>
          {navItems.map(item => (
            <li key={item.path}>
              <Link to={item.path} className={location.pathname === item.path ? 'active' : ''}>
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <main className="main-content">
        <header className="top-bar">
          <button onClick={toggleTheme} className="theme-toggle">
            {theme === 'light' ? '🌙 Dark' : '☀️ Light'}
          </button>
        </header>
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
