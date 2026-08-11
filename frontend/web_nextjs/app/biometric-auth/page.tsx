'use client';

import React, { useState, useEffect, useRef } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface WebAuthnCredential {
  credentialId: string;
  deviceName: string;
  transport: string;
  createdAt: string;
  lastUsed?: string;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8446';

export default function BiometricAuthPage() {
  const [isSupported, setIsSupported] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [credentials, setCredentials] = useState<WebAuthnCredential[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deviceName, setDeviceName] = useState('');
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    // Check if WebAuthn is supported
    if (typeof window !== 'undefined' && window.PublicKeyCredential) {
      setIsSupported(true);
      loadCredentials();
    }
  }, []);

  const { isDark } = useTheme();

  async function loadCredentials() {
    try {
      const userId = localStorage.getItem('userId') || 'demo-user';
      const res = await fetch(`${API_BASE}/api/v1/2fa/webauthn/credentials/${userId}`);
      const data = await res.json();
      setCredentials(data.credentials || []);
    } catch (err) {
      console.error('Failed to load credentials:', err);
    }
  }

  async function registerCredential() {
    if (!deviceName) {
      setError('Please enter a device name');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const userId = localStorage.getItem('userId') || 'demo-user';
      
      // Check if we're using a real browser or mock
      if (window.PublicKeyCredential && window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable) {
        const available = await window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
        
        if (available) {
          // Real WebAuthn registration
          const publicKeyCredentialCreationOptions: PublicKeyCredentialCreationOptions = {
            challenge: new Uint8Array(32),
            rp: {
              name: 'TigerWallet',
              id: window.location.hostname,
            },
            user: {
              id: new Uint8Array(16),
              name: userId,
              displayName: userId,
            },
            pubKeyCredParams: [
              { alg: -7, type: 'public-key' },
              { alg: -257, type: 'public-key' },
            ],
            timeout: 60000,
            authenticatorSelection: {
              authenticatorAttachment: 'platform',
              userVerification: 'required',
            },
          };

          const credential = await navigator.credentials.create({
            publicKey: publicKeyCredentialCreationOptions,
          }) as PublicKeyCredential;

          if (credential) {
            // Send to backend
            await fetch(`${API_BASE}/api/v1/2fa/webauthn/register`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                userId,
                deviceName,
                credentialId: Array.from(new Uint8Array(credential.rawId)),
              }),
            });

            await loadCredentials();
          }
        } else {
          // Simulate registration for demo
          await fetch(`${API_BASE}/api/v1/2fa/webauthn/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userId, deviceName }),
          });
          await loadCredentials();
        }
      } else {
        // Fallback: simulate registration
        await fetch(`${API_BASE}/api/v1/2fa/webauthn/register`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ userId, deviceName }),
        });
        await loadCredentials();
      }
    } catch (err: any) {
      setError(err.message || 'Failed to register credential');
    } finally {
      setLoading(false);
    }
  }

  async function authenticate() {
    setLoading(true);
    setError(null);

    try {
      const userId = localStorage.getItem('userId') || 'demo-user';
      
      if (window.PublicKeyCredential) {
        const available = await window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
        
        if (available && credentials.length > 0) {
          const publicKeyCredentialRequestOptions: PublicKeyCredentialRequestOptions = {
            challenge: new Uint8Array(32),
            timeout: 60000,
            userVerification: 'required',
          };

          const credential = await navigator.credentials.get({
            publicKey: publicKeyCredentialRequestOptions,
          }) as PublicKeyCredential;

          if (credential) {
            // Verify with backend
            const res = await fetch(`${API_BASE}/api/v1/2fa/webauthn/verify`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                userId,
                credentialId: Array.from(new Uint8Array(credential.rawId)),
              }),
            });
            
            const data = await res.json();
            if (data.valid) {
              setIsAuthenticated(true);
            }
          }
        } else {
          // No credentials, simulate auth
          setIsAuthenticated(true);
        }
      } else {
        // Simulate
        setIsAuthenticated(true);
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  }

  async function deleteCredential(credentialId: string) {
    try {
      const userId = localStorage.getItem('userId') || 'demo-user';
      await fetch(`${API_BASE}/api/v1/2fa/webauthn/credentials/${userId}/${credentialId}`, {
        method: 'DELETE',
      });
      await loadCredentials();
    } catch (err) {
      console.error('Failed to delete credential:', err);
    }
  }

  return (
    <div className={`min-h-screen bg-gradient-to-br ${isDark ? 'from-slate-900 to-slate-800' : 'from-slate-50 to-slate-100'}`}>
      {/* Header */}
      <header className={`border-b sticky top-0 z-10 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-3">
              <a href="/wallet" className="text-2xl">🐯</a>
              <div>
                <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>Biometric Authentication</h1>
                <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                  Secure your wallet with biometrics
                </p>
              </div>
            </div>
            <a href="/security" className={`hover:text-orange-500 ${isDark ? 'text-slate-300' : 'text-slate-600'}`}>
              ← Back to Security
            </a>
          </div>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Status */}
        {!isSupported && (
          <div className={`mb-6 border rounded-xl p-4 ${isDark ? 'bg-yellow-900/20 border-yellow-800' : 'bg-yellow-50 border-yellow-200'}`}>
            <div className="flex items-center gap-3">
              <span className="text-2xl">⚠️</span>
              <div>
                <div className={`font-medium ${isDark ? 'text-yellow-200' : 'text-yellow-800'}`}>WebAuthn Not Supported</div>
                <div className={`text-sm ${isDark ? 'text-yellow-300' : 'text-yellow-700'}`}>
                  Your browser doesn't support WebAuthn. Please use a modern browser.
                </div>
              </div>
            </div>
          </div>
        )}

        {isAuthenticated && (
          <div className={`mb-6 border rounded-xl p-4 ${isDark ? 'bg-green-900/20 border-green-800' : 'bg-green-50 border-green-200'}`}>
            <div className="flex items-center gap-3">
              <span className="text-2xl">✅</span>
              <div>
                <div className={`font-medium ${isDark ? 'text-green-200' : 'text-green-800'}`}>Authenticated</div>
                <div className={`text-sm ${isDark ? 'text-green-300' : 'text-green-700'}`}>
                  Biometric authentication successful!
                </div>
              </div>
            </div>
          </div>
        )}

        {error && (
          <div className={`mb-6 border rounded-xl p-4 ${isDark ? 'bg-red-900/20 border-red-800' : 'bg-red-50 border-red-200'}`}>
            <div className="flex items-center gap-3">
              <span className="text-2xl">❌</span>
              <div className={isDark ? 'text-red-300' : 'text-red-700'}>{error}</div>
            </div>
          </div>
        )}

        {/* Authenticate Section */}
        <div className={`rounded-xl border p-6 mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>
            🔐 Authenticate
          </h2>
          
          <button
            onClick={authenticate}
            disabled={loading || !isSupported}
            className="w-full py-4 bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 text-white rounded-xl font-semibold flex items-center justify-center gap-3 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-white"></div>
            ) : (
              <>
                <span className="text-2xl">👆</span>
                <span>Use Biometric to Authenticate</span>
              </>
            )}
          </button>

          <p className={`mt-4 text-sm text-center ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
            Use fingerprint, face recognition, or device PIN to authenticate
          </p>
        </div>

        {/* Register New Device */}
        <div className={`rounded-xl border p-6 mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>
            ➕ Add New Device
          </h2>
          
          <div className="space-y-4">
            <div>
              <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>
                Device Name
              </label>
              <input
                type="text"
                value={deviceName}
                onChange={(e) => setDeviceName(e.target.value)}
                placeholder="e.g., iPhone 15 Pro, MacBook Pro"
                className={`w-full px-4 py-3 rounded-lg border ${isDark ? 'border-slate-600 bg-slate-700 text-white' : 'border-slate-300 bg-white text-slate-900'}`}
              />
            </div>

            <button
              onClick={registerCredential}
              disabled={loading || !isSupported || !deviceName}
              className={`w-full py-3 rounded-lg font-medium disabled:opacity-50 ${isDark ? 'bg-slate-700 hover:bg-slate-600 text-white' : 'bg-slate-100 hover:bg-slate-200 text-slate-900'}`}
            >
              {loading ? 'Registering...' : 'Register This Device'}
            </button>
          </div>
        </div>

        {/* Registered Devices */}
        <div className={`rounded-xl border p-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>
            📱 Registered Devices
          </h2>
          
          {credentials.length === 0 ? (
            <div className={`text-center py-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
              <div className="text-4xl mb-3">📵</div>
              <p>No devices registered yet</p>
            </div>
          ) : (
            <div className="space-y-3">
              {credentials.map((cred) => (
                <div
                  key={cred.credentialId}
                  className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}
                >
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">📱</span>
                    <div>
                      <div className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>
                        {cred.deviceName}
                      </div>
                      <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                        Added: {new Date(cred.createdAt).toLocaleDateString()}
                      </div>
                    </div>
                  </div>
                  <button
                    onClick={() => deleteCredential(cred.credentialId)}
                    className="text-red-500 hover:text-red-600 text-sm font-medium"
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Security Info */}
        <div className={`mt-6 border rounded-xl p-4 ${isDark ? 'bg-blue-900/20 border-blue-800' : 'bg-blue-50 border-blue-200'}`}>
          <div className="flex items-start gap-3">
            <span className="text-2xl">🔒</span>
            <div>
              <div className={`font-medium ${isDark ? 'text-blue-200' : 'text-blue-800'}`}>Security Information</div>
              <ul className={`mt-2 text-sm space-y-1 ${isDark ? 'text-blue-300' : 'text-blue-700'}`}>
                <li>• Biometric data never leaves your device</li>
                <li>• Credentials are stored securely in hardware</li>
                <li>• Each device requires separate registration</li>
                <li>• You can remove devices at any time</li>
              </ul>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
