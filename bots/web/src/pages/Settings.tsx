// Settings Page - Bots
import React from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';

export default function Settings() {
  const { logout } = useAuth();
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="settings-page">
      <h1>Settings</h1>
      <section>
        <h2>Appearance</h2>
        <button onClick={toggleTheme}>Theme: {theme === 'light' ? 'Dark' : 'Light'}</button>
      </section>
      <section>
        <h2>Account</h2>
        <button onClick={logout}>Logout</button>
      </section>
    </div>
  );
}
