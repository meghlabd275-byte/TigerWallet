'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider';

// Base64url helpers for WebAuthn credential encoding/decoding.
function bufferToBase64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export default function BiometricAuth() {
  const { isDark } = useTheme();
  const [biometricType, setBiometricType] = useState<string | null>(null);
  const [webauthnSupported, setWebauthnSupported] = useState(false);
  const [credentialId, setCredentialId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    // Detect real WebAuthn availability (no random biometric type guessing).
    const supported = typeof window !== 'undefined' &&
      'PublicKeyCredential' in window &&
      typeof navigator.credentials !== 'undefined';
    setWebauthnSupported(supported);
    // The platform label is inferred from the user agent, not random.
    const ua = navigator.userAgent || '';
    let label = 'Platform authenticator';
    if (/iPhone|iPad|iPod/i.test(ua)) label = 'Face ID / Touch ID';
    else if (/Android/i.test(ua)) label = 'Fingerprint';
    else if (/Mac/i.test(ua)) label = 'Touch ID';
    else if (/Windows/i.test(ua)) label = 'Windows Hello';
    setBiometricType(label);

    // Detect whether a platform (biometric) authenticator is available.
    if (supported && 'isUserVerifyingPlatformAuthenticatorAvailable' in window.PublicKeyCredential) {
      window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
        .then((avail) => { if (!avail) setError('No biometric authenticator detected on this device.'); })
        .catch(() => {});
    }
  }, []);

  const handleEnable = async () => {
    setError('');
    setLoading(true);
    setMessage('');
    try {
      // Real WebAuthn registration (navigator.credentials.create).
      const challenge = new Uint8Array(32);
      crypto.getRandomValues(challenge);
      const userId = new Uint8Array(16);
      crypto.getRandomValues(userId);
      const cred = await navigator.credentials.create({
        publicKey: {
          challenge,
          rp: { name: 'TigerWallet' },
          user: {
            id: userId,
            name: 'tigerwallet-user',
            displayName: 'TigerWallet User',
          },
          pubKeyCredParams: [
            { type: 'public-key', alg: -7 },   // ES256
            { type: 'public-key', alg: -257 }, // RS256
          ],
          authenticatorSelection: {
            authenticatorAttachment: 'platform',
            userVerification: 'required',
          },
          timeout: 60000,
        },
      }) as PublicKeyCredential | null;
      if (cred && cred.rawId) {
        const id = bufferToBase64url(cred.rawId);
        setCredentialId(id);
        setMessage(`${biometricType} enabled.`);
      } else {
        setError('Registration was cancelled or failed.');
      }
    } catch (e: any) {
      setError(e?.message || 'Biometric enrollment failed.');
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async () => {
    // WebAuthn has no standard "delete" API; we clear the local credential
    // reference. The browser authenticator management UI is where the user
    // fully removes a credential.
    setCredentialId(null);
    setMessage('Disabled');
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-slate-900'}`}>
      <header className={`${isDark ? 'bg-slate-800' : 'bg-white'} border-b p-4`}><div className="flex items-center gap-4"><a href="/wallet" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Biometric Security</h1></div></header>
      <div className="max-w-md mx-auto p-8">
        <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6`}>
          <div className="text-center mb-6"><div className="text-6xl mb-4">🔐</div><h2 className="text-xl font-semibold">Biometric Authentication</h2><p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} mt-2`}>Use {biometricType || 'biometric'} to secure your wallet via WebAuthn.</p></div>
          {!webauthnSupported && <div className="bg-amber-100 text-amber-700 p-3 rounded-lg mb-4 text-center text-sm">WebAuthn is not supported in this browser.</div>}
          {error && <div className="bg-red-100 text-red-600 p-3 rounded-lg mb-4 text-center text-sm">{error}</div>}
          {message && <div className="bg-green-100 text-green-600 p-3 rounded-lg mb-4 text-center text-sm">{message}</div>}
          {credentialId ? (
            <button onClick={handleDisable} disabled={loading} className="w-full bg-red-500 text-white py-3 rounded-lg">{loading ? '...' : 'Disable'}</button>
          ) : (
            <button onClick={handleEnable} disabled={loading || !webauthnSupported} className="w-full bg-orange-500 text-white py-3 rounded-lg disabled:opacity-50">{loading ? 'Enrolling...' : 'Enable'}</button>
          )}
        </div>
      </div>
    </div>
  );
}
