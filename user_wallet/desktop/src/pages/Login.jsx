import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';

// Stable, per-device id so guestAuth provisions the same guest account on
// repeat visits instead of creating a new one each launch.
function getDeviceId() {
  let id = localStorage.getItem('userwallet-device-id');
  if (!id) {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    id = Array.from(bytes).map((b) => b.toString(16).padStart(2, '0')).join('');
    localStorage.setItem('userwallet-device-id', id);
  }
  return id;
}

function Login() {
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [isRegister, setIsRegister] = useState(false);
  const [showEmailForm, setShowEmailForm] = useState(false);
  const { login, register, guestAuth } = useAuth();
  const navigate = useNavigate();

  const startGuest = async (action) => {
    setError('');
    setBusy(true);
    try {
      await guestAuth(getDeviceId());
      navigate(`/wallets?action=${action}`);
    } catch (err) {
      setError(err.message || 'Guest authentication failed');
      setBusy(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      if (isRegister) {
        await register(email, username || email, password);
      } else {
        await login(email, password);
      }
      navigate('/dashboard');
    } catch (err) {
      setError(err.message || 'Authentication failed');
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>UserWallet</h1>

        {!showEmailForm ? (
          <>
            <h2>Get Started</h2>
            <p className="login-subtitle">No registration required — create or import a wallet instantly.</p>
            <div className="guest-actions">
              <button
                type="button"
                className="primary-btn"
                disabled={busy}
                onClick={() => startGuest('create')}
              >
                ➕ Create Wallet
              </button>
              <button
                type="button"
                className="primary-btn"
                disabled={busy}
                onClick={() => startGuest('import')}
              >
                📥 Import Wallet
              </button>
            </div>
            {error && <div className="error">{error}</div>}
            <button
              type="button"
              className="toggle-auth"
              onClick={() => { setShowEmailForm(true); setError(''); }}
            >
              Sign in with email
            </button>
          </>
        ) : (
          <>
            <h2>{isRegister ? 'Create Account' : 'Login'}</h2>
            <form onSubmit={handleSubmit}>
              {error && <div className="error">{error}</div>}
              <input
                type="email"
                placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
              {isRegister && (
                <input
                  type="text"
                  placeholder="Username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              )}
              <input
                type="password"
                placeholder="Password (min 8 chars)"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={8}
              />
              <button type="submit" disabled={busy}>{isRegister ? 'Register' : 'Login'}</button>
            </form>
            <button
              className="toggle-auth"
              onClick={() => {
                setIsRegister(!isRegister);
                setError('');
              }}
            >
              {isRegister ? 'Already have an account? Login' : "Don't have an account? Register"}
            </button>
            <button
              className="toggle-auth"
              onClick={() => { setShowEmailForm(false); setError(''); setIsRegister(false); }}
            >
              ← Back to quick start
            </button>
          </>
        )}
      </div>
    </div>
  );
}

export default Login;
