// Settings Page
import React, { useEffect, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

interface HealthState {
  status: string;
  service: string;
  licensed?: boolean;
  wl_client_id?: string;
}

export default function Settings() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const [health, setHealth] = useState<HealthState | null>(null);
  const [healthError, setHealthError] = useState('');

  useEffect(() => {
    api.health()
      .then(setHealth)
      .catch((err: unknown) => setHealthError(err instanceof Error ? err.message : 'health check failed'));
  }, []);

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <section className="settings-section">
        <h2>Backend Health</h2>
        {healthError ? (
          <div className="error">{healthError}</div>
        ) : health ? (
          <>
            <div className="setting-item">
              <label>Status</label>
              <span className={`status ${health.status === 'healthy' ? 'confirmed' : 'failed'}`}>{health.status}</span>
            </div>
            <div className="setting-item">
              <label>Service</label>
              <span>{health.service}</span>
            </div>
            <div className="setting-item">
              <label>Licensed</label>
              <span className={`status ${health.licensed ? 'confirmed' : 'failed'}`}>
                {health.licensed ? 'active' : 'inactive'}
              </span>
            </div>
            <div className="setting-item">
              <label>WL Client ID</label>
              <span className="mono">{health.wl_client_id || '—'}</span>
            </div>
          </>
        ) : (
          <p>Checking…</p>
        )}
      </section>

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
