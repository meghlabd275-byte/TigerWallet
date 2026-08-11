/**
 * TigerWallet - Authentication Service
 * Complete authentication with biometric support, 2FA, and secure session management
 */

import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';

// ============================================================================
// Types
// ============================================================================

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
  referralCode?: string;
}

export interface ResetPasswordRequest {
  email: string;
}

export interface VerifyEmailRequest {
  email: string;
  code: string;
}

export interface Enable2FARequest {
  method: 'totp' | 'sms' | 'email';
}

export interface Verify2FARequest {
  code: string;
  method: string;
}

export interface UpdatePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

export interface User {
  id: string;
  email: string;
  username: string;
  kycStatus: 'none' | 'pending' | 'verified' | 'rejected';
  twoFactorEnabled: boolean;
  createdAt: number;
  lastLogin: number;
}

export interface AuthResponse {
  token: string;
  refreshToken: string;
  expiresIn: number;
  user: User;
}

export interface BiometricAuthResult {
  success: boolean;
  publicKey?: string;
  credentialId?: string;
}

// ============================================================================
// API Client
// ============================================================================

const authClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
authClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('user_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor for token refresh
authClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      
      try {
        const refreshToken = localStorage.getItem('refresh_token');
        const response = await axios.post(`${API_BASE_URL}/auth/refresh`, {
          refreshToken,
        });
        
        const { token } = response.data;
        localStorage.setItem('user_token', token);
        
        originalRequest.headers.Authorization = `Bearer ${token}`;
        return authClient(originalRequest);
      } catch (refreshError) {
        // Refresh failed, logout user
        AuthService.logout();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }
    
    return Promise.reject(error);
  }
);

// ============================================================================
// Authentication Service
// ============================================================================

export class AuthService {
  /**
   * Register new user
   */
  static async register(request: RegisterRequest): Promise<AuthResponse> {
    try {
      const response = await authClient.post('/auth/register', request);
      const { token, refreshToken, user } = response.data;
      
      // Store tokens
      localStorage.setItem('user_token', token);
      localStorage.setItem('refresh_token', refreshToken);
      
      return { token, refreshToken, expiresIn: 86400, user };
    } catch (error: any) {
      console.error('Registration failed:', error);
      throw new Error(error.response?.data?.message || 'Registration failed');
    }
  }

  /**
   * Login with email/password
   */
  static async login(request: LoginRequest): Promise<AuthResponse> {
    try {
      const response = await authClient.post('/auth/login', request);
      const { token, refreshToken, user } = response.data;
      
      // Store tokens
      localStorage.setItem('user_token', token);
      localStorage.setItem('refresh_token', refreshToken);
      
      // Store user info
      localStorage.setItem('user_id', user.id);
      localStorage.setItem('user_email', user.email);
      
      return { token, refreshToken, expiresIn: 86400, user };
    } catch (error: any) {
      console.error('Login failed:', error);
      throw new Error(error.response?.data?.message || 'Login failed');
    }
  }

  /**
   * Login with biometric
   */
  static async loginWithBiometric(credentialId: string, signature: string): Promise<AuthResponse> {
    try {
      const response = await authClient.post('/auth/biometric/login', {
        credentialId,
        signature,
      });
      
      const { token, refreshToken, user } = response.data;
      
      localStorage.setItem('user_token', token);
      localStorage.setItem('refresh_token', refreshToken);
      localStorage.setItem('user_id', user.id);
      
      return { token, refreshToken, expiresIn: 86400, user };
    } catch (error: any) {
      console.error('Biometric login failed:', error);
      throw new Error('Biometric authentication failed');
    }
  }

  /**
   * Logout
   */
  static logout() {
    // Clear all auth data
    localStorage.removeItem('user_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user_id');
    localStorage.removeItem('user_email');
    localStorage.removeItem('biometric_enabled');
    
    // Redirect to login
    window.location.href = '/login';
  }

  /**
   * Check if user is authenticated
   */
  static isAuthenticated(): boolean {
    const token = localStorage.getItem('user_token');
    return !!token;
  }

  /**
   * Get current user
   */
  static async getCurrentUser(): Promise<User> {
    try {
      const response = await authClient.get('/auth/me');
      return response.data.user;
    } catch (error) {
      console.error('Failed to get current user:', error);
      throw error;
    }
  }

  /**
   * Request password reset
   */
  static async requestPasswordReset(request: ResetPasswordRequest): Promise<void> {
    try {
      await authClient.post('/auth/password/reset', request);
    } catch (error: any) {
      console.error('Password reset request failed:', error);
      throw new Error(error.response?.data?.message || 'Password reset request failed');
    }
  }

  /**
   * Reset password with token
   */
  static async resetPassword(token: string, newPassword: string): Promise<void> {
    try {
      await authClient.post('/auth/password/reset/confirm', {
        token,
        newPassword,
      });
    } catch (error: any) {
      console.error('Password reset failed:', error);
      throw new Error(error.response?.data?.message || 'Password reset failed');
    }
  }

  /**
   * Update password
   */
  static async updatePassword(request: UpdatePasswordRequest): Promise<void> {
    try {
      await authClient.post('/auth/password/update', request);
    } catch (error: any) {
      console.error('Password update failed:', error);
      throw new Error(error.response?.data?.message || 'Password update failed');
    }
  }

  /**
   * Verify email
   */
  static async verifyEmail(request: VerifyEmailRequest): Promise<void> {
    try {
      await authClient.post('/auth/email/verify', request);
    } catch (error: any) {
      console.error('Email verification failed:', error);
      throw new Error(error.response?.data?.message || 'Email verification failed');
    }
  }

  /**
   * Resend verification email
   */
  static async resendVerificationEmail(email: string): Promise<void> {
    try {
      await authClient.post('/auth/email/resend', { email });
    } catch (error: any) {
      console.error('Resend verification failed:', error);
      throw new Error(error.response?.data?.message || 'Failed to resend verification email');
    }
  }

  /**
   * Enable 2FA
   */
  static async enable2FA(request: Enable2FARequest): Promise<{ secret: string; qrCode: string }> {
    try {
      const response = await authClient.post('/auth/2fa/enable', request);
      return response.data;
    } catch (error: any) {
      console.error('Failed to enable 2FA:', error);
      throw new Error(error.response?.data?.message || 'Failed to enable 2FA');
    }
  }

  /**
   * Verify and activate 2FA
   */
  static async verify2FA(request: Verify2FARequest): Promise<void> {
    try {
      await authClient.post('/auth/2fa/verify', request);
    } catch (error: any) {
      console.error('2FA verification failed:', error);
      throw new Error(error.response?.data?.message || '2FA verification failed');
    }
  }

  /**
   * Disable 2FA
   */
  static async disable2FA(code: string): Promise<void> {
    try {
      await authClient.post('/auth/2fa/disable', { code });
    } catch (error: any) {
      console.error('Failed to disable 2FA:', error);
      throw new Error(error.response?.data?.message || 'Failed to disable 2FA');
    }
  }

  /**
   * Login with 2FA
   */
  static async loginWith2FA(code: string): Promise<AuthResponse> {
    try {
      const response = await authClient.post('/auth/2fa/login', { code });
      const { token, refreshToken, user } = response.data;
      
      localStorage.setItem('user_token', token);
      localStorage.setItem('refresh_token', refreshToken);
      
      return { token, refreshToken, expiresIn: 86400, user };
    } catch (error: any) {
      console.error('2FA login failed:', error);
      throw new Error(error.response?.data?.message || '2FA login failed');
    }
  }

  /**
   * Setup biometric authentication
   */
  static async setupBiometric(): Promise<BiometricAuthResult> {
    try {
      // Check if WebAuthn is supported
      if (!window.PublicKeyCredential) {
        throw new Error('Biometric authentication not supported');
      }

      // Create credential
      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: new Uint8Array(32),
          rp: { name: 'TigerWallet' },
          user: {
            id: new Uint8Array(16),
            name: localStorage.getItem('user_email') || 'user',
            displayName: localStorage.getItem('user_email') || 'User',
          },
          pubKeyCredParams: [
            { type: 'public-key', alg: -7 },
            { type: 'public-key', alg: -257 },
          ],
        },
      });

      if (!credential) {
        throw new Error('Failed to create biometric credential');
      }

      const publicKeyCredential = credential as PublicKeyCredential;
      const credentialId = btoa(String.fromCharCode(...new Uint8Array(publicKeyCredential.rawId)));

      // Store credential ID
      localStorage.setItem('biometric_enabled', 'true');
      localStorage.setItem('biometric_credential_id', credentialId);

      return {
        success: true,
        publicKey: '',
        credentialId,
      };
    } catch (error) {
      console.error('Biometric setup failed:', error);
      return { success: false };
    }
  }

  /**
   * Authenticate with biometric
   */
  static async authenticateWithBiometric(): Promise<BiometricAuthResult> {
    try {
      const credentialId = localStorage.getItem('biometric_credential_id');
      if (!credentialId) {
        throw new Error('No biometric credential found');
      }

      const assertion = await navigator.credentials.get({
        publicKey: {
          challenge: new Uint8Array(32),
          allowCredentials: [
            {
              id: Uint8Array.from(atob(credentialId), c => c.charCodeAt(0)),
              type: 'public-key',
            },
          ],
        },
      });

      if (!assertion) {
        throw new Error('Biometric authentication failed');
      }

      const publicKeyAssertion = assertion as PublicKeyCredential;
      const signature = btoa(String.fromCharCode(...new Uint8Array(publicKeyAssertion.response.signature)));

      // Verify with backend
      await this.loginWithBiometric(credentialId, signature);

      return {
        success: true,
        credentialId,
      };
    } catch (error) {
      console.error('Biometric authentication failed:', error);
      return { success: false };
    }
  }

  /**
   * Check if biometric is available
   */
  static isBiometricAvailable(): boolean {
    return !!(window as any).PublicKeyCredential;
  }

  /**
   * Check if biometric is enabled
   */
  static isBiometricEnabled(): boolean {
    return localStorage.getItem('biometric_enabled') === 'true';
  }
}

// Export default
export default AuthService;
