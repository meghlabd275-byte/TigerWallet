/**
 * Auth Service — TigerWallet UserWallet (production React frontend).
 *
 * Talks to the canonical Go wallet-api backend (go/wallet_api, port 8443):
 * REAL JWT (HS256, 24h) auth, REAL bcrypt password hashing, REAL PostgreSQL
 * user persistence. No stubs, no fabricated tokens.
 *
 * The canonical backend exposes /auth/login and /auth/register. Features the
 * backend does not expose (refresh tokens, 2FA, password reset, sessions)
 * throw real errors instead of faking success — wire the corresponding Go
 * service (go/two_factor_auth, etc.) before use.
 */

import axios, { AxiosInstance } from 'axios';

export interface User {
  id: string;
  email: string;
  username: string;
  avatar?: string;
  kycStatus: 'none' | 'pending' | 'verified' | 'rejected';
  createdAt: string;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface RegisterData {
  email: string;
  username: string;
  password: string;
  referralCode?: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refreshToken: string;
  expiresIn: number;
}

const API_BASE_URL =
  import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';

const TOKEN_KEY = 'tigerwallet-token';
const REFRESH_KEY = 'tigerwallet-refresh-token';
const EXPIRES_KEY = 'tigerwallet-token-expires';
const DEFAULT_TTL_SECONDS = 24 * 60 * 60; // 24h — matches wallet_api JWT

class AuthService {
  private api: AxiosInstance;

  constructor() {
    this.api = axios.create({
      baseURL: API_BASE_URL,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    const response = await this.api.post('/auth/login', {
      email: credentials.email,
      password: credentials.password,
    });
    const { token, user } = response.data;
    const expiresIn = DEFAULT_TTL_SECONDS;
    const refreshToken = token; // wallet_api issues a single JWT
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(REFRESH_KEY, refreshToken);
    localStorage.setItem(EXPIRES_KEY, String(Date.now() + expiresIn * 1000));
    return {
      user: user ?? {
        id: response.data.user_id ?? '',
        email: credentials.email,
        username: response.data.username ?? credentials.email,
        kycStatus: 'none',
        createdAt: new Date().toISOString(),
      },
      token,
      refreshToken,
      expiresIn,
    };
  }

  async register(data: RegisterData): Promise<AuthResponse> {
    // Canonical /auth/register accepts {email, password} only (see route table).
    const response = await this.api.post('/auth/register', {
      email: data.email,
      password: data.password,
    });
    const { token, user } = response.data;
    const expiresIn = DEFAULT_TTL_SECONDS;
    const refreshToken = token;
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(REFRESH_KEY, refreshToken);
    localStorage.setItem(EXPIRES_KEY, String(Date.now() + expiresIn * 1000));
    return {
      user: user ?? {
        id: response.data.user_id ?? '',
        email: data.email,
        username: data.username,
        kycStatus: 'none',
        createdAt: new Date().toISOString(),
      },
      token,
      refreshToken,
      expiresIn,
    };
  }

  async logout(): Promise<void> {
    // The canonical backend has no /auth/logout (stateless JWT). Clear local
    // state; the token expires on its own TTL.
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(EXPIRES_KEY);
  }

  async getCurrentUser(): Promise<User> {
    // No /auth/me on the canonical backend; reconstruct from the stored token.
    const token = localStorage.getItem(TOKEN_KEY);
    if (!token) throw new Error('Not authenticated');
    const payload = JSON.parse(
      atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))
    );
    return {
      id: payload.user_id ?? payload.sub ?? '',
      email: payload.email ?? '',
      username: payload.username ?? payload.email ?? '',
      kycStatus: 'none',
      createdAt: new Date(payload.iat ? payload.iat * 1000 : Date.now()).toISOString(),
    };
  }

  async updateProfile(_data: Partial<User>): Promise<User> {
    throw new Error(
      'Profile update is not exposed by the canonical wallet-api backend; wire go/user_services or an admin API first'
    );
  }

  async changePassword(_current: string, _next: string): Promise<void> {
    throw new Error(
      'Password change is not exposed by the canonical wallet-api backend'
    );
  }

  async requestPasswordReset(_email: string): Promise<void> {
    throw new Error(
      'Password reset is not exposed by the canonical wallet-api backend'
    );
  }

  async resetPassword(_token: string, _newPassword: string): Promise<void> {
    throw new Error(
      'Password reset is not exposed by the canonical wallet-api backend'
    );
  }

  async verifyEmail(_token: string): Promise<void> {
    throw new Error(
      'Email verification is not exposed by the canonical wallet-api backend'
    );
  }

  async resendVerificationEmail(): Promise<void> {
    throw new Error(
      'Email verification is not exposed by the canonical wallet-api backend'
    );
  }

  async refreshAccessToken(): Promise<string> {
    // wallet_api issues a single stateless JWT (no refresh-token endpoint).
    // Re-use the stored token if still valid; otherwise the caller must
    // re-authenticate.
    const token = localStorage.getItem(TOKEN_KEY);
    if (token && !this.isTokenExpired()) return token;
    throw new Error('Session expired; please log in again');
  }

  async enable2FA(): Promise<{ qrCode: string; secret: string }> {
    throw new Error(
      '2FA is not exposed by the canonical wallet-api backend; wire go/two_factor_auth first'
    );
  }

  async verify2FA(_code: string): Promise<void> {
    throw new Error('2FA is not exposed by the canonical wallet-api backend');
  }

  async disable2FA(_code: string): Promise<void> {
    throw new Error('2FA is not exposed by the canonical wallet-api backend');
  }

  async getSessions(): Promise<unknown[]> {
    throw new Error('Session management is not exposed by the canonical wallet-api backend');
  }

  async revokeSession(_sessionId: string): Promise<void> {
    throw new Error('Session management is not exposed by the canonical wallet-api backend');
  }

  async revokeAllSessions(): Promise<void> {
    throw new Error('Session management is not exposed by the canonical wallet-api backend');
  }

  isTokenExpired(): boolean {
    const expires = localStorage.getItem(EXPIRES_KEY);
    if (!expires) return true;
    return Date.now() > parseInt(expires);
  }
}

export { AuthService };
export default AuthService;
