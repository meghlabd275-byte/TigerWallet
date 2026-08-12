/**
 * Header - top app bar with page title and theme toggle.
 * Theme switching works on every page that renders Header, using the shared
 * ThemeContext + CSS variables so light/dark render correctly everywhere.
 */

import React from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

interface HeaderProps {
  title?: string;
}

const Header: React.FC<HeaderProps> = ({ title = 'TigerWallet' }) => {
  const { theme, toggleTheme } = useTheme();
  const { user, logout } = useAuth();

  return (
    <header
      className="fixed top-0 right-0 left-64 z-30 flex items-center justify-between px-6 h-16"
      style={{
        background: 'var(--color-bg-primary)',
        borderBottom: '1px solid var(--color-border)',
      }}
    >
      <h1
        className="text-xl font-semibold"
        style={{ color: 'var(--color-text-primary)' }}
      >
        {title}
      </h1>

      <div className="flex items-center gap-4">
        <button
          onClick={toggleTheme}
          className="flex items-center justify-center w-10 h-10 rounded-lg transition-colors hover:opacity-80"
          style={{
            background: 'var(--color-bg-tertiary)',
            color: 'var(--color-text-primary)',
          }}
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
        >
          {theme === 'dark' ? (
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="5" />
              <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" strokeLinecap="round" />
            </svg>
          ) : (
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          )}
        </button>

        {user && (
          <div className="flex items-center gap-3">
            <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              {user.email || user.username || 'Account'}
            </span>
            <button
              onClick={logout}
              className="px-3 py-1.5 text-sm rounded-lg transition-colors hover:opacity-80"
              style={{
                background: 'var(--color-bg-tertiary)',
                color: 'var(--color-text-primary)',
              }}
            >
              Sign out
            </button>
          </div>
        )}
      </div>
    </header>
  );
};

export default Header;
