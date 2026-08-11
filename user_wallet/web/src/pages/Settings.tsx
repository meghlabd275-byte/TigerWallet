// Settings Page
import React from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';

export default function Settings() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <section className="settings-section">
        <h2>Account</h2>
        <div className="setting-item">
          <label>Email</label>
          <span>{user?.email}</span>
        </div>
        <div className="setting-item">
          <label>Username</label>
          <span>{user?.username}</span>
        </div>
      </section>

      <section className="settings-section">
        <h2>Appearance</h2>
        <div className="setting-item">
          <label>Theme</label>
          <button onClick={toggleTheme} className="theme-btn">
            Current: {theme === 'light' ? 'Light' : 'Dark'}
          </button>
        </div>
      </section>

      <section className="settings-section">
        <h2>Security</h2>
        <button className="logout-btn" onClick={logout}>Logout</button>
      </section>
    </div>
  );
}
