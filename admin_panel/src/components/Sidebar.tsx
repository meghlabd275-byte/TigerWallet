// Sidebar Component - Complete Navigation
// Light/dark theme support

import React from 'react';
import { NavLink } from 'react-router-dom';
import { useTheme } from '../stores/ThemeStore';
import './Sidebar.css';

const Sidebar: React.FC = () => {
  const { theme } = useTheme();

  const menuItems = [
    { path: '/', icon: '📊', label: 'Dashboard' },
    { path: '/send', icon: '📤', label: 'Send' },
    { path: '/users', icon: '👥', label: 'Users' },
    { path: '/wallets', icon: '💼', label: 'Wallets' },
    { path: '/blockchain', icon: '⛓️', label: 'Blockchain' },
    { path: '/pairs', icon: '🔄', label: 'Trading Pairs' },
    { path: '/trading', icon: '📈', label: 'Trading Mgmt' },
    { path: '/margin', icon: '⚡', label: 'Margin Trading' },
    { path: '/liquidity', icon: '💧', label: 'Liquidity' },
    { path: '/fees', icon: '💰', label: 'Fees' },
    { path: '/p2p', icon: '🤝', label: 'P2P Trading' },
    { path: '/p2p-merchants', icon: '🏪', label: 'P2P Merchants' },
    { path: '/fiat', icon: '🏦', label: 'Fiat On-Ramp' },
    { path: '/card', icon: '💳', label: 'Crypto Card' },
    { path: '/transactions', icon: '📝', label: 'Transactions' },
    { path: '/whitelabel', icon: '🏢', label: 'White Label' },
    { path: '/kyc', icon: '✅', label: 'KYC' },
    { path: '/analytics', icon: '📊', label: 'Analytics' },
    { path: '/settings', icon: '⚙️', label: 'Settings' },
  ];

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="logo">
          <span className="logo-icon">🐯</span>
          <span className="logo-text">TigerWallet</span>
        </div>
        <span className="admin-badge">ADMIN</span>
      </div>

      <nav className="sidebar-nav">
        <ul className="nav-list">
          {menuItems.map((item) => (
            <li key={item.path}>
              <NavLink
                to={item.path}
                className={({ isActive }) => 
                  `nav-item ${isActive ? 'active' : ''}`
                }
              >
                <span className="nav-icon">{item.icon}</span>
                <span className="nav-label">{item.label}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      <div className="sidebar-footer">
        <div className="version-info">
          <span>Version 1.0.0</span>
          <span>Build 2026.07.28</span>
        </div>
      </div>
    </aside>
  );
};

export default Sidebar;
