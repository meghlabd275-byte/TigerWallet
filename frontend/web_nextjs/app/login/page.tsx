'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ThemeToggle } from '../components/ThemeToggle';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

interface AdminUser {
  id: string;
  email: string;
  username: string;
  role: 'super_admin' | 'admin';
  verified: boolean;
  twoFactorEnabled: boolean;
  createdAt: number;
  lastLogin: number;
  status: 'active' | 'suspended' | 'pending';
  permissions: string[];
}

interface LoginCredentials {
  email: string;
  password: string;
  code2FA?: string;
}

// A logged-in user (admin or standard). AdminUser is the canonical shape.
type User = AdminUser;

interface RegisterData {
  email: string;
  username: string;
  password: string;
  confirmPassword: string;
  whiteLabelId?: string;
}

// ============================================================================
// Security Constants
// ============================================================================

const SECURITY_CONFIG = {
  MIN_PASSWORD_LENGTH: 12,
  MAX_LOGIN_ATTEMPTS: 5,
  LOCKOUT_DURATION: 15 * 60 * 1000, // 15 minutes
  SESSION_DURATION: 24 * 60 * 60 * 1000, // 24 hours
  REFRESH_TOKEN_DURATION: 7 * 24 * 60 * 60 * 1000, // 7 days
  PASSWORD_REQUIREMENTS: {
    uppercase: true,
    lowercase: true,
    numbers: true,
    special: true,
    minLength: 12,
  },
};

// ============================================================================
// Validation Functions
// ============================================================================

const validateEmail = (email: string): boolean => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
};

const validatePassword = (password: string): { valid: boolean; errors: string[] } => {
  const errors: string[] = [];
  const config = SECURITY_CONFIG.PASSWORD_REQUIREMENTS;

  if (password.length < config.minLength) {
    errors.push(`Password must be at least ${config.minLength} characters`);
  }
  if (config.uppercase && !/[A-Z]/.test(password)) {
    errors.push('Password must contain at least one uppercase letter');
  }
  if (config.lowercase && !/[a-z]/.test(password)) {
    errors.push('Password must contain at least one lowercase letter');
  }
  if (config.numbers && !/[0-9]/.test(password)) {
    errors.push('Password must contain at least one number');
  }
  if (config.special && !/[!@#$%^&*(),.?":{}|<>]/.test(password)) {
    errors.push('Password must contain at least one special character');
  }

  return { valid: errors.length === 0, errors };
};

const validateUsername = (username: string): boolean => {
  // Username must be 3-20 characters, alphanumeric and underscores only
  const usernameRegex = /^[a-zA-Z0-9_]{3,20}$/;
  return usernameRegex.test(username);
};

// ============================================================================
// API Functions
// ============================================================================

const api = {
  async login(credentials: LoginCredentials): Promise<{ user: User; token: string; refreshToken: string }> {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Login failed');
    }

    return response.json();
  },

  async register(data: RegisterData): Promise<{ userId: string; status: string }> {
    const response = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Registration failed');
    }

    return response.json();
  },

  async verifyEmail(code: string): Promise<{ verified: boolean }> {
    const response = await fetch('/api/v1/auth/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Verification failed');
    }

    return response.json();
  },

  async enable2FA(): Promise<{ secret: string; qrCode: string }> {
    const response = await fetch('/api/v1/auth/2fa/enable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || '2FA enable failed');
    }

    return response.json();
  },

  async verify2FA(code: string): Promise<{ enabled: boolean }> {
    const response = await fetch('/api/v1/auth/2fa/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || '2FA verification failed');
    }

    return response.json();
  },

  async refreshToken(refreshToken: string): Promise<{ token: string }> {
    const response = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Token refresh failed');
    }

    return response.json();
  },

  async logout(): Promise<void> {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
  },

  async getCurrentUser(): Promise<User> {
    const response = await fetch('/api/v1/auth/me', {
      method: 'GET',
    });

    if (!response.ok) {
      throw new Error('Not authenticated');
    }

    return response.json();
  },

  async createAdmin(data: { email: string; username: string; role: string }): Promise<{ adminId: string }> {
    const response = await fetch('/api/v1/admin/create-admin', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Admin creation failed');
    }

    return response.json();
  },
};

// ============================================================================
// Session Management
// ============================================================================

const sessionManager = {
  setToken(token: string): void {
    localStorage.setItem('token', token);
    localStorage.setItem('tokenExpiry', (Date.now() + SECURITY_CONFIG.SESSION_DURATION).toString());
  },

  getToken(): string | null {
    const token = localStorage.getItem('token');
    const expiry = localStorage.getItem('tokenExpiry');

    if (!token || !expiry || Date.now() > parseInt(expiry)) {
      this.clearSession();
      return null;
    }

    return token;
  },

  setRefreshToken(token: string): void {
    localStorage.setItem('refreshToken', token);
  },

  getRefreshToken(): string | null {
    return localStorage.getItem('refreshToken');
  },

  clearSession(): void {
    localStorage.removeItem('token');
    localStorage.removeItem('tokenExpiry');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('user');
  },

  setUser(user: User): void {
    localStorage.setItem('user', JSON.stringify(user));
  },

  getUser(): User | null {
    const userStr = localStorage.getItem('user');
    if (!userStr) return null;
    try {
      return JSON.parse(userStr);
    } catch {
      return null;
    }
  },
};

// ============================================================================
// Main Component
// ============================================================================

export default function LoginPage() {
  const router = useRouter();
  const { theme } = useTheme();

  // State
  const [mode, setMode] = useState<'login' | 'register' | 'verify' | '2fa'>('login');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Form state
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [code2FA, setCode2FA] = useState('');
  const [verificationCode, setVerificationCode] = useState('');

  // Security state
  const [loginAttempts, setLoginAttempts] = useState(0);
  const [lockedUntil, setLockedUntil] = useState<number | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [passwordErrors, setPasswordErrors] = useState<string[]>([]);

  // 2FA state
  const [twoFactorSecret, setTwoFactorSecret] = useState('');
  const [twoFactorQR, setTwoFactorQR] = useState('');

  // ============================================================================
  // Effects
  // ============================================================================

  useEffect(() => {
    // Check if already logged in
    const token = sessionManager.getToken();
    if (token) {
      router.push('/wallet');
    }
  }, [router]);

  useEffect(() => {
    // Check lockout
    if (lockedUntil && Date.now() < lockedUntil) {
      const remaining = Math.ceil((lockedUntil - Date.now()) / 1000);
      setError(`Too many failed attempts. Try again in ${remaining} seconds`);

      const interval = setInterval(() => {
        if (Date.now() >= lockedUntil) {
          setLockedUntil(null);
          setLoginAttempts(0);
          setError(null);
          clearInterval(interval);
        }
      }, 1000);

      return () => clearInterval(interval);
    }
  }, [lockedUntil]);

  // ============================================================================
  // Handlers
  // ============================================================================

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    // Validate inputs
    if (!email || !password) {
      setError('Please enter email and password');
      return;
    }

    if (!validateEmail(email)) {
      setError('Please enter a valid email address');
      return;
    }

    // Check lockout
    if (lockedUntil && Date.now() < lockedUntil) {
      return;
    }

    setLoading(true);

    try {
      const credentials: LoginCredentials = {
        email,
        password,
      };

      const response = await api.login(credentials);

      // Store session
      sessionManager.setToken(response.token);
      sessionManager.setRefreshToken(response.refreshToken);
      sessionManager.setUser(response.user);

      // Reset attempts
      setLoginAttempts(0);

      setSuccess('Login successful! Redirecting...');

      // Redirect based on role
      if (response.user.role === 'super_admin') {
        router.push('/admin_wallet');
      } else if (response.user.role === 'admin') {
        router.push('/admin_panel');
      } else {
        router.push('/wallet');
      }
    } catch (err: any) {
      const newAttempts = loginAttempts + 1;
      setLoginAttempts(newAttempts);

      if (newAttempts >= SECURITY_CONFIG.MAX_LOGIN_ATTEMPTS) {
        setLockedUntil(Date.now() + SECURITY_CONFIG.LOCKOUT_DURATION);
        setError('Too many failed attempts. Account locked for 15 minutes');
      } else {
        setError(err.message || 'Login failed. Please check your credentials');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    // Validate inputs
    if (!email || !username || !password || !confirmPassword) {
      setError('Please fill in all fields');
      return;
    }

    if (!validateEmail(email)) {
      setError('Please enter a valid email address');
      return;
    }

    if (!validateUsername(username)) {
      setError('Username must be 3-20 characters, alphanumeric and underscores only');
      return;
    }

    const passwordValidation = validatePassword(password);
    if (!passwordValidation.valid) {
      setPasswordErrors(passwordValidation.errors);
      setError(passwordValidation.errors[0]);
      return;
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);

    try {
      const data: RegisterData = {
        email,
        username,
        password,
        confirmPassword,
      };

      await api.register(data);

      setSuccess('Registration successful! Please check your email to verify.');
      setMode('verify');
    } catch (err: any) {
      setError(err.message || 'Registration failed');
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!verificationCode) {
      setError('Please enter the verification code');
      return;
    }

    setLoading(true);

    try {
      await api.verifyEmail(verificationCode);
      setSuccess('Email verified! Please login.');
      setMode('login');
    } catch (err: any) {
      setError(err.message || 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

  const handleEnable2FA = async () => {
    setLoading(true);

    try {
      const result = await api.enable2FA();
      setTwoFactorSecret(result.secret);
      setTwoFactorQR(result.qrCode);
      setMode('2fa');
    } catch (err: any) {
      setError(err.message || 'Failed to enable 2FA');
    } finally {
      setLoading(false);
    }
  };

  const handleVerify2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!code2FA) {
      setError('Please enter the 2FA code');
      return;
    }

    setLoading(true);

    try {
      await api.verify2FA(code2FA);
      setSuccess('2FA enabled successfully!');
      setMode('login');
    } catch (err: any) {
      setError(err.message || '2FA verification failed');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await api.logout();
    } finally {
      sessionManager.clearSession();
      router.push('/login');
    }
  };

  const handlePasswordChange = (value: string) => {
    setPassword(value);
    const validation = validatePassword(value);
    setPasswordErrors(validation.errors);
  };

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="min-h-screen bg-slate-950 dark:bg-slate-950">
      <div className="absolute top-0 left-0 right-0 p-4 flex justify-between items-center z-10">
        <div className="logo text-2xl font-bold text-orange-500">🐯 TigerWallet</div>
        <ThemeToggle />
      </div>

      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="w-full max-w-md">
          {/* Error/Success Messages */}
          {error && (
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-500">
              {error}
            </div>
          )}
          {success && (
            <div className="mb-4 p-4 bg-green-500/10 border border-green-500/50 rounded-lg text-green-500">
              {success}
            </div>
          )}

          {/* Login Form */}
          {mode === 'login' && (
            <div className="bg-slate-900/80 dark:bg-slate-900/80 bg-white/50 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <h1 className="text-3xl font-bold text-white dark:text-white text-slate-900 mb-2">Welcome Back</h1>
              <p className="text-slate-400 dark:text-slate-500 mb-8">Sign in to your TigerWallet account</p>

              <form onSubmit={handleLogin}>
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Email Address
                  </label>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500"
                    placeholder="admin@tigerwallet.com"
                    required
                  />
                </div>

                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Password
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500 pr-12"
                      placeholder="••••••••••••"
                      required
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400"
                    >
                      {showPassword ? '🙈' : '👁️'}
                    </button>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={loading || (lockedUntil !== null && Date.now() < lockedUntil)}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-600 rounded-lg text-white font-semibold transition-colors"
                >
                  {loading ? 'Signing in...' : 'Sign In'}
                </button>
              </form>

              <div className="mt-6 text-center">
                <p className="text-slate-400 dark:text-slate-500 text-slate-600">
                  Don't have an account?{' '}
                  <button
                    onClick={() => {
                      setMode('register');
                      setError(null);
                    }}
                    className="text-orange-500 hover:underline"
                  >
                    Register
                  </button>
                </p>
              </div>

              <div className="mt-4 pt-4 border-t border-slate-800">
                <p className="text-xs text-slate-500 text-center">
                  🔒 Industrial-grade security with AES-256-GCM encryption
                </p>
              </div>
            </div>
          )}

          {/* Register Form */}
          {mode === 'register' && (
            <div className="bg-slate-900/80 dark:bg-slate-900/80 bg-white/50 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <h1 className="text-3xl font-bold text-white dark:text-white text-slate-900 mb-2">Create Account</h1>
              <p className="text-slate-400 dark:text-slate-500 mb-8">Join TigerWallet - The most secure Web3 wallet</p>

              <form onSubmit={handleRegister}>
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Email Address
                  </label>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500"
                    placeholder="you@example.com"
                    required
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Username
                  </label>
                  <input
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500"
                    placeholder="tiger_user"
                    required
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Password
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => handlePasswordChange(e.target.value)}
                      className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500 pr-12"
                      placeholder="••••••••••••"
                      required
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400"
                    >
                      {showPassword ? '🙈' : '👁️'}
                    </button>
                  </div>
                  {passwordErrors.length > 0 && (
                    <div className="mt-2 text-xs text-red-500">
                      {passwordErrors.map((err, i) => (
                        <div key={i}>• {err}</div>
                      ))}
                    </div>
                  )}
                </div>

                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Confirm Password
                  </label>
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500"
                    placeholder="••••••••••••"
                    required
                  />
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-600 rounded-lg text-white font-semibold transition-colors"
                >
                  {loading ? 'Creating Account...' : 'Create Account'}
                </button>
              </form>

              <div className="mt-6 text-center">
                <p className="text-slate-400 dark:text-slate-500 text-slate-600">
                  Already have an account?{' '}
                  <button
                    onClick={() => {
                      setMode('login');
                      setError(null);
                    }}
                    className="text-orange-500 hover:underline"
                  >
                    Sign In
                  </button>
                </p>
              </div>
            </div>
          )}

          {/* Verification Form */}
          {mode === 'verify' && (
            <div className="bg-slate-900/80 dark:bg-slate-900/80 bg-white/50 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <h1 className="text-3xl font-bold text-white dark:text-white text-slate-900 mb-2">Verify Email</h1>
              <p className="text-slate-400 dark:text-slate-500 mb-8">Enter the verification code sent to your email</p>

              <form onSubmit={handleVerify}>
                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Verification Code
                  </label>
                  <input
                    type="text"
                    value={verificationCode}
                    onChange={(e) => setVerificationCode(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500 text-center text-2xl tracking-widest"
                    placeholder="000000"
                    maxLength={6}
                    required
                  />
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-600 rounded-lg text-white font-semibold transition-colors"
                >
                  {loading ? 'Verifying...' : 'Verify'}
                </button>
              </form>
            </div>
          )}

          {/* 2FA Setup Form */}
          {mode === '2fa' && (
            <div className="bg-slate-900/80 dark:bg-slate-900/80 bg-white/50 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <h1 className="text-3xl font-bold text-white dark:text-white text-slate-900 mb-2">Enable 2FA</h1>
              <p className="text-slate-400 dark:text-slate-500 mb-8">Scan the QR code with your authenticator app</p>

              <div className="text-center mb-6">
                <img src={twoFactorQR} alt="2FA QR Code" className="mx-auto" />
              </div>

              <p className="text-sm text-slate-400 mb-4">
                Or enter this code manually: <code className="bg-slate-800 px-2 py-1 rounded">{twoFactorSecret}</code>
              </p>

              <form onSubmit={handleVerify2FA}>
                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-300 dark:text-slate-400 text-slate-700 mb-2">
                    Enter 2FA Code
                  </label>
                  <input
                    type="text"
                    value={code2FA}
                    onChange={(e) => setCode2FA(e.target.value)}
                    className="w-full px-4 py-3 bg-slate-800/50 dark:bg-slate-800/50 bg-white/50 border border-slate-700 rounded-lg text-white dark:text-white text-slate-900 focus:outline-none focus:border-orange-500 text-center text-2xl tracking-widest"
                    placeholder="000000"
                    maxLength={6}
                    required
                  />
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-600 rounded-lg text-white font-semibold transition-colors"
                >
                  {loading ? 'Verifying...' : 'Enable 2FA'}
                </button>
              </form>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}