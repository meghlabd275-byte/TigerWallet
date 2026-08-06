import React from 'react';
import { Outlet, Link, useLocation } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';

function Layout() {
  const { theme, toggleTheme } = useTheme();
  const location = useLocation();

  const navItems = [
    { path: '/dashboard', label: 'Dashboard', icon: '📊' },
    { path: '/wallets', label: 'Wallets', icon: '💰' },
    { path: '/transactions', label: 'Transactions', icon: '📋' },
    { path: '/settings', label: 'Settings', icon: '⚙️' }
  ];

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>UserWallet</h2>
        <ul>
          {navItems.map(item => (
            <li key={item.path}>
              <Link to={item.path} className={location.pathname === item.path ? 'active' : ''}>
                <span>{item.icon}</span> {item.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <main className="main-content">
        <header className="top-bar">
          <button onClick={toggleTheme} className="theme-toggle">
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
        </header>
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

export default Layout;
