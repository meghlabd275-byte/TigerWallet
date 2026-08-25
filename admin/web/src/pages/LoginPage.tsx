// Admin Login Page — real authentication against admin/go :9093 POST /api/v1/auth/login.
// Backend returns {token, refreshToken, expiresAt, admin (AdminUser)}.
// Token stored in localStorage as "admin_token" (matching api.ts).
import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import adminApi from '../services/api';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { resolvedTheme, toggleTheme } = useTheme();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const result = await adminApi.login(email, password);
      adminApi.setToken(result.token);
      // Force a reload so App re-evaluates the token gate.
      window.location.reload();
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className="login-page"
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'var(--bg-secondary)',
        padding: '1rem',
      }}
    >
      <button
        onClick={toggleTheme}
        className="theme-toggle"
        title={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`}
        aria-label="Toggle theme"
        style={{
          position: 'fixed',
          top: '1rem',
          right: '1rem',
          background: 'none',
          border: 'none',
          fontSize: '1.5rem',
          cursor: 'pointer',
          color: 'var(--text-secondary)',
        }}
      >
        {resolvedTheme === 'dark' ? '☀️' : '🌙'}
      </button>
      <div
        style={{
          width: '100%',
          maxWidth: '420px',
          backgroundColor: 'var(--bg-card)',
          borderRadius: '0.75rem',
          boxShadow: 'var(--shadow-lg)',
          border: '1px solid var(--border-primary)',
          padding: '2.5rem 2rem',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: '2rem' }}>
          <div style={{ fontSize: '3rem', marginBottom: '0.5rem' }}>🔧</div>
          <h1 style={{ fontSize: '1.5rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
            TigerWallet Admin
          </h1>
          <p style={{ color: 'var(--text-tertiary)', marginTop: '0.25rem', fontSize: '0.875rem' }}>
            Sign in to manage the platform
          </p>
        </div>

        {error && (
          <div
            style={{
              backgroundColor: 'var(--color-error)',
              color: '#fff',
              padding: '0.75rem 1rem',
              borderRadius: '0.5rem',
              marginBottom: '1rem',
              fontSize: '0.875rem',
              fontWeight: 500,
            }}
          >
            {error}
          </div>
        )}

        <form onSubmit={handleLogin}>
          <div style={{ marginBottom: '1rem' }}>
            <label
              style={{
                display: 'block',
                fontSize: '0.875rem',
                fontWeight: 500,
                color: 'var(--text-secondary)',
                marginBottom: '0.375rem',
              }}
            >
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@tigerwallet.com"
              required
              style={{
                width: '100%',
                padding: '0.75rem 1rem',
                borderRadius: '0.5rem',
                border: '1px solid var(--border-primary)',
                backgroundColor: 'var(--input-bg)',
                color: 'var(--text-primary)',
                fontSize: '0.9375rem',
                outline: 'none',
                boxSizing: 'border-box',
              }}
              onFocus={(e) => (e.target.style.borderColor = 'var(--color-primary)')}
              onBlur={(e) => (e.target.style.borderColor = 'var(--border-primary)')}
            />
          </div>
          <div style={{ marginBottom: '1.5rem' }}>
            <label
              style={{
                display: 'block',
                fontSize: '0.875rem',
                fontWeight: 500,
                color: 'var(--text-secondary)',
                marginBottom: '0.375rem',
              }}
            >
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              style={{
                width: '100%',
                padding: '0.75rem 1rem',
                borderRadius: '0.5rem',
                border: '1px solid var(--border-primary)',
                backgroundColor: 'var(--input-bg)',
                color: 'var(--text-primary)',
                fontSize: '0.9375rem',
                outline: 'none',
                boxSizing: 'border-box',
              }}
              onFocus={(e) => (e.target.style.borderColor = 'var(--color-primary)')}
              onBlur={(e) => (e.target.style.borderColor = 'var(--border-primary)')}
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            style={{
              width: '100%',
              padding: '0.75rem',
              borderRadius: '0.5rem',
              border: 'none',
              backgroundColor: loading ? 'var(--color-primary-dark)' : 'var(--color-primary)',
              color: '#fff',
              fontWeight: 600,
              fontSize: '1rem',
              cursor: loading ? 'default' : 'pointer',
              transition: 'background-color 0.2s',
            }}
          >
            {loading ? 'Signing in…' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}