'use client';

/**
 * White Label Admin Login Page — real authentication against
 * white_label_admin/go POST /api/v1/auth/login.
 * Backend returns { token, admin: { id, email, username, role, ... } }.
 * Token stored in localStorage as "whitelabel_admin_token" (matching api.ts).
 */
import React, { useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import whiteLabelAdminApi from '../services/api';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { theme, isDark, toggleTheme } = useTheme();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const result = await whiteLabelAdminApi.login(email, password);
      whiteLabelAdminApi.setToken(result.token);
      window.location.reload();
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className={`min-h-screen flex items-center justify-center ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}
      style={{ padding: '1rem' }}
    >
      <button
        onClick={toggleTheme}
        aria-label="Toggle theme"
        style={{
          position: 'fixed',
          top: '1rem',
          right: '1rem',
          fontSize: '1.5rem',
          background: 'none',
          border: 'none',
          cursor: 'pointer',
        }}
      >
        {isDark ? '☀️' : '🌙'}
      </button>
      <div
        className={`w-full max-w-md ${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg shadow-lg p-8`}
      >
        <div className="text-center mb-6">
          <div className="text-4xl mb-2">🏢</div>
          <h1 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            White Label Admin
          </h1>
          <p className={`mt-1 text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
            Sign in to your white-label dashboard
          </p>
        </div>

        {error && (
          <div className="mb-4 px-4 py-3 rounded bg-red-500 text-white text-sm font-medium">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin}>
          <div className="mb-4">
            <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@whitelabel.com"
              required
              className={`w-full px-4 py-3 rounded border ${
                isDark
                  ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400'
                  : 'bg-white border-gray-300 text-gray-900 placeholder-gray-400'
              }`}
            />
          </div>
          <div className="mb-6">
            <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              className={`w-full px-4 py-3 rounded border ${
                isDark
                  ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400'
                  : 'bg-white border-gray-300 text-gray-900 placeholder-gray-400'
              }`}
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 rounded bg-blue-600 hover:bg-blue-700 text-white font-semibold disabled:opacity-60 disabled:cursor-default"
          >
            {loading ? 'Signing in…' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}