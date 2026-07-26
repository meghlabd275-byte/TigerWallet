'use client';

import React, { useState, useEffect } from 'react';
import { PasskeyAuthenticator } from './authenticator';

/**
 * TigerWallet Passkey Login Page
 * Real WebAuthn biometric authentication
 */

export default function PasskeyLogin() {
  const [authenticator] = useState(() => new PasskeyAuthenticator());
  const [isSupported, setIsSupported] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [users, setUsers] = useState<Array<{ id: string; username: string; credentials: string[] }>>([]);
  const [showRegister, setShowRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');

  useEffect(() => {
    // Check WebAuthn support
    setIsSupported(authenticator.isSupported());
    
    // Load existing users
    setUsers(authenticator.getUsers());
  }, [authenticator]);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      const result = await authenticator.register(username, displayName);
      
      if (result.success) {
        setUsers(authenticator.getUsers());
        setShowRegister(false);
        setUsername('');
        setDisplayName('');
        alert('Passkey registered successfully!');
      } else {
        setError(result.error || 'Registration failed');
      }
    } catch (err: any) {
      setError(err.message || 'An error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogin = async (userId?: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const result = await authenticator.authenticate(userId);
      
      if (result.success) {
        // In production, send to backend for session creation
        alert(`Logged in as ${result.user?.username || 'User'}!`);
        // Redirect to wallet
        window.location.href = '/wallet';
      } else {
        setError(result.error || 'Authentication failed');
      }
    } catch (err: any) {
      setError(err.message || 'An error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  if (isSupported === null) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-orange-500"></div>
      </div>
    );
  }

  if (!isSupported) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center p-4">
        <div className="bg-white dark:bg-slate-800 rounded-2xl p-8 max-w-md w-full shadow-xl">
          <div className="text-center">
            <div className="text-6xl mb-4">⚠️</div>
            <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">Browser Not Supported</h2>
            <p className="text-slate-600 dark:text-slate-300">
              Passkey authentication requires a modern browser with WebAuthn support.
              Please use Chrome, Firefox, Safari, or Edge.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 to-amber-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-800 rounded-2xl p-8 max-w-md w-full shadow-xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="text-5xl mb-4">🐯</div>
          <h1 className="text-3xl font-bold text-slate-900 dark:text-white">TigerWallet</h1>
          <p className="text-slate-600 dark:text-slate-300 mt-2">Passkey Authentication</p>
        </div>

        {/* Error Display */}
        {error && (
          <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg p-4 mb-6">
            <p className="text-red-600 dark:text-red-400 text-sm">{error}</p>
          </div>
        )}

        {!showRegister ? (
          <>
            {/* Login Form */}
            <div className="space-y-4">
              <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">Sign in with Passkey</h2>
              
              {users.length > 0 ? (
                <div className="space-y-3">
                  {users.map(user => (
                    <button
                      key={user.id}
                      onClick={() => handleLogin(user.id)}
                      disabled={isLoading}
                      className="w-full p-4 bg-gradient-to-r from-orange-500 to-amber-500 hover:from-orange-600 hover:to-amber-600 text-white font-semibold rounded-xl transition-all disabled:opacity-50 flex items-center gap-3"
                    >
                      <span className="text-2xl">🔐</span>
                      <span>{user.username}</span>
                    </button>
                  ))}
                  
                  <button
                    onClick={() => handleLogin()}
                    disabled={isLoading}
                    className="w-full p-4 bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-200 font-semibold rounded-xl transition-all disabled:opacity-50"
                  >
                    {isLoading ? 'Authenticating...' : 'Use any passkey'}
                  </button>
                </div>
              ) : (
                <p className="text-slate-600 dark:text-slate-400 text-center py-4">
                  No passkeys registered yet. Create one to get started.
                </p>
              )}
            </div>

            {/* Register Link */}
            <div className="mt-6 text-center">
              <button
                onClick={() => setShowRegister(true)}
                className="text-orange-600 dark:text-orange-400 hover:underline font-medium"
              >
                Create new passkey
              </button>
            </div>
          </>
        ) : (
          <>
            {/* Register Form */}
            <form onSubmit={handleRegister} className="space-y-4">
              <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">Create Passkey</h2>
              
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Username
                </label>
                <input
                  type="text"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  required
                  className="w-full px-4 py-3 rounded-xl border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  placeholder="Enter username"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Display Name
                </label>
                <input
                  type="text"
                  value={displayName}
                  onChange={e => setDisplayName(e.target.value)}
                  required
                  className="w-full px-4 py-3 rounded-xl border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  placeholder="Your display name"
                />
              </div>

              <button
                type="submit"
                disabled={isLoading || !username || !displayName}
                className="w-full py-4 bg-gradient-to-r from-orange-500 to-amber-500 hover:from-orange-600 hover:to-amber-600 text-white font-bold rounded-xl transition-all disabled:opacity-50"
              >
                {isLoading ? 'Creating Passkey...' : 'Create Passkey'}
              </button>
            </form>

            {/* Login Link */}
            <div className="mt-6 text-center">
              <button
                onClick={() => setShowRegister(false)}
                className="text-orange-600 dark:text-orange-400 hover:underline font-medium"
              >
                Already have a passkey? Sign in
              </button>
            </div>
          </>
        )}

        {/* Security Note */}
        <div className="mt-8 pt-6 border-t border-slate-200 dark:border-slate-700">
          <div className="flex items-center justify-center gap-2 text-slate-500 dark:text-slate-400 text-sm">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M2.166 4.999A11.954 11.954 0 0010 1.944 11.954 11.954 0 0017.834 5c.11.65.166 1.32.166 2.001 0 5.225-3.34 9.67-8 11.317C5.34 16.67 2 12.225 2 7c0-.682.057-1.35.166-2.001zm11.541 3.708a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <span>Secured with biometric authentication</span>
          </div>
        </div>
      </div>
    </div>
  );
}
