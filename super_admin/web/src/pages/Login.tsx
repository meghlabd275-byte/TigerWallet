/**
 * TigerWallet Super Admin - Login Page
 * Secure authentication with 2FA support
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../context/ThemeContext';
import superAdminApi from '../services/api';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [twoFactorCode, setTwoFactorCode] = useState('');
  const [show2FA, setShow2FA] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { resolvedTheme } = useTheme();
  const navigate = useNavigate();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      if (show2FA && twoFactorCode) {
        const result = await superAdminApi.loginWith2FA(email, password, twoFactorCode);
        superAdminApi.setToken(result.token);
      } else {
        const result = await superAdminApi.login(email, password);
        superAdminApi.setToken(result.token);
      }
      navigate('/');
    } catch (err: any) {
      if (err.message.includes('2FA')) {
        setShow2FA(true);
        setError('Please enter your 2FA code');
      } else {
        setError(err.message || 'Login failed');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-primary p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="text-5xl mb-4">🐯</div>
          <h1 className="text-2xl font-bold text-primary">TigerWallet</h1>
          <p className="text-secondary mt-1">Super Admin Portal</p>
        </div>

        {/* Login Card */}
        <div className="card">
          <div className="card-body p-6">
            <h2 className="text-xl font-semibold text-primary mb-6 text-center">
              {show2FA ? 'Two-Factor Authentication' : 'Sign In'}
            </h2>

            {error && (
              <div className="alert alert-error mb-4">
                {error}
              </div>
            )}

            <form onSubmit={handleLogin}>
              {!show2FA ? (
                <>
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-secondary mb-2">
                      Email
                    </label>
                    <input
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="w-full px-4 py-3 rounded-lg border border-primary bg-primary text-primary"
                      placeholder="admin@tigerwallet.com"
                      required
                    />
                  </div>

                  <div className="mb-6">
                    <label className="block text-sm font-medium text-secondary mb-2">
                      Password
                    </label>
                    <input
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full px-4 py-3 rounded-lg border border-primary bg-primary text-primary"
                      placeholder="••••••••"
                      required
                    />
                  </div>
                </>
              ) : (
                <div className="mb-6">
                  <label className="block text-sm font-medium text-secondary mb-2">
                    2FA Code
                  </label>
                  <input
                    type="text"
                    value={twoFactorCode}
                    onChange={(e) => setTwoFactorCode(e.target.value)}
                    className="w-full px-4 py-3 rounded-lg border border-primary bg-primary text-primary text-center text-2xl tracking-widest"
                    placeholder="000000"
                    maxLength={6}
                    required
                  />
                  <p className="text-sm text-tertiary mt-2">
                    Enter the 6-digit code from your authenticator app
                  </p>
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="btn-primary w-full py-3"
              >
                {loading ? (
                  <span className="flex items-center justify-center gap-2">
                    <span className="loader"></span>
                    Signing in...
                  </span>
                ) : (
                  show2FA ? 'Verify' : 'Sign In'
                )}
              </button>
            </form>

            {show2FA && (
              <button
                onClick={() => {
                  setShow2FA(false);
                  setTwoFactorCode('');
                }}
                className="btn-ghost w-full mt-4"
              >
                Back to Login
              </button>
            )}
          </div>
        </div>

        {/* Footer */}
        <p className="text-center text-sm text-tertiary mt-6">
          © 2024 TigerWallet. All rights reserved.
        </p>
      </div>
    </div>
  );
}
