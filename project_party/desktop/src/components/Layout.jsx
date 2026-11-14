import React from 'react';
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

function Layout() {
  const { theme, toggleTheme } = useTheme();
  const { token, isAdmin, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const navItems = [
    { path: '/market', label: 'Market', icon: '📈' },
    { path: '/tokens', label: 'Tokens', icon: '🪙' },
    { path: '/listings', label: 'Listings', icon: '🏷️' },
    { path: '/launchpad', label: 'Launchpad', icon: '🚀' },
    { path: '/market-making', label: 'Market Making', icon: '🤝' },
    { path: '/fees', label: 'Fees', icon: '💰' },
    { path: '/favorites', label: 'Favorites', icon: '⭐' }
  ];
  if (isAdmin) navItems.push({ path: '/admin', label: 'Admin', icon: '🛡️' });

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>ProjectParty</h2>
        <ul>
          {navItems.map(item => (
            <li key={item.path}>
              <Link to={item.path} className={location.pathname.startsWith(item.path) ? 'active' : ''}>
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
          {token ? (
            <button onClick={() => { logout(); navigate('/login'); }} className="theme-toggle">Sign out</button>
          ) : (
            <Link to="/login" className="theme-toggle">Sign in</Link>
          )}
        </header>
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

export default Layout;
