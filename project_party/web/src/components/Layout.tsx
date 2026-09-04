// Layout — WL-ProjectParty
import React from 'react';
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

export default function Layout() {
  const { theme, toggleTheme } = useTheme();
  const { email, isAdmin, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const navItems = [
    { path: '/dashboard', label: 'Dashboard' },
    { path: '/tokens', label: 'Tokens' },
    { path: '/submit', label: 'Submit Token' },
    { path: '/listings', label: 'Listings' },
    { path: '/launchpad', label: 'Launchpad' },
    { path: '/market-making', label: 'Market Making' },
    { path: '/fees', label: 'Fees' },
    ...(isAdmin ? [{ path: '/admin', label: 'Admin' }] : []),
    { path: '/favorites', label: 'Favorites' },
    { path: '/settings', label: 'Settings' }
  ];

  const onLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>WL ProjectParty</h2>
        <ul>
          {navItems.map(item => (
            <li key={item.path}>
              <Link to={item.path} className={location.pathname === item.path ? 'active' : ''}>
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
        {email && <div className="muted" style={{ marginTop: '1rem', fontSize: '0.8rem', wordBreak: 'break-word' }}>{email}</div>}
      </nav>
      <main className="main-content">
        <header className="top-bar">
          <button onClick={toggleTheme} className="theme-toggle" title="Toggle theme">
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
          <button className="secondary" onClick={onLogout} title="Logout">Logout</button>
        </header>
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
