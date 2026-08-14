// Settings Page - ProjectParty
import React from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

export default function Settings() {
  const { theme, toggleTheme } = useTheme();
  const { logout } = useAuth();

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
