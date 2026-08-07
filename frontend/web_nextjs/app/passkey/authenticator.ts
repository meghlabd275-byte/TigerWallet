/**
 * TigerWallet Passkey/WebAuthn Authentication
 * Real biometric authentication implementation using WebAuthn API
 * Supports fingerprint, face recognition, and hardware security keys
 */

import { BrowserAdapter } from './browser_adapter';

// ============================================================================
// Types
// ============================================================================

export interface PasskeyCredential {
  id: string;
  rawId: string;
  type: 'public-key';
  response: AuthenticatorAttestationResponse | AuthenticatorAssertionResponse;
  clientDataJSON?: string;
}

export interface PasskeyRegistrationOptions {
  rp: {
    name: string;
    id: string;
    icon?: string;
  };
  user: {
    id: string;
    name: string;
    displayName: string;
  };
  challenge: string;
  pubKeyCredParams: PublicKeyCredentialParams[];
  timeout?: number;
  excludeCredentials?: PublicKeyCredentialDescriptor[];
  authenticatorSelection?: AuthenticatorSelectionCriteria;
  attestation?: AttestationConveyancePreference;
}

export interface PasskeyAuthenticationOptions {
  challenge: string;
  timeout?: number;
  rpId?: string;
  allowCredentials?: PublicKeyCredentialDescriptor[];
  userVerification?: UserVerificationRequirement;
}

export interface PasskeyUser {
  id: string;
  username: string;
  displayName: string;
  credentials: string[];
  createdAt: number;
}

export interface AuthResult {
  success: boolean;
  user?: PasskeyUser;
  error?: string;
  credentialId?: string;
}

// ============================================================================
// Passkey Authenticator - Real WebAuthn Implementation
// ============================================================================

export class PasskeyAuthenticator {
  private rpId: string;
  private rpName: string;
  private browserAdapter: BrowserAdapter;

  constructor(rpId?: string, rpName: string = 'TigerWallet') {
    this.rpId = rpId ?? (typeof window !== 'undefined' ? window.location.hostname : 'localhost');
    this.rpName = rpName;
    this.browserAdapter = new BrowserAdapter();
  }

  /**
   * Check if WebAuthn is supported
   */
  isSupported(): boolean {
    return this.browserAdapter.isWebAuthnSupported();
  }

  /**
   * Check if platform authenticator is available (biometrics)
   */
  async isPlatformAuthenticatorAvailable(): Promise<boolean> {
    if (!this.isSupported()) return false;
    
    try {
      const available = await navigator.credentials?.create({
        publicKey: {
          challenge: new Uint8Array(32),
          rp: { id: this.rpId, name: this.rpName },
          user: { id: new Uint8Array(16), name: 'test', displayName: 'Test' },
          pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
        },
      } as any);
      return !!available;
    } catch {
      return false;
    }
  }

  /**
   * Generate random challenge for WebAuthn
   */
  private generateChallenge(length: number = 32): string {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return this.bufferToBase64Url(array);
  }

  /**
   * Convert ArrayBuffer to Base64URL
   */
  private bufferToBase64Url(buffer: ArrayBuffer | Uint8Array): string {
    const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
    let binary = '';
    bytes.forEach(b => binary += String.fromCharCode(b));
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  /**
   * Convert Base64URL to ArrayBuffer
   */
  private base64UrlToBuffer(base64url: string): ArrayBuffer {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const padding = base64.length % 4;
    const paddedBase64 = padding ? base64 + '='.repeat(4 - padding) : base64;
    
    const binary = atob(paddedBase64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }

  /**
   * Register a new passkey credential
   */
  async register(
    username: string,
    displayName: string,
    userId?: string
  ): Promise<AuthResult> {
    if (!this.isSupported()) {
      return { success: false, error: 'WebAuthn is not supported in this browser' };
    }

    try {
      const challenge = this.generateChallenge();
      const userBuffer = userId ? this.base64UrlToBuffer(userId) : crypto.getRandomValues(new Uint8Array(16));
      const userIdBase64 = this.bufferToBase64Url(userBuffer);

      const options: PublicKeyCredentialCreationOptions = {
        rp: {
          id: this.rpId,
          name: this.rpName,
        },
        user: {
          id: userBuffer,
          name: username,
          displayName: displayName,
        },
        challenge: this.base64UrlToBuffer(challenge),
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },  // ES256
          { type: 'public-key', alg: -257 }, // RS256
        ],
        timeout: 60000,
        authenticatorSelection: {
          authenticatorAttachment: 'platform',
          userVerification: 'preferred',
          requireResidentKey: true,
        },
        attestation: 'direct',
      };

      const credential = await navigator.credentials?.create({
        publicKey: options,
      } as any) as PublicKeyCredential | null;

      if (!credential) {
        return { success: false, error: 'Registration was cancelled' };
      }

      // Store credential info
      const credentialId = this.bufferToBase64Url(credential.rawId);
      
      // Save to localStorage (in production, save to backend)
      this.saveCredential({
        id: credentialId,
        username,
        userId: userIdBase64,
        createdAt: Date.now(),
      });

      return {
        success: true,
        credentialId,
        user: {
          id: userIdBase64,
          username,
          displayName,
          credentials: [credentialId],
          createdAt: Date.now(),
        },
      };
    } catch (error: any) {
      console.error('Passkey registration error:', error);
      return { success: false, error: error.message || 'Registration failed' };
    }
  }

  /**
   * Authenticate with existing passkey
   */
  async authenticate(userId?: string): Promise<AuthResult> {
    if (!this.isSupported()) {
      return { success: false, error: 'WebAuthn is not supported in this browser' };
    }

    try {
      const challenge = this.generateChallenge();
      
      // Get stored credentials
      const credentials = this.getStoredCredentials();
      const allowCredentials: PublicKeyCredentialDescriptor[] = credentials
        .filter(c => !userId || c.userId === userId)
        .map(c => ({
          type: 'public-key',
          id: this.base64UrlToBuffer(c.id),
        }));

      const options: PublicKeyCredentialRequestOptions = {
        challenge: this.base64UrlToBuffer(challenge),
        timeout: 60000,
        rpId: this.rpId,
        allowCredentials: allowCredentials.length > 0 ? allowCredentials : undefined,
        userVerification: 'preferred',
      };

      const credential = await navigator.credentials?.get({
        publicKey: options,
      } as any) as PublicKeyCredential | null;

      if (!credential) {
        return { success: false, error: 'Authentication was cancelled' };
      }

      const credentialId = this.bufferToBase64Url(credential.rawId);
      const user = this.getUserByCredentialId(credentialId);

      return {
        success: true,
        credentialId,
        user: user || undefined,
      };
    } catch (error: any) {
      console.error('Passkey authentication error:', error);
      return { success: false, error: error.message || 'Authentication failed' };
    }
  }

  /**
   * Delete a passkey credential
   */
  async deleteCredential(credentialId: string): Promise<boolean> {
    try {
      const credentials = this.getStoredCredentials();
      const filtered = credentials.filter(c => c.id !== credentialId);
      localStorage.setItem('tigerwallet_passkeys', JSON.stringify(filtered));
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Get all stored credentials
   */
  getStoredCredentials(): Array<{ id: string; username: string; userId: string; createdAt: number }> {
    try {
      const stored = localStorage.getItem('tigerwallet_passkeys');
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  }

  /**
   * Save credential to storage
   */
  private saveCredential(credential: { id: string; username: string; userId: string; createdAt: number }): void {
    const credentials = this.getStoredCredentials();
    
    // Check if credential already exists
    const exists = credentials.some(c => c.id === credential.id);
    if (!exists) {
      credentials.push(credential);
      localStorage.setItem('tigerwallet_passkeys', JSON.stringify(credentials));
    }
  }

  /**
   * Get user by credential ID
   */
  private getUserByCredentialId(credentialId: string): PasskeyUser | undefined {
    const credentials = this.getStoredCredentials();
    const cred = credentials.find(c => c.id === credentialId);
    
    if (cred) {
      return {
        id: cred.userId,
        username: cred.username,
        displayName: cred.username,
        credentials: [cred.id],
        createdAt: cred.createdAt,
      };
    }
    return undefined;
  }

  /**
   * Get all registered users
   */
  getUsers(): PasskeyUser[] {
    const credentials = this.getStoredCredentials();
    const usersMap = new Map<string, PasskeyUser>();
    
    for (const cred of credentials) {
      if (!usersMap.has(cred.userId)) {
        usersMap.set(cred.userId, {
          id: cred.userId,
          username: cred.username,
          displayName: cred.username,
          credentials: [],
          createdAt: cred.createdAt,
        });
      }
      const user = usersMap.get(cred.userId)!;
      user.credentials.push(cred.id);
    }
    
    return Array.from(usersMap.values());
  }
}

// ============================================================================
// Browser Adapter for WebAuthn Support Detection
// ============================================================================

class BrowserAdapter {
  isWebAuthnSupported(): boolean {
    return !!(
      navigator.credentials?.create &&
      navigator.credentials?.get
    );
  }

  isSecureContext(): boolean {
    return window.isSecureContext;
  }

  getBrowserName(): string {
    const ua = navigator.userAgent;
    if (ua.includes('Chrome')) return 'Chrome';
    if (ua.includes('Firefox')) return 'Firefox';
    if (ua.includes('Safari')) return 'Safari';
    if (ua.includes('Edge')) return 'Edge';
    return 'Unknown';
  }
}

// ============================================================================
// Default Export
// ============================================================================

export const passkeyAuthenticator = new PasskeyAuthenticator();
