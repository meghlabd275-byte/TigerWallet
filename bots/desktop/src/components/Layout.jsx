import React from 'react';
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

function Layout() {
  const { theme, toggleTheme } = useTheme();
  const { token, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const navItems = [
    { path: '/dashboard', label: 'Dashboard', icon: '📊' },
    { path: '/bots', label: 'Bots', icon: '🤖' },
    { path: '/connectors', label: 'Connectors', icon: '🔌' },
    { path: '/subscriptions', label: 'Subscription', icon: '💳' },
    { path: '/fees', label: 'Fees', icon: '💰' },
    { path: '/settings', label: 'Settings', icon: '⚙️' }
  ];

  if (!token) {
    return (
      <div className={`layout ${theme}`}>
        <main className="main-content">
          <div className="page-content">
            <p>Not signed in.</p>
            <Link to="/login">Sign in</Link>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className={`layout ${theme}`}>
      <nav className="sidebar">
        <h2>TigerBots</h2>
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
          <button
            onClick={async () => { await logout(); navigate('/login'); }}
            className="theme-toggle"
          >
            Sign out
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
