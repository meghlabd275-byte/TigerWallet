// Get Started Page — UserWallet requires NO registration.
//
// On first open the app shows two primary actions: Create Wallet or Import
// Wallet. Selecting either provisions an anonymous guest account via
// /auth/guest (no email/password needed) and routes to the wallet-creation or
// wallet-import flow. A stored wallet unlocks straight into the app.
//
// Email/password login + register remain available behind a "Sign in with
// email" toggle as an OPTIONAL account-recovery path.
import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { useNavigate, Link } from 'react-router-dom';

// Stable per-browser device id (generated once, persisted in localStorage).
// Used as the guest-account device_id so re-installs reuse the same guest.
function getDeviceId(): string {
  const KEY = 'userwallet-device-id';
  let id = localStorage.getItem(KEY);
  if (!id) {
    const buf = new Uint8Array(16);
    crypto.getRandomValues(buf);
    id = Array.from(buf).map((b) => b.toString(16).padStart(2, '0')).join('');
    localStorage.setItem(KEY, id);
  }
  return id;
}

export default function Login() {
  const [mode, setMode] = useState<'start' | 'email'>('start');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const { login, guestAuth } = useAuth();
  const navigate = useNavigate();
  const { isDark, toggleTheme } = useTheme();

  const startCreate = async () => {
    setError('');
    setBusy(true);
    try {
      await guestAuth(getDeviceId());
      navigate('/wallets?action=create');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not start. Backend unreachable.');
    } finally {
      setBusy(false);
    }
  };

  const startImport = async () => {
    setError('');
    setBusy(true);
    try {
      await guestAuth(getDeviceId());
      navigate('/wallets?action=import');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not start. Backend unreachable.');
    } finally {
      setBusy(false);
    }
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await login(email, password);
      navigate('/dashboard');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h1>TigerWallet</h1>
          <button type="button" onClick={toggleTheme} className="theme-toggle" aria-label="Toggle theme">
            {isDark ? '☀️' : '🌙'}
          </button>
        </div>
        {mode === 'start' ? (
          <>
            <h2>Your Wallet, Your Keys</h2>
            <p className="subtitle">No registration required. Create a new wallet or import an existing one to get started.</p>
            {error && <div className="error">{error}</div>}
            <button className="primary-btn" onClick={startCreate} disabled={busy}>
              {busy ? 'Starting…' : '➕ Create Wallet'}
            </button>
            <button className="secondary-btn" onClick={startImport} disabled={busy}>
              📥 Import Wallet
            </button>
            <button className="link-btn" onClick={() => { setMode('email'); setError(''); }}>
              Sign in with email
            </button>
          </>
        ) : (
          <>
            <h2>Sign In</h2>
            <form onSubmit={handleEmailLogin}>
              {error && <div className="error">{error}</div>}
              <div className="form-group">
                <label>Email</label>
                <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              <div className="form-group">
                <label>Password</label>
                <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
              </div>
              <button type="submit" className="primary-btn" disabled={busy}>
                {busy ? 'Signing in…' : 'Login'}
              </button>
            </form>
            <Link to="/register" className="toggle-auth" style={{ display: 'block', textAlign: 'center' }}>
              Don't have an account? Register
            </Link>
            <button className="link-btn" onClick={() => { setMode('start'); setError(''); }}>
              ← Back
            </button>
          </>
        )}
      </div>
    </div>
  );
}
