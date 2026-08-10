/**
 * TigerWallet - MPC Login Component
 * Multi-Party Computation social login with key sharding
 *
 * Features:
 * - Social login (Google, Apple, Discord, Twitter)
 * - Key sharding (2-of-3, 3-of-5)
 * - Biometric authentication
 * - No seed phrase required
 */

import React, { useState, useCallback } from 'react';
import { useWallet } from '../../../app/wallet';

type Provider = 'google' | 'apple' | 'discord' | 'twitter' | 'email';

// Base URL of the TigerWallet MPC service (go/mpc server.go).
const MPC_API_BASE =
  (typeof window !== 'undefined' && (window as unknown as { __MPC_API_BASE__?: string }).__MPC_API_BASE__) ||
  process.env.NEXT_PUBLIC_MPC_API_URL ||
  'http://localhost:9099';

interface MPCWallet {
  address: string;
  publicKey: string;
  keyId: string;
  shardId: string;
  threshold: number;
  totalShards: number;
}

interface MPCLoginState {
  isLoading: boolean;
  error: string | null;
  wallet: MPCWallet | null;
  isAuthenticated: boolean;
}

interface SocialLoginConfig {
  clientId: string;
  redirectUri: string;
  keyThreshold: number;
  keyTotalShards: number;
}

export function useMPCLogin(config: SocialLoginConfig) {
  useWallet();
  const [state, setState] = useState<MPCLoginState>({
    isLoading: false,
    error: null,
    wallet: null,
    isAuthenticated: false,
  });

  // Generate OAuth URL for provider
  const getOAuthUrl = useCallback((provider: Provider): string => {
    const baseUrls: Record<Provider, string> = {
      google: 'https://accounts.google.com/o/oauth2/v2/auth',
      apple: 'https://appleid.apple.com/auth/authorize',
      discord: 'https://discord.com/api/oauth2/authorize',
      twitter: 'https://twitter.com/i/oauth2/authorize',
      email: 'https://tigerwallet.io/auth/email',
    };

    const scopes: Record<Provider, string> = {
      google: 'openid email profile',
      apple: 'name email',
      discord: 'identify email',
      twitter: 'tweet.read users.read',
      email: 'openid email',
    };

    const params = new URLSearchParams({
      client_id: config.clientId,
      redirect_uri: config.redirectUri,
      response_type: 'code',
      scope: scopes[provider],
      state: generateState(),
    });

    return `${baseUrls[provider]}?${params.toString()}`;
  }, [config]);

  // Login with provider
  const login = useCallback(async (provider: Provider): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      if (!config.clientId) {
        throw new Error(
          'Social login is not configured. Set a real OAuth client_id to enable social MPC login.'
        );
      }
      // Real OIDC authorization-code redirect. The IdP returns to redirectUri
      // with a `code`; a backend callback exchanges it for an id_token, after
      // which createMPCWallet binds the wallet to that identity.
      const oauthUrl = getOAuthUrl(provider);
      sessionStorage.setItem('mpc_login_provider', provider);
      sessionStorage.setItem('mpc_login_threshold', String(config.keyThreshold));
      sessionStorage.setItem('mpc_login_total_shards', String(config.keyTotalShards));
      window.location.href = oauthUrl;
      // The browser navigates away; state stays loading until the callback resolves.
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message || 'Login failed',
        isAuthenticated: false,
      }));
    }
  }, [getOAuthUrl, config]);

  // Login with email (passwordless)
  const loginWithEmail = useCallback(async (email: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      // Request a real magic-link from the wallet backend. The backend sends
      // an email and issues a one-time token; the user clicks the link, which
      // returns here with a token used to create the MPC wallet.
      const apiBase = process.env.NEXT_PUBLIC_WALLET_API_URL || '';
      const mlRes = await fetch(`${apiBase}/api/v1/auth/magic-link`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      });
      if (!mlRes.ok) {
        const d = await mlRes.text();
        throw new Error(`Magic link request failed (${mlRes.status}): ${d}`);
      }
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: 'Check your email for a magic sign-in link.',
      }));
      return;  // wallet is created when the user returns from the magic link.
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message || 'Login failed',
      }));
    }
  }, [config]);

  // Verify biometric
  const verifyBiometric = useCallback(async (): Promise<boolean> => {
    if (!window.PublicKeyCredential) {
      console.error('WebAuthn not supported');
      return false;
    }

    try {
      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: new Uint8Array(32),
          timeout: 60000,
          userVerification: 'preferred',
        },
      });

      return !!credential;
    } catch (error) {
      console.error('Biometric verification failed:', error);
      return false;
    }
  }, []);

  // Register biometric
  const registerBiometric = useCallback(async (): Promise<boolean> => {
    if (!window.PublicKeyCredential) {
      console.error('WebAuthn not supported');
      return false;
    }

    try {
      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: new Uint8Array(32),
          rp: {
            name: 'TigerWallet',
          },
          user: {
            id: new Uint8Array(16),
            name: 'user',
            displayName: 'User',
          },
          pubKeyCredParams: [
            { type: 'public-key', alg: -7 },
          ],
        },
      });

      // Save credential ID to backend
      console.log('Biometric registered');
      return !!credential;
    } catch (error) {
      console.error('Biometric registration failed:', error);
      return false;
    }
  }, []);

  // Logout
  const logout = useCallback((): void => {
    setState({
      isLoading: false,
      error: null,
      wallet: null,
      isAuthenticated: false,
    });
  }, []);

  return {
    ...state,
    login,
    loginWithEmail,
    verifyBiometric,
    registerBiometric,
    logout,
  };
}

// Helper functions
function generateState(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
}

async function createMPCWallet(
  idToken: string,
  threshold: number,
  totalShards: number
): Promise<MPCWallet> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (idToken) headers['Authorization'] = `Bearer ${idToken}`;

  const res = await fetch(`${MPC_API_BASE}/api/v1/mpc/create`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ threshold, totalShards }),
  });

  if (!res.ok) {
    const detail = await res.text();
    throw new Error(`MPC wallet creation failed (${res.status}): ${detail}`);
  }

  const data = await res.json();
  return {
    address: data.address,
    publicKey: data.publicKey,
    keyId: data.keyId,
    shardId: data.keyId,
    threshold: data.threshold,
    totalShards: data.totalShards,
  };
}

// MPC Login Button Component
interface MPCLoginButtonProps {
  provider: Provider;
  onLogin: (provider: Provider) => Promise<void>;
  isLoading: boolean;
}

export function MPCLoginButton({ provider, onLogin, isLoading }: MPCLoginButtonProps) {
  const providerConfig = {
    google: {
      label: 'Continue with Google',
      icon: (
        <svg className="w-5 h-5" viewBox="0 0 24 24">
          <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
          <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
          <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
          <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
        </svg>
      ),
      bg: 'bg-white',
      text: 'text-gray-700',
    },
    apple: {
      label: 'Continue with Apple',
      icon: (
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12.152 6.896c-.948 0-2.415-1.078-3.96-1.04-2.04.027-3.91 1.183-4.961 3.014-2.117 3.675-.546 9.103 1.519 12.09 1.013 1.454 2.208 3.09 3.792 3.039 1.52-.065 2.09-.987 3.935-.987 1.831 0 2.35.987 3.96.948 1.637-.026 2.676-1.48 3.676-2.948 1.156-1.688 1.636-3.325 1.662-3.415-.039-.013-3.182-1.221-3.22-4.857-.026-3.04 2.48-4.494 2.597-4.559-1.429-2.09-3.623-2.324-4.39-2.376-2-.156-3.675 1.09-4.61 1.09zM15.53 3.83c.843-1.012 1.4-2.427 1.245-3.83-1.207.052-2.662.805-3.532 1.818-.78.896-1.454 2.338-1.273 3.714 1.338.104 2.715-.688 3.559-1.701"/>
        </svg>
      ),
      bg: 'bg-black',
      text: 'text-white',
    },
    discord: {
      label: 'Continue with Discord',
      icon: (
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z"/>
        </svg>
      ),
      bg: 'bg-[#5865F2]',
      text: 'text-white',
    },
    twitter: {
      label: 'Continue with X',
      icon: (
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
          <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
        </svg>
      ),
      bg: 'bg-black',
      text: 'text-white',
    },
    email: {
      label: 'Continue with Email',
      icon: (
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
          <polyline points="22,6 12,13 2,6"/>
        </svg>
      ),
      bg: 'bg-blue-600',
      text: 'text-white',
    },
  };

  const config = providerConfig[provider];

  return (
    <button
      onClick={() => onLogin(provider)}
      disabled={isLoading}
      className={`${config.bg} ${config.text} px-4 py-3 rounded-lg font-medium flex items-center justify-center gap-3 hover:opacity-90 transition-opacity disabled:opacity-50`}
    >
      {isLoading ? (
        <div className="animate-spin rounded-full h-5 w-5 border-2 border-current border-t-transparent"></div>
      ) : (
        config.icon
      )}
      {config.label}
    </button>
  );
}

// MPC Login Modal Component
interface MPCLoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onLogin: (provider: Provider) => Promise<void>;
  onEmailLogin: (email: string) => Promise<void>;
  isLoading: boolean;
}

export function MPCLoginModal({
  isOpen,
  onClose,
  onLogin,
  onEmailLogin,
  isLoading
}: MPCLoginModalProps) {
  const [email, setEmail] = useState('');
  const [showEmail, setShowEmail] = useState(false);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose}></div>

      <div className="relative bg-white rounded-2xl shadow-2xl max-w-md w-full mx-4 p-8">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-gray-400 hover:text-gray-600"
        >
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <div className="text-center mb-8">
          <h2 className="text-2xl font-bold text-gray-900">
            Welcome to TigerWallet
          </h2>
          <p className="text-gray-600 mt-2">
            Sign in with your favorite provider - no seed phrase required
          </p>
        </div>

        {!showEmail ? (
          <div className="space-y-4">
            <MPCLoginButton
              provider="google"
              onLogin={onLogin}
              isLoading={isLoading}
            />
            <MPCLoginButton
              provider="apple"
              onLogin={onLogin}
              isLoading={isLoading}
            />
            <MPCLoginButton
              provider="discord"
              onLogin={onLogin}
              isLoading={isLoading}
            />
            <MPCLoginButton
              provider="twitter"
              onLogin={onLogin}
              isLoading={isLoading}
            />

            <div className="relative my-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-300"></div>
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-white text-gray-500">or</span>
              </div>
            </div>

            <button
              onClick={() => setShowEmail(true)}
              className="w-full bg-gray-100 text-gray-700 px-4 py-3 rounded-lg font-medium flex items-center justify-center gap-3 hover:bg-gray-200 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              Sign in with Email
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Email Address
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <button
              onClick={() => onEmailLogin(email)}
              disabled={isLoading || !email}
              className="w-full bg-blue-600 text-white px-4 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {isLoading ? 'Sending...' : 'Send Magic Link'}
            </button>

            <button
              onClick={() => setShowEmail(false)}
              className="w-full text-gray-600 py-2 hover:text-gray-800"
            >
              Back to providers
            </button>
          </div>
        )}

        <p className="text-xs text-gray-500 text-center mt-6">
          By continuing, you agree to TigerWallet's Terms of Service and Privacy Policy
        </p>
      </div>
    </div>
  );
}

export default MPCLoginButton;
