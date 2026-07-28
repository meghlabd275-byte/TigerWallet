/**
 * Auth Service - User Authentication
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

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

class AuthService {
  private api: AxiosInstance;

  constructor() {
    this.api = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    const response = await this.api.post('/auth/login', credentials);
    
    const { user, token, refreshToken, expiresIn } = response.data;
    
    // Store tokens
    localStorage.setItem('tigerwallet-token', token);
    localStorage.setItem('tigerwallet-refresh-token', refreshToken);
    localStorage.setItem('tigerwallet-token-expires', String(Date.now() + expiresIn * 1000));
    
    return { user, token, refreshToken, expiresIn };
  }

  async register(data: RegisterData): Promise<AuthResponse> {
    const response = await this.api.post('/auth/register', data);
    
    const { user, token, refreshToken, expiresIn } = response.data;
    
    localStorage.setItem('tigerwallet-token', token);
    localStorage.setItem('tigerwallet-refresh-token', refreshToken);
    localStorage.setItem('tigerwallet-token-expires', String(Date.now() + expiresIn * 1000));
    
    return { user, token, refreshToken, expiresIn };
  }

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem('tigerwallet-refresh-token');
    
    if (refreshToken) {
      try {
        await this.api.post('/auth/logout', { refreshToken });
      } catch (err) {
        // Ignore logout errors
      }
    }
    
    localStorage.removeItem('tigerwallet-token');
    localStorage.removeItem('tigerwallet-refresh-token');
    localStorage.removeItem('tigerwallet-token-expires');
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.api.get('/auth/me');
    return response.data.user;
  }

  async updateProfile(data: Partial<User>): Promise<User> {
    const response = await this.api.patch('/auth/profile', data);
    return response.data.user;
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await this.api.post('/auth/change-password', {
      currentPassword,
      newPassword,
    });
  }

  async requestPasswordReset(email: string): Promise<void> {
    await this.api.post('/auth/forgot-password', { email });
  }

  async resetPassword(token: string, newPassword: string): Promise<void> {
    await this.api.post('/auth/reset-password', { token, newPassword });
  }

  async verifyEmail(token: string): Promise<void> {
    await this.api.post('/auth/verify-email', { token });
  }

  async resendVerificationEmail(): Promise<void> {
    await this.api.post('/auth/resend-verification');
  }

  async refreshAccessToken(): Promise<string> {
    const refreshToken = localStorage.getItem('tigerwallet-refresh-token');
    
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }
    
    const response = await this.api.post('/auth/refresh', { refreshToken });
    
    const { token, refreshToken: newRefreshToken, expiresIn } = response.data;
    
    localStorage.setItem('tigerwallet-token', token);
    localStorage.setItem('tigerwallet-refresh-token', newRefreshToken);
    localStorage.setItem('tigerwallet-token-expires', String(Date.now() + expiresIn * 1000));
    
    return token;
  }

  async enable2FA(): Promise<{ qrCode: string; secret: string }> {
    const response = await this.api.post('/auth/2fa/enable');
    return response.data;
  }

  async verify2FA(code: string): Promise<void> {
    await this.api.post('/auth/2fa/verify', { code });
  }

  async disable2FA(code: string): Promise<void> {
    await this.api.post('/auth/2fa/disable', { code });
  }

  async getSessions(): Promise<any[]> {
    const response = await this.api.get('/auth/sessions');
    return response.data.sessions;
  }

  async revokeSession(sessionId: string): Promise<void> {
    await this.api.delete(`/auth/sessions/${sessionId}`);
  }

  async revokeAllSessions(): Promise<void> {
    await this.api.delete('/auth/sessions');
  }

  isTokenExpired(): boolean {
    const expires = localStorage.getItem('tigerwallet-token-expires');
    if (!expires) return true;
    return Date.now() > parseInt(expires);
  }
}

export default AuthService;
