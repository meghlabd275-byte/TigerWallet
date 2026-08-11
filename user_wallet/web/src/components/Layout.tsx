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
