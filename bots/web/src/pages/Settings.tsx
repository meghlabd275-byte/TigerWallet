// Settings Page - Bots
import React, { useEffect, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

interface HealthInfo {
  status: string;
  service: string;
  licensed: boolean;
  reason: string;
  wl_client_id: string;
  product: string;
}

export default function Settings() {
  const { logout, user } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [healthError, setHealthError] = useState('');

  useEffect(() => {
    api.health().then(setHealth).catch(err => setHealthError(err.message || 'health check failed'));
  }, []);

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <section>
        <h2>Appearance</h2>
        <p className="hint">Current theme: <strong>{theme}</strong></p>
        <button onClick={toggleTheme}>Switch to {theme === 'light' ? 'Dark' : 'Light'}</button>
      </section>

      <section>
        <h2>Backend Health</h2>
        {healthError && <div className="error">{healthError}</div>}
        {health ? (
          <div className="detail-grid">
            <div><span className="detail-label">Service</span>{health.service}</div>
            <div><span className="detail-label">Status</span>
              <span className={`status ${health.status === 'healthy' ? 'running' : 'stopped'}`}>{health.status}</span>
            </div>
            <div><span className="detail-label">Licensed</span>
              <span className={`status ${health.licensed ? 'running' : 'stopped'}`}>
                {health.licensed ? 'yes' : 'no'}
              </span>
            </div>
            <div><span className="detail-label">Reason</span>{health.reason || '—'}</div>
            <div><span className="detail-label">WL Client ID</span><code>{health.wl_client_id}</code></div>
            <div><span className="detail-label">Product</span>{health.product}</div>
          </div>
        ) : !healthError ? (
          <p className="hint">Checking...</p>
        ) : null}
      </section>

      <section>
        <h2>Account</h2>
        {user && (
          <p className="hint">
            Signed in as <strong>{user.email}</strong>{user.role ? ` (${user.role})` : ''}
          </p>
        )}
        <button onClick={logout}>Logout</button>
      </section>
    </div>
  );
}
