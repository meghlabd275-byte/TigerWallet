// Layout - ProjectParty
import React from 'react';
import { Outlet, Link, useLocation } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';

export default function Layout() {
  const { theme, toggleTheme } = useTheme();
  const location = useLocation();

  const navItems = [
    { path: '/dashboard', label: 'Dashboard' },
    { path: '/coins', label: 'Coins' },
    { path: '/tokens', label: 'Tokens' },
    { path: '/favorites', label: 'Favorites' },
    { path: '/submit', label: 'Submit' },
    { path: '/listings', label: 'Listings' },
    { path: '/launchpad', label: 'Launchpad' },
    { path: '/market-making', label: 'Market Making' },
    { path: '/pricing', label: 'Pricing' },
    { path: '/analytics', label: 'Analytics' },
    { path: '/compliance', label: 'Compliance' },
    { path: '/fees', label: 'Fees' },
    { path: '/settings', label: 'Settings' }
  ];

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>ProjectParty</h2>
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
